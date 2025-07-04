#!/bin/sh
# 禁用立即退出，让我们能看到所有错误
set +e

echo "============ 准备环境 ============"
# 创建根目录下的符号链接 - 这是解决问题的关键
echo "创建环境变量文件符号链接: /.env -> /app/.env"
ln -sf /app/.env /.env

# 创建backend目录结构(如果不存在)
echo "确保backend目录和日志目录存在"
mkdir -p /app/backend/logs
touch /app/backend/logs/portal.log  # 确保日志文件存在
chmod 755 /app/backend/logs
chmod 644 /app/backend/logs/portal.log  # 设置合适的文件权限

# 打印环境变量信息
echo "检查环境变量..."
if [ -f "/app/.env" ]; then
    echo "环境变量文件存在于 /app/.env"
    # 查看环境变量文件内容的前几行（不显示敏感信息）
    head -n 3 /app/.env | grep -v PASSWORD | grep -v SECRET | grep -v TOKEN
else
    echo "警告: 环境变量文件不存在于 /app/.env"
fi

# 导出环境变量到系统，但不覆盖已存在的变量
echo "导出环境变量到系统..."
if [ -f "/app/.env" ]; then
    # 仅当环境变量不存在时才从.env文件加载
    while IFS='=' read -r key value || [ -n "$key" ]; do
        # 跳过注释和空行
        case $key in
            \#*) continue ;;
            "") continue ;;
        esac
        
        # 除非是MYSQL_HOST或变量不存在，否则不覆盖
        if [ "$key" = "MYSQL_HOST" ] || [ -z "$(eval echo \$$key)" ]; then
            export "$key=$value"
        fi
    done < /app/.env
    
    # 确保MYSQL_HOST正确设置
    export MYSQL_HOST=mysql
    echo "已成功导出环境变量，并确保MYSQL_HOST=mysql"
else
    echo "注意: 将使用容器中已有的环境变量"
fi
# 检查关键环境变量是否存在
echo "检查关键环境变量..."
[ -z "$MYSQL_HOST" ] && echo "警告: MYSQL_HOST 未设置" || echo "MYSQL_HOST=$MYSQL_HOST"
[ -z "$JWT_SECRET" ] && echo "警告: JWT_SECRET 未设置" || echo "JWT_SECRET 已设置"

# 执行数据库初始化脚本
echo "============ 执行数据库初始化 ============"
if [ -f "/app/init-db.sh" ]; then
    echo "开始执行数据库初始化脚本..."
    /app/init-db.sh
    if [ $? -ne 0 ]; then
        echo "警告: 数据库初始化失败，但将继续启动应用..."
    else
        echo "数据库初始化成功"
    fi
else
    echo "错误: 找不到数据库初始化脚本，跳过初始化"
fi

# 启动Nginx
echo "启动Nginx..."
nginx
NGINX_STATUS=$?
if [ $NGINX_STATUS -ne 0 ]; then
    echo "Nginx启动失败，状态码: $NGINX_STATUS"
    nginx -t  # 测试配置
else
    echo "Nginx已启动"
fi

# 启动后端
echo "启动Go后端..."
cd /app
./server &
BACKEND_PID=$!
echo "后端进程ID: $BACKEND_PID"

# 显示所有运行的进程
echo "============ 所有启动的进程 ============"
ps aux

# 持续监控关键进程
echo "============ 持续监控中 ============"
while true; do
    sleep 10
    
    # 检查后端是否在运行
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        echo "警告: 后端进程已退出，尝试重启..."
        cd /app
        ./server &
        BACKEND_PID=$!
        echo "新的后端进程ID: $BACKEND_PID"
    fi
    
    # 检查Nginx是否在运行
    if ! pidof nginx >/dev/null; then
        echo "警告: Nginx进程已退出，尝试重启..."
        nginx
    fi
done