# 构建阶段
FROM golang:tip-alpine3.22 AS builder

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct

WORKDIR /app

# 换阿里源 + 更新包列表 + 安装 build-base（支持 cgo）
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache build-base

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译（启用 CGO 支持 sqlite3）
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o qq-bot .

# 运行阶段
FROM golang:tip-alpine3.22

# 换阿里源，安装 tzdata 和 sqlite3 运行依赖
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update && \
    apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 创建非 root 用户
RUN adduser -D -g '' qqbot

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/qq-bot .
COPY --from=builder /app/database /app/database

# 创建数据目录和日志目录并设置权限
RUN mkdir -p /app/data /app/logs && chown -R qqbot:qqbot /app

# 暴露端口
EXPOSE 8080

# 切换到非 root 用户
USER qqbot

# 运行
CMD ["./qq-bot"]
