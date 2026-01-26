#!/bin/bash

# YouKong 数据库初始化脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=============================="
echo "YouKong 数据库初始化"
echo "=============================="

# 检查参数
if [ -z "$DB_HOST" ] || [ -z "$DB_USER" ] || [ -z "$DB_PASSWORD" ] || [ -z "$DB_NAME" ]; then
  echo -e "${RED}请设置数据库环境变量:${NC}"
  echo "export DB_HOST=your_db_host"
  echo "export DB_USER=your_db_user"
  echo "export DB_PASSWORD=your_db_password"
  echo "export DB_NAME=youkong"
  exit 1
fi

echo -e "${YELLOW}连接数据库: $DB_HOST${NC}"

# 创建数据库 (如果不存在)
echo -e "${YELLOW}创建数据库...${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASSWORD" -e "CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 执行迁移脚本
echo -e "${YELLOW}执行迁移脚本...${NC}"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" < /opt/youkong/migrations/001_init.sql

echo -e "${GREEN}=============================="
echo "数据库初始化完成!"
echo -e "==============================${NC}"
