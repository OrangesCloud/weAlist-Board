# Kubernetes 준비 사항

이 애플리케이션은 Kubernetes 환경에 배포할 수 있도록 설계되었습니다.

## 구현된 Cloud Native 기능

### 1. Health Check Endpoints ✅

Kubernetes Probe를 위한 3가지 엔드포인트:

```yaml
# Kubernetes Deployment 예시
spec:
  containers:
  - name: kanban-api
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8000
      initialDelaySeconds: 10
      periodSeconds: 10

    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8000
      initialDelaySeconds: 5
      periodSeconds: 5
```

**엔드포인트:**
- `GET /health` - 기본 헬스체크
- `GET /health/live` - Liveness Probe (Pod 재시작 여부)
- `GET /health/ready` - Readiness Probe (트래픽 수신 여부, DB/Redis 연결 확인)

---

### 2. Database Migration (Alembic) ✅

프로덕션 환경에서는 Alembic을 사용한 마이그레이션 관리:

```yaml
# Kubernetes Job으로 마이그레이션 실행
apiVersion: batch/v1
kind: Job
metadata:
  name: kanban-db-migration
spec:
  template:
    spec:
      containers:
      - name: migration
        image: kanban-api:latest
        command: ["alembic", "upgrade", "head"]
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kanban-secrets
              key: database-url
      restartPolicy: OnFailure
```

**주요 명령어:**
```bash
# 마이그레이션 생성
alembic revision --autogenerate -m "설명"

# 마이그레이션 적용
alembic upgrade head

# 롤백
alembic downgrade -1
```

---

### 3. Structured Logging (JSON) ✅

로그를 JSON 형식으로 출력하여 ELK, Loki, CloudWatch 등과 쉽게 연동:

```json
{
  "timestamp": "2025-01-18T10:30:00.123Z",
  "level": "INFO",
  "logger": "app.main",
  "message": "Starting application",
  "service": "Kanban Service",
  "environment": "production"
}
```

**환경별 로그 포맷:**
- **개발**: 읽기 쉬운 텍스트 포맷
- **프로덕션**: JSON 포맷 (파싱 가능)

---

### 4. Graceful Shutdown ✅

SIGTERM 시그널을 받으면 진행 중인 요청을 완료한 후 종료:

```yaml
# Deployment에서 terminationGracePeriodSeconds 설정
spec:
  template:
    spec:
      terminationGracePeriodSeconds: 30
      containers:
      - name: kanban-api
        # ...
```

**동작 방식:**
1. K8s가 SIGTERM 전송
2. 새 요청 수신 중지
3. 진행 중인 요청 완료 대기
4. DB 연결 정리
5. 종료

---

### 5. 12-Factor App 준수 ✅

| Factor | 구현 | 설명 |
|--------|------|------|
| I. Codebase | ✅ | Git으로 버전 관리 |
| II. Dependencies | ✅ | requirements.txt로 명시 |
| III. Config | ✅ | 환경변수로 관리 (.env) |
| IV. Backing services | ✅ | PostgreSQL, Redis를 리소스로 취급 |
| V. Build, release, run | ✅ | Docker multi-stage build |
| VI. Processes | ✅ | Stateless (세션 없음) |
| VII. Port binding | ✅ | 8000 포트 바인딩 |
| VIII. Concurrency | ✅ | 프로세스 모델 (HPA 가능) |
| IX. Disposability | ✅ | 빠른 시작/종료, Graceful Shutdown |
| X. Dev/prod parity | ✅ | Docker로 환경 일치 |
| XI. Logs | ✅ | stdout/stderr로 스트림 |
| XII. Admin processes | ✅ | Alembic migration |

---

## Kubernetes 배포 예시

### ConfigMap
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kanban-config
data:
  ENV: "production"
  DEBUG: "False"
  PROJECT_NAME: "Kanban Service"
  CORS_ORIGINS: "https://example.com"
```

### Secret
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kanban-secrets
type: Opaque
stringData:
  database-url: "postgresql://user:pass@postgres:5432/kanban"
  redis-url: "redis://:pass@redis:6379/0"
  secret-key: "your-secret-key"
```

### Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kanban-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kanban-api
  template:
    metadata:
      labels:
        app: kanban-api
    spec:
      containers:
      - name: kanban-api
        image: kanban-api:latest
        ports:
        - containerPort: 8000
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: kanban-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: kanban-secrets
              key: redis-url
        - name: ENV
          valueFrom:
            configMapKeyRef:
              name: kanban-config
              key: ENV
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8000
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Service
```yaml
apiVersion: v1
kind: Service
metadata:
  name: kanban-api
spec:
  selector:
    app: kanban-api
  ports:
  - port: 80
    targetPort: 8000
  type: ClusterIP
```

### HPA (Horizontal Pod Autoscaler)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: kanban-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: kanban-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## 다음 단계 (향후 추가 가능)

### 1. Prometheus Metrics
```python
# /metrics 엔드포인트 추가
# prometheus-client 사용
```

### 2. Distributed Tracing
```python
# OpenTelemetry 통합
# Jaeger 또는 Zipkin
```

### 3. Rate Limiting
```python
# slowapi 또는 Redis 기반 rate limiter
```

### 4. Circuit Breaker
```python
# 외부 서비스 호출 시 Circuit Breaker 패턴
```

---

## 참고 자료

- [12-Factor App](https://12factor.net/)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)
- [FastAPI Deployment](https://fastapi.tiangolo.com/deployment/)
- [Alembic Documentation](https://alembic.sqlalchemy.org/)
