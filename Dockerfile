# 使用官方 Go 镜像作为构建环境
FROM golang:1.21 AS builder

# 设置工作目录
WORKDIR /app

# 将 go.mod 和 go.sum 复制到工作目录
COPY go.mod .
COPY go.sum .

# 设置 GOPROXY 环境变量
ENV GOPROXY=https://goproxy.cn,direct

# 下载依赖
RUN go mod download

# 将项目中的所有文件复制到工作目录
COPY . .

# 编译应用程序
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./server

# 使用 scratch 作为运行环境
FROM scratch

# 从构建器镜像中复制编译好的应用程序
COPY --from=builder /app/main .

# 复制静态文件和资源
COPY --from=builder /app/static ./static
COPY --from=builder /app/resources ./resources

# 暴露端口
EXPOSE 8333

# 运行应用程序
CMD ["./main"]
