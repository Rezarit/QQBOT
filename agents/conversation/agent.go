package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"qq-bot-star/utils/logger"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Agent 对话智能体
type Agent struct {
	chatModel           model.BaseChatModel
	histories           map[string][]*schema.Message // key: sessionID, value: 历史记录
	historiesMu         sync.RWMutex                 // 保护 histories 的读写锁
	maxHistory          int                          // 每个会话最大历史轮数
	defaultSystemPrompt string                       // 默认系统提示词
	sillyTavernAPIURL   string                       // SillyTavern API地址
	sillyTavernAPIKey   string                       // SillyTavern API密钥
	sillyTavernModel    string                       // SillyTavern 使用的模型
	client              *http.Client                 // HTTP客户端，用于调用SillyTavern API
	toolsNode           *compose.ToolsNode           // 工具执行节点（可选）
	tools               []tool.BaseTool              // 工具列表（可选）
}

// SillyTavernMessage SillyTavern消息结构
type SillyTavernMessage struct {
	Role    string `json:"role"`    // 角色: user, assistant
	Content string `json:"content"` // 内容
}

// SillyTavernRequest SillyTavern API请求结构
type SillyTavernRequest struct {
	Messages []SillyTavernMessage `json:"messages"`
	Model    string               `json:"model"`
	Stream   bool                 `json:"stream"`
}

// SillyTavernResponse SillyTavern API响应结构
type SillyTavernResponse struct {
	Choices []struct {
		Message SillyTavernMessage `json:"message"`
	} `json:"choices"`
}

// Config 对话智能体配置
type Config struct {
	ChatModel           model.BaseChatModel
	MaxHistory          int
	DefaultSystemPrompt string
	SillyTavernAPIURL   string
	SillyTavernAPIKey   string
	SillyTavernModel    string
	Tools               []tool.BaseTool // 工具列表（可选）
}

// NewAgent 创建一个对话智能体
func NewAgent(config Config) *Agent {
	// 初始化默认值
	if config.MaxHistory <= 0 {
		config.MaxHistory = 10
	}
	if config.DefaultSystemPrompt == "" {
		config.DefaultSystemPrompt = "你是一个可爱的QQ机器人助手，性格活泼开朗，喜欢用表情包，回答简洁有趣。"
	}

	agent := &Agent{
		chatModel:           config.ChatModel,
		histories:           make(map[string][]*schema.Message),
		maxHistory:          config.MaxHistory,
		defaultSystemPrompt: config.DefaultSystemPrompt,
		sillyTavernAPIURL:   config.SillyTavernAPIURL,
		sillyTavernAPIKey:   config.SillyTavernAPIKey,
		sillyTavernModel:    config.SillyTavernModel,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		tools: config.Tools,
	}

	// 如果有工具，初始化 ToolsNode 并绑定工具到 ChatModel
	if len(config.Tools) > 0 {
		toolsNode, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
			Tools: config.Tools,
		})
		if err != nil {
			logger.Warnf("初始化 ToolsNode 失败，将不使用工具调用: %v", err)
		} else {
			agent.toolsNode = toolsNode

			// 绑定工具信息到 ChatModel (使用类型断言)
			var toolInfos []*schema.ToolInfo
			for _, t := range config.Tools {
				info, err := t.Info(context.Background())
				if err != nil {
					logger.Warnf("获取工具信息失败: %v", err)
					continue
				}
				toolInfos = append(toolInfos, info)
			}

			if len(toolInfos) > 0 {
				// 类型断言：检查 ChatModel 是否支持 BindTools
				type bindable interface {
					BindTools([]*schema.ToolInfo) error
				}

				if bindableModel, ok := config.ChatModel.(bindable); ok {
					err = bindableModel.BindTools(toolInfos)
					if err != nil {
						logger.Warnf("绑定工具到 ChatModel 失败: %v", err)
					} else {
						logger.Infof("成功绑定 %d 个工具到 ChatModel", len(toolInfos))
					}
				} else {
					logger.Warnf("当前 ChatModel 类型不支持 BindTools，将不使用工具调用")
					agent.toolsNode = nil // 不支持的话就不用 ToolsNode 了
				}
			}
		}
	}

	return agent
}

// getSessionID 生成会话ID
// - 私聊: user_{userID}
// - 群聊: group_{groupID}_user_{userID}
func getSessionID(isGroup bool, groupID, userID int64) string {
	if isGroup {
		return fmt.Sprintf("group_%d_user_%d", groupID, userID)
	}
	return fmt.Sprintf("user_%d", userID)
}

// Process 处理文本
// isGroup: 是否群聊
// groupID: 群ID（群聊时有效）
// userID: 用户QQ号
// nickname: 用户昵称
func (a *Agent) Process(ctx context.Context, text string, isGroup bool, groupID, userID int64, nickname string) (string, error) {
	sessionID := getSessionID(isGroup, groupID, userID)

	// 加写锁
	a.historiesMu.Lock()
	defer a.historiesMu.Unlock()

	// 获取或初始化该会话的历史记录
	history, exists := a.histories[sessionID]
	if !exists {
		history = []*schema.Message{}
		logger.Infof("新会话创建: %s", sessionID)
	}

	// 裁剪历史记录
	history = a.trimHistory(history)

	// 添加用户消息
	history = append(history, schema.UserMessage(text))

	// 如果配置了SillyTavern API，使用SillyTavern处理
	if a.sillyTavernAPIURL != "" {
		logger.Infof("使用 SillyTavern API 处理消息 - 会话: %s", sessionID)
		response, err := a.processWithSillyTavern(ctx, sessionID, history, userID, nickname)
		if err != nil {
			logger.Errorf("SillyTavern API 调用失败 - 会话: %s, 错误: %v", sessionID, err)
			// 失败时回退到LLM处理
			return a.processWithLLM(ctx, sessionID, history, exists, userID, nickname, isGroup, groupID)
		}
		// 添加助手消息到历史
		history = append(history, &schema.Message{
			Role:    "assistant",
			Content: response,
		})
		a.histories[sessionID] = history
		return response, nil
	}

	// 没有配置SillyTavern API，使用LLM处理
	return a.processWithLLM(ctx, sessionID, history, exists, userID, nickname, isGroup, groupID)
}

// processWithLLM 使用LLM处理消息
func (a *Agent) processWithLLM(ctx context.Context, sessionID string, history []*schema.Message, exists bool, userID int64, nickname string, isGroup bool, groupID int64) (string, error) {
	// 构建消息列表（包含当前用户QQ号和昵称）
	messages := a.buildMessages(ctx, history, userID, nickname, isGroup, groupID)

	logger.Infof("调用 LLM - 会话: %s, 消息数: %d, QQ: %d, 昵称: %s", sessionID, len(messages), userID, nickname)

	// 如果有 ToolsNode，使用工具调用
	if a.toolsNode != nil {
		return a.processWithTools(ctx, sessionID, history, messages, exists)
	}

	// 没有工具，直接调用 LLM
	return a.processWithoutTools(ctx, sessionID, history, messages, exists)
}

// processWithSillyTavern 使用SillyTavern API处理消息
func (a *Agent) processWithSillyTavern(ctx context.Context, sessionID string, history []*schema.Message, userID int64, nickname string) (string, error) {
	// 构建SillyTavern消息列表
	stMessages := make([]SillyTavernMessage, 0, len(history))

	// 添加系统提示
	stMessages = append(stMessages, SillyTavernMessage{
		Role:    "system",
		Content: a.defaultSystemPrompt,
	})

	// 添加历史消息
	for _, msg := range history {
		if msg.Role == "user" {
			stMessages = append(stMessages, SillyTavernMessage{
				Role:    "user",
				Content: msg.Content,
			})
		} else if msg.Role == "assistant" {
			stMessages = append(stMessages, SillyTavernMessage{
				Role:    "assistant",
				Content: msg.Content,
			})
		}
	}

	// 构建请求
	request := SillyTavernRequest{
		Messages: stMessages,
		Model:    a.sillyTavernModel,
		Stream:   false,
	}

	// 序列化请求
	data, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.sillyTavernAPIURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if a.sillyTavernAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.sillyTavernAPIKey)
	}

	// 发送请求
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API调用失败，状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var response SillyTavernResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取回复
	if len(response.Choices) == 0 || response.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("SillyTavern返回空回复")
	}

	return response.Choices[0].Message.Content, nil
}

// processWithTools 带工具调用的处理流程
func (a *Agent) processWithTools(ctx context.Context, sessionID string, history []*schema.Message, messages []*schema.Message, exists bool) (string, error) {
	maxToolCalls := 3 // 最大工具调用次数
	currentMessages := make([]*schema.Message, len(messages))
	copy(currentMessages, messages)

	for i := 0; i < maxToolCalls; i++ {
		logger.Infof("工具调用 - 第 %d 轮，消息数: %d", i+1, len(currentMessages))

		// 调用 LLM
		response, err := a.chatModel.Generate(ctx, currentMessages)
		if err != nil {
			logger.Errorf("LLM 调用失败 - 会话: %s, 错误: %v", sessionID, err)
			if exists {
				a.histories[sessionID] = history[:len(history)-1]
			}
			return "哎呀，我脑子卡住了，等会儿再试试吧～", err
		}

		// 添加助手消息到历史
		currentMessages = append(currentMessages, response)

		// 检查是否有工具调用
		if len(response.ToolCalls) == 0 {
			// 没有工具调用，直接返回结果
			history = append(history, response)
			a.histories[sessionID] = history
			logger.Infof("LLM 回复成功（无工具调用） - 会话: %s", sessionID)
			return response.Content, nil
		}

		// 有工具调用，执行工具
		logger.Infof("执行工具调用 - 会话: %s, 工具数: %d", sessionID, len(response.ToolCalls))

		toolMessages, err := a.toolsNode.Invoke(ctx, response)
		if err != nil {
			logger.Errorf("工具调用失败 - 会话: %s, 错误: %v", sessionID, err)
			if exists {
				a.histories[sessionID] = history[:len(history)-1]
			}
			return "哎呀，调用工具失败了，等会儿再试试吧～", err
		}

		// 添加工具结果消息
		currentMessages = append(currentMessages, toolMessages...)

		logger.Infof("工具调用完成 - 会话: %s, 继续对话", sessionID)
	}

	// 超过最大工具调用次数，返回最后一条消息
	logger.Warnf("超过最大工具调用次数 - 会话: %s", sessionID)
	history = append(history, currentMessages[len(currentMessages)-1])
	a.histories[sessionID] = history
	return history[len(history)-1].Content, nil
}

// processWithoutTools 不带工具调用的处理流程
func (a *Agent) processWithoutTools(ctx context.Context, sessionID string, history []*schema.Message, messages []*schema.Message, exists bool) (string, error) {
	// 调用 LLM
	response, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		logger.Errorf("LLM 调用失败 - 会话: %s, 错误: %v", sessionID, err)
		// 失败时回滚，不保存这次用户消息
		if exists {
			a.histories[sessionID] = history[:len(history)-1]
		}
		return "哎呀，我脑子卡住了，等会儿再试试吧～", err
	}

	// 保存回复
	reply := response.Content
	history = append(history, schema.AssistantMessage(reply, nil))
	a.histories[sessionID] = history

	logger.Infof("LLM 回复成功 - 会话: %s", sessionID)
	return reply, nil
}

// buildMessages 构建消息列表
func (a *Agent) buildMessages(ctx context.Context, history []*schema.Message, userID int64, nickname string, isGroup bool, groupID int64) []*schema.Message {
	// 使用默认系统提示词
	systemPrompt := a.defaultSystemPrompt

	// 在系统提示词中加上当前对话用户的基本信息
	if nickname != "" {
		systemPrompt = fmt.Sprintf(`%s

当前正在跟你对话的用户信息：
- 用户 QQ 号（user_id）：%d
- 群 ID（group_id）：%d
- 昵称：%s

**记住**：不要直白地告诉用户他的 QQ 号或昵称！要用自然的方式跟他交流！`, systemPrompt, userID, groupID, nickname)
	} else {
		systemPrompt = fmt.Sprintf(`%s

当前正在跟你对话的用户信息：
- 用户 QQ 号（user_id）：%d
- 群 ID（group_id）：%d

**记住**：不要直白地告诉用户他的 QQ 号！要用自然的方式跟他交流！`, systemPrompt, userID, groupID)
	}

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
	}
	messages = append(messages, history...)
	return messages
}

// trimHistory 截断历史记录
func (a *Agent) trimHistory(history []*schema.Message) []*schema.Message {
	maxLen := a.maxHistory * 2 // 每个用户消息对应一个机器人回复
	if len(history) > maxLen {
		return history[len(history)-maxLen:]
	}
	return history
}

// ClearHistory 清除指定会话的历史记录
func (a *Agent) ClearHistory(isGroup bool, groupID, userID int64) {
	sessionID := getSessionID(isGroup, groupID, userID)
	a.historiesMu.Lock()
	defer a.historiesMu.Unlock()
	delete(a.histories, sessionID)
	logger.Infof("清除会话历史: %s", sessionID)
}

// ClearAllHistory 清除所有会话的历史记录
func (a *Agent) ClearAllHistory() {
	a.historiesMu.Lock()
	defer a.historiesMu.Unlock()
	count := len(a.histories)
	a.histories = make(map[string][]*schema.Message)
	logger.Infof("清除所有会话历史，共 %d 个会话", count)
}
