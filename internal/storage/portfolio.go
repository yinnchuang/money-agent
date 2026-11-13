package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

const portfolioFile = "portfolio.md"

// PortfolioPath 返回 portfolio.md 的绝对路径
func PortfolioPath(dataDir string) string {
	return filepath.Join(dataDir, portfolioFile)
}

// LoadPortfolio 读取 portfolio.md，不存在返回默认模板
func LoadPortfolio(dataDir string) (string, error) {
	path := PortfolioPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultPortfolio(), nil
		}
		return "", fmt.Errorf("读取 portfolio.md 失败: %w", err)
	}
	return string(data), nil
}

// SavePortfolio 写入 portfolio.md
func SavePortfolio(dataDir, content string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	path := PortfolioPath(dataDir)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入 portfolio.md 失败: %w", err)
	}
	return nil
}

func defaultPortfolio() string {
	return `# 资产追踪

## 当前总资产
¥0.00

## 持仓
| 基金代码 | 名称 | 持仓成本 | 现价 | 持仓份额 | 收益率 |
|----------|------|----------|------|----------|--------|

## 最近操作
（暂无记录）
`
}
