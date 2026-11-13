package questionnaire

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"money-agent/internal/storage"
)

// Question 问卷题
type Question struct {
	Text     string
	Options  []string // 显示给用户的选项（编号 1,2,3...）
	HelpText string   // 选项的解释
}

var questions = []Question{
	{
		Text: "你能接受的最大年度投资亏损是多少？",
		Options: []string{
			"不超过 5%（极保守）",
			"5% ~ 15%（中等）",
			"15% ~ 30%（积极）",
			"超过 30%（激进）",
		},
	},
	{
		Text: "你的投资经验有多久？",
		Options: []string{
			"不到 1 年（新手）",
			"1 ~ 3 年",
			"3 ~ 5 年",
			"5 年以上（经验丰富）",
		},
	},
	{
		Text: "你的主要投资目标是什么？",
		Options: []string{
			"保值为主，不亏就行",
			"稳健增值，略高于通胀",
			"长期（5年+）资产增值",
			"追求高收益，能承担较大风险",
		},
	},
	{
		Text: "你平时会用哪些投资方式？（可多选，用逗号分隔）",
		Options: []string{
			"银行存款 / 货币基金",
			"债券 / 债券基金",
			"混合基金",
			"股票基金 / ETF",
			"个股",
		},
	},
	{
		Text: "你偏好的投资行业或赛道是？",
		Options: []string{
			"没有特别偏好，分散配置",
			"科技 / AI / 新能源",
			"消费 / 医药 / 红利",
			"周期 / 资源 / 金融",
		},
	},
}

// Run 在终端上跑问卷，生成 user_soul.md 写入 dataDir
func Run(dataDir string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("  Money Agent · 首次理财画像问卷")
	fmt.Println("  回答完这 5 道题，我就能更好地帮你")
	fmt.Println("========================================")
	fmt.Println()

	answers := make([]string, len(questions))
	for i, q := range questions {
		fmt.Printf("【第 %d 题】%s\n", i+1, q.Text)
		for j, opt := range q.Options {
			fmt.Printf("  %d) %s\n", j+1, opt)
		}
		fmt.Print("请输入选项编号（默认 1）: ")
		line, err := reader.ReadString('\n')
		if err != nil {
			line = "1"
		}
		line = strings.TrimSpace(line)
		if line == "" {
			line = "1"
		}
		answers[i] = line
		fmt.Println()
	}

	content := buildSoulMD(answers, questions)

	fmt.Println("----------------------------------------")
	fmt.Println("✅ 画像已生成，预览如下：")
	fmt.Println("----------------------------------------")
	fmt.Println(content)
	fmt.Println("----------------------------------------")

	if err := storage.SaveSoul(dataDir, content); err != nil {
		return fmt.Errorf("保存 user_soul.md 失败: %w", err)
	}
	fmt.Printf("\n已写入 %s\n\n", storage.SoulPath(dataDir))
	fmt.Println("现在可以运行 `money-agent chat` 开始对话了 🎯")
	return nil
}

func buildSoulMD(answers []string, questions []Question) string {
	// 把编号映射成选项文本
	resolve := func(q Question, ans string) string {
		idx := 0
		fmt.Sscanf(ans, "%d", &idx)
		if idx < 1 || idx > len(q.Options) {
			return q.Options[0]
		}
		return q.Options[idx-1]
	}

	riskLevel := resolve(questions[0], answers[0])
	experience := resolve(questions[1], answers[1])
	goal := resolve(questions[2], answers[2])
	methods := resolve(questions[3], answers[3])
	preference := resolve(questions[4], answers[4])

	return fmt.Sprintf(`# 用户理财画像

> 由 Money Agent 首次问卷生成，可随时手动编辑

## 风险偏好
- 可接受最大年度亏损：%s
- 投资经验：%s

## 财务目标
- 主要目标：%s

## 常用投资方式
- %s

## 行业 / 赛道偏好
- %s

## 备注
（可手动补充你的特殊要求、禁忌、资金规模等）
`, riskLevel, experience, goal, methods, preference)
}
