from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from app.config import settings
from app.database import engine, Base
from app.logging_config import setup_logging, get_logger

# 로깅 설정
setup_logging()
logger = get_logger(__name__)

from app.models import (
    Workspace,
    Project,
    Ticket,
    Task
)

from app.api import workspaces, projects, tickets, tasks, health

app = FastAPI(
    title=settings.PROJECT_NAME,
    version=settings.VERSION,
    description="Kanban Board Management API - Cloud Native Ready",
    docs_url="/docs" if settings.DEBUG else None,
    redoc_url="/redoc" if settings.DEBUG else None,
    openapi_tags=[
        {"name": "health", "description": "Health check endpoints for K8s probes"},
        {"name": "workspaces", "description": "Workspace management"},
        {"name": "projects", "description": "Project management"},
        {"name": "tickets", "description": "Ticket management"},
        {"name": "tasks", "description": "Task management"},
    ]
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.on_event("startup")
async def startup_event():
    """
    애플리케이션 시작 시 실행

    개발 환경: 자동으로 테이블 생성 (빠른 개발용)
    프로덕션: Alembic 마이그레이션 사용 (alembic upgrade head)
    """
    logger.info(
        "Starting application",
        extra={
            "service": settings.PROJECT_NAME,
            "version": settings.VERSION,
            "environment": settings.ENV
        }
    )

    if settings.ENV == "development":
        # 개발 환경에서만 자동 테이블 생성
        logger.info("Creating database tables (development mode)")
        Base.metadata.create_all(bind=engine)
    else:
        logger.info("Skipping table creation (use Alembic for production)")


@app.on_event("shutdown")
async def shutdown_event():
    """
    애플리케이션 종료 시 실행 (Graceful Shutdown)

    - DB 연결 정리
    - 진행 중인 요청 완료 대기
    - K8s에서 SIGTERM 받을 때 실행됨
    """
    logger.info("Shutting down application gracefully")

    # DB 엔진 정리
    engine.dispose()
    logger.info("Database connections closed")

@app.get("/")
async def root():
    return {
        "message": f"Welcome to {settings.PROJECT_NAME}",
        "version": settings.VERSION,
        "docs": "/docs" if settings.DEBUG else "disabled",
        "health": "/health"
    }

# Health check routes (K8s probes)
app.include_router(health.router, tags=["health"])

# API routes
app.include_router(workspaces.router, prefix="/api/workspaces", tags=["workspaces"])
app.include_router(projects.router, prefix="/api/projects", tags=["projects"])
app.include_router(tickets.router, prefix="/api/tickets", tags=["tickets"])
app.include_router(tasks.router, prefix="/api/tasks", tags=["tasks"])
