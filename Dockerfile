# 构建阶段
FROM golang:tip-alpine3.22 AS builder

# 设置工作目录
WORKDIR /app

# 设置国内镜像源，加速构建
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go mod 和 sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建可执行文件
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o qq-bot-star main.go

# 第二阶段：运行阶段
FROM alpine:latest

# 设置时区
ENV TZ=Asia/Shanghai

# 安装 ca-certificates 和 tzdata
RUN apk --no-cache add ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 从构建阶段复制可执行文件
COPY --from=builder /app/qq-bot-star .

# 复制数据库 schema
COPY --from=builder /app/database ./database

# 创建 data 目录
RUN mkdir -p /app/data /app/logs

# 暴露端口（如果需要的话）
EXPOSE 8080

# 启动命令
CMD ["/app/qq-bot-star"]
