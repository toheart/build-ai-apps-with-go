package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatCompletionRequest struct {
	Model      string    `json:"model"`
	Messages   []Message `json:"messages"`
	Tools      []Tool    `json:"tools,omitempty"`
	ToolChoice string    `json:"tool_choice,omitempty"`
}

type ChatCompletionResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	Role      string     `json:"role"`
	Content   *string    `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type Config struct {
	APIKey       string
	BaseURL      string
	Model        string
	MaxToolSteps int
}

func main() {
	loadDotEnv(".env")

	config, err := readConfig()
	if err != nil {
		fmt.Printf("配置错误：%v\n", err)
		os.Exit(1)
	}

	tools := buildTools()
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个简洁、可靠的中文助理。当问题需要读取当前时间或做加法计算时，优先调用合适工具，不要自己编造结果。拿到工具结果后，再用简短自然语言回答用户。",
		},
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("最小 Agent 已启动，输入 exit、quit 或 退出 可结束对话。")
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

		messages = append(messages, Message{Role: "user", Content: input})
		fmt.Println()

		answer, updatedMessages, err := runAgentTurn(config, messages, tools)
		if err != nil {
			fmt.Printf("请求失败：%v\n\n", err)
			messages = messages[:len(messages)-1]
			continue
		}

		messages = updatedMessages
		fmt.Printf("助手：%s\n\n", answer)
	}
}

func readConfig() (Config, error) {
	config := Config{
		APIKey:       strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		BaseURL:      strings.TrimSpace(os.Getenv("OPENAI_API_BASE")),
		Model:        strings.TrimSpace(os.Getenv("OPENAI_MODEL")),
		MaxToolSteps: 3,
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

	config.BaseURL = strings.TrimRight(config.BaseURL, "/")
	return config, nil
}

func buildTools() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "get_current_time",
				Description: "获取当前系统时间，适合回答现在几点、当前时间之类的问题",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "add_numbers",
				Description: "计算两个数字相加的结果，适合处理简单加法",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"a": map[string]any{"type": "number", "description": "第一个加数"},
						"b": map[string]any{"type": "number", "description": "第二个加数"},
					},
					"required": []string{"a", "b"},
				},
			},
		},
	}
}

func runAgentTurn(config Config, messages []Message, tools []Tool) (string, []Message, error) {
	workingMessages := append([]Message(nil), messages...)

	for step := 0; step < config.MaxToolSteps; step++ {
		response, err := askModel(config, workingMessages, tools)
		if err != nil {
			return "", messages, err
		}

		assistantMessage := Message{Role: response.Role}
		if response.Content != nil {
			assistantMessage.Content = strings.TrimSpace(*response.Content)
		}
		if len(response.ToolCalls) > 0 {
			assistantMessage.ToolCalls = response.ToolCalls
		}

		workingMessages = append(workingMessages, assistantMessage)

		if len(response.ToolCalls) == 0 {
			answer := strings.TrimSpace(assistantMessage.Content)
			if answer == "" {
				return "", messages, errors.New("模型返回了空内容")
			}
			return answer, workingMessages, nil
		}

		for _, toolCall := range response.ToolCalls {
			fmt.Printf("[tool_call] %s(%s)\n", toolCall.Function.Name, compactJSON(toolCall.Function.Arguments))

			result, err := executeToolCall(toolCall)
			if err != nil {
				result = fmt.Sprintf(`{"error":%q}`, err.Error())
			}

			fmt.Printf("[tool_result] %s\n", result)
			workingMessages = append(workingMessages, Message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    result,
			})
		}

		fmt.Println()
	}

	return "", messages, fmt.Errorf("工具调用超过最大轮数 %d", config.MaxToolSteps)
}

func askModel(config Config, messages []Message, tools []Tool) (ResponseMessage, error) {
	requestBody := ChatCompletionRequest{
		Model:      config.Model,
		Messages:   messages,
		Tools:      tools,
		ToolChoice: "auto",
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return ResponseMessage{}, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := config.BaseURL + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return ResponseMessage{}, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ResponseMessage{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ResponseMessage{}, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return ResponseMessage{}, fmt.Errorf("model API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return ResponseMessage{}, fmt.Errorf("unmarshal response: %w, body=%s", err, strings.TrimSpace(string(body)))
	}

	if len(response.Choices) == 0 {
		return ResponseMessage{}, errors.New("response.choices 为空")
	}

	return response.Choices[0].Message, nil
}

func executeToolCall(toolCall ToolCall) (string, error) {
	var args map[string]any
	if strings.TrimSpace(toolCall.Function.Arguments) != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("parse tool arguments: %w", err)
		}
	}

	switch toolCall.Function.Name {
	case "get_current_time":
		result := map[string]string{
			"current_time": time.Now().Format("2006-01-02 15:04:05 -07:00"),
		}
		return mustJSON(result), nil
	case "add_numbers":
		a, err := readNumber(args, "a")
		if err != nil {
			return "", err
		}
		b, err := readNumber(args, "b")
		if err != nil {
			return "", err
		}
		result := map[string]any{
			"a":      a,
			"b":      b,
			"result": a + b,
		}
		return mustJSON(result), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}
}

func readNumber(args map[string]any, key string) (float64, error) {
	value, ok := args[key]
	if !ok {
		return 0, fmt.Errorf("missing argument: %s", key)
	}

	switch number := value.(type) {
	case float64:
		return number, nil
	case string:
		parsed, err := strconv.ParseFloat(number, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %s: %w", key, err)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("invalid argument type for %s", key)
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `{"error":"marshal tool result failed"}`
	}
	return string(data)
}

func compactJSON(value string) string {
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return value
	}
	data, err := json.Marshal(decoded)
	if err != nil {
		return value
	}
	return string(data)
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
