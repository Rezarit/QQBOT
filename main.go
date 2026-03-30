package main

import (
	"os"
	"os/signal"
	"syscall"

	"qq-bot-star/bootstrap"
	"qq-bot-star/config"
)

func main() {
	// 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic("加载配置失败: " + err.Error())
	}

	// 初始化应用
	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		panic("初始化应用失败: " + err.Error())
	}
	defer app.Stop()

	go func() {
		app.Start()
	}()

	// 等待信号量信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}
