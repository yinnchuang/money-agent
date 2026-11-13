package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"money-agent/internal/config"
	"money-agent/internal/llm"
	"money-agent/internal/storage"
	"money-agent/internal/tools"
)

// Agent 对话代理，组合 LLM client + 工具 + 记忆
type Agent struct {
	cfg      *config.Config
	llm      llm.Client
	registry *tools.Registry
	history  []llm.Message
}

func New(cfg *config.Config, llmClient llm.Client, registry *tools.Registry) *Agent {
	return &Agent{
		cfg:      cfg,
		llm:      llmClient,
		registry: registry,
		history:  make([]llm.Message, 0),
	}
}

// Chat 单轮对话，内部会自动读取 soul + portfolio 拼入系统提示
func (a *Agent) Chat(ctx context.Context, userInput string) (string, error) {
	messages, err := a.buildMessages(ctx)
	if err != nil {
		return "", err
	}
	messages = append(messages, llm.Message{Role: "user", Content: userInput})

	// 多轮 tool-call 循环（最多 5 轮，防止 LLM 死循环）
	for i := 0; i < 5; i++ {
		reply, err := a.llm.Chat(messages, a.registry.All())
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		// 检查是否需要调工具：DeepSeek/OpenAI 兼容的函数调用会被我们在 deepseek.go 里转成 [tool_call] 前缀
		if toolName, toolArgs, ok := parseToolCall(reply); ok {
			// 把 LLM 的工具调用请求记进历史
			messages = append(messages, llm.Message{Role: "assistant", Content: reply})

			toolResult, err := a.registry.RunByName(ctx, toolName, toolArgs)
			if err != nil {
				toolResult = fmt.Sprintf("工具 %s 执行失败: %v", toolName, err)
			}
			// 把工具结果喂回去
			messages = append(messages, llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("工具 %s 返回结果：\n%s\n\n请基于以上结果继续回答用户的问题。注意：如果需要更新 portfolio.md，请使用 update 操作并传入完整的新内容。", toolName, toolResult),
			})
			continue
		}

		// 普通回复，写进历史并返回
		a.history = append(a.history,
			llm.Message{Role: "user", Content: userInput},
			llm.Message{Role: "assistant", Content: reply},
		)
		return reply, nil
	}

	return "抱歉，我反复调用了工具还是没搞定，请换个说法试试？", nil
}

// buildMessages 组装 system + 历史消息
func (a *Agent) buildMessages(ctx context.Context) ([]llm.Message, error) {
	systemPrompt, err := a.buildSystemPrompt()
	if err != nil {
		return nil, err
	}
	messages := []llm.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, a.history...)
	return messages, nil
}

// buildSystemPrompt 把用户画像 + 资产状态拼进 system prompt
func (a *Agent) buildSystemPrompt() (string, error) {
	var sb strings.Builder

	sb.WriteString(`你是 Money Agent，一个帮助用户更好理财的对话助手。你的风格：
- 用中文、口语化、不说废话
- 给出具体数字和明确建议，不要模棱两可
- 涉及投资建议时提醒风险，不做绝对承诺
- 如果需要查看或更新用户的资产数据，请使用工具调用

可用工具：
- soul: 查看用户理财画像
- portfolio: 查看/更新资产追踪（view 或 update）

注意事项：
- 回答长度控制在 200-400 字
- 如果用户让你"看看我有多少钱"或类似表述，直接调用 portfolio(view)
- 如果用户让你"帮我分析一下应该怎么配置"，先调用 soul 和 portfolio 再给建议
`)

	// 拼接 user_soul.md 内容（如果存在）
	soul, err := storage.LoadSoul(a.cfg.DataDir)
	if err != nil {
		return "", err
	}
	if soul != "" {
		sb.WriteString("\n\n## 用户理财画像（user_soul.md）\n")
		sb.WriteString(soul)
	}

	// 拼接 portfolio.md 当前状态
	portfolio, err := storage.LoadPortfolio(a.cfg.DataDir)
	if err != nil {
		return "", err
	}
	sb.WriteString("\n\n## 用户当前资产（portfolio.md）\n")
	sb.WriteString(portfolio)

	return sb.String(), nil
}

// Reset 清空对话历史（用于 /reset 命令等）
func (a *Agent) Reset() {
	a.history = nil
}

// parseToolCall 从 LLM 回复中提取 [tool_call] 标记。
// LLM 可能返回 "帮你查一下\n[tool_call] name=portfolio args={...}" 这样的混合内容。
// 返回 (toolName, argsJSON, 是否命中)
func parseToolCall(reply string) (string, string, bool) {
	// 优先找 [tool_call] 标记（deepseek.go 转换后的格式）
	tcIdx := strings.Index(reply, "[tool_call]")
	if tcIdx >= 0 {
		after := reply[tcIdx+len("[tool_call]"):]
		// 跳过前导空白
		after = strings.TrimLeft(after, " \t\n")
		// 取第一行（可能有多个 tool_call，先处理第一个）
		if nlIdx := strings.Index(after, "\n"); nlIdx >= 0 {
			after = after[:nlIdx]
		}
		after = strings.TrimSpace(after)

		// 格式: name=portfolio args={"action":"view"}
		// 先按 " args=" 拆分取前半
		namePart := after
		args := "{}"
		if idx := strings.Index(after, " args="); idx >= 0 {
			namePart = strings.TrimSpace(after[:idx])
			args = strings.TrimSpace(after[idx+len(" args="):])
		}
		// 再从 namePart 里去掉 "name=" 前缀
		if strings.HasPrefix(namePart, "name=") {
			namePart = strings.TrimSpace(namePart[len("name="):])
		}
		if namePart != "" {
			return namePart, args, true
		}
	}

	// 尝试解析裸 JSON
	trimmed := strings.TrimSpace(reply)
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"name"`) {
		var raw struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(trimmed), &raw); err == nil && raw.Name != "" {
			return raw.Name, string(raw.Arguments), true
		}
	}

	return "", "", false
}

// NewWithTimeout 方便调用时设置超时
func (a *Agent) ChatWithTimeout(userInput string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return a.Chat(ctx, userInput)
}
