package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Message 消息结构
type Message struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Text     string `json:"text"`
	Time     int64  `json:"time"`
}

// Summary 总结结构
type Summary struct {
	Date     string   `json:"date"`
	GroupID  int64    `json:"group_id"`
	UserID   int64    `json:"user_id"`
	Nickname string   `json:"nickname"`
	Summary  string   `json:"summary"`
	Messages []string `json:"messages"`
	Time     int64    `json:"time"`
}

// StoreMessage 存储消息
func StoreMessage(groupID, userID int64, nickname, text string) error {
	date := time.Now().Format("2006-01-02")
	dir := fmt.Sprintf("data/daily/%s/%d", date, groupID)
	os.MkdirAll(dir, 0755)
	
	file := fmt.Sprintf("%s/%d.json", dir, userID)
	var messages []Message
	
	// 读取现有消息
	if _, err := os.Stat(file); err == nil {
		data, _ := os.ReadFile(file)
		json.Unmarshal(data, &messages)
	}
	
	// 添加新消息
	messages = append(messages, Message{
		UserID:   userID,
		Nickname: nickname,
		Text:     text,
		Time:     time.Now().Unix(),
	})
	
	// 写回文件
	data, _ := json.MarshalIndent(messages, "", "  ")
	return os.WriteFile(file, data, 0644)
}

// GetMessages 获取指定日期、群、用户的消息
func GetMessages(date string, groupID, userID int64) ([]Message, error) {
	file := fmt.Sprintf("data/daily/%s/%d/%d.json", date, groupID, userID)
	var messages []Message
	
	if _, err := os.Stat(file); err != nil {
		return messages, nil
	}
	
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(data, &messages)
	return messages, nil
}

// StoreSummary 存储总结
func StoreSummary(summary Summary) error {
	date := summary.Date
	dir := fmt.Sprintf("data/summaries/%s/%d", date, summary.GroupID)
	os.MkdirAll(dir, 0755)
	
	file := fmt.Sprintf("%s/%d.json", dir, summary.UserID)
	data, _ := json.MarshalIndent(summary, "", "  ")
	return os.WriteFile(file, data, 0644)
}

// ListGroups 获取指定日期的所有群
func ListGroups(date string) ([]int64, error) {
	dir := fmt.Sprintf("data/daily/%s", date)
	var groups []int64
	
	if _, err := os.Stat(dir); err != nil {
		return groups, nil
	}
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			var groupID int64
			fmt.Sscanf(entry.Name(), "%d", &groupID)
			groups = append(groups, groupID)
		}
	}
	
	return groups, nil
}

// ListUsers 获取指定日期和群的所有用户
func ListUsers(date string, groupID int64) ([]int64, error) {
	dir := fmt.Sprintf("data/daily/%s/%d", date, groupID)
	var users []int64
	
	if _, err := os.Stat(dir); err != nil {
		return users, nil
	}
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			var userID int64
			fmt.Sscanf(entry.Name(), "%d.json", &userID)
			users = append(users, userID)
		}
	}
	
	return users, nil
}
