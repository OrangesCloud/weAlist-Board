# weAlist Kanban Service

칸반 보드 관리 시스템 (Workspace → Project → Ticket → Task)

> **Cloud Native Ready**: Kubernetes 배포를 위한 베이스 애플리케이션

이 프로젝트는 향후 Kubernetes 환경으로 마이그레이션하기 위한 베이스 애플리케이션입니다.
현재는 Docker Compose로 실행하며, K8s 배포에 필요한 기능들이 이미 구현되어 있습니다.

## 기술 스택

### Backend
- Python 3.11
- FastAPI (async/await)
- SQLAlchemy 2.0 (ORM)
- Alembic (DB Migration)
- Pydantic v2 (Validation)

### Infrastructure
- PostgreSQL 16 (공통 인프라)
- Redis 7 (공통 인프라)
- Docker & Docker Compose

### Cloud Native Features
- ✅ Health Check Endpoints (Liveness/Readiness Probes)
- ✅ Structured Logging (JSON)
- ✅ Graceful Shutdown
- ✅ Database Migration (Alembic)
- ✅ 12-Factor App Compliance
- ✅ Stateless Design (샤딩 대비)

## 실행 방법

### 1. 공통 인프라 설정 (최초 1회)

```bash
# infrastructure 디렉토리로 이동
cd infrastructure

# 환경변수 파일 생성
cp .env.example .env

# .env 파일에서 비밀번호 변경 (중요!)
# POSTGRES_PASSWORD와 REDIS_PASSWORD를 변경하세요

# 인프라 시작
docker-compose up -d
```

### 2. Kanban 서비스 실행

```bash
# services/kanban 디렉토리로 이동
cd services/kanban

# 환경변수 파일 생성
cp .env.example .env

# .env 파일에서 비밀번호를 infrastructure/.env와 동일하게 설정

# Kanban 서비스 시작
docker-compose up -d

# 로그 확인
docker-compose logs -f
```

## API

- **Workspace**: 워크스페이스 관리
- **Project**: 프로젝트 관리 (상태, 우선순위)
- **Ticket**: 티켓 관리 (상태, 우선순위)
- **Task**: 작업 관리 (완료 처리 포함)

### API 문서
- **Swagger UI**: http://localhost:8000/docs
- **Health Check**: http://localhost:8000/health
- **Liveness Probe**: http://localhost:8000/health/live
- **Readiness Probe**: http://localhost:8000/health/ready

## 테스트

```bash
# 테스트 실행 (커버리지 포함)
docker-compose exec kanban pytest --cov

# 테스트만 실행
docker-compose exec kanban pytest -v
```

- 총 36개 테스트
- 커버리지 97%+

## 개발

```bash
# Kanban 서비스 컨테이너 중지
cd services/kanban
docker-compose down

# 재빌드
docker-compose up -d --build

# 인프라 중지 (주의: 다른 서비스도 영향받음!)
cd infrastructure
docker-compose down
```

## 다른 팀과 협업

이 프로젝트는 weAlist 공통 인프라를 사용합니다.
다른 팀(예: Member 팀)과 인프라를 공유하는 방법은 `infrastructure/SHARING.md`를 참고하세요.

---

## 프로젝트 문서

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 샤딩 대비 설계 (FK 제거)
- **[K8S_READY.md](K8S_READY.md)** - Kubernetes 준비 사항 및 배포 가이드
- **[infrastructure/SHARING.md](infrastructure/SHARING.md)** - 팀 간 인프라 공유 가이드

---

## 향후 계획 (Phase 2)

이 베이스 애플리케이션을 기반으로 다음 작업 예정:

1. **Kubernetes 마이그레이션**
   - Helm Chart 작성
   - ConfigMap/Secret 분리
   - HPA (Horizontal Pod Autoscaler) 설정

2. **CI/CD 파이프라인**
   - GitHub Actions
   - 자동 빌드 & 배포
   - 컨테이너 이미지 레지스트리

3. **모니터링 & 로깅**
   - Prometheus + Grafana
   - ELK Stack 또는 Loki
   - 분산 추적 (Jaeger)

4. **보안 강화**
   - Network Policy
   - RBAC 설정
   - Secret 암호화 (Sealed Secrets)
   - Security Scanning

---

## 기여

이 프로젝트에서 사용한 도구 및 참고 자료:
- Claude Code (코드 작성 보조)
- FastAPI Documentation
- Kubernetes Best Practices
- 12-Factor App Methodology
