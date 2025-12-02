# Swagger API 문서 가이드

## 목차

- [개요](#개요)
- [Swagger UI 접속](#swagger-ui-접속)
- [API 카테고리](#api-카테고리)
- [Swagger 문서 생성](#swagger-문서-생성)
- [Swagger 문서 검증](#swagger-문서-검증)
- [Godoc 주석 작성 가이드](#godoc-주석-작성-가이드)
- [개발자 워크플로우](#개발자-워크플로우)
- [문제 해결](#문제-해결)

## 개요

이 프로젝트는 Swagger/OpenAPI 2.0을 사용하여 API 문서를 자동 생성합니다. Go 코드의 godoc 주석을 파싱하여 swagger.yaml, swagger.json, docs.go 파일을 생성합니다.

### 주요 기능

- **자동 문서 생성**: Go 코드의 주석에서 자동으로 API 문서 생성
- **인터랙티브 UI**: Swagger UI를 통한 API 테스트 및 탐색
- **타입 안정성**: DTO 스키마 자동 추출 및 검증
- **검증 도구**: 문서 완전성을 보장하는 자동화된 검증 스크립트

## Swagger UI 접속

서버 실행 후 브라우저에서 다음 URL에 접속하세요:

```
http://localhost:8000/swagger/index.html
```

### Swagger UI 기능

1. **API 엔드포인트 탐색**
   - 모든 API 엔드포인트를 카테고리별로 확인
   - 각 엔드포인트의 HTTP 메서드, 경로, 설명 확인

2. **요청/응답 스키마**
   - 요청 바디 구조 확인
   - 응답 데이터 구조 확인
   - 필수/선택 필드 구분

3. **API 테스트**
   - "Try it out" 버튼으로 직접 API 호출
   - 요청 파라미터 입력
   - 실시간 응답 확인

4. **예시 데이터**
   - 각 엔드포인트의 요청/응답 예시 제공
   - 실제 사용 가능한 데이터 형식 확인

## API 카테고리

### Projects
- Workspace별 프로젝트 관리
- 프로젝트 생성, 조회, 수정, 삭제
- 프로젝트 검색 (이름/설명)
- 프로젝트 초기 설정 조회
- 기본 프로젝트 조회

### Project Members
- 프로젝트 멤버 목록 조회
- 멤버 제거 (OWNER/ADMIN)
- 멤버 역할 변경 (OWNER)

### Project Join Requests
- 프로젝트 가입 요청 생성
- 가입 요청 목록 조회 (OWNER/ADMIN)
- 가입 요청 승인/거부 (OWNER/ADMIN)

### Boards
- Board CRUD 작업
- Stage, Importance, Role 관리
- Custom Fields 필터링

### Field Options
- 커스텀 필드 옵션 관리
- 필드 타입별 옵션 조회
- 옵션 생성, 수정, 삭제

### Participants
- Board 참여자 관리
- 참여자 추가/제거

### Comments
- Board 댓글 기능
- 댓글 작성/수정/삭제

## Swagger 문서 생성

### 기본 생성 명령어

```bash
# Make 명령어 사용 (권장)
make swagger

# 또는 직접 실행
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

### 중요 플래그

- `-g cmd/api/main.go`: 메인 파일 경로 지정
- `-o docs`: 출력 디렉토리 지정
- `--parseDependency`: 외부 패키지의 타입 파싱
- `--parseInternal`: 내부 패키지의 타입 파싱

**주의**: `--parseDependency`와 `--parseInternal` 플래그를 반드시 포함해야 모든 DTO 스키마가 올바르게 생성됩니다.

### 생성되는 파일

- `docs/swagger.yaml`: YAML 형식의 API 문서
- `docs/swagger.json`: JSON 형식의 API 문서
- `docs/docs.go`: Go 코드로 임베드된 문서

## Swagger 문서 검증

### 검증 스크립트

프로젝트는 세 가지 검증 스크립트를 제공합니다:

#### 1. 엔드포인트 커버리지 검증

```bash
./scripts/validate-swagger.sh
```

- router.go의 모든 라우트가 swagger.yaml에 문서화되어 있는지 확인
- 커버리지 퍼센티지 계산
- 100% 커버리지 미달 시 오류 반환

#### 2. DTO 스키마 검증

```bash
./scripts/validate-dto-schemas.sh
```

- 핸들러에서 사용하는 모든 DTO가 swagger definitions에 정의되어 있는지 확인
- 누락된 스키마 리포트
- 누락 시 오류 반환

#### 3. Godoc 주석 품질 검증

```bash
./scripts/validate-godoc.sh
```

- 모든 핸들러에 필수 godoc 주석이 있는지 확인
- @Summary, @Router, @Tags 주석 검증
- 불완전한 주석 리포트

### 전체 검증 실행

```bash
# 모든 검증 스크립트 실행
./scripts/validate-swagger.sh && \
./scripts/validate-dto-schemas.sh && \
./scripts/validate-godoc.sh
```

## Godoc 주석 작성 가이드

### 기본 템플릿

모든 핸들러 함수는 다음 형식의 godoc 주석을 포함해야 합니다:

```go
// HandlerName godoc
// @Summary      간단한 요약 (한 줄)
// @Description  상세한 설명 (여러 줄 가능)
// @Tags         tag-name
// @Accept       json
// @Produce      json
// @Param        paramName paramType dataType required "description" [options]
// @Success      200 {object} response.SuccessResponse{data=dto.ResponseType} "success message"
// @Failure      400 {object} response.ErrorResponse "error message"
// @Failure      401 {object} response.ErrorResponse "unauthorized"
// @Failure      403 {object} response.ErrorResponse "forbidden"
// @Failure      404 {object} response.ErrorResponse "not found"
// @Failure      500 {object} response.ErrorResponse "internal server error"
// @Router       /api/path [method]
func (h *Handler) HandlerName(c *gin.Context) {
    // ...
}
```

### 주석 태그 설명

#### @Summary (필수)
- 엔드포인트의 간단한 요약 (한 줄)
- 한글 또는 영문 사용 가능
- 예: `Board 생성`, `Get board list`

#### @Description (권장)
- 엔드포인트의 상세한 설명
- 여러 줄 작성 가능
- 특별한 동작이나 제약사항 명시

#### @Tags (필수)
- API 그룹화를 위한 태그
- 소문자와 하이픈 사용 (예: `boards`, `field-options`)
- 하나의 엔드포인트는 하나의 태그만 사용

#### @Accept / @Produce
- 요청/응답 컨텐츠 타입
- 일반적으로 `json` 사용
- POST, PUT, PATCH는 @Accept 포함

#### @Param
파라미터 정의 형식:
```
@Param name location type required "description" [options]
```

**Location 타입:**
- `path`: URL 경로 파라미터 (예: `/boards/:boardId`)
- `query`: 쿼리 스트링 파라미터 (예: `?page=1`)
- `body`: 요청 바디 (JSON)
- `header`: HTTP 헤더

**예시:**
```go
// @Param boardId path string true "Board ID (UUID)"
// @Param page query int false "Page number" default(1)
// @Param request body dto.CreateBoardRequest true "Board 생성 요청"
// @Param Authorization header string true "Bearer token"
```

**Enum 값 지정:**
```go
// @Param fieldType query string true "Field Type" Enums(stage, role, importance)
// @Param status query string false "Status" Enums(active, inactive) default(active)
```

#### @Success
성공 응답 정의:
```
@Success statusCode {type} dataType "description"
```

**예시:**
```go
// 단순 응답
// @Success 200 {object} response.SuccessResponse "success"

// 데이터 포함 응답
// @Success 200 {object} response.SuccessResponse{data=dto.BoardResponse}

// 배열 응답
// @Success 200 {object} response.SuccessResponse{data=[]dto.BoardResponse}

// 중첩 구조
// @Success 200 {object} response.SuccessResponse{data=dto.ProjectInitSettingsResponse}
```

#### @Failure
에러 응답 정의:
```
@Failure statusCode {type} dataType "description"
```

**표준 에러 응답:**
```go
// @Failure 400 {object} response.ErrorResponse "잘못된 요청"
// @Failure 401 {object} response.ErrorResponse "인증 실패"
// @Failure 403 {object} response.ErrorResponse "권한 없음"
// @Failure 404 {object} response.ErrorResponse "리소스를 찾을 수 없음"
// @Failure 409 {object} response.ErrorResponse "충돌 (중복 데이터)"
// @Failure 500 {object} response.ErrorResponse "서버 내부 오류"
```

#### @Router (필수)
라우트 경로 및 HTTP 메서드:
```
@Router /api/path [method]
```

**예시:**
```go
// @Router /boards [post]
// @Router /boards/{boardId} [get]
// @Router /boards/{boardId} [put]
// @Router /boards/{boardId} [delete]
```

### 실제 예시

#### GET 엔드포인트
```go
// GetFieldOptions godoc
// @Summary      필드 옵션 목록 조회
// @Description  특정 필드 타입의 옵션 목록을 조회합니다
// @Tags         field-options
// @Produce      json
// @Param        fieldType query string true "Field Type" Enums(stage, role, importance)
// @Success      200 {object} response.SuccessResponse{data=[]dto.FieldOptionResponse}
// @Failure      400 {object} response.ErrorResponse "잘못된 fieldType 파라미터"
// @Failure      500 {object} response.ErrorResponse
// @Router       /field-options [get]
func (h *FieldOptionHandler) GetFieldOptions(c *gin.Context) {
    // ...
}
```

#### POST 엔드포인트
```go
// CreateBoard godoc
// @Summary      Board 생성
// @Description  새로운 Board를 생성합니다
// @Tags         boards
// @Accept       json
// @Produce      json
// @Param        request body dto.CreateBoardRequest true "Board 생성 요청"
// @Success      201 {object} response.SuccessResponse{data=dto.BoardResponse}
// @Failure      400 {object} response.ErrorResponse "잘못된 요청 데이터"
// @Failure      404 {object} response.ErrorResponse "프로젝트를 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse
// @Router       /boards [post]
func (h *BoardHandler) CreateBoard(c *gin.Context) {
    // ...
}
```

#### PATCH 엔드포인트
```go
// UpdateFieldOption godoc
// @Summary      필드 옵션 수정
// @Description  필드 옵션의 정보를 수정합니다
// @Tags         field-options
// @Accept       json
// @Produce      json
// @Param        optionId path string true "Option ID (UUID)"
// @Param        request body dto.UpdateFieldOptionRequest true "필드 옵션 수정 요청"
// @Success      200 {object} response.SuccessResponse{data=dto.FieldOptionResponse}
// @Failure      400 {object} response.ErrorResponse "잘못된 요청 데이터"
// @Failure      404 {object} response.ErrorResponse "옵션을 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse
// @Router       /field-options/{optionId} [patch]
func (h *FieldOptionHandler) UpdateFieldOption(c *gin.Context) {
    // ...
}
```

#### DELETE 엔드포인트
```go
// DeleteFieldOption godoc
// @Summary      필드 옵션 삭제
// @Description  필드 옵션을 삭제합니다 (시스템 기본 옵션은 삭제 불가)
// @Tags         field-options
// @Produce      json
// @Param        optionId path string true "Option ID (UUID)"
// @Success      200 {object} response.SuccessResponse "삭제 성공"
// @Failure      400 {object} response.ErrorResponse "잘못된 옵션 ID"
// @Failure      403 {object} response.ErrorResponse "시스템 기본 옵션은 삭제 불가"
// @Failure      404 {object} response.ErrorResponse "옵션을 찾을 수 없음"
// @Failure      500 {object} response.ErrorResponse
// @Router       /field-options/{optionId} [delete]
func (h *FieldOptionHandler) DeleteFieldOption(c *gin.Context) {
    // ...
}
```

### DTO 구조체 주석

DTO 구조체에도 주석을 추가하여 Swagger 문서의 품질을 높일 수 있습니다:

```go
// CreateBoardRequest Board 생성 요청
type CreateBoardRequest struct {
    ProjectID  uuid.UUID `json:"projectId" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
    Title      string    `json:"title" binding:"required,max=200" example:"Implement User Authentication"`
    Content    string    `json:"content" binding:"max=5000" example:"Add JWT-based authentication system"`
    Stage      string    `json:"stage" binding:"required,oneof=in_progress pending approved review" example:"in_progress"`
    Importance string    `json:"importance" binding:"required,oneof=urgent normal" example:"urgent"`
    Role       string    `json:"role" binding:"required,oneof=developer planner" example:"developer"`
}
```

**주요 태그:**
- `json`: JSON 필드명
- `binding`: 검증 규칙 (required, max, min, oneof 등)
- `example`: Swagger UI에 표시될 예시 값

## 개발자 워크플로우

### 새 API 엔드포인트 추가 시

1. **핸들러 함수 작성**
   ```go
   func (h *Handler) NewEndpoint(c *gin.Context) {
       // 구현
   }
   ```

2. **Godoc 주석 추가**
   - 위의 템플릿을 참고하여 완전한 주석 작성
   - 모든 필수 태그 포함 (@Summary, @Tags, @Router)

3. **라우터에 등록**
   ```go
   router.GET("/api/new-endpoint", handler.NewEndpoint)
   ```

4. **Swagger 문서 재생성**
   ```bash
   make swagger
   ```

5. **검증 실행**
   ```bash
   ./scripts/validate-swagger.sh
   ./scripts/validate-godoc.sh
   ```

6. **Swagger UI 확인**
   - 서버 실행 후 http://localhost:8000/swagger/index.html 접속
   - 새 엔드포인트가 올바르게 표시되는지 확인

### DTO 수정 시

1. **DTO 구조체 수정**
   ```go
   type MyDTO struct {
       NewField string `json:"newField" binding:"required"`
   }
   ```

2. **Swagger 문서 재생성**
   ```bash
   make swagger
   ```

3. **스키마 검증**
   ```bash
   ./scripts/validate-dto-schemas.sh
   ```

4. **Swagger UI에서 스키마 확인**

### 커밋 전 체크리스트

- [ ] 모든 새 핸들러에 godoc 주석 추가
- [ ] `make swagger` 실행하여 문서 재생성
- [ ] 모든 검증 스크립트 통과
- [ ] Swagger UI에서 수동 확인
- [ ] 변경된 docs/ 파일 커밋에 포함

## 문제 해결

### Swagger 생성 실패

**증상**: `swag init` 명령어 실행 시 오류 발생

**해결 방법**:
1. swag 도구 설치 확인
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. 플래그 확인
   ```bash
   swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
   ```

3. Go 모듈 정리
   ```bash
   go mod tidy
   ```

### DTO 스키마 누락

**증상**: Swagger UI에서 DTO 스키마가 표시되지 않음

**원인**: `--parseDependency` 또는 `--parseInternal` 플래그 누락

**해결 방법**:
```bash
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

### 엔드포인트가 Swagger에 표시되지 않음

**원인**:
1. Godoc 주석 누락 또는 형식 오류
2. @Router 태그 누락 또는 경로 불일치

**해결 방법**:
1. 핸들러 함수의 godoc 주석 확인
2. @Router 경로가 router.go와 일치하는지 확인
3. 검증 스크립트 실행
   ```bash
   ./scripts/validate-godoc.sh
   ./scripts/validate-swagger.sh
   ```

### 검증 스크립트 실패

**증상**: 검증 스크립트가 오류 반환

**해결 방법**:
1. 스크립트 출력 확인하여 누락된 항목 파악
2. 누락된 godoc 주석 추가
3. 누락된 DTO 스키마 확인
4. Swagger 문서 재생성
5. 검증 재실행

### Swagger UI 접속 불가

**증상**: http://localhost:8000/swagger/index.html 접속 시 404 오류

**원인**:
1. 서버가 실행되지 않음
2. Swagger 문서가 생성되지 않음
3. Swagger 미들웨어 미등록

**해결 방법**:
1. 서버 실행 확인
   ```bash
   make run
   ```

2. Swagger 문서 생성 확인
   ```bash
   ls -la docs/
   ```

3. main.go에 Swagger 임포트 확인
   ```go
   import _ "board-service/docs"
   ```

## 참고 자료

- [Swag 공식 문서](https://github.com/swaggo/swag)
- [OpenAPI 2.0 Specification](https://swagger.io/specification/v2/)
- [Gin Swagger 통합](https://github.com/swaggo/gin-swagger)

## 문의

Swagger 문서 관련 문의사항이나 개선 제안은 GitHub Issues를 이용해주세요.
