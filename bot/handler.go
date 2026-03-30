package bot

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"qq-bot-star/agents/conversation"
	"qq-bot-star/domain"
	"qq-bot-star/utils/logger"
)

type Handler struct {
	sender            *Sender
	conversationAgent *conversation.Agent
}

// NewHandler 创建一个处理程序
func NewHandler(sender *Sender, convAgent *conversation.Agent) *Handler {
	return &Handler{
		sender:            sender,
		conversationAgent: convAgent,
	}
}

// HandleMessage 处理消息
func (h *Handler) HandleMessage(msg domain.OneBotMessage) {
	logger.Debugf("收到消息 - 类型: %s, 群ID: %d, 机器人QQ: %d, 发送者QQ: %d",
		msg.MessageType, msg.GroupID, msg.SelfID, msg.UserID)

	if msg.MessageType == "group" {
		isAt := msg.IsAtMe()
		logger.Debugf("群聊消息，是否@机器人: %v", isAt)
		if !isAt {
			logger.Debugf("群聊消息但未@机器人，忽略: %s", msg.Sender.Nickname)
			return
		}
	}

	text := msg.ExtractTextWithoutAt()
	logger.Infof("收到来自 %s(%d) 的消息: %s", msg.Sender.Nickname, msg.UserID, text)

	replyText, err := h.conversationAgent.Process(context.Background(), text)
	if err != nil {
		logger.Errorf("处理消息失败: %v", err)
	}

	logger.Debugf("准备回复 - 群聊: %v, 群ID: %d, 内容: %s",
		msg.MessageType == "group", msg.GroupID, replyText)

	// 按句子分割消息
	sentences := splitSentences(replyText)
	for i, sentence := range sentences {
		// 过滤空消息
		trimmed := strings.TrimSpace(sentence)
		if trimmed == "" {
			continue
		}
		var sendErr error
		if msg.MessageType == "group" {
			sendErr = h.sender.SendGroupMessage(msg.GroupID, trimmed)
		} else {
			sendErr = h.sender.SendPrivateMessage(msg.UserID, trimmed)
		}

		if sendErr != nil {
			logger.Errorf("发送消息失败: %v", sendErr)
			break
		} else {
			logger.Infof("发送消息成功: %s", trimmed)
		}

		// 添加随机延迟，避免发言太快被封号
		if i < len(sentences)-1 {
			// 生成1到3秒之间的随机延迟
			randomDelay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
			time.Sleep(randomDelay)
		}
	}
}

// splitSentences 按句子分割文本
func splitSentences(text string) []string {
	var sentences []string
	var currentSentence string

	for _, char := range text {
		currentSentence += string(char)
		// 按标点符号分割
		if char == '！' || char == '？' || char == '。' || char == '!' || char == '?' || char == '.' {
			sentences = append(sentences, currentSentence)
			currentSentence = ""
		}
	}

	// 添加最后一句（如果有）
	if currentSentence != "" {
		sentences = append(sentences, currentSentence)
	}

	return sentences
}
