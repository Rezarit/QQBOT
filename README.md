# QQ 机器人（NapCat + AstrBot）

Docker Compose 部署，两个容器同网络，NapCat 反向 WS 连到 AstrBot。

## 目录

```
.
├── docker-compose.yml
├── .env                  # 本地变量（WEBUI_TOKEN，已 gitignore，需自己建）
├── persona-feiyu.md      # 「大肥鱼」人格设定：外形 / 生图提示词 / 人格提示词 / 表情包分类
└── data\                 # 运行时数据（已 gitignore）
    ├── qq\               # QQ 登录态（扫码后生成，别删）
    ├── napcat\config\    # NapCat 配置（onebot11.json / webui.json）
    └── astrbot\          # AstrBot 配置与数据（cmd_config.json 等）
```

二维码读到 `qrcode\qrcode.png`，NapCat WebUI 的 token 记在 `.napcat_token`（两者都在本地，不入库）。

## 用到的上游项目

源码没有放进本仓库，容器跑的都是官方镜像：

- [AstrBot](https://github.com/AstrBotDevs/AstrBot) — LLM 机器人框架（`soulter/astrbot:latest`）
- [NapCatQQ](https://github.com/NapNeko/NapCatQQ) — OneBot v11 实现（`mlikiowa/napcat-docker:latest`）

两个上游都兼容 OneBot v11，想换框架/实现只改 `docker-compose.yml` 即可。

## 首次部署

```bash
cd /d/qqbot
echo "WEBUI_TOKEN=$(head -c 12 /dev/urandom | xxd -p)" > .env   # 或随便填一个强密码
docker compose up -d
docker compose logs -f napcat   # 看二维码
```

## 常用命令

```bash
cd /d/qqbot
docker compose up -d          # 启动
docker compose ps             # 状态
docker compose logs -f napcat # 看日志（二维码在这里）
docker compose logs -f astrbot
docker compose restart        # 重启
docker compose down           # 停止（数据保留）
```

## 访问地址

- AstrBot WebUI：http://localhost:6185 （初始账号 `astrbot`，密码见 `docker compose logs astrbot` 启动横幅）
- NapCat WebUI：http://127.0.0.1:6099/webui?token=<.napcat_token 里的值>

## 接下来要做的

1. 用**小号**扫描 `qrcode\qrcode.png`（或看 `docker compose logs napcat` 里的二维码）登录 QQ。
2. AstrBot WebUI → `服务提供商` → 新建，选 DeepSeek/OpenAI 兼容，填 API Key。
3. AstrBot WebUI → `对话` → 把默认模型设成刚加的那个。
4. 日志里出现 `aiocqhttp(OneBot v11) 适配器已连接` 即打通。群里 @ 机器人就能聊。
5. 想用「大肥鱼」人格：AstrBot WebUI → `人格` → 新建，内容照 `persona-feiyu.md` 第三节粘，导入步骤见该文档第六节。

## 端口

| 服务 | 端口 | 暴露范围 |
|---|---|---|
| AstrBot WebUI | 6185 | 0.0.0.0 |
| NapCat WebUI | 6099 | 仅 127.0.0.1 |
| NapCat → AstrBot 反向 WS | 6199 | 仅 docker 内网 |
