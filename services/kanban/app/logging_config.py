import logging
import json
import sys
from datetime import datetime
from typing import Any, Dict
from app.config import settings


class JSONFormatter(logging.Formatter):
    """
    JSON 형식으로 로그를 출력하는 Formatter
    ELK Stack, Loki, CloudWatch 등에서 파싱하기 좋음
    """

    def format(self, record: logging.LogRecord) -> str:
        log_data: Dict[str, Any] = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": record.levelname,
            "logger": record.name,
            "message": record.getMessage(),
            "service": settings.PROJECT_NAME,
            "environment": settings.ENV,
        }

        # 추가 정보
        if hasattr(record, "extra"):
            log_data.update(record.extra)

        # 예외 정보
        if record.exc_info:
            log_data["exception"] = self.formatException(record.exc_info)

        # 함수, 파일, 라인 정보
        if settings.DEBUG:
            log_data["function"] = record.funcName
            log_data["file"] = record.pathname
            log_data["line"] = record.lineno

        return json.dumps(log_data, ensure_ascii=False)


def setup_logging() -> None:
    """
    로깅 설정

    개발 환경: 읽기 쉬운 포맷
    프로덕션: JSON 포맷 (로그 수집 도구용)
    """
    # Root logger 설정
    root_logger = logging.getLogger()

    # 기존 핸들러 제거
    root_logger.handlers.clear()

    # 로그 레벨 설정
    log_level = logging.DEBUG if settings.DEBUG else logging.INFO
    root_logger.setLevel(log_level)

    # Handler 생성
    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(log_level)

    # Formatter 설정
    if settings.ENV == "production":
        # 프로덕션: JSON 포맷
        formatter = JSONFormatter()
    else:
        # 개발: 읽기 쉬운 포맷
        formatter = logging.Formatter(
            fmt="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
            datefmt="%Y-%m-%d %H:%M:%S"
        )

    handler.setFormatter(formatter)
    root_logger.addHandler(handler)

    # uvicorn 로거 설정
    for logger_name in ["uvicorn", "uvicorn.access", "uvicorn.error"]:
        logger = logging.getLogger(logger_name)
        logger.handlers.clear()
        logger.addHandler(handler)
        logger.setLevel(log_level)
        logger.propagate = False

    # SQLAlchemy 로그 레벨 조정
    logging.getLogger("sqlalchemy.engine").setLevel(
        logging.INFO if settings.DEBUG else logging.WARNING
    )


# 애플리케이션용 logger 가져오기
def get_logger(name: str) -> logging.Logger:
    """
    애플리케이션용 logger 반환

    사용 예:
        from app.logging_config import get_logger
        logger = get_logger(__name__)
        logger.info("Hello", extra={"user_id": 123})
    """
    return logging.getLogger(name)
