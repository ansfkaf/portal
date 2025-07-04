# 多阶段构建的Dockerfile，包含React前端和Go后端

# 第一阶段：构建前端
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend

# 复制package文件并安装依赖
COPY frontend/package*.json ./
RUN npm ci

# 复制前端源代码
COPY frontend/ ./

# 构建前端
RUN npm run build

# 第二阶段：构建后端
FROM golang:1.22.1-alpine AS backend-builder
WORKDIR /app/backend

# 安装构建依赖
RUN apk add --no-cache gcc musl-dev

# 复制go.mod和go.sum并下载依赖
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# 复制后端源代码
COPY backend/ ./

# 构建后端二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# 第三阶段：最终镜像
FROM debian:stable-slim

# 防止 tzdata 等交互配置
ENV DEBIAN_FRONTEND=noninteractive
# 设置时区为东八区（上海/中国）
ENV TZ=Asia/Shanghai

# 安装 Nginx、MySQL 客户端和其他必须工具
RUN apt-get update && \
apt-get install -y --no-install-recommends \
    nginx \
    default-mysql-client \
    ca-certificates \
    golang \
    tzdata \
&& apt-get clean && rm -rf /var/lib/apt/lists/* \
&& ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 设置 Nginx
COPY --from=frontend-builder /app/frontend/dist /var/www/html
COPY nginx.conf /etc/nginx/nginx.conf

# 创建应用目录结构
WORKDIR /app
COPY --from=backend-builder /app/backend/server /app/server

# 创建日志目录
RUN mkdir -p /app/backend/logs && chmod 755 /app/backend/logs

# 同步后端源码（如果需要 go run ... 之类）
COPY --from=backend-builder /app/backend /app/backend

# 复制 .env 文件到 /app/.env（如果需要）
COPY .env /app/.env

# 复制初始化脚本和启动脚本
COPY init-db.sh /app/init-db.sh
COPY app.sh /app.sh
RUN chmod +x /app.sh /app/init-db.sh

# 修改 nginx 配置中的用户
RUN sed -i 's/user nginx;/user www-data;/' /etc/nginx/nginx.conf || echo "Nginx user config not found"

# 暴露端口
EXPOSE 81 8080

# 设置 Go 工具链环境变量
ENV GOTOOLCHAIN=auto

# 入口命令
CMD ["/app.sh"]