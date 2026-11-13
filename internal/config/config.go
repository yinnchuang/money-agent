package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/subosito/gotenv"
)

// Config 集中管理运行期配置
type Config struct {
	// LLM 配置
	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string

	// 数据目录（相对项目根，存放 user_soul.md / portfolio.md）
	DataDir string
}

// Load 从 .env + 进程环境变量加载配置。
// 进程环境变量优先级高于 .env（gotenv.Load 不覆盖已有变量）。
func Load() (*Config, error) {
	// 尝试加载项目根目录下的 .env（如果不存在也不报错，方便 CI 直接用环境变量）
	envPath := filepath.Join(findProjectRoot(), ".env")
	if _, err := os.Stat(envPath); err == nil {
		_ = gotenv.Load(envPath)
	}

	cfg := &Config{
		LLMAPIKey:  os.Getenv("OPENAI_API_KEY"),
		LLMBaseURL: getEnvOrDefault("OPENAI_BASE_URL", "https://api.deepseek.com"),
		LLMModel:   getEnvOrDefault("OPENAI_MODEL", "deepseek-chat"),
		DataDir:    getEnvOrDefault("DATA_DIR", filepath.Join(findProjectRoot(), "data")),
	}

	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY 未配置，请在 .env 或环境变量中设置")
	}

	return cfg, nil
}

// LoadWithoutLLM 用于 init 等不需要 LLM 的命令，容忍 API key 缺失
func LoadWithoutLLM() (*Config, error) {
	envPath := filepath.Join(findProjectRoot(), ".env")
	if _, err := os.Stat(envPath); err == nil {
		_ = gotenv.Load(envPath)
	}
	return &Config{
		LLMAPIKey:  os.Getenv("OPENAI_API_KEY"),
		LLMBaseURL: getEnvOrDefault("OPENAI_BASE_URL", "https://api.deepseek.com"),
		LLMModel:   getEnvOrDefault("OPENAI_MODEL", "deepseek-chat"),
		DataDir:    getEnvOrDefault("DATA_DIR", filepath.Join(findProjectRoot(), "data")),
	}, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// findProjectRoot 向上查找包含 go.mod 或 .env 的目录
func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if parent := filepath.Dir(dir); parent == dir {
			return "."
		} else {
			dir = parent
		}
	}
}
