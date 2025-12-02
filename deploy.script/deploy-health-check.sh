#!/bin/bash
# CodeDeploy Hook: ValidateService
# 새로 배포된 Board Service의 헬스 체크

set -euo pipefail

SERVICE_PORT=8000 # Board Service 포트
HEALTH_ENDPOINT="/api/boards/health" # Board Service 헬스체크 엔드포인트
MAX_ATTEMPTS=30  # 서비스 시작 시간을 고려하여 증가
WAIT_SECONDS=5

echo "🏥 Starting health check for board-service on port ${SERVICE_PORT}..."

# 먼저 컨테이너가 실행 중인지 확인
echo "🔍 Checking if container is running..."
CONTAINER_EXISTS=$(sudo docker ps | grep "wealist-board-service" | wc -l)
if [ "$CONTAINER_EXISTS" -eq 0 ]; then
    echo "❌ Container wealist-board-service is not running!"
    sudo docker ps -a | grep "wealist-board-service" || echo "Container not found"
    exit 1
fi
echo "✅ Container is running"

for i in $(seq 1 $MAX_ATTEMPTS); do
    # 컨테이너 상태 확인
    CONTAINER_STATUS=$(sudo docker inspect --format='{{.State.Status}}' wealist-board-service 2>/dev/null || echo "not_found")
    
    if [ "$CONTAINER_STATUS" != "running" ]; then
        echo "⚠️ Attempt $i/$MAX_ATTEMPTS: Container status is ${CONTAINER_STATUS}. Waiting ${WAIT_SECONDS}s..."
        sleep $WAIT_SECONDS
        continue
    fi
    
    # Health endpoint 확인
    STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${SERVICE_PORT}${HEALTH_ENDPOINT}" || true)
    
    if [ "$STATUS_CODE" -eq 200 ]; then
        echo "✅ Health check succeeded! (HTTP 200)"
        exit 0
    elif [ "$STATUS_CODE" -eq 000 ]; then
        echo "⏳ Attempt $i/$MAX_ATTEMPTS: Service not yet reachable. Waiting ${WAIT_SECONDS}s..."
    else
        echo "⚠️ Attempt $i/$MAX_ATTEMPTS: Service responded with HTTP ${STATUS_CODE}. Waiting ${WAIT_SECONDS}s..."
    fi
    
    sleep $WAIT_SECONDS
done

echo "❌ Health check failed after ${MAX_ATTEMPTS} attempts."
echo "📋 Container logs (last 20 lines):"
sudo docker logs wealist-board-service --tail 20
exit 1