# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖并整理 (只有 go.mod/go.sum 变化时才会重新执行)
RUN go mod download && go mod tidy

# 复制源代码 (只有源代码变化时才会重新执行)
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 复制静态文件
COPY --from=builder /app/static ./static

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./main"]

