package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"money-agent/internal/llm"
)

// Tool agent 可调用的工具接口
type Tool interface {
	// Name 工具名称，LLM 通过此名字调用
	Name() string
	// Description 描述 + 参数 JSON Schema，给 LLM 看的
	Definition() llm.ToolDefinition
	// Run 执行工具，返回结果文本；ctx 用于超时控制
	Run(ctx context.Context, argsJSON string) (string, error)
}

// Registry 工具注册中心
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// All 返回所有工具定义，喂给 LLM
func (r *Registry) All() []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// RunByName 根据名字执行工具。args 可以是 JSON 字符串或纯文本
func (r *Registry) RunByName(ctx context.Context, name, argsJSON string) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}
	return t.Run(ctx, argsJSON)
}

// ParseArgs 把 JSON args 解析到目标结构体（方便工具自己定义参数）
func ParseArgs(argsJSON string, target any) error {
	if argsJSON == "" {
		return nil
	}
	return json.Unmarshal([]byte(argsJSON), target)
}
