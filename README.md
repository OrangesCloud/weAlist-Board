# Board Service

> 클라우드 네이티브 마이크로서비스 기반 프로젝트 관리 시스템

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 개요

weAlist의 **보드 관리 마이크로서비스**로, 프로젝트 및 칸반 보드 기능을 제공합니다. Clean Architecture와 12-Factor App 원칙을 따라 설계되었으며, 독립적인 배포와 수평 확장이 가능합니다.

### 핵심 특징

- ⚡️ **고성능**: Go/Gin 프레임워크 기반 경량 서비스
- 🏗️ **Clean Architecture**: 계층 분리 및 의존성 역전 원칙 적용
- 🔄 **무상태(Stateless)**: JWT 인증 기반 수평 확장 가능
- 📦 **컨테이너화**: Docker 멀티스테이지 빌드로 최적화
- 🚀 **CI/CD**: GitHub Actions 자동 배포 파이프라인
- 📊 **관찰 가능성**: 구조화된 로깅, 메트릭, 헬스 체크

## 기술 스택

| 분류 | 기술 |
|------|------|
| **언어** | Go 1.25+ |
| **프레임워크** | Gin (HTTP), GORM (ORM) |
| **데이터베이스** | PostgreSQL 17 |
| **캐시** | Redis 7 |
| **인증** | JWT (HS512) |
| **로깅** | Uber Zap |
| **문서화** | Swagger/OpenAPI |
| **컨테이너** | Docker, Docker Compose |
| **CI/CD** | GitHub Actions, AWS ECR |

## 빠른 시작

### 1. 사전 요구사항

- Docker & Docker Compose (필수)
- Go 1.25 이상 (로컬 실행 시)

### 2. 환경 설정

```bash
# 환경 변수 파일 복사
cp docker/env/.env.dev.example docker/env/.env.dev

# 필요시 환경 변수 수정
# - DATABASE_URL: PostgreSQL 연결 문자열
# - SECRET_KEY: JWT 서명 키 (최소 64바이트)
# - USER_SERVICE_URL: User Service 엔드포인트 (독립 실행 시 생략 가능)
```

### 3. 실행 방법

#### Docker Compose - 독립 개발 환경 (권장)

board-service를 user-service 없이 독립적으로 실행합니다. Kubernetes Pod 환경을 시뮬레이션합니다.

```bash
# 개발 환경 시작 (board-service + PostgreSQL + Redis + MinIO)
docker compose -f docker/compose/docker-compose.dev.yml up -d

# 로그 확인
docker compose -f docker/compose/docker-compose.dev.yml logs -f board-service

# 서비스 중지
docker compose -f docker/compose/docker-compose.dev.yml down

# 볼륨 포함 완전 삭제
docker compose -f docker/compose/docker-compose.dev.yml down -v
```

**포트 접근:**
- Board Service API: http://localhost:8000
- PostgreSQL: localhost:5432
- Redis: localhost:6379
- MinIO API: http://localhost:9000
- MinIO Console: http://localhost:9001

**특징:**
- ✅ user-service 의존성 없음
- ✅ 모든 포트 직접 노출
- ✅ 빠른 시작 (60초 이내)
- ✅ 독립적인 개발 및 테스트

#### Docker Compose - 통합 환경

user-service와 함께 전체 스택을 실행합니다.

```bash
# 전체 서비스 시작 (user-service + board-service + nginx)
docker compose -f docker/compose/docker-compose.yml up -d

# 로그 확인
docker compose logs -f board-service

# 서비스 중지
docker compose down
```

#### 로컬 실행 (Go 직접 실행)

```bash
# 의존성 설치
go mod download

# 데이터베이스 마이그레이션
./scripts/db/apply_migrations.sh dev

# 서버 실행
go run cmd/api/main.go
```

### 4. 헬스 체크

```bash
# 서비스 상태 확인
curl http://localhost:8000/health

# API 문서 확인 (개발 환경)
open http://localhost:8000/swagger/index.html
```

## 주요 기능

### 프로젝트 관리
- 워크스페이스별 프로젝트 생성 및 조회
- 기본 프로젝트 자동 생성
- 프로젝트 멤버 권한 관리 (OWNER/ADMIN/MEMBER)

### 보드 관리
- 칸반 보드 CRUD (생성, 조회, 수정, 삭제)
- Fractional Indexing 기반 순서 관리 (O(1) 위치 변경)
- 커스텀 필드 지원 (Stage, Importance, Role)
- Soft Delete로 데이터 복구 가능

### 협업 기능
- 보드 참여자 관리 (추가/제거)
- 댓글 작성 및 스레드
- 파일 첨부 (S3 Presigned URL 방식)

### 실시간 동기화
- WebSocket 기반 실시간 업데이트
- 프로젝트별 채널 격리

## API 엔드포인트

### 기본 URL
```
로컬: http://localhost:8000/api
AWS:  https://api.wealist.co.kr/api/boards
```

### 주요 엔드포인트

| 기능 | 메서드 | 경로 | 설명 |
|------|--------|------|------|
| **프로젝트** | POST | `/projects` | 프로젝트 생성 |
| | GET | `/projects/workspace/:id` | 워크스페이스 프로젝트 목록 |
| **보드** | POST | `/boards` | 보드 생성 |
| | GET | `/boards/:id` | 보드 상세 조회 |
| | GET | `/boards/project/:id` | 프로젝트 보드 목록 |
| | PUT | `/boards/:id` | 보드 수정 |
| | PUT | `/boards/:id/move` | 보드 위치 이동 |
| | DELETE | `/boards/:id` | 보드 삭제 (soft) |
| **참여자** | POST | `/participants` | 참여자 추가 |
| | GET | `/participants/board/:id` | 참여자 목록 |
| **댓글** | POST | `/comments` | 댓글 작성 |
| | GET | `/comments/board/:id` | 댓글 목록 |
| **첨부파일** | POST | `/attachments/presigned-url` | 업로드 URL 생성 |

**전체 API 문서**: [Swagger UI](http://localhost:8000/swagger/index.html) 참조

## 프로젝트 구조

```
board-service/
├── cmd/api/              # 애플리케이션 진입점
├── internal/
│   ├── handler/          # HTTP 핸들러 (Presentation)
│   ├── service/          # 비즈니스 로직 (Application)
│   ├── repository/       # 데이터 접근 (Infrastructure)
│   ├── domain/           # 도메인 모델 (Domain)
│   ├── dto/              # 요청/응답 DTO
│   ├── middleware/       # 미들웨어 (인증, 로깅, CORS)
│   ├── config/           # 설정 관리
│   └── database/         # DB 연결 및 초기화
├── migrations/           # SQL 마이그레이션
├── docs/                 # Swagger 문서
├── scripts/              # 유틸리티 스크립트
│   ├── db/              # 데이터베이스 관리
│   └── integration-test.sh
├── docker/              # Docker 설정
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

**계층 설명**:
- **Handler**: HTTP 요청/응답 처리, DTO 검증
- **Service**: 비즈니스 로직, 트랜잭션 관리
- **Repository**: 데이터베이스 CRUD 연산
- **Domain**: 도메인 모델 및 비즈니스 규칙

> 자세한 아키텍처는 [ARCHITECTURE.md](ARCHITECTURE.md) 참조

## 아키텍처

### 마이크로서비스 구성

```
┌─────────────────────────────────────────────────┐
│              Application Load Balancer           │
│           https://api.wealist.co.kr             │
└────────────────┬────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ▼                 ▼
┌──────────────┐   ┌──────────────┐
│ User Service │   │Board Service │
│ (Spring Boot)│   │   (Go/Gin)   │
│   :8080      │   │    :8000     │
└──────┬───────┘   └──────┬───────┘
       │                  │
       ▼                  ▼
┌──────────────────────────────────┐
│     PostgreSQL (독립 DB)         │
│  ┌─────────────────────────┐    │
│  │ wealist_user_db         │    │
│  │ wealist_board_db        │    │
│  └─────────────────────────┘    │
└──────────────────────────────────┘
```

### 핵심 설계 원칙

#### 1. 서비스 독립성
- ✅ **독립 데이터베이스**: 각 서비스가 전용 DB 소유
- ✅ **No Foreign Keys**: 애플리케이션 레벨 관계 관리
- ✅ **API 통신**: 서비스 간 RESTful API로만 통신
- ✅ **독립 배포**: 서비스별 독립적 배포 및 버전 관리

#### 2. Clean Architecture
- **의존성 방향**: 외부 → 내부 (Handler → Service → Repository → Domain)
- **인터페이스 기반**: 구현체 교체 가능 (테스트 용이)
- **Rich Domain Model**: 비즈니스 로직을 도메인에 캡슐화 (26개 메서드)

#### 3. 확장성
- **무상태 설계**: 세션 없이 JWT 기반 인증
- **수평 확장**: 인스턴스 추가로 처리량 증가
- **샤딩 준비**: UUID 기반 분산 ID, FK 없음

> 자세한 내용은 [ARCHITECTURE.md](ARCHITECTURE.md) 참조

## 개발 가이드

### 독립 개발 워크플로우

board-service는 user-service 없이 독립적으로 개발할 수 있습니다.

```bash
# 1. 개발 환경 시작
docker compose -f docker/compose/docker-compose.dev.yml up -d

# 2. Health check 확인
curl http://localhost:8000/health

# 3. 코드 수정 후 재빌드
docker compose -f docker/compose/docker-compose.dev.yml up -d --build board-service

# 4. 로그 실시간 확인
docker compose -f docker/compose/docker-compose.dev.yml logs -f board-service

# 5. 테스트 실행
go test ./...

# 6. 개발 완료 후 정리
docker compose -f docker/compose/docker-compose.dev.yml down
```

**주요 특징:**
- ✅ user-service 의존성 없음 - 독립적 개발 가능
- ✅ 빠른 시작 - 60초 이내 모든 서비스 준비
- ✅ 직접 포트 접근 - nginx 없이 API 직접 호출
- ✅ 데이터베이스 직접 접근 - psql, Redis CLI 사용 가능
- ✅ Kubernetes 환경 시뮬레이션 - Pod 독립성 테스트

### 사용 가능한 명령어

```bash
# 빌드
make build              # 바이너리 빌드
make build-linux        # Linux용 빌드 (Docker)

# 테스트
make test               # 전체 테스트 실행
make test-coverage      # 커버리지 리포트 (HTML)

# 코드 품질
make fmt                # 코드 포맷팅
make lint               # Lint 검사
make check              # fmt + vet + lint

# 데이터베이스
make db-create          # 데이터베이스 생성
make migrate-up         # 마이그레이션 실행
make migrate-down       # 마이그레이션 롤백

# Docker
make docker-build       # 이미지 빌드
make docker-compose-up  # 전체 서비스 시작

# 통합 테스트
./scripts/integration-test.sh   # 전체 API 테스트
```

### 환경 변수

#### 원본 형식 (wealist-project 호환, 권장)

```bash
# 서버
SERVER_PORT=8000
ENV=dev                 # dev 또는 prod

# 데이터베이스
DATABASE_URL=postgresql://postgres:password@localhost:5432/wealist_board_db?sslmode=disable

# JWT
SECRET_KEY=your-secret-key-at-least-64-bytes

# 외부 서비스
USER_SERVICE_URL=http://user-service:8080

# CORS
CORS_ORIGINS=http://localhost:3000

# 로깅
LOG_LEVEL=info          # debug, info, warn, error

# S3 (첨부파일)
S3_BUCKET=wealist-dev-files
S3_REGION=ap-northeast-2
```

#### 현재 형식 (하위 호환)

```bash
# 서버
SERVER_PORT=8000
SERVER_MODE=debug       # debug 또는 release

# 데이터베이스
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=wealist_board_db
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRE_TIME=24h

# 외부 서비스
USER_API_BASE_URL=http://user-service:8080

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000

# 로깅
LOG_LEVEL=info
LOG_OUTPUT_PATH=stdout
```

> **설정 우선순위**: 환경 변수 > .env 파일 > config.yaml > 기본값

### 데이터베이스 마이그레이션

#### 개발 환경

```bash
# AutoMigrate 사용 (빠른 프로토타이핑)
ENV=dev USE_AUTO_MIGRATE=true go run cmd/api/main.go

# 스키마 덤프
./scripts/db/dump_schema.sh dev
```

#### 프로덕션 환경

```bash
# 수동 마이그레이션 (안전)
./scripts/db/apply_migrations.sh prod

# 롤백
./scripts/db/rollback.sh prod 20250106120000
```

#### 새 마이그레이션 생성

```bash
# 1. 도메인 모델 수정
# internal/domain/board.go

# 2. 개발 환경에서 AutoMigrate로 테스트
ENV=dev USE_AUTO_MIGRATE=true go run cmd/api/main.go

# 3. 스키마 덤프
./scripts/db/dump_schema.sh dev

# 4. 마이그레이션 파일 생성
# migrations/004_add_feature.sql
# migrations/004_add_feature_down.sql

# 5. 로컬 테스트
./scripts/db/apply_migrations.sh dev

# 6. 커밋 후 CI/CD가 자동 적용
git add migrations/
git commit -m "Add migration for new feature"
```

### Swagger 문서 생성

```bash
# Swagger 문서 재생성
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# 또는
make swagger

# 검증
./scripts/validate-swagger.sh
```

**Godoc 주석 예시**:

```go
// CreateBoard godoc
// @Summary      보드 생성
// @Description  새로운 보드를 생성합니다
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBoardRequest true "보드 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.BoardResponse}
// @Failure      400 {object} response.ErrorResponse
// @Router       /api/boards [post]
func (h *BoardHandler) CreateBoard(c *gin.Context) {
    // ...
}
```

## 배포 환경

weAlist는 **3가지 배포 환경**을 지원합니다:

| 환경 | 용도 | 인프라 | 접근 방식 | 자동 배포 |
|------|------|--------|-----------|----------|
| **Local** | 개발/디버깅 | Docker Compose | `localhost:8000` | ❌ |
| **EC2 Dev** | 통합 테스트 | EC2 단일 인스턴스 | `http://<EC2-IP>:8000` | ✅ CI/CD |
| **Production** | 운영 서비스 | AWS ALB + ECS/EC2 | `https://api.wealist.co.kr/api/boards` | ⚠️ 수동 |

---

### 1. Local 개발 환경 (Standalone)

**목적**: 로컬 머신에서 board-service 독립 개발 및 디버깅

**특징**:
- ✅ user-service 의존성 없음
- ✅ Kubernetes Pod 환경 시뮬레이션
- ✅ 빠른 시작 (60초 이내)
- ✅ 모든 포트 직접 노출

```bash
# 프로젝트 루트에서
cd board-service

# 독립 개발 환경 시작 (board-service + PostgreSQL + Redis + MinIO)
docker compose -f docker/compose/docker-compose.dev.yml up -d

# 로그 확인
docker compose -f docker/compose/docker-compose.dev.yml logs -f board-service

# 서비스 중지
docker compose -f docker/compose/docker-compose.dev.yml down
```

**접속**:
```bash
# API 직접 접근 (nginx 없음)
curl http://localhost:8000/health
curl http://localhost:8000/api/boards/...

# Swagger 문서
open http://localhost:8000/swagger/index.html

# PostgreSQL 직접 접근
psql -h localhost -p 5432 -U postgres -d wealist_board_db

# Redis 직접 접근
redis-cli -h localhost -p 6379 -a password

# MinIO Console
open http://localhost:9001
```

**환경 변수** (`docker/env/.env.dev`):
```bash
ENV=dev
SERVER_BASE_PATH=""                    # ALB 없음
DATABASE_URL=postgresql://postgres:password@postgres:5432/wealist_board_db
REDIS_URL=redis://:password@redis:6379/1
USER_SERVICE_URL=                      # 빈 값 (독립 실행)
USE_AUTO_MIGRATE=true                  # 자동 마이그레이션
LOG_LEVEL=debug
```

---

### 2. EC2 Dev 환경

**목적**: 팀 통합 테스트 및 QA 환경

**특징**:
- ✅ CI/CD 자동 배포 (main 브랜치 푸시 시)
- ✅ All-in-one 구성 (서비스 + DB + 모니터링)
- ✅ 낮은 비용 (~$15-20/월, t3.small)
- ⚠️ 프로덕션 사용 비권장

**EC2 인스턴스에서 수동 배포**:
```bash
# SSH 접속
ssh ubuntu@<EC2-PUBLIC-IP>

# wealist 디렉토리로 이동
cd /home/ubuntu/wealist

# 환경 변수 로드
source /home/ubuntu/.env.ec2-dev

# Docker Compose로 배포
docker-compose -f docker/compose/docker-compose.ec2-dev.yml up -d

# 헬스 체크
curl http://localhost:8000/health

# 로그 확인
docker-compose -f docker/compose/docker-compose.ec2-dev.yml logs -f board-service
```

**접속**:
```bash
# EC2 Public IP로 접근
curl http://<EC2-PUBLIC-IP>:8000/health
curl http://<EC2-PUBLIC-IP>:8000/api/boards/...
```

**환경 변수**:
```bash
ENV=dev
SERVER_BASE_PATH=""                    # ALB 없음
DATABASE_URL=postgresql://postgres:password@postgres:5432/wealist_board_db
USER_SERVICE_URL=http://user-service:8080
USE_AUTO_MIGRATE=false                 # 수동 마이그레이션
```

---

### 3. Production 환경

**목적**: 실제 운영 서비스

**특징**:
- ✅ AWS ALB를 통한 HTTPS 접근
- ✅ RDS PostgreSQL (Multi-AZ)
- ✅ ElastiCache Redis
- ✅ Auto Scaling
- ⚠️ 수동 배포 (Release Tag 생성 후)

**접속**:
```bash
# HTTPS로 접근 (ALB SSL Termination)
curl https://api.wealist.co.kr/api/boards/health
curl https://api.wealist.co.kr/api/boards/api/projects/...
```

**환경 변수**:
```bash
ENV=prod
SERVER_BASE_PATH="/api/boards"         # ALB 경로 재작성
DATABASE_URL=postgresql://user:pass@rds-endpoint.ap-northeast-2.rds.amazonaws.com/wealist_board_db
REDIS_URL=redis://elasticache-endpoint.cache.amazonaws.com:6379
USER_SERVICE_URL=http://user-service-internal:8080
USE_AUTO_MIGRATE=false                 # 수동 마이그레이션만
LOG_LEVEL=warn                         # 프로덕션 로그 최소화
```

**ALB Path-Based Routing**:
```
클라이언트:  https://api.wealist.co.kr/api/boards/health
     ↓
ALB:         /api/boards/health → /health (경로 재작성)
     ↓
Board Service: GET /health
```

**배포 프로세스**:
```bash
# 1. Release Tag 생성
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 2. GitHub Actions에서 수동 승인 후 배포
# 3. CloudWatch 모니터링 확인
# 4. Rollback 준비 (이전 버전 유지)
```

---

### CI/CD 파이프라인

**자동 배포 플로우**:

```
1. Push to main branch
   ↓
2. GitHub Actions CI
   - Go test 실행
   - Docker 이미지 빌드
   - ECR에 푸시
   ↓
3. GitHub Actions CD (자동 트리거)
   - Parameter Store에서 환경 변수 로드
   - ECR에서 이미지 풀
   - 데이터베이스 마이그레이션 실행
   - board-service 재시작
   - 헬스 체크 (60초 타임아웃)
   ↓
4. 배포 완료
```

**수동 배포**:

GitHub Actions UI에서 "CD - Dev Board Service" 워크플로우 실행

### AWS 프로덕션 (ALB)

```bash
# ALB 라우팅 확인
./scripts/verify-alb-setup.sh

# Target Group 헬스 체크
./scripts/check-alb-health.sh

# 서비스 상태 확인
curl https://api.wealist.co.kr/api/boards/health
```

## 트러블슈팅

### 데이터베이스 연결 실패

```bash
# PostgreSQL 실행 확인
pg_isready -h localhost -p 5432

# 데이터베이스 존재 확인
psql -U postgres -l | grep wealist_board_db

# 데이터베이스 재생성
make db-reset
```

### JWT 인증 실패

```bash
# Secret Key 확인 (User Service와 일치해야 함)
echo $SECRET_KEY

# JWT_SECRET 환경 변수 설정
export SECRET_KEY="your-shared-secret-key-at-least-64-bytes"
```

### Docker 컨테이너 문제

```bash
# 컨테이너 로그 확인
docker-compose logs board-service

# 컨테이너 재시작
docker-compose restart board-service

# 전체 재구성
docker-compose down -v
docker-compose up -d
```

### User Service 통신 오류

**독립 개발 환경 (docker-compose.dev.yml)**:

user-service 없이 실행하므로 통신 오류는 정상입니다. board-service는 user-service 의존성 없이 동작하도록 설계되었습니다.

```bash
# Health check는 user-service 없이도 성공
curl http://localhost:8000/health
# 응답: {"status":"healthy","dependencies":{"database":"connected","redis":"connected"}}

# user-service 필요한 API는 503 반환 (정상 동작)
curl http://localhost:8000/api/workspaces/123/profile
# 응답: {"error":"Service Unavailable","message":"User service is currently unavailable"}
```

**통합 환경 (docker-compose.yml)**:

```bash
# User Service URL 확인
echo $USER_SERVICE_URL

# Docker 네트워크에서는 서비스 이름 사용
# ✅ USER_SERVICE_URL=http://user-service:8080
# ❌ USER_SERVICE_URL=http://localhost:8080

# 연결 테스트
curl $USER_SERVICE_URL/health
```

### 독립 개발 환경 문제

```bash
# 컨테이너 상태 확인
docker compose -f docker/compose/docker-compose.dev.yml ps

# 모든 서비스가 healthy인지 확인
docker compose -f docker/compose/docker-compose.dev.yml ps | grep healthy

# 특정 서비스 로그 확인
docker compose -f docker/compose/docker-compose.dev.yml logs postgres
docker compose -f docker/compose/docker-compose.dev.yml logs redis
docker compose -f docker/compose/docker-compose.dev.yml logs minio

# 완전 재시작
docker compose -f docker/compose/docker-compose.dev.yml down -v
docker compose -f docker/compose/docker-compose.dev.yml up -d
```

## 보안

### 최신 보안 업데이트

- ✅ **2025-11-29**: `golang.org/x/crypto` v0.45.0 업그레이드
  - SSH GSSAPI 무제한 메모리 소비 취약점 해결
  - SSH Agent 잘못된 메시지 패닉 취약점 해결

### 보안 검증

```bash
# 의존성 취약점 검사
go list -json -m all | nancy sleuth

# 정적 분석
golangci-lint run

# 보안 감사
gosec ./...
```

## 성능

### 벤치마크 (t3.small, PostgreSQL RDS)

| 엔드포인트 | 평균 응답시간 | P95 | RPS |
|-----------|--------------|-----|-----|
| GET /health | 2ms | 5ms | 5000+ |
| GET /boards/:id | 15ms | 30ms | 800+ |
| POST /boards | 25ms | 50ms | 500+ |
| GET /boards/project/:id | 20ms | 40ms | 600+ |

### 최적화 전략

- ✅ Redis 캐싱 (프로젝트 메타데이터, 권한)
- ✅ 인덱스 최적화 (project_id, position)
- ✅ Fractional Indexing (O(1) 위치 변경)
- ✅ 연결 풀링 (Max 25, Idle 5)

## 문서

- 📐 [아키텍처 가이드](ARCHITECTURE.md) - 클라우드 네이티브 설계 상세
- 📚 [API 마이그레이션 가이드](MIGRATION_GUIDE.md) - API 변경 사항
- 📦 [S3 업로드 가이드](docs/PRESIGNED_URL_API_GUIDE.md) - 파일 첨부 구현
- 🔧 [설정 가이드](docs/CONFIGURATION.md) - 환경 변수 상세
- 🚀 [CI/CD 가이드](docs/CI_CD_INTEGRATION.md) - 배포 자동화

## 라이선스

MIT License

## 기여

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 문의

프로젝트 관련 문의사항이나 버그 리포트는 [GitHub Issues](https://github.com/your-org/wealist/issues)를 이용해주세요.
