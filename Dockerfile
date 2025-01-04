# 使用官方Golang镜像作为构建环境
FROM golang:1.22

# 设置环境变量
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct
ENV GIN_MODE=release
# ENV PORT=8000
ENV GOOS=linux

# 设置工作目录
WORKDIR /app

# 将Go模块配置和源代码复制到容器中
COPY go.mod ./
COPY go.sum ./
RUN go mod download

# 将源代码复制到容器中
COPY . .

# 构建Go应用程序
RUN go build -o myapp .



# 从builder镜像中复制编译后的二进制文件
COPY . .

# 暴露端口供Web应用使用
EXPOSE 8000

# 容器启动时运行Web应用程序
CMD ["./myapp"]

