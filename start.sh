#!/bin/bash

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$PROJECT_ROOT/.env"
if [ ! -f "$ENV_FILE" ]; then
    echo "[Error] Missing $ENV_FILE. Copy .env.example to .env and fill in the secrets first."
    exit 1
fi
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a
cd "$PROJECT_ROOT" || exit 1

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}   PFChat 项目启动脚本 (Linux/Mac)${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 启动 MySQL (Docker)
echo -e "${YELLOW}[5/5] 启动服务...${NC}"
echo ""
echo -e "${BLUE}[MySQL] 启动 MySQL 数据库...${NC}"
cd docker || exit

# 使用 docker-compose 或 docker compose
if command -v docker-compose &> /dev/null; then
    sudo docker-compose --env-file "$ENV_FILE" up -d
else
    sudo docker compose --env-file "$ENV_FILE" up -d
fi

if [ $? -ne 0 ]; then
    echo -e "${RED}[错误] MySQL 启动失败${NC}"
    cd ..
    exit 1
fi
cd ..
echo -e "${GREEN}[✓] MySQL 已启动${NC}"
sleep 5
echo ""

# 启动后端
echo -e "${BLUE}[后端] 启动 Go 后端服务...${NC}"
cd backend || exit

# 使用 nohup 在后台运行
nohup go run . > ../logs/backend.log 2>&1 &
BACKEND_PID=$!
echo $BACKEND_PID > ../logs/backend.pid

if [ $? -ne 0 ]; then
    echo -e "${RED}[错误] 后端启动失败${NC}"
    cd ..
    exit 1
fi
cd ..
echo -e "${GREEN}[✓] 后端服务已启动 (PID: $BACKEND_PID, 端口: 8080)${NC}"
sleep 3
echo ""

# 启动前端
echo -e "${BLUE}[前端] 启动 Vue 前端服务...${NC}"
cd frontend || exit

# 检查是否安装了依赖
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}[提示] 检测到未安装依赖，正在安装...${NC}"
    npm install
    if [ $? -ne 0 ]; then
        echo -e "${RED}[错误] 依赖安装失败${NC}"
        cd ..
        exit 1
    fi
fi

# 使用 nohup 在后台运行
nohup npm run dev > ../logs/frontend.log 2>&1 &
FRONTEND_PID=$!
echo $FRONTEND_PID > ../logs/frontend.pid

if [ $? -ne 0 ]; then
    echo -e "${RED}[错误] 前端启动失败${NC}"
    cd ..
    exit 1
fi
cd ..
echo -e "${GREEN}[✓] 前端服务已启动 (PID: $FRONTEND_PID, 端口: 3000)${NC}"
echo ""

# 完成
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}   ✨ 所有服务启动完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}📍 访问地址:${NC}"
echo -e "   前端: ${GREEN}http://localhost:3000${NC}"
echo -e "   后端: ${GREEN}http://localhost:8080${NC}"
echo ""
echo -e "${BLUE}📝 管理员账号:${NC}"
echo -e "   如需初始创建，请在 .env 中设置 DEFAULT_ADMIN_USERNAME 和 DEFAULT_ADMIN_PASSWORD${NC}"
echo ""
echo -e "${BLUE}📊 进程信息:${NC}"
echo -e "   后端 PID: ${GREEN}$BACKEND_PID${NC}"
echo -e "   前端 PID: ${GREEN}$FRONTEND_PID${NC}"
echo ""
echo -e "${BLUE}📋 日志文件:${NC}"
echo -e "   后端: ${GREEN}logs/backend.log${NC}"
echo -e "   前端: ${GREEN}logs/frontend.log${NC}"
echo ""
echo -e "${BLUE}💡 管理命令:${NC}"
echo -e "   查看日志: ${GREEN}tail -f logs/backend.log${NC}"
echo -e "           ${GREEN}tail -f logs/frontend.log${NC}"
echo -e "   停止服务: ${GREEN}./stop.sh${NC}"
echo ""
