package agent

import (
	"context"
	"fmt"
	"sync"
)

// ToolRegistry 工具注册中心
type ToolRegistry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewToolRegistry 创建工具注册中心
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("工具名称不能为空")
	}

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("工具 %s 已存在", tool.Name)
	}

	if tool.Handler == nil {
		return fmt.Errorf("工具 %s 必须有处理函数", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// MustRegister 注册工具（失败时 panic）
func (r *ToolRegistry) MustRegister(tool *Tool) {
	if err := r.Register(tool); err != nil {
		panic(err)
	}
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// Execute 执行工具
func (r *ToolRegistry) Execute(ctx context.Context, name string, args map[string]interface{}) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("工具 %s 不存在", name),
		}, fmt.Errorf("工具 %s 不存在", name)
	}

	return tool.Handler(ctx, args)
}

// GetAll 获取所有工具
func (r *ToolRegistry) GetAll() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetNames 获取所有工具名称
func (r *ToolRegistry) GetNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ToOpenAIFormat 转换为 OpenAI/OpenRouter 工具格式
func (r *ToolRegistry) ToOpenAIFormat() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]map[string]interface{}, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool.ToOpenAIFormat())
	}
	return tools
}

// Count 获取工具数量
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Unregister 注销工具
func (r *ToolRegistry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		return true
	}
	return false
}

// Clear 清空所有工具
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = make(map[string]*Tool)
}
