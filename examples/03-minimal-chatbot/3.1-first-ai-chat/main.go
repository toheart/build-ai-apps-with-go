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

func main() {
	loadDotEnv(".env")

	modeFlag := flag.String("mode", "", "response mode: sync or stream")
	flag.Parse()

	config, err := readConfig(strings.TrimSpace(*modeFlag))
	if err != nil {
		fmt.Printf("配置错误：%v\n", err)
		os.Exit(1)
	}

	question, err := readQuestion()
	if err != nil {
		fmt.Printf("读取输入失败：%v\n", err)
		os.Exit(1)
	}

	// 这份示例同时保留了 sync 和 stream 两种实现。
	// sync 模式更容易帮助初学者理解“一次请求 -> 一次完整响应”的基本结构。
	// stream 模式更贴近真实聊天产品的交互体验，用户可以更早看到模型逐步生成的内容。
	// 运行时可以通过 `-mode sync` 或 `-mode stream` 自行选择；默认使用 stream。
	fmt.Println()
	fmt.Printf("模型回复（%s）：\n", config.ResponseMode)
	answer, err := askModel(config, question)
	if err != nil {
		fmt.Printf("请求失败：%v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(answer) == "" {
		fmt.Printf("请求失败：%v\n", errors.New("模型返回了空内容"))
		os.Exit(1)
	}

	fmt.Println()
	if config.ResponseMode == modeSync {
		fmt.Println(answer)
	}
	fmt.Println()
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

func readQuestion() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入你的问题：")
	question, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	question = strings.TrimSpace(question)
	if question == "" {
		return "", errors.New("问题不能为空")
	}

	return question, nil
}

func askModel(config Config, question string) (string, error) {
	if config.ResponseMode == modeSync {
		return syncModel(config, question)
	}

	return streamModel(config, question)
}

func syncModel(config Config, question string) (string, error) {
	resp, err := doChatRequest(config, question, false)
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

func streamModel(config Config, question string) (string, error) {
	resp, err := doChatRequest(config, question, true)
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

	return strings.TrimSpace(answer.String()), nil
}

func doChatRequest(config Config, question string, stream bool) (*http.Response, error) {
	requestBody := ChatCompletionRequest{
		Model: config.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: question,
			},
		},
		Stream: stream,
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
