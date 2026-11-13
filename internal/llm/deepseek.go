package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DeepSeekClient 通过 OpenAI 兼容接口调用 DeepSeek
type DeepSeekClient struct {
	APIKey  string
	BaseURL string // e.g. https://api.deepseek.com
	Model   string // e.g. deepseek-chat
	HTTP    *http.Client
}

func NewDeepSeekClient(apiKey, baseURL, model string) *DeepSeekClient {
	return &DeepSeekClient{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{},
	}
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []apiMessage     `json:"messages"`
	Tools       []toolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type toolDefinition struct {
	Type     string         `json:"type"`
	Function functionSchema `json:"function"`
}

type functionSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content      string          `json:"content"`
			ToolCalls    []toolCallItem  `json:"tool_calls"`
			FunctionCall *functionCall   `json:"function_call,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type toolCallItem struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Chat 调用 DeepSeek 的 chat completions 接口
func (c *DeepSeekClient) Chat(messages []Message, tools []ToolDefinition) (string, error) {
	apiMsgs := make([]apiMessage, 0, len(messages))
	for _, m := range messages {
		apiMsgs = append(apiMsgs, apiMessage{Role: m.Role, Content: m.Content})
	}

	req := chatRequest{
		Model:       c.Model,
		Messages:    apiMsgs,
		Temperature: 0.7,
	}

	if len(tools) > 0 {
		req.Tools = make([]toolDefinition, 0, len(tools))
		for _, t := range tools {
			req.Tools = append(req.Tools, toolDefinition{
				Type: "function",
				Function: functionSchema{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, string(data))
	}

	var result chatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("LLM 错误: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空响应")
	}

	msg := result.Choices[0].Message
	content := msg.Content

	// 优先处理 tool_calls 数组（OpenAI 新版格式）
	if len(msg.ToolCalls) > 0 {
		var sb strings.Builder
		sb.WriteString(content)
		if content != "" {
			sb.WriteString("\n")
		}
		for _, tc := range msg.ToolCalls {
			sb.WriteString(fmt.Sprintf("[tool_call] name=%s args=%s\n", tc.Function.Name, tc.Function.Arguments))
		}
		return sb.String(), nil
	}

	// 兼容旧的 function_call 格式
	if content == "" && msg.FunctionCall != nil {
		fc := msg.FunctionCall
		content = fmt.Sprintf("[tool_call] name=%s args=%s", fc.Name, fc.Arguments)
	}
	return content, nil
}
