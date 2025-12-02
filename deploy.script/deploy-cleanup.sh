#!/bin/bash
# CodeDeploy Hook: BeforeInstall
# 이전 배포에서 남은 Board Service 컨테이너 중지 및 이미지 정리

set +e # 오류가 나도 계속 진행 (컨테이너가 없을 수 있음)

PROJECT_ROOT="/home/ec2-user/wealist"
COMPOSE_FILE="${PROJECT_ROOT}/docker/compose/docker-compose.ec2-prod.yml"
echo "🧹 Cleaning up old board-service containers..."

# 1. Docker Compose 명령어가 무엇인지 확인합니다.
if docker compose version &> /dev/null; then
  COMPOSE_CMD="docker compose"
else
  COMPOSE_CMD="docker-compose"
fi

# 2. 기존 Board Service 컨테이너만 직접 중지 및 삭제 (다른 서비스에 영향 없음)
echo "🛑 Stopping board-service container only..."
if sudo docker ps -a | grep -q "wealist-board-service"; then
    sudo docker stop wealist-board-service 2>/dev/null || true
    sudo docker rm -f wealist-board-service 2>/dev/null || true
    echo "  ✅ Board service container removed"
else
    echo "  No existing board-service container found"
fi

# ⚠️ docker-compose 파일 삭제 제거: 다른 서비스가 사용 중일 수 있음
# CodeDeploy가 자동으로 덮어쓰기 함

# 3. 🧹 Board Service 관련 임시 헬스체크 컨테이너만 정리
echo "🧹 Cleaning up board-service temporary health check containers..."

# Board Service 관련 임시 컨테이너만 정리
sudo docker rm -f temp-board-health 2>/dev/null || true

# 8000 포트(Board Service)만 정리 - User Service(8080)는 건드리지 않음
echo "  - Checking port 8000 (board-service)..."
PID=$(sudo lsof -ti:8000 2>/dev/null || true)
if [ -n "$PID" ]; then
    echo "    Found process ${PID} using port 8000, killing..."
    sudo kill -9 $PID 2>/dev/null || true
fi

echo "✅ Cleanup complete."
set -e