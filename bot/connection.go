package bot

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"time"

	"qq-bot-star/agents/conversation"
	"qq-bot-star/domain"
	"qq-bot-star/utils/logger"

	"github.com/gorilla/websocket"
)

type Bot struct {
	conn      *websocket.Conn
	handler   *Handler
	sender    *Sender
	done      chan struct{}
	interrupt chan os.Signal
}

// NewBot 创建一个机器人
func NewBot(wsURL string, convAgent *conversation.Agent) (*Bot, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}

	// 重试连接，直到成功
	var conn *websocket.Conn
	retries := 0
	maxRetries := 30 // 最多重试 30 次，每次间隔 2 秒，总共 1 分钟

	for retries < maxRetries {
		logger.Infof("连接到 NapCatQQ: %s (尝试 %d/%d)", u.String(), retries+1, maxRetries)
		var dialErr error
		conn, _, dialErr = websocket.DefaultDialer.Dial(u.String(), nil)
		if dialErr == nil {
			break
		}

		logger.Warnf("连接失败: %v, 2 秒后重试", dialErr)
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			break
		}

		logger.Warnf("连接失败: %v, 2 秒后重试", err)
		time.Sleep(2 * time.Second)
		retries++
	}

	if err != nil {
		return nil, fmt.Errorf("连接 NapCatQQ 失败: %v", err)
	}

	bot := &Bot{
		conn:      conn,
		done:      make(chan struct{}),
		interrupt: make(chan os.Signal, 1),
	}

	bot.sender = NewSender(conn)
	bot.handler = NewHandler(bot.sender, convAgent)

	signal.Notify(bot.interrupt, os.Interrupt)

	return bot, nil
}

// Start 启动机器人
func (b *Bot) Start() {
	go b.readLoop()
	logger.Info("连接成功！按 Ctrl+C 退出")

	for {
		select {
		case <-b.done:
			return
		case <-b.interrupt:
			b.Stop()
			return
		}
	}
}

// readLoop 读取消息循环
func (b *Bot) readLoop() {
	defer close(b.done)
	for {
		_, message, err := b.conn.ReadMessage()
		if err != nil {
			logger.Errorf("读取消息失败: %v", err)
			return
		}
		logger.Debugf("收到消息: %s", message)

		var msg domain.OneBotMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			logger.Errorf("解析消息失败: %v", err)
			continue
		}

		if msg.PostType == "message" {
			go b.handler.HandleMessage(msg)
		}
	}
}

// Stop 停止机器人
func (b *Bot) Stop() {
	logger.Info("正在关闭连接...")
	err := b.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		logger.Errorf("关闭连接失败: %v", err)
	}
	select {
	case <-b.done:
	case <-time.After(time.Second):
	}
	b.conn.Close()
}
