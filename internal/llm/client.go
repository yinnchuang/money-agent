package llm

// Message 统一的对话消息结构
type Message struct {
	Role    string // "system" | "user" | "assistant" | "tool"
	Content string
}

// Client LLM 抽象接口，方便后续替换不同模型
type Client interface {
	// Chat 发送多轮对话，返回模型回复
	Chat(messages []Message, tools []ToolDefinition) (string, error)
}

// ToolDefinition 描述一个可供 LLM 调用的工具
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema
}
