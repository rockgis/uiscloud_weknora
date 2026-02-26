# WeKnora 최종 운영 가이드

**버전**: 2.0 (최종판)
**작성일**: 2025-12-21
**프로젝트**: WeKnora - LLM 기반 RAG 프레임워크
**환경**: macOS 로컬 개발 환경

---

## 📑 목차

- [1. 프로젝트 개요](#1-프로젝트-개요)
- [2. 시스템 구성 요소](#2-시스템-구성-요소)
- [3. 빠른 시작 가이드](#3-빠른-시작-가이드)
- [4. 시스템 재시작 절차](#4-시스템-재시작-절차)
- [5. 전체 테스트 실행](#5-전체-테스트-실행)
- [6. 웹 UI 사용 가이드](#6-웹-ui-사용-가이드)
- [7. 시스템 종료 가이드](#7-시스템-종료-가이드)
- [8. 자동화 스크립트](#8-자동화-스크립트)
- [9. 트러블슈팅](#9-트러블슈팅)
- [10. 참고 자료](#10-참고-자료)

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
| 프론트엔드 | Vue 3, Vite 7.2, TypeScript 5.8, TDesign |
| 문서 파서 | Python, gRPC, PaddleOCR |
| 데이터베이스 | PostgreSQL (ParadeDB), Redis |
| 벡터 저장소 | pgvector, Qdrant |
| 그래프 DB | Neo4j |
| LLM | Ollama (로컬) |

---

## 2. 시스템 구성 요소

### 2.1 인프라 서비스 (Docker)

| 서비스 | 포트 | 설명 | 상태 확인 |
|--------|------|------|----------|
| PostgreSQL | 5432 | ParadeDB (pgvector 포함) | `docker ps | grep postgres` |
| Redis | 6379 | 캐시 및 큐 | `docker ps | grep redis` |
| Qdrant | 6333 | 벡터 데이터베이스 | `docker ps | grep qdrant` |
| Neo4j | 7474, 7687 | 그래프 데이터베이스 | `docker ps | grep neo4j` |
| Jaeger | 16686 | 분산 트레이싱 | `docker ps | grep jaeger` |
| DocReader | 50051 | 문서 파싱 gRPC | `docker ps | grep docreader` |
| MinIO | 9000, 9001 | 객체 스토리지 | `docker ps | grep minio` |

### 2.2 애플리케이션 서비스

| 서비스 | 포트 | 설명 | 상태 확인 |
|--------|------|------|----------|
| 백엔드 | 8080 | Go REST API | `curl http://localhost:8080/health` |
| 프론트엔드 | 5173 | Vue 3 SPA | `curl http://localhost:5173` |
| Ollama | 11434 | LLM 추론 엔진 | `curl http://localhost:11434/api/version` |

### 2.3 등록된 AI 모델

| 모델명 | 타입 | 설명 | 기본값 |
|--------|------|------|--------|
| nomic-embed-text:latest | Embedding | 768차원 벡터 임베딩 | - |
| llama3.1:8b | LLM | 일반 대화 모델 | ⭐ |
| gemma2:9b | KnowledgeQA | Agent 모드용 고성능 모델 | - |
| qwen3:8b | KnowledgeQA | 다국어 지원 모델 | - |

---

## 3. 빠른 시작 가이드

### 3.1 사전 요구 사항

```bash
# 필수 소프트웨어
- Docker Desktop (실행 중)
- Go 1.24+
- Node.js 22+ / pnpm
- Python 3.11+
- Ollama (설치 및 실행 중)

# Ollama 모델 다운로드
ollama pull llama3.1:8b
ollama pull gemma2:9b
ollama pull qwen3:8b
ollama pull nomic-embed-text
```

### 3.2 환경 설정

```bash
# .env 파일 확인 (주요 설정)
DB_HOST=localhost
DB_PORT=5432
OLLAMA_BASE_URL=http://localhost:11434
DOCREADER_ADDR=localhost:50051
```

### 3.3 시작 명령어

```bash
# 1단계: 인프라 서비스 시작
make dev-start

# 2단계: 백엔드 시작 (새 터미널)
make dev-app

# 3단계: 프론트엔드 시작 (새 터미널)
make dev-frontend

# 4단계: 상태 확인
./scripts/tests/check_status.sh
```

### 3.4 초기 설정

```bash
# AI 모델 자동 등록
./scripts/setup/setup_models.sh

# 추가 모델 등록 (선택 사항)
./scripts/setup/add_llm_models.sh
```

### 3.5 웹 UI 접속

- **URL**: http://localhost:5173
- **로그인**: admin@example.com / admin123

---

## 4. 시스템 재시작 절차

### 4.1 완전 재시작 (클린 시작)

```bash
# 1. 프론트엔드 종료
kill -15 $(lsof -ti:5173)

# 2. 백엔드 종료
kill -15 $(lsof -ti:8080)

# 3. Docker 인프라 종료
make dev-stop

# 4. 남은 컨테이너 정리
docker stop $(docker ps -q --filter "name=WeKnora")
docker rm $(docker ps -aq --filter "name=WeKnora")

# 5. 인프라 재시작
make dev-start

# 6. 백엔드 재시작 (5초 대기 후)
sleep 5 && make dev-app &

# 7. 프론트엔드 재시작 (10초 대기 후)
sleep 10 && make dev-frontend &

# 8. 상태 확인 (15초 대기 후)
sleep 15 && ./scripts/tests/check_status.sh
```

### 4.2 재시작 검증 결과

```
==========================================
  WeKnora 시스템 상태 확인
==========================================

✓ 백엔드 정상 (포트 8080)
✓ 프론트엔드 정상 (포트 5173)
✓ Ollama 정상 (포트 11434)

Docker 인프라 서비스:
✓ WeKnora-postgres-dev: Up (healthy)
✓ WeKnora-redis-dev: Up
✓ WeKnora-qdrant-dev: Up
✓ WeKnora-neo4j-dev: Up
✓ WeKnora-jaeger-dev: Up
✓ WeKnora-docreader-dev: Up (healthy)
✓ WeKnora-minio-dev: Up (healthy)
```

---

## 5. 전체 테스트 실행

### 5.1 시스템 통합 테스트

```bash
chmod +x scripts/tests/test_system.sh
./scripts/tests/test_system.sh
```

#### 테스트 항목 및 결과

| # | 테스트 항목 | 결과 | 세부 사항 |
|---|-------------|------|-----------|
| 1 | 백엔드 서버 상태 | ✓ | http://localhost:8080 |
| 2 | 프론트엔드 서버 상태 | ✓ | http://localhost:5173 |
| 3 | 사용자 인증 | ✓ | JWT 토큰 발급 성공 |
| 4 | Ollama 연결 | ✓ | http://localhost:11434 |
| 5 | AI 모델 확인 | ✓ | 4개 모델 등록됨 |
| 6 | 지식베이스 생성/삭제 | ✓ | CRUD 작업 성공 |
| 7 | Docker 인프라 | ✓ | 7개 컨테이너 정상 |

**통과율: 7/7 (100%)**

### 5.2 AI 대화 기능 테스트

```bash
chmod +x scripts/tests/test_ai_chat.sh
./scripts/tests/test_ai_chat.sh
```

#### 테스트 항목 및 결과

| # | 테스트 항목 | 결과 | 세부 사항 |
|---|-------------|------|-----------|
| 1 | 대화 세션 생성 | ✓ | Session ID 발급 성공 |
| 2 | LLM 응답 | ✓ | llama3.1:8b 한국어 응답 |
| 3 | Embedding | ✓ | 768차원 벡터 생성 |
| 4 | 세션 정리 | ✓ | 삭제 성공 |

**통과율: 4/4 (100%)**

### 5.3 전체 테스트 요약

```
==========================================
  전체 테스트 결과
==========================================

총 테스트: 11개
통과: 11개
실패: 0개
통과율: 100%

테스트 실행 시간: 약 30초
```

---

## 6. 웹 UI 사용 가이드

### 6.1 로그인

1. 브라우저에서 http://localhost:5173 접속
2. 로그인 정보 입력:
   - **이메일**: admin@example.com
   - **비밀번호**: admin123
3. "로그인" 버튼 클릭

### 6.2 지식베이스 생성

#### 6.2.1 웹 UI에서 생성

1. 좌측 메뉴에서 **"지식베이스"** 클릭
2. **"+ 새 지식베이스"** 버튼 클릭
3. 정보 입력:
   - **이름**: 지식베이스 이름
   - **설명**: 지식베이스 설명
   - **타입**: document 또는 faq 선택
4. **"생성"** 버튼 클릭

#### 6.2.2 API로 생성 (테스트 완료)

```bash
# 로그인
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}'

# 지식베이스 생성
curl -X POST http://localhost:8080/api/v1/knowledge-bases \
  -H "Authorization: Bearer {TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "웹 UI 테스트 지식베이스",
    "description": "테스트용 지식베이스입니다.",
    "type": "document"
  }'
```

#### 6.2.3 생성 테스트 결과

```json
{
  "success": true,
  "data": {
    "id": "dc8842fb-4c98-47bd-91a2-1f9132dca126",
    "name": "웹 UI 테스트 지식베이스 (수정됨)",
    "type": "document",
    "description": "업데이트 테스트: 지식베이스 설명이 수정되었습니다.",
    "tenant_id": 10000,
    "created_at": "2025-12-21T01:25:11.357529+09:00",
    "updated_at": "2025-12-21T01:25:57.171928+09:00",
    "knowledge_count": 0,
    "chunk_count": 0,
    "is_processing": false
  }
}
```

### 6.3 문서 업로드

1. 생성된 지식베이스 클릭
2. **"문서 업로드"** 버튼 클릭
3. 파일 선택 (PDF, DOCX, TXT 등)
4. 자동 파싱 및 벡터 인덱싱 시작

### 6.4 AI 대화

1. 좌측 메뉴에서 **"대화"** 클릭
2. **"새 대화"** 버튼 클릭
3. 대화 모드 선택:
   - **Normal**: 일반 대화
   - **Knowledge QA**: 지식베이스 검색
   - **Agent**: 도구 사용 모드
4. 질문 입력 후 전송

### 6.5 CRUD 작업 검증

| 작업 | 엔드포인트 | 결과 |
|------|-----------|------|
| CREATE | POST /api/v1/knowledge-bases | ✓ 성공 |
| READ (List) | GET /api/v1/knowledge-bases | ✓ 성공 |
| READ (Detail) | GET /api/v1/knowledge-bases/{id} | ✓ 성공 |
| UPDATE | PUT /api/v1/knowledge-bases/{id} | ✓ 성공 |
| DELETE | DELETE /api/v1/knowledge-bases/{id} | ✓ 성공 |

---

## 7. 시스템 종료 가이드

### 7.1 정상 종료 (권장)

```bash
# 1. 프론트엔드 종료 (Ctrl+C 또는)
kill -15 $(lsof -ti:5173)

# 2. 백엔드 종료 (Ctrl+C 또는)
kill -15 $(lsof -ti:8080)

# 3. Docker 인프라 종료
make dev-stop

# 4. 종료 확인
docker ps --filter "name=WeKnora"
# (아무것도 출력되지 않으면 성공)
```

### 7.2 강제 종료

```bash
# 모든 프로세스 강제 종료
kill -9 $(lsof -ti:8080)
kill -9 $(lsof -ti:5173)

# 모든 Docker 컨테이너 강제 종료 및 삭제
docker stop $(docker ps -q --filter "name=WeKnora")
docker rm $(docker ps -aq --filter "name=WeKnora")

# 네트워크 정리 (필요시)
docker network prune -f
```

### 7.3 Ollama 종료 (선택 사항)

```bash
# Ollama 프로세스 종료
kill -15 $(lsof -ti:11434)

# 또는 macOS 앱 종료
# Ollama 메뉴바 아이콘 → Quit Ollama
```

### 7.4 데이터 보존

**종료 시 보존되는 데이터:**
- PostgreSQL 데이터베이스 (Docker volume)
- Redis 데이터 (영구 저장 설정 시)
- Qdrant 벡터 인덱스
- Neo4j 그래프 데이터
- MinIO 객체 스토리지
- 업로드된 문서 파일

**삭제되는 데이터:**
- 메모리 캐시
- 세션 데이터 (JWT 토큰 유효)
- 임시 파일

### 7.5 완전 삭제 (초기화)

```bash
# ⚠️ 경고: 모든 데이터가 삭제됩니다!

# Docker 볼륨 삭제
docker volume rm $(docker volume ls -q | grep WeKnora)

# 데이터베이스 파일 삭제 (로컬 설정 시)
# rm -rf ./data/*

# 로그 파일 삭제
# rm -rf ./logs/*
```

---

## 8. 자동화 스크립트

### 8.1 스크립트 개요

| 스크립트 | 경로 | 설명 | 실행 시간 |
|---------|------|------|----------|
| check_status.sh | scripts/tests/ | 빠른 상태 확인 | ~3초 |
| test_system.sh | scripts/tests/ | 전체 시스템 테스트 | ~30초 |
| test_ai_chat.sh | scripts/tests/ | AI 대화 기능 테스트 | ~15초 |
| setup_models.sh | scripts/setup/ | AI 모델 자동 등록 | ~10초 |
| add_llm_models.sh | scripts/setup/ | 추가 LLM 등록 | ~5초 |

### 8.2 상태 확인 스크립트

**파일**: `scripts/tests/check_status.sh`

```bash
#!/bin/bash
# WeKnora 시스템 상태 확인 스크립트

echo "=========================================="
echo "  WeKnora 시스템 상태 확인"
echo "=========================================="
echo ""

# 서버 상태
curl -s http://localhost:8080/health | grep -q ok && echo "✓ 백엔드 정상 (포트 8080)" || echo "✗ 백엔드 오류"
curl -s http://localhost:5173 | grep -q WeKnora && echo "✓ 프론트엔드 정상 (포트 5173)" || echo "✗ 프론트엔드 오류"
curl -s http://localhost:11434/api/version | grep -q version && echo "✓ Ollama 정상 (포트 11434)" || echo "✗ Ollama 오류"

echo ""
echo "Docker 인프라 서비스:"
docker ps --format "✓ {{.Names}}: {{.Status}}" | grep WeKnora

echo ""
echo "=========================================="
echo "  상태 확인 완료"
echo "=========================================="
```

**사용법**:
```bash
chmod +x scripts/tests/check_status.sh
./scripts/tests/check_status.sh
```

### 8.3 시스템 테스트 스크립트

**파일**: `scripts/tests/test_system.sh` (134줄)

**주요 기능**:
1. 백엔드/프론트엔드 서버 상태 확인
2. 사용자 인증 테스트 (JWT)
3. Ollama 연결 테스트
4. 등록된 AI 모델 확인
5. 지식베이스 CRUD 테스트
6. Docker 인프라 상태 확인

### 8.4 AI 대화 테스트 스크립트

**파일**: `scripts/tests/test_ai_chat.sh` (97줄)

**주요 기능**:
1. 대화 세션 생성/삭제
2. Ollama LLM 응답 테스트
3. Embedding 모델 테스트 (768차원)
4. 한국어 질문 응답 검증

### 8.5 모델 설정 스크립트

**파일**: `scripts/setup/setup_models.sh`

**등록 모델**:
- nomic-embed-text:latest (Embedding, 기본값)
- llama3.1:8b (LLM, 기본값)

**파일**: `scripts/setup/add_llm_models.sh`

**추가 모델**:
- gemma2:9b (Agent 모드용)
- qwen3:8b (다국어 지원)

---

## 9. 트러블슈팅

### 9.1 포트 충돌

**증상**: "address already in use" 오류

**해결**:
```bash
# 8080 포트 사용 중인 프로세스 확인
lsof -i :8080

# 프로세스 종료
kill -9 <PID>

# 또는 모든 관련 프로세스 종료
lsof -ti:8080 | xargs kill -9
```

### 9.2 Docker 컨테이너 시작 실패

**증상**: "Cannot connect to Docker daemon"

**해결**:
```bash
# Docker Desktop 실행 확인
open -a Docker

# Docker 상태 확인 (1분 대기)
sleep 60 && docker ps

# Docker 재시작
killall Docker && open -a Docker
```

### 9.3 데이터베이스 연결 오류

**증상**: "connection refused" 또는 "invalid port"

**해결**:
```bash
# .env 파일 확인
cat .env | grep DB_

# 필수 변수 확인
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=postgres
# DB_PASSWORD=postgres
# DB_NAME=weknora

# PostgreSQL 컨테이너 확인
docker logs WeKnora-postgres-dev
```

### 9.4 Ollama 연결 실패

**증상**: "ollama service unavailable"

**해결**:
```bash
# Ollama 실행 확인
curl http://localhost:11434/api/version

# Ollama 재시작
killall ollama
ollama serve &

# .env 확인 (Docker 환경 주소 사용 금지)
# ✗ OLLAMA_BASE_URL=http://host.docker.internal:11434
# ✓ OLLAMA_BASE_URL=http://localhost:11434
```

### 9.5 프론트엔드 빌드 오류

**증상**: "Module not found" 또는 의존성 오류

**해결**:
```bash
cd frontend

# 의존성 재설치
rm -rf node_modules
pnpm install

# 캐시 정리
pnpm store prune

# 개발 서버 재시작
pnpm dev
```

### 9.6 백엔드 컴파일 오류

**증상**: "undefined: xxx" 또는 타입 오류

**해결**:
```bash
# Go 모듈 정리
go mod tidy
go mod download

# 캐시 정리
go clean -cache -modcache

# 재빌드
go build -o main cmd/server/main.go
```

### 9.7 AI 모델 미등록

**증상**: 로그인 후 "모델이 설정되지 않았습니다"

**해결**:
```bash
# Ollama에 모델 다운로드
ollama pull llama3.1:8b
ollama pull nomic-embed-text

# 모델 자동 등록
./scripts/setup/setup_models.sh

# 등록 확인
curl -H "Authorization: Bearer {TOKEN}" \
  http://localhost:8080/api/v1/models
```

### 9.8 지식베이스 생성 실패

**증상**: "Config required" 오류

**해결**:
API 요청 시 `config` 필드 포함:
```json
{
  "name": "지식베이스 이름",
  "description": "설명",
  "type": "document",
  "config": {
    "chunking_config": {
      "chunk_size": 0,
      "chunk_overlap": 0
    },
    "embedding_model_id": "",
    "summary_model_id": ""
  }
}
```

---

## 10. 참고 자료

### 10.1 주요 URL

| 서비스 | URL | 용도 |
|--------|-----|------|
| 웹 UI | http://localhost:5173 | 사용자 인터페이스 |
| API | http://localhost:8080 | REST API |
| Swagger | http://localhost:8080/swagger/index.html | API 문서 |
| Jaeger UI | http://localhost:16686 | 분산 트레이싱 |
| Neo4j Browser | http://localhost:7474 | 그래프 DB 관리 |
| MinIO Console | http://localhost:9001 | 객체 스토리지 관리 |

### 10.2 주요 API 엔드포인트

#### 인증
- POST `/api/v1/auth/register` - 사용자 등록
- POST `/api/v1/auth/login` - 로그인
- POST `/api/v1/auth/refresh` - 토큰 갱신

#### 지식베이스
- GET `/api/v1/knowledge-bases` - 목록 조회
- POST `/api/v1/knowledge-bases` - 생성
- GET `/api/v1/knowledge-bases/{id}` - 상세 조회
- PUT `/api/v1/knowledge-bases/{id}` - 수정
- DELETE `/api/v1/knowledge-bases/{id}` - 삭제

#### AI 모델
- GET `/api/v1/models` - 모델 목록
- POST `/api/v1/models` - 모델 등록
- PUT `/api/v1/models/{id}/default` - 기본 모델 설정

#### 대화
- POST `/api/v1/sessions` - 세션 생성
- POST `/api/v1/conversations` - 대화 시작
- GET `/api/v1/conversations/{id}/stream` - SSE 스트리밍

### 10.3 프로젝트 구조

```
WeKnora/
├── cmd/server/              # 애플리케이션 진입점
├── internal/                # 내부 패키지
│   ├── agent/              # ReACT Agent 구현
│   ├── application/        # 비즈니스 로직
│   ├── handler/            # HTTP 핸들러 (17개)
│   ├── types/              # 데이터 모델 (~30개)
│   ├── middleware/         # JWT, CORS, 트레이싱
│   ├── searchutil/         # 하이브리드 검색
│   └── database/           # GORM 추상화
├── frontend/               # Vue 3 프론트엔드
│   ├── src/api/           # Axios 클라이언트
│   ├── src/views/         # 페이지 컴포넌트
│   ├── src/components/    # 재사용 컴포넌트
│   └── src/stores/        # Pinia 상태 관리
├── docreader/             # Python 문서 파서
├── scripts/               # 자동화 스크립트
│   ├── setup/            # 설정 스크립트
│   └── tests/            # 테스트 스크립트
├── docs/                  # 문서
└── migrations/            # DB 마이그레이션
```

### 10.4 코드베이스 통계

| 언어 | 파일 수 | 주요 용도 |
|------|---------|----------|
| Go | 238 | 백엔드 API, 비즈니스 로직 |
| Vue/TS | 103 | 프론트엔드 UI |
| Python | 45 | 문서 파싱 (gRPC) |
| SQL | 3 | 데이터베이스 마이그레이션 |
| Shell | 7 | 자동화 스크립트 |

### 10.5 중요 수정 사항

**Panic 이슈 수정 (11개 → 0개)**:
- `internal/application/service/tenant.go` - API 키 생성 (3개)
- `internal/application/service/user.go` - JWT 비밀키 (1개)
- `internal/application/service/dataset.go` - Parquet 로딩 (5개)
- `internal/models/utils/slices.go` - 슬라이스 청킹 (1개)
- `internal/container/container.go` - DI 컨테이너 (1개)

**환경 변수 수정**:
- `OLLAMA_BASE_URL`: `host.docker.internal` → `localhost`
- 추가: `DB_PORT`, `DB_HOST`, `DOCREADER_ADDR`

### 10.6 성능 메트릭

| 메트릭 | 값 | 측정 시점 |
|--------|-----|----------|
| 백엔드 Health Check | 17µs | 재시작 후 |
| 로그인 응답 시간 | ~50ms | 인증 테스트 |
| 지식베이스 생성 | ~200ms | CRUD 테스트 |
| LLM 응답 (llama3.1) | ~2-3초 | AI 대화 테스트 |
| Embedding 생성 | ~100ms | 768차원 벡터 |

---

## 부록 A. 빠른 참조 명령어

### 시작
```bash
make dev-start          # 인프라 시작
make dev-app            # 백엔드 시작
make dev-frontend       # 프론트엔드 시작
```

### 종료
```bash
kill -15 $(lsof -ti:8080 :5173)  # 애플리케이션 종료
make dev-stop                     # 인프라 종료
```

### 테스트
```bash
./scripts/tests/check_status.sh   # 빠른 상태 확인
./scripts/tests/test_system.sh    # 전체 테스트
./scripts/tests/test_ai_chat.sh   # AI 테스트
```

### 설정
```bash
./scripts/setup/setup_models.sh     # 모델 등록
./scripts/setup/add_llm_models.sh   # 추가 모델
```

---

## 부록 B. 최종 시스템 상태 (2025-12-21)

### 재시작 완료 시각
- 2025-12-21 01:23:00 (KST)

### 서비스 상태
```
✓ 백엔드 정상 (포트 8080)
✓ 프론트엔드 정상 (포트 5173)
✓ Ollama 정상 (포트 11434)
✓ PostgreSQL Up (healthy)
✓ Redis Up
✓ Qdrant Up
✓ Neo4j Up
✓ Jaeger Up
✓ DocReader Up (healthy)
✓ MinIO Up (healthy)
```

### 테스트 결과
- 시스템 통합 테스트: 7/7 통과 (100%)
- AI 대화 기능 테스트: 4/4 통과 (100%)
- 전체 통과율: 11/11 (100%)

### 등록된 계정
- 사용자: admin@example.com
- 비밀번호: admin123
- Tenant ID: 10000

### 등록된 모델
- nomic-embed-text:latest (Embedding)
- llama3.1:8b (LLM, 기본값)
- gemma2:9b (KnowledgeQA)
- qwen3:8b (KnowledgeQA)

### 생성된 지식베이스
- ID: dc8842fb-4c98-47bd-91a2-1f9132dca126
- 이름: "웹 UI 테스트 지식베이스 (수정됨)"
- 타입: document
- 상태: CRUD 작업 모두 검증 완료

---

**문서 끝**

이 가이드는 WeKnora 시스템의 전체 라이프사이클을 다룹니다:
- ✅ 설치 및 초기 설정
- ✅ 시스템 재시작
- ✅ 전체 테스트 실행
- ✅ 웹 UI 사용
- ✅ 시스템 종료
- ✅ 트러블슈팅

**작성자**: Claude Code
**최종 업데이트**: 2025-12-21 01:30:00 KST
