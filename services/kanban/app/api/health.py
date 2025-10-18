from fastapi import APIRouter, Depends, status
from sqlalchemy.orm import Session
from sqlalchemy import text
from app.database import get_db
from app.config import settings
import redis

router = APIRouter()

@router.get("/health")
async def health_check():
    """기본 Health Check - 서비스가 실행 중인지만 확인"""
    return {
        "status": "healthy",
        "service": settings.PROJECT_NAME,
        "version": settings.VERSION,
        "environment": settings.ENV
    }

@router.get("/health/live")
async def liveness_probe():
    """
    Kubernetes Liveness Probe
    - 애플리케이션이 살아있는지 확인
    - 실패 시 Pod 재시작
    """
    return {
        "status": "alive",
        "service": settings.PROJECT_NAME
    }

@router.get("/health/ready")
async def readiness_probe(db: Session = Depends(get_db)):
    """
    Kubernetes Readiness Probe
    - 애플리케이션이 트래픽을 받을 준비가 되었는지 확인
    - DB, Redis 등 의존성 체크
    - 실패 시 트래픽 라우팅 중지 (재시작 안함)
    """
    health_status = {
        "status": "ready",
        "service": settings.PROJECT_NAME,
        "checks": {}
    }

    all_healthy = True

    # 1. Database 연결 확인
    try:
        db.execute(text("SELECT 1"))
        health_status["checks"]["database"] = "healthy"
    except Exception as e:
        health_status["checks"]["database"] = f"unhealthy: {str(e)}"
        all_healthy = False

    # 2. Redis 연결 확인
    try:
        redis_client = redis.from_url(settings.REDIS_URL, decode_responses=True)
        redis_client.ping()
        health_status["checks"]["redis"] = "healthy"
        redis_client.close()
    except Exception as e:
        health_status["checks"]["redis"] = f"unhealthy: {str(e)}"
        all_healthy = False

    if not all_healthy:
        health_status["status"] = "not_ready"
        return health_status, status.HTTP_503_SERVICE_UNAVAILABLE

    return health_status
