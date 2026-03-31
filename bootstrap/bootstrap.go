package bootstrap

import (
	"context"

	"qq-bot-star/agents/conversation"
	"qq-bot-star/agents/knowledge"
	"qq-bot-star/bot"
	"qq-bot-star/config"
	"qq-bot-star/services/summary"
	"qq-bot-star/utils/logger"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

type App struct {
	Bot *bot.Bot
}

// NewApp 创建一个新的应用实例
func NewApp(cfg *config.Config) (*App, error) {
	// 初始化日志
	if err := logger.Init(cfg.Log); err != nil {
		return nil, err
	}
	logger.Info("日志初始化成功")

	// 初始化 LLM 模型
	chatModel, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  cfg.LLM.APIKey,
		BaseURL: cfg.LLM.BaseURL,
		Model:   cfg.LLM.Model,
	})
	if err != nil {
		return nil, err
	}
	logger.Info("LLM 模型初始化成功")

	// 初始化知识库 Agent
	knowledgeAgent, err := knowledge.NewAgent(context.Background(), knowledge.Config{
		Config: cfg,
	})
	if err != nil {
		return nil, err
	}
	logger.Info("知识库 Agent 初始化成功")

	// 获取工具列表
	tools, err := knowledgeAgent.GetTools()
	if err != nil {
		logger.Warnf("获取工具列表失败，将不使用工具调用: %v", err)
		tools = nil
	}

	// 初始化对话 Agent
	convAgent := conversation.NewAgent(conversation.Config{
		ChatModel:           chatModel,
		MaxHistory:          cfg.ConversationAgent.MaxHistory,
		DefaultSystemPrompt: cfg.ConversationAgent.SystemPrompt,
		KnowledgeAgent:      knowledgeAgent,
		Tools:               tools,
	})
	logger.Info("对话 Agent 初始化成功")

	// 初始化总结服务
	summaryService := summary.NewService(convAgent)
	logger.Info("总结服务初始化成功")

	// 初始化机器人
	botInstance, err := bot.NewBot(cfg.Bot.WSURL, convAgent, knowledgeAgent)
	if err != nil {
		return nil, err
	}
	logger.Info("机器人初始化成功")

	// 启动总结服务
	summaryService.StartDailySummary()

	// 返回应用实例
	return &App{
		Bot: botInstance,
	}, nil
}

func (a *App) Start() {
	logger.Info("应用启动中...")
	a.Bot.Start()
}

func (a *App) Stop() {
	logger.Info("应用停止中...")
	logger.Sync()
}
