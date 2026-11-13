package tools

import (
	"context"
	"fmt"

	"money-agent/internal/llm"
	"money-agent/internal/storage"
)

// SoulTool 让 agent 能查看 user_soul.md
type SoulTool struct {
	DataDir string
}

func (t *SoulTool) Name() string { return "soul" }

func (t *SoulTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        "soul",
		Description: "查看用户的理财画像（user_soul.md），了解用户的风险偏好、投资经验、财务目标等。通常在开始新一轮对话前调用一次。",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (t *SoulTool) Run(ctx context.Context, argsJSON string) (string, error) {
	content, err := storage.LoadSoul(t.DataDir)
	if err != nil {
		return "", err
	}
	if content == "" {
		return "用户尚未完成首次理财画像问卷，请先运行 `money-agent init`", nil
	}
	return fmt.Sprintf("以下是 user_soul.md 内容：\n\n%s", content), nil
}
