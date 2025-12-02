#!/bin/bash
# =============================================================================
# CodeDeploy Hook: ApplicationStart (Board Service)
# SSM Parameter Store에서 Prod 환경 변수를 로드하고 Docker Compose를 실행합니다.
# =============================================================================

set -euo pipefail

# 1. 상수 정의
PROJECT_ROOT="/home/ec2-user/wealist"
COMPOSE_FILE="${PROJECT_ROOT}/docker/compose/docker-compose.ec2-prod.yml"
SERVICE_NAME="board-service"
AWS_REGION="ap-northeast-2"
PARAMETER_BASE_PATH="/wealist/prod"

echo "🚀 Board Service Production Deployment Start"
echo "Project Root: ${PROJECT_ROOT}"

# AWS CLI가 SSM 및 기타 API 호출에 사용할 기본 리전 환경 변수 강제 설정
export AWS_DEFAULT_REGION="${AWS_REGION}"

# 2. SSM Parameter 로드 함수 정의
load_param() {
    local name="$1"
    aws ssm get-parameter --name "${PARAMETER_BASE_PATH}/${name}" --query 'Parameter.Value' --output text 2>/dev/null || echo ""
}

load_secret() {
    local name="$1"
    aws ssm get-parameter --name "${PARAMETER_BASE_PATH}/${name}" --with-decryption --query 'Parameter.Value' --output text 2>/dev/null || echo ""
}

# 3. IMDS 로드 대기 및 인프라 환경 변수 로드 시작
echo "⏳ Waiting for IAM Role credentials to load via IMDS (15s delay)..."
sleep 15

echo "🔑 Loading secrets and endpoints from SSM Parameter Store..."

# AWS_ACCOUNT_ID 추출 및 검증
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text 2>/dev/null)
if [ -z "$AWS_ACCOUNT_ID" ] || [ "$AWS_ACCOUNT_ID" == "null" ]; then
    echo "❌ FATAL: Could not retrieve AWS Account ID using STS."
    exit 1
fi
export AWS_REGION="${AWS_REGION}"

echo "✅ AWS Account ID loaded: ${AWS_ACCOUNT_ID}"

# DB/Cache 엔드포인트
export RDS_HOST=$(load_param "db/rds_host")
export REDIS_HOST=$(load_param "cache/redis_host")
export REDIS_PORT=$(load_param "cache/redis_port")

# DB 이름
export USER_DB_NAME=$(load_secret "db/user_db_name")
export BOARD_DB_NAME=$(load_secret "db/board_db_name")

# 시크릿 정보
export JWT_SECRET=$(load_secret "jwt/jwt_secret")
export POSTGRES_SUPERUSER=$(load_param "db/rds_master_username")
export POSTGRES_SUPERUSER_PASSWORD=$(load_secret "db/rds_master_password")
export REDIS_PASSWORD=$(load_secret "cache/redis_auth_token")

# Redis AUTH가 비활성화된 경우 빈 문자열로 처리
if [ "$REDIS_PASSWORD" == "NONE" ]; then
    export REDIS_PASSWORD=""
fi

# Board Service DB 접속 시크릿
export BOARD_DB_USER=$(load_secret "db/board_db_user")
export BOARD_DB_PASSWORD=$(load_secret "db/board_db_password")

# User Service DB 접속 시크릿 (User Service가 필요할 경우)
export USER_DB_USER=$(load_secret "db/user_db_user")
export USER_DB_PASSWORD=$(load_secret "db/user_db_password")

# OAuth 및 S3 설정
export S3_BUCKET=$(load_param "s3/bucket")
export S3_REGION="${AWS_REGION}"
export CORS_ORIGINS="*"

# Exporter Ports
export NODE_EXPORTER_PORT=9100

# 4. 이미지 버전 환경 변수 설정
echo "🏷️ Loading image versions from SSM Parameter Store..."

# Board Service 버전 (현재 배포할 버전)
export BOARD_SERVICE_VERSION=$(load_param "version/board_service")
if [ -z "$BOARD_SERVICE_VERSION" ]; then
  echo "⚠️ Board Service version not found in SSM. Using latest tag."
  export BOARD_SERVICE_VERSION="latest"
fi


echo "Board Service Tag: ${BOARD_SERVICE_VERSION}"

# 5. Docker Compose 실행
if docker compose version &> /dev/null; then
  COMPOSE_CMD="docker compose"
else
  COMPOSE_CMD="docker-compose"
fi

# ECR 로그인
echo "🐳 Logging into ECR..."
aws ecr get-login-password --region ${AWS_REGION} | \
  sudo docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com

# 6. 최신 이미지 Pull (board-service만)
echo "🐳 Pulling image: ${SERVICE_NAME}:${BOARD_SERVICE_VERSION}"
sudo -E $COMPOSE_CMD -f "${COMPOSE_FILE}" pull "${SERVICE_NAME}"

# 7. Docker Compose 실행 (board-service만 재시작)
echo "🔄 Starting Board Service defined in ${COMPOSE_FILE}..."

SERVICES_TO_RESTART="board-service"

# 기존 Board Service 컨테이너만 직접 중지 및 제거 (다른 서비스에 영향 없음)
echo "🛑 Stopping and removing existing Board Service container (direct Docker commands)..."
if sudo docker ps -a | grep -q "wealist-board-service"; then
    echo "  Found existing board-service container, removing it..."
    sudo docker stop wealist-board-service 2>/dev/null || true
    sudo docker rm -f wealist-board-service 2>/dev/null || true
    echo "  ✅ Old container removed"
else
    echo "  No existing board-service container found"
fi

# 컨테이너 정리 대기
sleep 2

# 🚨 --no-deps --force-recreate 를 사용하여 Board Service만 재시작 (다른 서비스에 영향 없음)
echo "🚀 Starting new board-service container..."
sudo -E $COMPOSE_CMD -f "${COMPOSE_FILE}" up -d --no-deps --force-recreate ${SERVICES_TO_RESTART}

# 컨테이너 시작 대기
sleep 3

# 컨테이너 상태 확인
echo "📊 Checking container status..."

# Node Exporter가 실행 중이 아니면 시작 (최초 배포 시에만)
echo "🔍 Ensuring node-exporter is running..."
sudo -E $COMPOSE_CMD -f "${COMPOSE_FILE}" up -d --no-deps node-exporter 2>/dev/null || true

echo "✅ Deployment initiated. CodeDeploy will now run ValidateService."