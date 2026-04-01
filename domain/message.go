package domain

import (
	"fmt"
)

type OneBotMessage struct {
	Time        int64       `json:"time"`
	SelfID      int64       `json:"self_id"`
	PostType    string      `json:"post_type"`
	MessageType string      `json:"message_type"`
	SubType     string      `json:"sub_type"`
	MessageID   int64       `json:"message_id"`
	UserID      int64       `json:"user_id"`
	Message     interface{} `json:"message"`
	RawMessage  string      `json:"raw_message"`
	Font        int         `json:"font"`
	Sender      struct {
		UserID   int64  `json:"user_id"`
		Nickname string `json:"nickname"`
		Card     string `json:"card"`
		Sex      string `json:"sex"`
		Age      int    `json:"age"`
		Area     string `json:"area"`
		Level    string `json:"level"`
		Role     string `json:"role"`
		Title    string `json:"title"`
	} `json:"sender"`
	GroupID int64 `json:"group_id"`
}

type MessageSegment struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// IsAtMe 检查消息是否@了机器人
func (m *OneBotMessage) IsAtMe() bool {
	if m.MessageType != "group" {
		return true
	}

	segments, ok := m.Message.([]interface{})
	if !ok {
		return false
	}

	selfIDStr := fmt.Sprintf("%d", m.SelfID)

	for _, seg := range segments {
		if segMap, ok := seg.(map[string]interface{}); ok {
			if segType, ok := segMap["type"].(string); ok {
				if segType == "at" {
					if data, ok := segMap["data"].(map[string]interface{}); ok {
						if qq, ok := data["qq"].(float64); ok {
							if int64(qq) == m.SelfID {
								return true
							}
						}
						if qqStr, ok := data["qq"].(string); ok {
							if qqStr == "all" {
								return true
							}
							if qqStr == selfIDStr {
								return true
							}
						}
					}
				}
			}
		}
	}

	return false
}

// ExtractTextWithoutAt 提取文本，并去除@机器人的部分
func (m *OneBotMessage) ExtractTextWithoutAt() string {
	var text string

	segments, ok := m.Message.([]interface{})
	if !ok {
		if str, ok := m.Message.(string); ok {
			return str
		}
		return ""
	}

	for _, seg := range segments {
		if segMap, ok := seg.(map[string]interface{}); ok {
			if segType, ok := segMap["type"].(string); ok {
				if segType == "text" {
					if data, ok := segMap["data"].(map[string]interface{}); ok {
						if t, ok := data["text"].(string); ok {
							text += t
						}
					}
				} else if segType == "image" {
					// 处理图片消息，添加图片描述
					text += "[图片]"
					// 可以在这里提取图片 URL 或其他信息
					if data, ok := segMap["data"].(map[string]interface{}); ok {
						if url, ok := data["url"].(string); ok {
							// 可以将 URL 存储到其他地方
							_ = url
						}
					}
				}
			}
		}
	}

	return text
}

// ExtractContentWithImages 提取文本和图片信息，用于传递给 LLM
func (m *OneBotMessage) ExtractContentWithImages() string {
	var content string

	segments, ok := m.Message.([]interface{})
	if !ok {
		if str, ok := m.Message.(string); ok {
			return str
		}
		return ""
	}

	for _, seg := range segments {
		if segMap, ok := seg.(map[string]interface{}); ok {
			if segType, ok := segMap["type"].(string); ok {
				if segType == "text" {
					if data, ok := segMap["data"].(map[string]interface{}); ok {
						if t, ok := data["text"].(string); ok {
							content += t
						}
					}
				} else if segType == "image" {
					// 处理图片消息，只添加 [图片] 字样
					content += "[图片]"
				}
			}
		}
	}

	return content
}
