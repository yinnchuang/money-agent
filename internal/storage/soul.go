package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

const soulFile = "user_soul.md"

// SoulPath 返回 user_soul.md 的绝对路径
func SoulPath(dataDir string) string {
	return filepath.Join(dataDir, soulFile)
}

// LoadSoul 读取 user_soul.md，不存在返回空串（不是错误）
func LoadSoul(dataDir string) (string, error) {
	path := SoulPath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("读取 user_soul.md 失败: %w", err)
	}
	return string(data), nil
}

// SaveSoul 写入 user_soul.md
func SaveSoul(dataDir, content string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}
	path := SoulPath(dataDir)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("写入 user_soul.md 失败: %w", err)
	}
	return nil
}

// SoulExists 判断是否已完成首次问卷
func SoulExists(dataDir string) bool {
	_, err := os.Stat(SoulPath(dataDir))
	return err == nil
}
