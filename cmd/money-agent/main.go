package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"money-agent/internal/agent"
	"money-agent/internal/config"
	"money-agent/internal/llm"
	"money-agent/internal/questionnaire"
	"money-agent/internal/storage"
	"money-agent/internal/tools"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "money-agent",
		Short: "Money Agent - 帮你做好理财的对话助手",
	}

	rootCmd.AddCommand(initCmd(), chatCmd(), portfolioCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

// initCmd 首次启动的问卷生成
func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "首次运行，生成 user_soul.md 理财画像",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithoutLLM()
			if err != nil {
				return err
			}
			if storage.SoulExists(cfg.DataDir) {
				fmt.Println("⚠️  user_soul.md 已存在，如需重新生成请手动删除后再跑 init")
				fmt.Printf("   位置：%s\n", storage.SoulPath(cfg.DataDir))
				return nil
			}
			return questionnaire.Run(cfg.DataDir)
		},
	}
}

// chatCmd 进入对话模式
func chatCmd() *cobra.Command {
	var oneShot string
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "进入对话模式",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// 还没 init 过的话提醒一下（但不强制）
			if !storage.SoulExists(cfg.DataDir) {
				fmt.Println("💡 还没生成 user_soul.md，建议先跑 `money-agent init`")
				fmt.Println()
			}

			llmClient := llm.NewDeepSeekClient(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel)
			registry := tools.NewRegistry()
			registry.Register(&tools.PortfolioTool{DataDir: cfg.DataDir})
			registry.Register(&tools.SoulTool{DataDir: cfg.DataDir})
			ag := agent.New(cfg, llmClient, registry)

			if oneShot != "" {
				return runOnce(ag, oneShot)
			}
			return runREPL(ag)
		},
	}
	cmd.Flags().StringVarP(&oneShot, "message", "m", "", "单轮模式：直接传入问题，不进入交互")
	return cmd
}

// portfolioCmd 查看当前资产快照
func portfolioCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "portfolio",
		Short: "查看 / 更新 portfolio.md",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadWithoutLLM()
			if err != nil {
				return err
			}
			content, err := storage.LoadPortfolio(cfg.DataDir)
			if err != nil {
				return err
			}
			fmt.Println(content)
			return nil
		},
	}
}

func runOnce(ag *agent.Agent, input string) error {
	fmt.Println("🤔 Money Agent 思考中...")
	reply, err := ag.ChatWithTimeout(input, 60)
	if err != nil {
		return err
	}
	fmt.Println(reply)
	return nil
}

func runREPL(ag *agent.Agent) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("💰 Money Agent 已就绪（输入 quit / exit / Ctrl-D 退出）")
	fmt.Println()

	for {
		fmt.Print("你> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return nil
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if input == "quit" || input == "exit" {
			fmt.Println("👋 再见！")
			return nil
		}
		if input == "/reset" {
			ag.Reset()
			fmt.Println("🔄 对话已重置")
			continue
		}
		if input == "/help" {
			fmt.Println("可用命令：/reset（清空对话）/help（帮助）quit（退出）")
			continue
		}

		fmt.Print("🤔 Money Agent 思考中...\n\n")
		reply, err := ag.ChatWithTimeout(input, 60)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ 出错了: %v\n", err)
			continue
		}
		fmt.Println(reply)
		fmt.Println()
	}
}
