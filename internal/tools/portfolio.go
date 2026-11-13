package tools

import (
	"context"
	"fmt"

	"money-agent/internal/llm"
	"money-agent/internal/storage"
)

// PortfolioTool 让 agent 能查看 / 更新 portfolio.md
type PortfolioTool struct {
	DataDir string
}

func (t *PortfolioTool) Name() string { return "portfolio" }

func (t *PortfolioTool) Definition() llm.ToolDefinition {
	return llm.ToolDefinition{
		Name:        "portfolio",
		Description: "查看或更新用户的资产追踪（portfolio.md）。action=view 查看当前快照；action=update 更新总资产或持仓列表时，需要把完整的新内容作为 content 传入。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"enum":        []string{"view", "update"},
					"description": "view=查看当前资产快照；update=更新 portfolio.md",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "action=update 时必填，完整的 portfolio.md 内容（Markdown 格式）",
				},
			},
			"required": []string{"action"},
		},
	}
}

type portfolioArgs struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}

func (t *PortfolioTool) Run(ctx context.Context, argsJSON string) (string, error) {
	var args portfolioArgs
	if err := ParseArgs(argsJSON, &args); err != nil {
		return "", fmt.Errorf("参数解析失败: %w", err)
	}

	switch args.Action {
	case "view":
		content, err := storage.LoadPortfolio(t.DataDir)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("以下是当前 portfolio.md 内容：\n\n%s", content), nil

	case "update":
		if args.Content == "" {
			return "", fmt.Errorf("action=update 时 content 不能为空")
		}
		if err := storage.SavePortfolio(t.DataDir, args.Content); err != nil {
			return "", err
		}
		return "portfolio.md 已更新", nil

	default:
		return "", fmt.Errorf("未知 action: %s（支持 view / update）", args.Action)
	}
}
