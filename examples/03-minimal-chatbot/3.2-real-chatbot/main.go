package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type ChatCompletionStreamResponse struct {
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Delta        DeltaMessage `json:"delta"`
	FinishReason string       `json:"finish_reason"`
}

type DeltaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const (
	modeSync   = "sync"
	modeStream = "stream"
)

type Config struct {
	APIKey       string
	BaseURL      string
	Model        string
	ResponseMode string
}

func main() {
	loadDotEnv(".env")

	modeFlag := flag.String("mode", "", "response mode: sync or stream")
	flag.Parse()

	config, err := readConfig(strings.TrimSpace(*modeFlag))
	if err != nil {
		fmt.Printf("配置错误：%v\n", err)
		os.Exit(1)
	}

	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个简洁、友好的中文助理。回答前请先结合整段消息历史，尤其要承接用户前文已经说过的事实。回复尽量短，不要无关扩展。",
		},
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("最小 Chatbot 已启动，输入 exit、quit 或 退出 可结束对话。")
	fmt.Println()

	for {
		input, err := readInput(reader)
		if err != nil {
			fmt.Printf("读取输入失败：%v\n", err)
			os.Exit(1)
		}

		if shouldExit(input) {
			fmt.Println("对话结束，再见。")
			return
		}

		messages = append(messages, Message{
			Role:    "user",
			Content: input,
		})

		fmt.Println()
		fmt.Print("助手：")

		answer, err := askModel(config, messages)
		if err != nil {
			fmt.Printf("\n请求失败：%v\n\n", err)
			messages = messages[:len(messages)-1]
			continue
		}

		if config.ResponseMode == modeSync {
			fmt.Print(answer)
		}

		fmt.Println()
		fmt.Println()

		messages = append(messages, Message{
			Role:    "assistant",
			Content: answer,
		})
	}
}

func readConfig(modeOverride string) (Config, error) {
	config := Config{
		APIKey:  strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		BaseURL: strings.TrimSpace(os.Getenv("OPENAI_API_BASE")),
		Model:   strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
	}

	if config.APIKey == "" {
		return Config{}, errors.New("缺少 OPENAI_API_KEY")
	}
	if config.BaseURL == "" {
		return Config{}, errors.New("缺少 OPENAI_API_BASE")
	}
	if config.Model == "" {
		return Config{}, errors.New("缺少 OPENAI_MODEL")
	}

	config.ResponseMode = pickResponseMode(modeOverride, strings.TrimSpace(os.Getenv("OPENAI_RESPONSE_MODE")))
	if config.ResponseMode != modeSync && config.ResponseMode != modeStream {
		return Config{}, fmt.Errorf("不支持的响应模式 %q，可选值为 sync 或 stream", config.ResponseMode)
	}

	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return config, nil
}

func pickResponseMode(modeOverride string, envMode string) string {
	if modeOverride != "" {
		return strings.ToLower(modeOverride)
	}

	if envMode != "" {
		return strings.ToLower(envMode)
	}

	return modeStream
}

func readInput(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("你：")
		input, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}

		input = strings.TrimSpace(input)
		if input != "" {
			return input, nil
		}

		if errors.Is(err, io.EOF) {
			return "exit", nil
		}
	}
}

func shouldExit(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "exit", "quit", "退出":
		return true
	default:
		return false
	}
}

func askModel(config Config, messages []Message) (string, error) {
	if config.ResponseMode == modeSync {
		return syncModel(config, messages)
	}

	return streamModel(config, messages)
}

func syncModel(config Config, messages []Message) (string, error) {
	resp, err := doChatRequest(config, messages, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("model API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("unmarshal response: %w, body=%s", err, strings.TrimSpace(string(body)))
	}

	if len(response.Choices) == 0 {
		return "", errors.New("response.choices 为空")
	}

	answer := strings.TrimSpace(response.Choices[0].Message.Content)
	if answer == "" {
		return "", errors.New("模型返回了空内容")
	}

	return answer, nil
}

func streamModel(config Config, messages []Message) (string, error) {
	resp, err := doChatRequest(config, messages, true)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("read error response body: %w", readErr)
		}
		return "", fmt.Errorf("model API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	reader := bufio.NewReader(resp.Body)
	var answer strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read stream chunk: %w", err)
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			chunk := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if chunk == "[DONE]" {
				break
			}

			var streamResponse ChatCompletionStreamResponse
			if err := json.Unmarshal([]byte(chunk), &streamResponse); err != nil {
				return "", fmt.Errorf("unmarshal stream chunk: %w, chunk=%s", err, chunk)
			}

			if len(streamResponse.Choices) > 0 {
				content := streamResponse.Choices[0].Delta.Content
				if content != "" {
					fmt.Print(content)
					answer.WriteString(content)
				}
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	answerText := strings.TrimSpace(answer.String())
	if answerText == "" {
		return "", errors.New("模型返回了空内容")
	}

	return answerText, nil
}

func doChatRequest(config Config, messages []Message, stream bool) (*http.Response, error) {
	requestBody := ChatCompletionRequest{
		Model:    config.Model,
		Messages: messages,
		Stream:   stream,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := config.BaseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func loadDotEnv(filename string) {
	path := filepath.Clean(filename)
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"`)

		if key == "" {
			continue
		}

		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}
