package summary

import (
	"context"
	"fmt"
	"os"
	"time"

	"qq-bot-star/agents/conversation"
	"qq-bot-star/utils/logger"
	"qq-bot-star/utils/storage"
)

type Service struct {
	conversationAgent *conversation.Agent
}

func NewService(convAgent *conversation.Agent) *Service {
	return &Service{
		conversationAgent: convAgent,
	}
}

// StartDailySummary 启动每日总结任务
func (s *Service) StartDailySummary() {
	// 立即执行一次
	s.GenerateDailySummary()

	// 设置定时任务，每天凌晨执行
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			s.GenerateDailySummary()
		}
	}()
}

// GenerateDailySummary 生成每日总结
func (s *Service) GenerateDailySummary() {
	// 获取昨天的日期
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	logger.Infof("开始生成 %s 的总结", yesterday)

	// 获取所有群
	groups, err := storage.ListGroups(yesterday)
	if err != nil {
		logger.Errorf("获取群列表失败: %v", err)
		return
	}

	for _, groupID := range groups {
		// 获取群内所有用户
		users, err := storage.ListUsers(yesterday, groupID)
		if err != nil {
			logger.Errorf("获取用户列表失败: %v", err)
			continue
		}

		for _, userID := range users {
			// 获取用户消息
			messages, err := storage.GetMessages(yesterday, groupID, userID)
			if err != nil {
				logger.Errorf("获取消息失败: %v", err)
				continue
			}

			if len(messages) == 0 {
				continue
			}

			// 生成总结
			summary := s.generateSummary(messages, yesterday, groupID, userID)

			// 存储总结
			err = storage.StoreSummary(summary)
			if err != nil {
				logger.Errorf("存储总结失败: %v", err)
			} else {
				logger.Infof("生成总结成功 - 日期: %s, 群: %d, 用户: %d", yesterday, groupID, userID)
			}
		}
	}

	// 清理昨天的原始消息数据
	s.cleanupDailyData(yesterday)

	logger.Infof("每日总结生成完成")
}

// cleanupDailyData 清理每日数据
func (s *Service) cleanupDailyData(date string) {
	dir := fmt.Sprintf("data/daily/%s", date)
	err := os.RemoveAll(dir)
	if err != nil {
		logger.Errorf("清理每日数据失败: %v", err)
	} else {
		logger.Infof("清理每日数据成功 - 日期: %s", date)
	}
}

// generateSummary 生成总结
func (s *Service) generateSummary(messages []storage.Message, date string, groupID, userID int64) storage.Summary {
	// 提取消息文本
	var messageTexts []string
	var nickname string

	for _, msg := range messages {
		messageTexts = append(messageTexts, msg.Text)
		if nickname == "" {
			nickname = msg.Nickname
		}
	}

	// 构建提示词
	prompt := fmt.Sprintf(`请对以下聊天记录进行总结，提取关键信息和重要事件：
%s

总结要求：
1. 简要概括当天的聊天内容
2. 提取重要的信息和事件
3. 保持客观中立
4. 语言简洁明了
5. 不要包含无关信息
`, fmt.Sprintf("%v", messageTexts))

	// 使用对话智能体生成总结
	summaryText, err := s.conversationAgent.Process(context.Background(), prompt, false, 0, 0, "")
	if err != nil {
		summaryText = "生成总结失败"
	}

	return storage.Summary{
		Date:     date,
		GroupID:  groupID,
		UserID:   userID,
		Nickname: nickname,
		Summary:  summaryText,
		Messages: messageTexts,
		Time:     time.Now().Unix(),
	}
}
