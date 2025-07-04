#!/bin/sh
set -e

echo "============ 数据库初始化脚本启动 ============"

MYSQL_HOST=${MYSQL_HOST:-mysql}
MYSQL_PORT=${MYSQL_PORT:-3306}
MYSQL_DATABASE=${MYSQL_DATABASE:-portal}
MYSQL_USERNAME=${MYSQL_USERNAME:-root}
MYSQL_PASSWORD=${MYSQL_PASSWORD:-123456}

# 等待 MySQL 服务就绪
echo "等待 MySQL 服务就绪..."
MAX_TRIES=30
TRIES=0
while [ $TRIES -lt $MAX_TRIES ]; do
    if mysqladmin ping -h"$MYSQL_HOST" -P"$MYSQL_PORT" \
        -u"$MYSQL_USERNAME" -p"$MYSQL_PASSWORD" \
        --silent --protocol=TCP
    then
        echo "MySQL连接成功"
        break
    fi
    TRIES=$((TRIES+1))
    echo "等待MySQL连接... 尝试 $TRIES/$MAX_TRIES"
    sleep 2
done

if [ $TRIES -eq $MAX_TRIES ]; then
    echo "错误: 无法连接到MySQL，超过最大尝试次数"
    exit 1
fi

echo "创建数据库（如不存在）: $MYSQL_DATABASE"
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USERNAME" -p"$MYSQL_PASSWORD" \
      --protocol=TCP -e "CREATE DATABASE IF NOT EXISTS $MYSQL_DATABASE;"

echo "开始执行迁移脚本并创建管理员..."
# 使用 go run 执行迁移脚本
cd /app/backend && go run utils/db/migrate.go -create-admin=true

echo "============ 数据库初始化完成 ============"
exit 0