package bot

import (
	"encoding/json"
	"fmt"
	"sync/atomic"

	"qq-bot-star/domain"
	"qq-bot-star/utils/logger"

	"github.com/gorilla/websocket"
)

type Sender struct {
	conn      *websocket.Conn
	messageID int64
}

// NewSender 创建一个发送器
func NewSender(conn *websocket.Conn) *Sender {
	return &Sender{
		conn: conn,
	}
}

// SendGroupMessage 发送群消息
func (s *Sender) SendGroupMessage(groupID int64, text string) error {
	return s.send("send_msg", map[string]interface{}{
		"group_id": groupID,
		"message":  text,
	})
}

// SendPrivateMessage 发送私聊消息
func (s *Sender) SendPrivateMessage(userID int64, text string) error {
	return s.send("send_msg", map[string]interface{}{
		"user_id": userID,
		"message": text,
	})
}

// send 发送消息
func (s *Sender) send(action string, params map[string]interface{}) error {
	msgID := atomic.AddInt64(&s.messageID, 1)

	actionMsg := domain.OneBotAction{
		Action: action,
		Params: params,
		Echo:   "action_" + fmt.Sprintf("%d", msgID),
	}

	data, err := json.Marshal(actionMsg)
	if err != nil {
		logger.Errorf("序列化消息失败: %v", err)
		return err
	}

	err = s.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		logger.Errorf("发送消息失败: %v", err)
		return err
	}

	logger.Debug("发送消息成功")
	return nil
}
