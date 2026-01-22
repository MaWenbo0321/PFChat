#!/bin/bash
# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   停止 PFChat 项目服务${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 停止前端
if [ -f "logs/frontend.pid" ]; then
    FRONTEND_PID=$(cat logs/frontend.pid)
    echo -e "${YELLOW}[前端] 停止前端服务 (PID: $FRONTEND_PID)...${NC}"
    sudo kill $FRONTEND_PID 2>/dev/null
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[✓] 前端服务已停止${NC}"
    else
        echo -e "${RED}[!] 前端进程不存在或已停止${NC}"
    fi
    rm logs/frontend.pid
else
    echo -e "${YELLOW}[!] 未找到前端 PID 文件${NC}"
fi
echo ""

# 停止后端
if [ -f "logs/backend.pid" ]; then
    BACKEND_PID=$(cat logs/backend.pid)
    echo -e "${YELLOW}[后端] 停止后端服务 (PID: $BACKEND_PID)...${NC}"
    sudo kill $BACKEND_PID 2>/dev/null
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[✓] 后端服务已停止${NC}"
    else
        echo -e "${RED}[!] 后端进程不存在或已停止${NC}"
    fi
    rm logs/backend.pid
else
    echo -e "${YELLOW}[!] 未找到后端 PID 文件${NC}"
fi
echo ""

# 停止 MySQL
echo -e "${YELLOW}[MySQL] 停止 MySQL 数据库...${NC}"
cd docker || exit
if command -v docker-compose &> /dev/null; then
    sudo docker-compose down
else
    sudo docker compose down
fi
if [ $? -eq 0 ]; then
    echo -e "${GREEN}[✓] MySQL 已停止${NC}"
else
    echo -e "${RED}[!] MySQL 停止失败${NC}"
fi
cd ..
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   ✨ 所有服务已停止！${NC}"
echo -e "${GREEN}========================================${NC}"