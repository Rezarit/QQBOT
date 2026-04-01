package knowledge

import (
	"context"
)

// Agent 知识库智能体
type Agent struct {
}

// Config 知识库智能体配置
type Config struct {
}

// NewAgent 创建一个知识库智能体
func NewAgent(ctx context.Context, cfg Config) (*Agent, error) {
	return &Agent{}, nil
}
