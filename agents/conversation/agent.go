package conversation

import (
	"context"
	"qq-bot-star/utils/logger"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Agent 对话智能体
type Agent struct {
	chatModel    model.BaseChatModel
	history      []*schema.Message
	historyMutex sync.Mutex
	maxHistory   int
	systemPrompt string
}

// Config 对话智能体配置
type Config struct {
	ChatModel    model.BaseChatModel
	MaxHistory   int
	SystemPrompt string
}

// NewAgent 创建一个对话智能体
func NewAgent(config Config) *Agent {
	// 初始化默认值
	if config.MaxHistory <= 0 {
		config.MaxHistory = 10
	}
	if config.SystemPrompt == "" {
		config.SystemPrompt = "你是一个可爱的QQ机器人助手，性格活泼开朗，喜欢用表情包，回答简洁有趣。"
	}

	// 初始化对话智能体
	return &Agent{
		chatModel:    config.ChatModel,
		history:      []*schema.Message{},
		maxHistory:   config.MaxHistory,
		systemPrompt: config.SystemPrompt,
	}
}

// Process 处理文本
func (a *Agent) Process(ctx context.Context, text string) (string, error) {
	// 加锁保护历史记录
	a.historyMutex.Lock()
	defer a.historyMutex.Unlock()

	// 先裁剪，保证不会太长
	a.trimHistory()

	// 添加用户消息到历史记录
	a.history = append(a.history, schema.UserMessage(text))

	// 构建消息列表
	messages := a.buildMessages()

	logger.Infof("调用 LLM 处理文本，消息数: %d", len(messages))

	// 调用 LLM 处理文本
	response, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		logger.Errorf("LLM 调用失败: %v", err)
		// 失败时把刚才加的用户消息也删掉，保持历史一致
		if len(a.history) > 0 {
			a.history = a.history[:len(a.history)-1]
		}
		return "哎呀，我脑子卡住了，等会儿再试试吧～", err
	}

	// 提取回复内容
	reply := response.Content
	a.history = append(a.history, schema.AssistantMessage(reply, nil))

	return reply, nil
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages() []*schema.Message {
	messages := []*schema.Message{
		schema.SystemMessage(a.systemPrompt),
	}
	messages = append(messages, a.history...)
	return messages
}

// trimHistory 截断历史记录
func (a *Agent) trimHistory() {
	if len(a.history) > a.maxHistory*2 {
		a.history = a.history[len(a.history)-a.maxHistory*2:]
	}
}

// ClearHistory 清除历史记录
func (a *Agent) ClearHistory() {
	a.historyMutex.Lock()
	defer a.historyMutex.Unlock()
	a.history = []*schema.Message{}
}
