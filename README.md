# QQ Bot Star 🤖✨

一个基于 Go + OneBot + Eino 框架的智能 QQ 机器人，包含三个核心 Agent！

---

## 项目架构

```
source_code/
├── main.go                 # 入口文件：启动机器人
├── go.mod / go.sum        # Go 模块依赖
│
├── domain/                 # 领域模型层
│   ├── bot.go              # OneBot 动作结构
│   └── message.go          # OneBot 消息结构
│
├── bot/                    # 机器人核心层
│   ├── connection.go       # WebSocket 连接管理（连接 NapCatQQ）
│   ├── handler.go          # 消息分发处理
│   └── sender.go           # 消息发送封装
│
├── agents/                 # 三个智能 Agent！
│   ├── conversation/       # 对话 Agent - 跟群员互动聊天
│   ├── knowledge/          # 知识库 Agent - 记忆有趣的聊天记录
│   └── operations/         # 运维 Agent - 监控报错、生成报告
│
├── storage/                # 数据存储层
│   └── (待实现)            # 给知识库 Agent 用的存储
│
├── config/                 # 配置层
│   └── (待实现)            # 各种配置项
│
└── utils/                  # 工具函数层
    └── (待实现)            # 通用工具
```

---

## 技术栈

- **Go 1.24+** - 后端语言
- **gorilla/websocket** - WebSocket 通信
- **OneBot 协议** - QQ 机器人协议
- **NapCatQQ** - OneBot 实现
- **Eino** - Go 语言 LLM 应用框架（待集成）

---

## 三个 Agent 说明

### 1. 对话 Agent 💬
负责跟群员互动，处理对话相关问题。

### 2. 知识库 Agent 🧠
负责记忆，保存群聊里有趣的聊天记录，支持查询。

### 3. 运维 Agent 🔧
负责检查自身报错，把错误信息整理成报告返回，辅助开发。

---

## 快速开始

### 前置条件
1. 安装 Go 1.24+
2. 启动 NapCatQQ 并开启 WebSocket 服务（默认 ws://127.0.0.1:3001）

### 运行项目

```bash
cd source_code
go run main.go
```

---

## 开发进度

- [x] 项目架构重构
- [x] 机器人核心层（连接、收发消息）
- [ ] 对话 Agent 实现
- [ ] 知识库 Agent 实现
- [ ] 运维 Agent 实现
- [ ] Eino 框架集成
- [ ] 部署到腾讯云

---

## 实习建议 💡

（先把架构搭好，再一个个实现 Agent，稳扎稳打！）
