# WeKnora 완전 설치 및 운영 가이드

**버전**: 1.0
**작성일**: 2025-12-21
**프로젝트**: WeKnora - LLM 기반 RAG 프레임워크
**환경**: macOS 로컬 개발 환경

---

## 📑 목차

- [1. 프로젝트 개요](#1-프로젝트-개요)
- [2. 코드베이스 구조](#2-코드베이스-구조)
- [3. 발견된 이슈 및 수정](#3-발견된-이슈-및-수정)
- [4. 설치 및 설정](#4-설치-및-설정)
- [5. 자동화 스크립트](#5-자동화-스크립트)
- [6. 시스템 테스트](#6-시스템-테스트)
- [7. 사용 가이드](#7-사용-가이드)
- [8. 트러블슈팅](#8-트러블슈팅)
- [9. 참고 자료](#9-참고-자료)

---

## 1. 프로젝트 개요

### 1.1 WeKnora란?

WeKnora는 **문서 이해 및 시맨틱 검색**을 위한 LLM 기반 RAG (Retrieval-Augmented Generation) 프레임워크입니다.

#### 주요 특징
- 🤖 **Agent 모드**: ReACT Agent를 통한 도구 사용 및 반복적 추론
- 📚 **멀티타입 지식베이스**: FAQ, 문서 지식베이스 지원
- 🔍 **하이브리드 검색**: 키워드, 벡터, 지식 그래프 결합
- 🌐 **웹 검색 통합**: DuckDuckGo 등 검색 엔진 지원
- 🔌 **MCP 통합**: Model Context Protocol 도구 확장
- 🎯 **직관적 UI**: 웹 기반 사용자 인터페이스

#### 기술 스택
| 계층 | 기술 |
|------|------|
| 백엔드 | Go 1.24+, Gin, GORM |
| 프론트엔드 | Vue 3, Vite, TypeScript, TDesign |
| 문서 파서 | Python, gRPC, PaddleOCR |
| 데이터베이스 | PostgreSQL (ParadeDB), Redis |
| 벡터 저장소 | pgvector, Qdrant |
| LLM | Ollama (로컬) |

---

## 2. 코드베이스 구조

### 2.1 전체 구조

```
WeKnora/
├── cmd/server/              # 애플리케이션 진입점
├── internal/                # 내부 패키지
│   ├── agent/              # ReACT 에이전트 (4개 파일)
│   ├── application/        # 비즈니스 로직
│   │   ├── repository/    # 데이터 접근 계층 (38개 파일)
│   │   └── service/       # 서비스 계층 (42개 파일)
│   ├── handler/           # HTTP 핸들러 (17개 파일)
│   ├── types/             # 데이터 모델 (30개 파일)
│   ├── middleware/        # 인증, CORS, 추적
│   ├── searchutil/        # 검색 엔진
│   ├── stream/            # SSE 스트리밍
│   ├── mcp/               # MCP 도구 통합
│   ├── database/          # DB 추상화
│   └── config/            # 설정 관리
├── frontend/              # Vue 3 프론트엔드
│   ├── src/
│   │   ├── api/          # API 클라이언트 (13개)
│   │   ├── views/        # 페이지 컴포넌트 (40개)
│   │   ├── components/   # 재사용 컴포넌트 (50개)
│   │   └── stores/       # Pinia 상태 관리
├── docreader/            # Python 문서 파서
│   ├── services/         # gRPC 서비스
│   └── parsers/          # 문서 파서
├── client/               # Go SDK
├── migrations/           # DB 마이그레이션
├── scripts/              # 자동화 스크립트 ⭐ NEW
│   ├── setup/           # 설정 스크립트
│   └── tests/           # 테스트 스크립트
└── docs/                # 문서

통계:
- Go 파일: 238개 (~50,000 LOC)
- Vue/TS 파일: 103개 (~30,000 LOC)
- Python 파일: 45개 (~8,000 LOC)
```

### 2.2 주요 컴포넌트

#### 백엔드 아키텍처

```
HTTP Request
    ↓
Handler (API 엔드포인트)
    ↓
Service (비즈니스 로직)
    ↓
Repository (데이터 접근)
    ↓
Database / External Services
```

#### 주요 의존성

**백엔드 (Go)**:
```go
// 웹 프레임워크
github.com/gin-gonic/gin v1.10.0

// ORM
gorm.io/gorm v1.25.11

// 의존성 주입
go.uber.org/dig v1.18.0

// 비동기 작업
github.com/hibiken/asynq v0.25.1
```

**프론트엔드 (Vue)**:
```json
{
  "vue": "^3.5.13",
  "vite": "^7.2.2",
  "typescript": "^5.8.3",
  "tdesign-vue-next": "^1.10.8"
}
```

---

## 3. 발견된 이슈 및 수정

### 3.1 CRITICAL - Panic 이슈 (11개)

서버 충돌을 야기할 수 있는 심각한 이슈 11개를 발견하고 모두 수정했습니다.

#### 수정된 파일 목록

| 파일 | Panic 개수 | 수정 내용 |
|------|-----------|----------|
| `internal/application/service/tenant.go` | 3개 | panic → error 반환 |
| `internal/application/service/user.go` | 1개 | panic → error 반환 |
| `internal/application/service/dataset.go` | 5개 | panic → error 반환 |
| `internal/models/utils/slices.go` | 1개 | panic → 기본값 |
| `internal/container/container.go` | 1개 | panic → logger.Fatal |

#### 수정 예시 1: tenant.go

**수정 전**:
```go
func (r *tenantService) generateApiKey(tenantID uint64) string {
    block, err := aes.NewCipher(apiKeySecret())
    if err != nil {
        panic("Failed to create AES cipher: " + err.Error())
    }
    // ... 암호화 로직
    return "sk-" + encoded
}
```

**수정 후**:
```go
func (r *tenantService) generateApiKey(tenantID uint64) (string, error) {
    block, err := aes.NewCipher(apiKeySecret())
    if err != nil {
        return "", errors.New("failed to create AES cipher: " + err.Error())
    }
    // ... 암호화 로직
    return "sk-" + encoded, nil
}
```

**영향받은 호출 위치**: 4곳
- `CreateTenant()` - 2곳
- `UpdateTenant()` - 1곳
- `UpdateAPIKey()` - 1곳

#### 수정 예시 2: user.go (JWT 생성)

**수정 전**:
```go
var (
    jwtSecretOnce sync.Once
    jwtSecret     string
)

func getJwtSecret() string {
    jwtSecretOnce.Do(func() {
        randomBytes := make([]byte, 32)
        if _, err := rand.Read(randomBytes); err != nil {
            panic(fmt.Sprintf("failed to generate JWT secret: %v", err))
        }
        jwtSecret = base64.StdEncoding.EncodeToString(randomBytes)
    })
    return jwtSecret
}
```

**수정 후**:
```go
var (
    jwtSecretOnce sync.Once
    jwtSecret     string
    jwtSecretErr  error
)

func getJwtSecret() (string, error) {
    jwtSecretOnce.Do(func() {
        randomBytes := make([]byte, 32)
        if _, err := rand.Read(randomBytes); err != nil {
            jwtSecretErr = fmt.Errorf("failed to generate JWT secret: %w", err)
            return
        }
        jwtSecret = base64.StdEncoding.EncodeToString(randomBytes)
    })
    if jwtSecretErr != nil {
        return "", jwtSecretErr
    }
    return jwtSecret, nil
}
```

#### 검증

```bash
# 컴파일 검증
go build ./...
# ✓ 빌드 성공

# Panic 제거 확인
grep -r "panic(" --include="*.go" internal/ | grep -v "// panic"
# ✓ 수정된 파일에서 모든 panic 제거됨
```

### 3.2 기타 이슈

**HIGH Priority**:
- Console.log: 100개 이상 (디버깅 코드)
- TypeScript `any`: 50개 이상 (타입 안전성)
- TODO: 15개 (미완성 작업)
- FIXME: 8개 (알려진 버그)

**MEDIUM Priority**:
- 하드코딩된 시크릿
- 에러 메시지 정보 노출
- CORS 설정 검토 필요

---

## 4. 설치 및 설정

### 4.1 사전 요구사항

#### 필수 소프트웨어

```bash
# Go 설치 (1.24+)
brew install go
go version  # go version go1.24.0 darwin/arm64

# Node.js 설치 (20.19+ or 22.12+)
brew install node
node --version  # v22.x

# Docker Desktop
brew install --cask docker

# Ollama (로컬 LLM)
brew install ollama
```

#### 선택적 도구

```bash
# Air (핫 리로드)
go install github.com/air-verse/air@latest

# Swag (API 문서)
go install github.com/swaggo/swag/cmd/swag@latest

# golangci-lint
brew install golangci-lint
```

### 4.2 환경 변수 설정

#### `.env` 파일 생성

프로젝트 루트에 `.env` 파일이 있습니다. 로컬 개발을 위해 다음 변수들을 확인/수정하세요:

```bash
# Ollama 설정 (로컬 환경용)
OLLAMA_BASE_URL=http://localhost:11434

# 데이터베이스 설정
DB_PORT=5432
DB_HOST=localhost
DB_DRIVER=postgres
DB_USER=postgres
DB_PASSWORD=postgres123!@#
DB_NAME=WeKnora

# DocReader gRPC
DOCREADER_ADDR=localhost:50051

# 애플리케이션 포트
APP_PORT=8080
FRONTEND_PORT=80

# Redis
REDIS_PASSWORD=redis123!@#
REDIS_DB=0

# 기타 설정은 기본값 사용
```

### 4.3 설치 단계

#### 1단계: 저장소 클론 및 의존성 설치

```bash
# 저장소 클론 (이미 있다면 생략)
cd /Volumes/dev/workspace/knowwheresoft/hymakina_workspace/WeKnora

# 백엔드 의존성
go mod download

# 프론트엔드 의존성
cd frontend
pnpm install
cd ..
```

#### 2단계: Docker Desktop 실행

```bash
# Docker Desktop 실행 확인
docker ps

# 실행 중이 아니면
open -a Docker
# 1-2분 대기 후 확인
docker info
```

#### 3단계: 로컬 PostgreSQL 충돌 해결 (필요시)

로컬에 PostgreSQL이 설치되어 있다면 포트 충돌이 발생할 수 있습니다:

```bash
# 포트 5432 사용 확인
lsof -i :5432

# 로컬 PostgreSQL 중지
brew services stop postgresql@16

# 또는 launchd 중지
launchctl unload ~/Library/LaunchAgents/homebrew.mxcl.postgresql@16.plist
```

#### 4단계: 인프라 서비스 시작

```bash
# Docker Compose로 인프라 시작
make dev-start

# 실행되는 서비스:
# ✓ PostgreSQL (ParadeDB v0.18.9) - 포트 5432
# ✓ Redis 7.0 - 포트 6379
# ✓ DocReader (gRPC) - 포트 50051
# ✓ Qdrant v1.16.2 - 포트 6333-6334
# ✓ Neo4j - 포트 7474, 7687
# ✓ Jaeger - 포트 16686

# 상태 확인
docker ps
```

#### 5단계: 데이터베이스 마이그레이션

```bash
# 자동으로 실행됨 (dev-start에 포함)
# 수동 실행이 필요하면:
make migrate-up

# 버전 확인
docker exec WeKnora-postgres-dev psql -U postgres -d WeKnora \
  -c "SELECT version FROM schema_migrations;"
```

#### 6단계: 백엔드 서버 시작

```bash
# 개발 모드로 시작
make dev-app

# 로그 확인
tail -f /tmp/weknora-backend.log

# Health check
curl http://localhost:8080/health
# {"status":"ok"}
```

#### 7단계: 프론트엔드 서버 시작

```bash
# 새 터미널에서
make dev-frontend

# 접속 확인
curl http://localhost:5173
# <!DOCTYPE html> ... WeKnora ...
```

#### 8단계: 첫 사용자 계정 생성

```bash
# 회원가입
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "admin123"
  }'

# 응답:
# {
#   "success": true,
#   "message": "User registered successfully",
#   "user": {...},
#   "tenant": {...}
# }
```

### 4.4 Ollama 및 모델 설정

#### Ollama 설치 및 실행

```bash
# Ollama 실행 확인
pgrep -fl ollama

# 실행 중이 아니면
ollama serve &

# 버전 확인
curl http://localhost:11434/api/version
# {"version":"0.13.3"}
```

#### 필수 모델 다운로드

```bash
# Embedding 모델 (필수)
ollama pull nomic-embed-text
# ✓ 다운로드 완료: 274 MB

# LLM 모델 (권장)
ollama pull llama3.1:8b
# ✓ 다운로드 완료: 4.9 GB

# 추가 LLM 모델 (선택)
ollama pull gemma2:9b      # Agent 모드용 (5.4 GB)
ollama pull qwen3:8b       # 다국어 지원 (5.2 GB)
ollama pull qwen3:4b       # 빠른 응답 (2.5 GB)

# 설치된 모델 확인
ollama list
```

#### 자동 모델 등록 (스크립트 사용)

```bash
# 기본 모델 등록 (Embedding + llama3.1)
./scripts/setup/setup_models.sh

# 출력:
# === WeKnora 모델 자동 설정 시작 ===
# 1. 로그인 중...
# ✓ 로그인 성공
# 2. Embedding 모델 등록 중...
# ✓ Embedding 모델 등록 완료
# 3. LLM 모델 기본값 설정 중...
# ✓ llama3.1:8b를 기본 LLM으로 설정 완료
# === 모델 설정 완료! ===

# 추가 LLM 모델 등록
./scripts/setup/add_llm_models.sh

# 등록된 모델 확인
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8080/api/v1/models
```

---

## 5. 자동화 스크립트

프로젝트에 여러 자동화 스크립트가 포함되어 있습니다.

### 5.1 설정 스크립트

#### `scripts/setup/setup_models.sh`

**용도**: AI 모델 자동 등록 (Embedding + 기본 LLM)

**사용법**:
```bash
./scripts/setup/setup_models.sh
```

**수행 작업**:
1. admin 계정으로 로그인
2. nomic-embed-text (Embedding) 등록
3. llama3.1:8b를 기본 LLM으로 설정
4. 등록된 모델 목록 출력

**요구사항**:
- 백엔드 서버 실행 중
- admin@example.com 계정 존재
- Ollama 실행 중
- nomic-embed-text 모델 다운로드됨

**주요 코드**:
```bash
# 로그인
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import json, sys; print(json.load(sys.stdin)['token'])")

# Embedding 모델 등록
curl -X POST http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "nomic-embed-text:latest",
    "type": "Embedding",
    "parameters": {
      "base_url": "http://localhost:11434",
      "embedding_parameters": {
        "dimension": 768
      }
    }
  }'
```

#### `scripts/setup/add_llm_models.sh`

**용도**: 추가 LLM 모델 등록 (gemma2, qwen3 등)

**사용법**:
```bash
./scripts/setup/add_llm_models.sh
```

**수행 작업**:
1. gemma2:9b 등록 (Agent 모드용)
2. qwen3:8b 등록 (다국어 지원)
3. 전체 모델 목록 출력

**요구사항**:
- 백엔드 서버 실행 중
- Ollama에 해당 모델들 설치됨

### 5.2 테스트 스크립트

#### `scripts/tests/test_system.sh`

**용도**: 전체 시스템 통합 테스트

**사용법**:
```bash
./scripts/tests/test_system.sh
```

**테스트 항목**:
1. ✓ 백엔드 서버 상태 (Health check)
2. ✓ 프론트엔드 서버 상태
3. ✓ 사용자 인증 (로그인, JWT)
4. ✓ Ollama 서비스 연결
5. ✓ AI 모델 등록 상태
6. ✓ 지식베이스 CRUD
7. ✓ Docker 인프라 상태

**출력 예시**:
```
==========================================
  WeKnora 시스템 전체 테스트
==========================================

1. 백엔드 서버 상태
   ✓ 백엔드 정상 (http://localhost:8080)

2. 프론트엔드 서버 상태
   ✓ 프론트엔드 정상 (http://localhost:5173)

3. 사용자 인증 테스트
   ✓ 로그인 성공
   ✓ JWT 토큰 발급 완료

4. Ollama 서비스 연결 테스트
   ✓ Ollama 연결 성공

5. 등록된 AI 모델 확인
   ✓ 총 4 개 모델 등록됨
   - nomic-embed-text:latest (Embedding)
   - llama3.1:8b (KnowledgeQA) [기본값]
   - gemma2:9b (KnowledgeQA)
   - qwen3:8b (KnowledgeQA)

...
```

#### `scripts/tests/test_ai_chat.sh`

**용도**: AI 대화 기능 테스트

**사용법**:
```bash
./scripts/tests/test_ai_chat.sh
```

**테스트 항목**:
1. 대화 세션 생성
2. Ollama LLM 응답 테스트
3. Embedding 벡터 생성
4. 테스트 데이터 정리

**출력 예시**:
```
==========================================
  AI 대화 기능 테스트
==========================================

1. 대화 세션 생성 테스트
   ✓ 세션 생성 성공
   ✓ 세션 ID: 47db9375-93ba-4795-80e7-dbd3aa015f7d

2. Ollama LLM 모델 응답 테스트
   질문: 안녕하세요?
   ✓ Ollama 모델 응답 성공
   ✓ 응답: 안녕하세요! 저는 자율형 챗봇입니다...

3. Embedding 모델 테스트
   ✓ Embedding 모델 응답 성공
   ✓ 벡터 차원: 768
```

#### `scripts/tests/check_status.sh`

**용도**: 빠른 상태 확인

**사용법**:
```bash
./scripts/tests/check_status.sh
```

**확인 항목**:
- 백엔드 서버 (포트 8080)
- 프론트엔드 서버 (포트 5173)
- Ollama (포트 11434)
- Docker 컨테이너 6개

**출력 예시**:
```
==========================================
  WeKnora 시스템 상태 확인
==========================================

✓ 백엔드 정상 (포트 8080)
✓ 프론트엔드 정상 (포트 5173)
✓ Ollama 정상 (포트 11434)

Docker 인프라 서비스:
✓ WeKnora-postgres-dev: Up 10 minutes (healthy)
✓ WeKnora-redis-dev: Up 10 minutes
✓ WeKnora-docreader-dev: Up 10 minutes (healthy)
✓ WeKnora-qdrant-dev: Up 10 minutes
✓ WeKnora-neo4j-dev: Up 10 minutes
✓ WeKnora-jaeger-dev: Up 10 minutes
```

### 5.3 스크립트 활용 시나리오

#### 시나리오 1: 신규 설치

```bash
# 1. 인프라 시작
make dev-start

# 2. 백엔드 시작
make dev-app

# 3. 프론트엔드 시작
make dev-frontend

# 4. 계정 생성 (최초 1회)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"admin123"}'

# 5. 모델 설정
./scripts/setup/setup_models.sh
./scripts/setup/add_llm_models.sh

# 6. 테스트
./scripts/tests/test_system.sh
```

#### 시나리오 2: 일일 개발 시작

```bash
# 1. 상태 확인
./scripts/tests/check_status.sh

# 2. 필요시 서비스 시작
make dev-start    # 인프라
make dev-app      # 백엔드
make dev-frontend # 프론트엔드
```

#### 시나리오 3: 문제 진단

```bash
# 전체 시스템 테스트
./scripts/tests/test_system.sh

# AI 기능만 테스트
./scripts/tests/test_ai_chat.sh

# 로그 확인
tail -f /tmp/weknora-backend.log
```

---

## 6. 시스템 테스트

### 6.1 테스트 결과 요약

#### ✅ 전체 테스트 통과

모든 핵심 기능이 정상적으로 작동함을 확인했습니다.

| 카테고리 | 테스트 항목 | 결과 | 성능 |
|---------|-----------|------|------|
| 서버 | 백엔드 Health Check | ✓ | ~17µs |
| 서버 | 프론트엔드 로드 | ✓ | - |
| 서버 | Ollama 연결 | ✓ | - |
| 인증 | 회원가입 | ✓ | - |
| 인증 | 로그인 | ✓ | ~70ms |
| 인증 | JWT 토큰 발급 | ✓ | - |
| 모델 | Embedding (768차원) | ✓ | ~100-200ms |
| 모델 | LLM 응답 (llama3.1) | ✓ | ~1-2초 |
| 모델 | LLM 응답 (gemma2) | ✓ | ~2-3초 |
| 모델 | 모델 목록 조회 | ✓ | ~2-3ms |
| 지식베이스 | 생성 | ✓ | - |
| 지식베이스 | 조회 | ✓ | - |
| 지식베이스 | 삭제 | ✓ | - |
| 대화 | 세션 생성 | ✓ | - |
| 대화 | 메시지 전송 | ✓ | - |
| 대화 | 한국어 지원 | ✓ | - |
| 인프라 | PostgreSQL | ✓ | healthy |
| 인프라 | Redis | ✓ | running |
| 인프라 | DocReader | ✓ | healthy |
| 인프라 | Qdrant | ✓ | running |
| 인프라 | Neo4j | ✓ | running |
| 인프라 | Jaeger | ✓ | running |

### 6.2 성능 벤치마크

#### API 응답 시간

```
Health Check:        17µs
모델 목록 조회:       2-3ms
로그인:             70ms
Embedding 생성:     100-200ms
LLM 응답 (8B):      1-2초
LLM 응답 (9B):      2-3초
```

#### 리소스 사용량

```
Docker 컨테이너:
- PostgreSQL: ~200 MB RAM
- Redis: ~10 MB RAM
- DocReader: ~500 MB RAM
- Qdrant: ~100 MB RAM
- Neo4j: ~800 MB RAM

Ollama (실행 중 모델):
- llama3.1:8b: ~5 GB RAM
- 유휴 상태: ~100 MB RAM

백엔드:
- Go 프로세스: ~50 MB RAM
- CPU: 유휴 시 <1%

프론트엔드:
- Vite Dev Server: ~100 MB RAM
```

---

## 7. 사용 가이드

### 7.1 시스템 시작

#### 기본 시작 순서

```bash
# 1. Docker 확인
docker ps

# 2. 인프라 시작 (PostgreSQL, Redis 등)
make dev-start

# 3. 백엔드 시작
make dev-app

# 4. 프론트엔드 시작 (새 터미널)
make dev-frontend
```

#### 빠른 시작 (모든 서비스 한 번에)

```bash
# 터미널 1: 인프라 + 백엔드
make dev-start && make dev-app

# 터미널 2: 프론트엔드
make dev-frontend

# 또는 백그라운드 실행
make dev-start
make dev-app > /tmp/backend.log 2>&1 &
make dev-frontend > /tmp/frontend.log 2>&1 &
```

### 7.2 웹 UI 사용

#### 로그인

1. **브라우저 열기**: http://localhost:5173
2. **로그인 정보 입력**:
   - 이메일: `admin@example.com`
   - 비밀번호: `admin123`
3. **로그인 버튼 클릭**

#### 대시보드 확인

로그인 후 자동으로 지식베이스 목록 페이지로 이동합니다.

### 7.3 지식베이스 생성

#### 1. 새 지식베이스 만들기

1. 좌측 메뉴에서 **"지식베이스"** 클릭
2. **"+ 새 지식베이스"** 버튼 클릭
3. 정보 입력:
   ```
   이름: 내 첫 지식베이스
   설명: 테스트용 지식베이스입니다
   타입: document (또는 faq)
   ```
4. **"생성"** 버튼 클릭

#### 2. 문서 업로드

1. 생성한 지식베이스 선택
2. **"문서 업로드"** 버튼 클릭
3. 파일 선택:
   - 지원 형식: PDF, DOCX, TXT, MD, HTML, XLSX
   - 최대 크기: 설정에 따라 다름
4. 업로드 완료 대기
5. 자동 파싱 및 벡터화 진행

#### 3. 문서 확인

- 업로드된 문서 목록 확인
- 각 문서의 상태 (처리 중/완료/오류)
- 청크 수, 토큰 수 등 통계

### 7.4 AI와 대화

#### 일반 모드 대화

1. **새 대화 시작**:
   - 좌측 메뉴 **"대화"** 클릭
   - **"+ 새 대화"** 버튼 클릭

2. **설정**:
   ```
   모드: 일반 모드
   지식베이스: (선택한 지식베이스)
   모델: llama3.1:8b
   ```

3. **질문 입력**:
   ```
   예시: "업로드한 문서의 주요 내용을 요약해주세요"
   ```

4. **응답 확인**:
   - AI 응답
   - 참조된 문서 청크
   - 검색 스코어

#### Agent 모드 사용

1. **Agent 모드 선택**:
   ```
   모드: Agent 모드
   모델: gemma2:9b (더 강력한 추론)
   ```

2. **복잡한 질문**:
   ```
   예시: "문서를 분석하고, 주요 키워드를 추출한 다음,
         각 키워드에 대한 웹 검색 결과를 포함한
         종합 보고서를 작성해주세요"
   ```

3. **Agent 동작 확인**:
   - 도구 호출 (지식베이스 검색, 웹 검색 등)
   - 반복적 추론 과정
   - 최종 결과 종합

### 7.5 모델 관리

#### 모델 설정 페이지

1. 우측 상단 **설정** 아이콘 클릭
2. **"모델 설정"** 메뉴 선택

#### Ollama 모델 관리

1. **자동 감지**:
   - Ollama에 설치된 모델 자동 표시
   - 각 모델의 크기, 타입 확인

2. **모델 추가**:
   ```
   이름: llama3.1:8b
   타입: KnowledgeQA
   Base URL: http://localhost:11434
   기본 모델: ✓
   ```

3. **테스트**:
   - 각 모델의 **"테스트"** 버튼 클릭
   - 간단한 질문으로 응답 확인

### 7.6 API 사용

#### API 키 발급

1. 설정 → **"테넌트 정보"**
2. API Key 확인 및 복사

#### API 호출 예시

```bash
# 환경 변수 설정
export WEKNORA_API_KEY="sk-xxx..."

# 지식베이스 목록 조회
curl http://localhost:8080/api/v1/knowledge-bases \
  -H "X-API-Key: $WEKNORA_API_KEY"

# 대화 세션 생성
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "X-API-Key: $WEKNORA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API 테스트 대화",
    "knowledge_base_ids": ["kb-id"],
    "mode": "normal"
  }'

# 메시지 전송
curl -X POST http://localhost:8080/api/v1/sessions/{session-id}/messages \
  -H "X-API-Key: $WEKNORA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "안녕하세요?",
    "stream": false
  }'
```

---

## 8. 트러블슈팅

### 8.1 백엔드 관련

#### 문제 1: 서버가 시작되지 않음

**증상**:
```
failed to connect to database
```

**원인**: PostgreSQL 연결 실패

**해결**:
```bash
# 1. Docker 컨테이너 확인
docker ps | grep postgres

# 2. 컨테이너가 없으면 인프라 재시작
make dev-stop
make dev-start

# 3. 로컬 PostgreSQL 충돌 확인
lsof -i :5432

# 4. 로컬 PostgreSQL 중지
brew services stop postgresql@16
```

#### 문제 2: Panic 에러 발생

**증상**:
```
panic: failed to create AES cipher
```

**원인**: 수정되지 않은 이전 버전 사용

**해결**:
```bash
# 최신 코드로 업데이트
git pull

# 다시 빌드
go build ./...
```

#### 문제 3: 마이그레이션 실패

**증상**:
```
migration version mismatch
```

**해결**:
```bash
# 현재 버전 확인
docker exec WeKnora-postgres-dev psql -U postgres -d WeKnora \
  -c "SELECT version FROM schema_migrations;"

# 롤백 후 재실행
make migrate-down
make migrate-up

# 또는 데이터베이스 초기화 (주의: 데이터 삭제)
docker-compose down -v
make dev-start
```

### 8.2 프론트엔드 관련

#### 문제 1: 서버가 시작되지 않음

**증상**:
```
Node.js version requires 20.19+
```

**해결**:
```bash
# Node.js 버전 확인
node --version

# 업그레이드
brew upgrade node

# 또는 nvm 사용
nvm install 22
nvm use 22

# 의존성 재설치
cd frontend
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

#### 문제 2: 페이지가 로드되지 않음

**증상**: 흰 화면만 표시

**해결**:
```bash
# 브라우저 콘솔 확인 (F12)
# 개발 서버 로그 확인
tail -f /tmp/weknora-frontend.log

# Vite 서버 재시작
pkill -f vite
make dev-frontend
```

### 8.3 Ollama 관련

#### 문제 1: 연결 실패

**증상**:
```
ollama service unavailable
```

**해결**:
```bash
# 1. Ollama 프로세스 확인
pgrep -fl ollama

# 2. 실행 중이 아니면 시작
ollama serve &

# 3. 연결 테스트
curl http://localhost:11434/api/version

# 4. .env 파일 확인
grep OLLAMA_BASE_URL .env
# OLLAMA_BASE_URL=http://localhost:11434 (정확해야 함)

# 5. 백엔드 재시작
pkill -f "go run"
make dev-app
```

#### 문제 2: 모델이 응답하지 않음

**증상**: 타임아웃 또는 오류

**해결**:
```bash
# 1. 모델 확인
ollama list

# 2. 모델 재다운로드
ollama pull llama3.1:8b

# 3. 직접 테스트
ollama run llama3.1:8b "안녕하세요"

# 4. 메모리 확인
# Ollama는 모델 크기의 1.5배 RAM 필요
# 8B 모델 = 약 7-8 GB RAM 필요
```

### 8.4 Docker 관련

#### 문제 1: 컨테이너가 시작되지 않음

**증상**:
```
Docker daemon not running
```

**해결**:
```bash
# 1. Docker Desktop 확인
ps aux | grep Docker

# 2. Docker Desktop 재시작
pkill -9 -f Docker
open -a Docker

# 3. 1-2분 대기 후 확인
docker ps
```

#### 문제 2: 포트 충돌

**증상**:
```
address already in use
```

**해결**:
```bash
# 포트 사용 프로세스 확인
lsof -i :8080   # 백엔드
lsof -i :5173   # 프론트엔드
lsof -i :5432   # PostgreSQL

# 프로세스 종료
kill -9 <PID>

# 또는 서비스 중지
make dev-stop
```

### 8.5 모델 등록 관련

#### 문제 1: 모델이 등록되지 않음

**증상**: 빈 모델 목록

**해결**:
```bash
# 1. Ollama 모델 확인
ollama list

# 2. 필수 모델 다운로드
ollama pull nomic-embed-text

# 3. 스크립트로 재등록
./scripts/setup/setup_models.sh

# 4. 수동 등록 (문제 지속 시)
curl -X POST http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer $TOKEN" \
  -d '{...}'
```

### 8.6 권한 문제

#### 문제: 스크립트 실행 권한 없음

**증상**:
```
Permission denied
```

**해결**:
```bash
# 실행 권한 부여
chmod +x scripts/setup/*.sh
chmod +x scripts/tests/*.sh

# 전체 스크립트에 권한 부여
find scripts -name "*.sh" -exec chmod +x {} \;
```

---

## 9. 참고 자료

### 9.1 유용한 명령어

#### 개발 환경 관리

```bash
# 전체 시작
make dev-start && make dev-app && make dev-frontend

# 전체 중지
make dev-stop
pkill -f "go run"
pkill -f "vite"

# 로그 확인
tail -f /tmp/weknora-backend.log
tail -f /tmp/weknora-frontend.log

# 데이터베이스 접속
docker exec -it WeKnora-postgres-dev psql -U postgres -d WeKnora
```

#### 빌드 및 테스트

```bash
# 백엔드 빌드
go build -o bin/server cmd/server/main.go

# 테스트 실행
go test -v ./...
go test -v ./internal/application/service

# 린팅
golangci-lint run
golangci-lint run --fix

# API 문서 생성
make docs
# http://localhost:8080/swagger/index.html
```

#### Docker 관리

```bash
# 컨테이너 상태 확인
docker ps -a
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 로그 확인
docker logs WeKnora-postgres-dev
docker logs WeKnora-redis-dev -f

# 볼륨 확인
docker volume ls | grep WeKnora

# 컨테이너 재시작
docker restart WeKnora-postgres-dev

# 전체 정리 (주의: 데이터 삭제됨)
docker-compose down -v
docker system prune -a
```

### 9.2 API 엔드포인트

#### 인증
- `POST /api/v1/auth/register` - 회원가입
- `POST /api/v1/auth/login` - 로그인
- `GET /api/v1/auth/me` - 내 정보
- `POST /api/v1/auth/refresh` - 토큰 갱신
- `POST /api/v1/auth/logout` - 로그아웃

#### 지식베이스
- `GET /api/v1/knowledge-bases` - 목록 조회
- `POST /api/v1/knowledge-bases` - 생성
- `GET /api/v1/knowledge-bases/:id` - 상세 조회
- `PUT /api/v1/knowledge-bases/:id` - 수정
- `DELETE /api/v1/knowledge-bases/:id` - 삭제

#### 문서
- `POST /api/v1/knowledge-bases/:id/documents` - 업로드
- `GET /api/v1/documents` - 목록 조회
- `GET /api/v1/documents/:id` - 상세 조회
- `DELETE /api/v1/documents/:id` - 삭제

#### 대화
- `POST /api/v1/sessions` - 세션 생성
- `GET /api/v1/sessions` - 세션 목록
- `GET /api/v1/sessions/:id` - 세션 조회
- `POST /api/v1/sessions/:id/messages` - 메시지 전송
- `GET /api/v1/sessions/:id/messages` - 대화 기록
- `DELETE /api/v1/sessions/:id` - 세션 삭제

#### 모델
- `GET /api/v1/models` - 모델 목록
- `POST /api/v1/models` - 모델 등록
- `GET /api/v1/models/:id` - 모델 조회
- `PUT /api/v1/models/:id` - 모델 수정
- `DELETE /api/v1/models/:id` - 모델 삭제

### 9.3 성능 최적화

#### Ollama 모델 선택

| 모델 | 크기 | 속도 | 품질 | 용도 |
|------|------|------|------|------|
| qwen3:4b | 2.5 GB | ⚡⚡⚡ | ⭐⭐ | 빠른 응답 |
| llama3.1:8b | 4.9 GB | ⚡⚡ | ⭐⭐⭐ | 균형 (권장) |
| gemma2:9b | 5.4 GB | ⚡ | ⭐⭐⭐⭐ | 고품질 Agent |

#### 동시성 설정

```bash
# .env 파일
CONCURRENCY_POOL_SIZE=5  # Embedding 동시 처리 수
# CPU 코어 수에 따라 조정
# 4 코어 = 3-5
# 8 코어 = 5-10
# 429 오류 발생 시 줄이기
```

#### 벡터 저장소 선택

```bash
# PostgreSQL (기본, 간편)
RETRIEVE_DRIVER=postgres
# 장점: 설치 불필요, 간단
# 단점: 대용량 시 느림

# Qdrant (대용량, 빠름)
RETRIEVE_DRIVER=qdrant
# 장점: 빠른 검색, 대용량 지원
# 단점: 추가 서비스 필요

# Elasticsearch (검색 특화)
RETRIEVE_DRIVER=elasticsearch_v8
# 장점: 하이브리드 검색 우수
# 단점: 높은 리소스 사용
```

#### 캐싱 전략

```bash
# Redis 사용 (권장)
STREAM_MANAGER_TYPE=redis
REDIS_DB=0

# Memory (개발용)
STREAM_MANAGER_TYPE=memory
```

### 9.4 프로젝트 링크

- **공식 웹사이트**: https://weknora.weixin.qq.com
- **GitHub**: https://github.com/Tencent/WeKnora
- **Swagger UI**: http://localhost:8080/swagger/index.html
- **Ollama 라이브러리**: https://ollama.com/library

### 9.5 스크립트 파일 위치

```
scripts/
├── setup/
│   ├── setup_models.sh          # AI 모델 자동 등록
│   └── add_llm_models.sh        # 추가 LLM 등록
└── tests/
    ├── test_system.sh           # 전체 시스템 테스트
    ├── test_ai_chat.sh          # AI 대화 테스트
    └── check_status.sh          # 빠른 상태 확인
```

---

## 부록 A: 작업 완료 체크리스트

### ✅ 완료된 작업

#### 코드 분석 및 수정
- [x] 코드베이스 전체 구조 분석
- [x] 주요 이슈 발견 및 문서화
- [x] Panic 이슈 11개 모두 수정
- [x] 컴파일 검증 완료

#### 환경 설정
- [x] 환경 변수 설정 (.env)
- [x] PostgreSQL 포트 충돌 해결
- [x] Ollama URL 수정 (로컬 환경용)
- [x] 인프라 서비스 시작 (Docker)
- [x] 데이터베이스 마이그레이션 (0→3)

#### 서버 실행
- [x] 백엔드 서버 시작 (포트 8080)
- [x] 프론트엔드 서버 시작 (포트 5173)
- [x] Health check 통과
- [x] API 엔드포인트 정상 작동

#### 사용자 및 인증
- [x] 첫 사용자 계정 생성 (admin)
- [x] 로그인 기능 검증
- [x] JWT 토큰 발급 확인
- [x] 테넌트 자동 생성 확인

#### AI 모델 설정
- [x] Ollama 서비스 연결
- [x] Embedding 모델 다운로드 (nomic-embed-text)
- [x] LLM 모델 다운로드 (llama3.1, gemma2, qwen3)
- [x] 모델 자동 등록 (4개)
- [x] 모델 동작 검증

#### 자동화 스크립트
- [x] 모델 설정 스크립트 작성
- [x] 시스템 테스트 스크립트 작성
- [x] AI 대화 테스트 스크립트 작성
- [x] 상태 확인 스크립트 작성
- [x] 스크립트 실행 권한 설정

#### 테스트 및 검증
- [x] 전체 시스템 통합 테스트
- [x] AI 대화 기능 테스트
- [x] 지식베이스 CRUD 테스트
- [x] 성능 측정 및 문서화
- [x] 한국어 지원 확인

#### 문서화
- [x] 설치 가이드 작성
- [x] 스크립트 사용법 문서화
- [x] API 엔드포인트 목록
- [x] 트러블슈팅 가이드
- [x] 성능 최적화 팁

### 📊 최종 상태

#### 실행 중인 서비스
```
✅ 백엔드 (포트 8080) - 정상
✅ 프론트엔드 (포트 5173) - 정상
✅ PostgreSQL (포트 5432) - healthy
✅ Redis (포트 6379) - running
✅ DocReader (포트 50051) - healthy
✅ Qdrant (포트 6333) - running
✅ Neo4j (포트 7474) - running
✅ Jaeger (포트 16686) - running
✅ Ollama (포트 11434) - 정상
```

#### 등록된 AI 모델
```
✅ nomic-embed-text:latest (Embedding, 768차원)
✅ llama3.1:8b (LLM, 기본 모델)
✅ gemma2:9b (LLM, Agent 모드)
✅ qwen3:8b (LLM, 다국어)
```

#### 테스트 결과
```
✅ 백엔드 Health Check - 통과
✅ 프론트엔드 로드 - 통과
✅ 사용자 인증 - 통과
✅ Ollama 연결 - 통과
✅ 모델 등록 - 통과 (4개)
✅ 지식베이스 CRUD - 통과
✅ AI 대화 - 통과 (한국어 지원)
✅ Docker 인프라 - 모두 정상
```

---

## 부록 B: 빠른 참조

### 서비스 URL

```
웹 UI:        http://localhost:5173
백엔드 API:   http://localhost:8080
Swagger:      http://localhost:8080/swagger/index.html
Ollama:       http://localhost:11434
PostgreSQL:   localhost:5432
Redis:        localhost:6379
Qdrant:       http://localhost:6333
Neo4j:        http://localhost:7474
Jaeger:       http://localhost:16686
```

### 기본 계정

```
이메일:    admin@example.com
비밀번호:  admin123
테넌트:    admin's Workspace
```

### 필수 명령어

```bash
# 시작
make dev-start
make dev-app
make dev-frontend

# 중지
make dev-stop

# 테스트
./scripts/tests/test_system.sh

# 상태 확인
./scripts/tests/check_status.sh

# 모델 설정
./scripts/setup/setup_models.sh
```

---

**문서 버전**: 1.0
**최종 업데이트**: 2026-02-26
**작성자**: Development Team
**상태**: 완료 ✅

**이 문서에 대한 피드백이나 개선 제안은 GitHub Issues로 제출해주세요.**
