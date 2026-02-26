# WeKnora 아키텍처 문서

> 버전: 0.2.2 | 최종 업데이트: 2026-02-26

---

## 목차

1. [프로젝트 개요](#1-프로젝트-개요)
2. [기술 스택](#2-기술-스택)
3. [전체 시스템 구조](#3-전체-시스템-구조)
4. [백엔드 아키텍처 (Go)](#4-백엔드-아키텍처-go)
5. [채팅 파이프라인](#5-채팅-파이프라인)
6. [ReACT 에이전트](#6-react-에이전트)
7. [데이터베이스 스키마](#7-데이터베이스-스키마)
8. [프론트엔드 아키텍처 (Vue 3)](#8-프론트엔드-아키텍처-vue-3)
9. [문서 파서 서비스 (docreader)](#9-문서-파서-서비스-docreader)
10. [인프라 및 배포](#10-인프라-및-배포)
11. [핵심 기능 흐름](#11-핵심-기능-흐름)
12. [보안 구조](#12-보안-구조)
13. [설정 관리](#13-설정-관리)

---

## 1. 프로젝트 개요

WeKnora는 문서 이해 및 시맨틱 검색을 위한 **LLM 기반 RAG (Retrieval-Augmented Generation) 프레임워크**다. 멀티모달 문서 전처리, 시맨틱 벡터 인덱싱, 하이브리드 검색, LLM 추론을 결합하며 멀티테넌시를 지원한다.

### 핵심 기능

| 기능 | 설명 |
|------|------|
| 문서 파싱 | PDF, Word, Excel, CSV, 이미지, 웹페이지 등 18종 포맷 지원 |
| 하이브리드 검색 | BM25 키워드 + 벡터 유사도 + 지식 그래프 통합 검색 |
| ReACT 에이전트 | 다중 도구를 활용한 멀티턴 추론 (최대 30회 반복) |
| FAQ 지식베이스 | 구조화된 Q&A 관리 및 검색 |
| MCP 통합 | Model Context Protocol로 외부 도구 확장 |
| 멀티테넌시 | 테넌트별 데이터 격리 및 모델 공유 |
| SSE 스트리밍 | 실시간 응답 스트리밍 |
| 평가 시스템 | BLEU, ROUGE, MRR, NDCG, MAP 등 RAG 품질 평가 |

---

## 2. 기술 스택

### 백엔드

| 구성요소 | 기술 | 버전 |
|---------|------|------|
| 언어 | Go | 1.24+ |
| 웹 프레임워크 | Gin | v1.11.0 |
| ORM | GORM | v1.x |
| 비동기 큐 | Asynq (Redis 기반) | v0.25.1 |
| 분산 추적 | OpenTelemetry + Jaeger | - |
| 문서 파서 통신 | gRPC | - |
| MCP 클라이언트 | mark3labs/mcp-go | v0.43.0 |

### 데이터 저장소

| 역할 | 기술 | 용도 |
|------|------|------|
| 주 데이터베이스 | PostgreSQL (ParadeDB v0.18.9) | 모든 비즈니스 데이터 |
| 벡터 저장소 (기본) | pgvector (PostgreSQL 확장) | 임베딩 저장 및 검색 |
| 벡터 저장소 (대안) | Qdrant v1.16.2 | 고성능 벡터 검색 |
| 전문 검색 | Elasticsearch v7/v8 | BM25 키워드 검색 |
| 그래프 DB | Neo4j 2025.10.1 | 지식 그래프 |
| 캐시/큐 | Redis 7.0 | 메시지 큐, 세션 캐시 |
| 파일 저장소 | 로컬 / MinIO / Tencent COS | 업로드 파일 |

### 프론트엔드

| 구성요소 | 기술 | 버전 |
|---------|------|------|
| 프레임워크 | Vue 3 | ^3.5.13 |
| 번들러 | Vite | - |
| 언어 | TypeScript | - |
| UI 컴포넌트 | TDesign Vue Next | v1.17.2 |
| 상태 관리 | Pinia | ^3.0.1 |
| HTTP 클라이언트 | Axios | ^1.8.4 |
| 국제화 | vue-i18n | ^11.1.12 |

### 문서 파서 (docreader)

| 구성요소 | 기술 |
|---------|------|
| 언어 | Python |
| 서버 프로토콜 | gRPC |
| 패키지 관리 | uv (pyproject.toml) |

---

## 3. 전체 시스템 구조

```
┌─────────────────────────────────────────────────────────────────┐
│                         클라이언트                               │
│   브라우저 (Vue 3 + TypeScript)          MCP 클라이언트 / SDK    │
└───────────────────┬────────────────────────────────────────────-┘
                    │ HTTP / SSE
┌───────────────────▼─────────────────────────────────────────────┐
│                    Go 백엔드 (Gin)  :8080                        │
│                                                                  │
│  ┌─────────┐  ┌──────────┐  ┌──────────┐  ┌─────────────────┐  │
│  │Middleware│  │  Router  │  │ Handlers │  │  Swagger /docs  │  │
│  │ JWT/Auth │  │  /api/v1 │  │  (14개+) │  │                 │  │
│  │ CORS     │  │          │  │          │  │                 │  │
│  │ Tracing  │  └────┬─────┘  └────┬─────┘  └─────────────────┘  │
│  └─────────┘       │             │                              │
│                     └──────┬──────┘                             │
│                             ▼                                    │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 비즈니스 서비스 레이어                     │    │
│  │                                                          │    │
│  │  SessionService │ KnowledgeService │ AgentService        │    │
│  │  UserService    │ ChunkService     │ ModelService        │    │
│  │  TenantService  │ MCPService       │ WebSearchService    │    │
│  │  EvaluationService │ TagService   │ FAQService           │    │
│  └────────┬────────────────┬──────────────────────────────┘    │
│           │                │                                    │
│  ┌────────▼────────┐  ┌────▼──────────────────────────────┐    │
│  │  채팅 파이프라인  │  │         에이전트 엔진               │    │
│  │  (16개 플러그인) │  │    ReACT 루프 (최대 30회)           │    │
│  │  PluginTracing   │  │    14개+ 도구 (Tools)              │    │
│  │  PluginSearch    │  │    EventBus 기반 스트리밍           │    │
│  │  PluginRerank    │  └───────────────────────────────────┘    │
│  │  PluginMerge     │                                           │
│  │  PluginChat      │                                           │
│  └────────┬─────────┘                                           │
│           │                                                     │
│  ┌────────▼─────────────────────────────────────────────────┐   │
│  │                   레포지토리 레이어                         │   │
│  │  Tenant │ User │ KnowledgeBase │ Knowledge │ Chunk        │   │
│  │  Session │ Message │ Model │ MCP │ Neo4j                  │   │
│  └─────────────────────────────────────────────────────────┘   │
└──────┬─────────────────────────────┬────────────────────────────┘
       │ SQL/pgvector                 │ gRPC
┌──────▼─────────┐            ┌──────▼──────────────────────────┐
│   PostgreSQL   │            │      docreader (Python)  :50051  │
│   (ParadeDB)   │            │      18개 문서 파서               │
│   + Qdrant     │            │      OCR, 이미지 처리             │
│   + Elasticsearch│          └─────────────────────────────────┘
│   + Neo4j      │
│   + Redis      │
└────────────────┘
```

---

## 4. 백엔드 아키텍처 (Go)

### 4.1 진입점: `cmd/server/main.go`

```
main()
├── config.LoadConfig()              # Viper 설정 로드
├── container.BuildContainer()       # 의존성 주입 컨테이너 구성
│   ├── 인프라 초기화 (DB, Redis, gRPC)
│   ├── 레포지토리 생성 (11개)
│   ├── 서비스 생성 (16개+)
│   ├── 파이프라인 플러그인 등록 (16개)
│   └── 핸들러 생성 (14개+)
├── router.NewRouter()               # Gin 라우터 설정
├── http.Server.ListenAndServe()     # HTTP 서버 시작
├── signal.Notify()                  # OS 신호 감시 (SIGINT, SIGTERM)
└── resourceCleaner.Cleanup()        # 종료 시 리소스 정리
```

### 4.2 의존성 주입: `internal/container/`

DI 컨테이너가 생성 순서에 따라 모든 컴포넌트를 조립한다.

```
Layer 1 - 인프라
  ResourceCleaner → Config → Tracer → Database → Redis → AntsPool

Layer 2 - 외부 서비스
  DocReaderClient (gRPC) → OllamaService → Neo4jClient → StreamManager

Layer 3 - 레포지토리 (11개)
  TenantRepo → UserRepo → AuthTokenRepo
  KnowledgeBaseRepo → KnowledgeRepo → ChunkRepo
  SessionRepo → MessageRepo → ModelRepo
  MCPServiceRepo → Neo4jRepo

Layer 4 - 서비스 (16개+)
  UserService → TenantService → ModelService
  KnowledgeService → ChunkService → KnowledgeBaseService
  MCPService → WebSearchService → AgentService
  SessionService → MessageService → EvaluationService

Layer 5 - 파이프라인 플러그인 (16개)
  PluginTracing → PluginSearch → PluginRerank → PluginMerge
  → PluginIntoChatMessage → PluginChat → PluginStreamFilter

Layer 6 - HTTP 핸들러 (14개+)
  AuthHandler → TenantHandler → KnowledgeBaseHandler
  KnowledgeHandler → ChunkHandler → FAQHandler → TagHandler
  SessionHandler → MessageHandler → ModelHandler
  EvaluationHandler → InitializationHandler
  SystemHandler → MCPServiceHandler → WebSearchHandler

Layer 7 - 라우터 및 비동기 큐
  Router (Gin) → AsynqClient → AsynqServer
```

### 4.3 API 엔드포인트 목록

모든 API는 `/api/v1` 접두사를 가지며 JWT 인증이 필요하다 (인증 엔드포인트 제외).

#### 인증 (`/api/v1/auth/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/auth/register` | 사용자 등록 |
| POST | `/auth/login` | 로그인 (JWT 발급) |
| POST | `/auth/refresh` | 액세스 토큰 갱신 |
| GET | `/auth/validate` | 토큰 유효성 검증 |
| POST | `/auth/logout` | 로그아웃 |
| GET | `/auth/me` | 현재 사용자 정보 |
| POST | `/auth/change-password` | 비밀번호 변경 |

#### 테넌트 (`/api/v1/tenants/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/tenants` | 테넌트 생성 |
| GET | `/tenants` | 테넌트 목록 |
| GET | `/tenants/:id` | 테넌트 상세 |
| PUT | `/tenants/:id` | 테넌트 수정 |
| DELETE | `/tenants/:id` | 테넌트 삭제 |
| GET | `/tenants/all` | 전체 테넌트 (관리자) |
| GET | `/tenants/search` | 테넌트 검색 |
| GET | `/tenants/kv/:key` | KV 값 조회 |
| PUT | `/tenants/kv/:key` | KV 값 수정 |

#### 지식베이스 (`/api/v1/knowledge-bases/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/knowledge-bases` | 지식베이스 생성 |
| GET | `/knowledge-bases` | 목록 조회 |
| GET | `/knowledge-bases/:id` | 상세 조회 |
| PUT | `/knowledge-bases/:id` | 수정 |
| DELETE | `/knowledge-bases/:id` | 삭제 |
| GET | `/knowledge-bases/:id/hybrid-search` | 하이브리드 검색 |
| POST | `/knowledge-bases/copy` | 복사 |
| GET | `/knowledge-bases/copy/progress/:task_id` | 복사 진행도 |
| GET | `/knowledge-bases/:id/tags` | 태그 목록 |
| POST | `/knowledge-bases/:id/tags` | 태그 생성 |
| PUT | `/knowledge-bases/:id/tags/:tag_id` | 태그 수정 |
| DELETE | `/knowledge-bases/:id/tags/:tag_id` | 태그 삭제 |

#### 지식 (`/api/v1/knowledge/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/knowledge-bases/:id/knowledge/file` | 파일로 지식 생성 |
| POST | `/knowledge-bases/:id/knowledge/url` | URL로 지식 생성 |
| POST | `/knowledge-bases/:id/knowledge/manual` | 수동 입력 |
| GET | `/knowledge-bases/:id/knowledge` | 지식 목록 |
| GET | `/knowledge/batch` | 일괄 조회 |
| GET | `/knowledge/:id` | 상세 조회 |
| DELETE | `/knowledge/:id` | 삭제 |
| PUT | `/knowledge/:id` | 수정 |
| PUT | `/knowledge/manual/:id` | 마크다운 수정 |
| GET | `/knowledge/:id/download` | 파일 다운로드 |
| PUT | `/knowledge/image/:id/:chunk_id` | 이미지 수정 |
| PUT | `/knowledge/tags` | 태그 일괄 수정 |

#### FAQ (`/api/v1/knowledge-bases/:id/faq/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/entries` | FAQ 목록 |
| GET | `/entries/export` | FAQ 내보내기 |
| POST | `/entries` | FAQ 일괄 추가 |
| POST | `/entry` | FAQ 단일 생성 |
| PUT | `/entries/:entry_id` | FAQ 수정 |
| PUT | `/entries/fields` | 필드 일괄 수정 |
| PUT | `/entries/tags` | 태그 일괄 수정 |
| DELETE | `/entries` | FAQ 삭제 |
| POST | `/search` | FAQ 검색 |

#### 청크 (`/api/v1/chunks/`)

| 메서드 | 경로 | 설명 |
|--------|------|------|
| GET | `/chunks/:knowledge_id` | 청크 목록 |
| GET | `/chunks/by-id/:id` | ID로 조회 |
| DELETE | `/chunks/:knowledge_id/:id` | 청크 삭제 |
| DELETE | `/chunks/:knowledge_id` | 전체 청크 삭제 |
| PUT | `/chunks/:knowledge_id/:id` | 청크 수정 |
| DELETE | `/chunks/by-id/:id/questions` | 생성된 질문 삭제 |

#### 채팅 / 세션

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/sessions` | 세션 생성 |
| GET | `/sessions/:id` | 세션 조회 |
| GET | `/sessions` | 세션 목록 |
| PUT | `/sessions/:id` | 세션 수정 |
| DELETE | `/sessions/:id` | 세션 삭제 |
| POST | `/sessions/:id/generate_title` | 제목 자동 생성 |
| POST | `/sessions/:id/stop` | 세션 중지 |
| GET | `/sessions/continue-stream/:id` | 스트림 계속 |
| POST | `/knowledge-chat/:session_id` | 지식 기반 QA (SSE) |
| POST | `/agent-chat/:session_id` | 에이전트 QA (SSE) |
| POST | `/knowledge-search` | 검색만 (세션 없음) |
| GET | `/messages/:session_id/load` | 메시지 로드 |
| DELETE | `/messages/:session_id/:id` | 메시지 삭제 |

#### 모델 / 시스템

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/models` | 모델 생성 |
| GET | `/models` | 모델 목록 |
| GET | `/models/:id` | 모델 상세 |
| PUT | `/models/:id` | 모델 수정 |
| DELETE | `/models/:id` | 모델 삭제 |
| GET | `/system/info` | 시스템 정보 |
| POST | `/evaluation` | 평가 실행 |
| GET | `/evaluation` | 평가 결과 |

#### MCP 서비스

| 메서드 | 경로 | 설명 |
|--------|------|------|
| POST | `/mcp-services` | MCP 서비스 생성 |
| GET | `/mcp-services` | 목록 |
| GET | `/mcp-services/:id` | 상세 |
| PUT | `/mcp-services/:id` | 수정 |
| DELETE | `/mcp-services/:id` | 삭제 |
| POST | `/mcp-services/:id/test` | 연결 테스트 |
| GET | `/mcp-services/:id/tools` | 도구 목록 |
| GET | `/mcp-services/:id/resources` | 리소스 목록 |
| GET | `/web-search/providers` | 웹 검색 제공자 목록 |

### 4.4 레이어 구조

```
HTTP 요청
    ↓
Middleware (JWT 검증, 로깅, 추적, 복구)
    ↓
Handler (요청 파싱, 검증, 응답 포맷팅)
    ↓
Service (비즈니스 로직, 트랜잭션 관리)
    ↓
Repository (데이터 접근, SQL/ORM)
    ↓
Database (PostgreSQL, Qdrant, Elasticsearch, Neo4j)
```

### 4.5 이벤트 시스템: `internal/event/`

비동기 통신을 위한 인메모리 이벤트 버스. 주로 SSE 스트리밍에 사용된다.

```go
// 이벤트 버스 핵심 구조
type EventBus struct {
    subscribers map[string][]EventHandler
    mu          sync.RWMutex
}

// 사용 패턴
bus := event.NewEventBus()
bus.Subscribe("agent:thought", handler)   // 구독
bus.Publish("agent:thought", data)        // 발행
```

**주요 이벤트 타입**:
- `agent:thought` — 에이전트 추론 단계
- `agent:tool_use` — 도구 호출
- `agent:tool_result` — 도구 결과
- `agent:final_answer` — 최종 답변
- `chat:chunk` — 채팅 청크 스트리밍
- `chat:done` — 채팅 완료

### 4.6 LLM 컨텍스트 관리: `internal/application/service/llmcontext/`

대화 이력의 토큰 초과를 방지하는 압축 시스템.

```
ContextManager
├── ContextStorage (저장소)
│   ├── MemoryStorage    # 인메모리 (단일 인스턴스)
│   └── RedisStorage     # Redis (분산 환경)
│
└── CompressionStrategy (압축 전략)
    ├── SlidingWindow    # 최근 N개 메시지만 유지
    └── SmartCompression # LLM으로 이전 대화 요약 (모델 필요)
```

**ContextManagerFactory** 설정 기반 자동 선택:
- 캐시 타입 = `redis` → RedisStorage
- 캐시 타입 = `memory` → MemoryStorage
- 압축 전략 = `smart` + 채팅 모델 있음 → SmartCompression
- 압축 전략 = 기타 → SlidingWindow (기본)

---

## 5. 채팅 파이프라인

`internal/application/service/chat_pipline/`에 구현된 이벤트 기반 파이프라인.

### 5.1 파이프라인 구조

```
ChatManage (공유 상태 객체)
├── Query: 사용자 질문
├── RewrittenQuery: 재작성된 질문
├── SearchResults: 검색 결과
├── Entities: 추출된 엔티티
├── Messages: LLM 메시지 목록
├── Context: 검색 결과 컨텍스트
└── EventBus: 이벤트 버스 (스트리밍용)
```

### 5.2 플러그인 실행 순서

```
1. PluginTracing
   └─ OpenTelemetry 스팬 생성 및 추적 시작

2. PluginRewrite  [선택적]
   └─ 대화 이력을 참조하여 질문 재작성 (대명사 해소)
      예: "그게 뭐야?" → "RAG 파이프라인이 뭐야?"

3. PluginExtractEntity  [선택적]
   └─ 질문에서 핵심 엔티티 추출

4. PluginSearch
   ├─ 검색 엔진에서 문서 검색
   │   ├─ BM25 키워드 검색 (Elasticsearch)
   │   ├─ 벡터 유사도 검색 (Qdrant / pgvector)
   │   └─ FAQ 검색 (FAQ 지식베이스인 경우)
   └─ SearchResults 채우기

5. PluginSearchEntity  [선택적]
   └─ 추출된 엔티티로 그래프 검색

6. PluginSearchParallel  [선택적]
   └─ 병렬로 여러 지식베이스 검색

7. PluginRerank  [선택적]
   ├─ 검색 결과를 Rerank 모델로 재정렬
   └─ 관련도 점수 재계산

8. PluginFilterTopK
   └─ 상위 K개 결과만 유지

9. PluginMerge
   ├─ 중복 청크 제거
   ├─ 검색 결과 병합
   └─ Context 문자열 생성

10. PluginIntoChatMessage
    └─ 검색 결과 + 질문 → LLM 메시지 포맷 변환

11. PluginChatCompletion / PluginChatCompletionStream
    ├─ LLM API 호출 (Ollama / OpenAI 호환)
    └─ 답변 생성 (동기 또는 스트리밍)

12. PluginStreamFilter
    └─ 스트리밍 결과 필터링 및 SSE 전송
```

### 5.3 검색 엔진 선택 (Retriever Registry)

`internal/application/service/retriever/`

```
RetrieverRegistry
├── keywords_vector_hybrid_indexer.go  # BM25 + 벡터 하이브리드
├── composite.go                       # 복합 검색 (여러 엔진 조합)
└── registry.go                        # 엔진 등록 및 팩토리

지원 엔진:
- elasticsearch  → BM25 전문 검색
- qdrant         → 벡터 유사도 검색 (Cosine)
- postgres       → pgvector 벡터 검색
- neo4j          → 그래프 관계 검색
- hybrid         → 위 조합 (점수 가중 합산)
```

---

## 6. ReACT 에이전트

`internal/agent/`에 구현된 ReACT (Reasoning + Acting) 패턴 에이전트.

### 6.1 에이전트 엔진 구조

```
AgentEngine
├── chatModel: Chat          # 추론용 LLM
├── tools: []Tool            # 사용 가능한 도구 목록
├── maxIterations: 30        # 최대 반복 횟수
├── eventBus: *event.EventBus # 스트리밍 이벤트 버스
└── Execute(ctx, query) error # 에이전트 실행
```

### 6.2 ReACT 루프

```
Execute(query)
    ↓
초기 메시지 구성 (System Prompt + Tools + Query)
    ↓
Loop (최대 30회):
    ├─ LLM 호출 → Thought + Action
    │   └─ EventBus.Publish("agent:thought", thought)
    │
    ├─ Tool 파싱 (함수 호출 포맷)
    │   └─ EventBus.Publish("agent:tool_use", toolName, args)
    │
    ├─ Tool 실행
    │   ├─ knowledge_search     → 지식베이스 검색
    │   ├─ grep_chunks          → 청크 키워드 검색
    │   ├─ list_knowledge_chunks → 청크 목록 조회
    │   ├─ get_document_info    → 문서 메타데이터 조회
    │   ├─ query_knowledge_graph → 그래프 쿼리 (Cypher)
    │   ├─ database_query       → SQL 직접 쿼리
    │   ├─ web_search           → 웹 검색 (DuckDuckGo)
    │   ├─ web_fetch            → 웹 페이지 가져오기
    │   ├─ thinking             → 순차적 사고 (내부 추론)
    │   ├─ todo_write           → 계획 작성
    │   └─ mcp_tool             → 외부 MCP 도구 호출
    │
    ├─ Tool 결과 메시지에 추가
    │   └─ EventBus.Publish("agent:tool_result", result)
    │
    └─ 종료 조건 확인:
        ├─ "Final Answer" → 답변 발행 후 종료
        ├─ 최대 반복 도달 → 마지막 상태 반환
        └─ 오류 → 에러 처리 후 종료
```

### 6.3 에이전트 도구 정의

```go
// 8개 UI 노출 도구
AvailableToolDefinitions() = [
    {Name: "thinking",              Label: "사고",           Description: "동적이고 반성적인 문제 해결 사고 도구"},
    {Name: "todo_write",            Label: "계획 수립",       Description: "구조화된 연구 계획 생성"},
    {Name: "grep_chunks",           Label: "키워드 검색",     Description: "특정 키워드가 포함된 문서 및 청크 신속 탐색"},
    {Name: "knowledge_search",      Label: "의미 검색",       Description: "질문을 이해하고 의미적으로 관련된 콘텐츠 탐색"},
    {Name: "list_knowledge_chunks", Label: "문서 청크 보기",  Description: "문서의 전체 청크 내용 조회"},
    {Name: "query_knowledge_graph", Label: "지식 그래프 조회", Description: "지식 그래프에서 관계 조회"},
    {Name: "get_document_info",     Label: "문서 정보 조회",  Description: "문서 메타데이터 확인"},
    {Name: "database_query",        Label: "데이터베이스 조회", Description: "데이터베이스에서 정보 조회"},
]
```

### 6.4 MCP 통합: `internal/mcp/`

```
MCPManager
├── GetOrCreateClient(service) → MCPClient
│   ├─ Stdio: 새 연결마다 새 프로세스/클라이언트
│   ├─ SSE: 연결 풀링 (재사용)
│   └─ HTTP Streamable: 연결 풀링 (재사용)
│
├── cleanupIdleConnections()   # 유휴 연결 정리
│
└── MCPTransportType:
    ├─ stdio   # 로컬 프로세스 (npx, uvx)
    ├─ sse     # SSE 원격 서버
    └─ http    # HTTP 스트리밍 원격 서버
```

---

## 7. 데이터베이스 스키마

### 7.1 주요 테이블

#### `tenants`
```sql
CREATE TABLE tenants (
    id              SERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    api_key         VARCHAR(64) NOT NULL UNIQUE,
    retriever_engines JSONB DEFAULT '[]',       -- 검색 엔진 설정
    status          VARCHAR(50) DEFAULT 'active',
    business        VARCHAR(255) NOT NULL,
    storage_quota   BIGINT DEFAULT 10737418240, -- 10GB
    storage_used    BIGINT DEFAULT 0,
    agent_config    JSONB DEFAULT NULL,         -- 에이전트 설정
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

#### `users`
```sql
CREATE TABLE users (
    id                    VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    username              VARCHAR(100) NOT NULL UNIQUE,
    email                 VARCHAR(255) NOT NULL UNIQUE,
    password_hash         VARCHAR(255) NOT NULL,
    avatar                VARCHAR(500),
    tenant_id             INTEGER REFERENCES tenants(id),
    is_active             BOOLEAN DEFAULT true,
    can_access_all_tenants BOOLEAN DEFAULT false,
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

#### `models`
```sql
CREATE TABLE models (
    id          VARCHAR(64) PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id   INTEGER NOT NULL,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(50) NOT NULL,   -- llm, embedding, rerank, vlm
    source      VARCHAR(50) NOT NULL,  -- openai, ollama, custom
    description TEXT,
    parameters  JSONB NOT NULL,        -- API URL, 키, 파라미터
    is_default  BOOLEAN DEFAULT false,
    status      VARCHAR(50) DEFAULT 'active',
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

#### `knowledge_bases`
```sql
CREATE TABLE knowledge_bases (
    id                      VARCHAR(36) PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                    VARCHAR(255) NOT NULL,
    tenant_id               INTEGER NOT NULL,
    type                    VARCHAR(50),   -- document, faq
    description             TEXT,
    chunking_config         JSONB,         -- 청킹 설정
    image_processing_config JSONB,         -- 이미지 처리 설정
    embedding_model_id      VARCHAR(64),
    summary_model_id        VARCHAR(64),
    rerank_model_id         VARCHAR(64),
    retrieve_config         JSONB,         -- 검색 설정
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

#### `knowledges`
```sql
CREATE TABLE knowledges (
    id               VARCHAR(36) PRIMARY KEY,
    tenant_id        INTEGER NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL REFERENCES knowledge_bases(id),
    type             VARCHAR(50) NOT NULL, -- file, url, manual
    title            VARCHAR(255) NOT NULL,
    file_type        VARCHAR(50),
    file_size        BIGINT,
    file_path        VARCHAR(512),
    url              VARCHAR(512),
    status           VARCHAR(50) DEFAULT 'pending',
    summary_status   VARCHAR(50),          -- 요약 상태 (v0.2.2)
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

#### `chunks`
```sql
CREATE TABLE chunks (
    id           VARCHAR(36) PRIMARY KEY,
    knowledge_id VARCHAR(36) NOT NULL REFERENCES knowledges(id),
    tenant_id    INTEGER NOT NULL,
    content      TEXT NOT NULL,
    embedding    vector(1536),  -- pgvector 임베딩
    seq_number   INTEGER,
    start_pos    INTEGER,
    end_pos      INTEGER,
    images       JSONB,          -- 이미지 메타데이터
    is_generated BOOLEAN DEFAULT false,  -- 생성된 청크 여부
    is_enabled   BOOLEAN DEFAULT true,
    created_at, updated_at TIMESTAMPTZ
);

-- 벡터 검색 인덱스
CREATE INDEX idx_chunks_embedding ON chunks USING ivfflat (embedding vector_cosine_ops);
```

#### `sessions` / `messages`
```sql
CREATE TABLE sessions (
    id               VARCHAR(36) PRIMARY KEY,
    tenant_id        INTEGER NOT NULL,
    knowledge_base_id VARCHAR(36),
    title            VARCHAR(255),
    status           VARCHAR(50) DEFAULT 'active',
    summary_model_id VARCHAR(64),
    created_at, updated_at, deleted_at TIMESTAMPTZ
);

CREATE TABLE messages (
    id         VARCHAR(36) PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL REFERENCES sessions(id),
    role       VARCHAR(50),  -- user, assistant, system
    content    TEXT,
    metadata   JSONB,        -- 검색 결과, 도구 결과 등
    created_at, updated_at TIMESTAMPTZ
);
```

#### `mcp_services`
```sql
CREATE TABLE mcp_services (
    id             VARCHAR(36) PRIMARY KEY,
    tenant_id      INTEGER NOT NULL,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    transport_type VARCHAR(50),  -- stdio, sse, http
    config         JSONB,        -- 서비스 설정
    enabled        BOOLEAN DEFAULT false,
    created_at, updated_at, deleted_at TIMESTAMPTZ
);
```

### 7.2 테이블 관계

```
tenants (1)
├── (N) users
├── (N) models
├── (N) knowledge_bases (1)
│            └── (N) knowledges (1)
│                         └── (N) chunks
│                                   └── vector embedding (pgvector)
├── (N) sessions (1)
│            └── (N) messages
└── (N) mcp_services

users (1) ── (N) auth_tokens
```

---

## 8. 프론트엔드 아키텍처 (Vue 3)

### 8.1 디렉토리 구조

```
frontend/src/
├── api/                    # API 클라이언트 레이어
│   ├── auth/               # 인증 API
│   ├── knowledge-base/     # 지식베이스 API
│   ├── chat/               # 채팅 API (SSE 포함)
│   │   └── stream.ts       # SSE 스트림 헬퍼
│   ├── model/              # 모델 관리 API
│   ├── initialization/     # 초기화 API (Ollama 상태 등)
│   ├── tenant/             # 테넌트 API
│   ├── system/             # 시스템 정보 API
│   ├── mcp-service.ts      # MCP 서비스 API
│   └── web-search.ts       # 웹 검색 API
│
├── stores/                 # Pinia 상태 관리
│   ├── auth.ts             # 사용자, 테넌트, 토큰, 지식베이스 상태
│   ├── knowledge.ts        # 지식 목록 상태
│   ├── menu.ts             # 사이드바 메뉴 상태, i18n locale 감지
│   ├── settings.ts         # 시스템 설정, 모델 초기화 상태
│   └── ui.ts               # UI 테마, 다크 모드
│
├── views/                  # 페이지 컴포넌트
│   ├── auth/
│   │   └── Login.vue       # 로그인/회원가입 (애니메이션 배경, 언어 선택)
│   ├── chat/
│   │   ├── index.vue       # 채팅 메인 화면
│   │   └── components/
│   │       ├── sendMsg.vue              # 메시지 입력
│   │       ├── botmsg.vue               # 봇 메시지
│   │       ├── AgentStreamDisplay.vue   # 에이전트 스트리밍 표시
│   │       ├── ToolResultRenderer.vue   # 도구 결과 렌더러
│   │       └── tool-results/            # 도구별 결과 컴포넌트
│   │           ├── SearchResults.vue
│   │           ├── WebSearchResults.vue
│   │           ├── GraphQueryResults.vue
│   │           ├── DatabaseQuery.vue
│   │           ├── ThinkingDisplay.vue
│   │           ├── PlanDisplay.vue
│   │           └── ...
│   ├── knowledge/
│   │   ├── KnowledgeBaseList.vue       # 지식베이스 목록
│   │   ├── KnowledgeBase.vue           # 지식 목록 (문서/FAQ)
│   │   ├── KnowledgeBaseEditorModal.vue # 지식베이스 생성/편집
│   │   ├── components/
│   │   │   └── FAQEntryManager.vue     # FAQ 관리
│   │   └── settings/
│   │       ├── KBModelConfig.vue       # 모델 설정
│   │       ├── KBChunkingSettings.vue  # 청킹 설정
│   │       ├── KBAdvancedSettings.vue  # 고급 설정
│   │       └── GraphSettings.vue       # 그래프 설정
│   ├── settings/
│   │   ├── Settings.vue               # 설정 메인
│   │   ├── ModelSettings.vue          # 모델 관리
│   │   ├── AgentSettings.vue          # 에이전트 설정 (가장 큰 파일)
│   │   ├── McpSettings.vue            # MCP 서비스 관리
│   │   ├── WebSearchSettings.vue      # 웹 검색 설정
│   │   ├── OllamaSettings.vue         # Ollama 관리
│   │   ├── GeneralSettings.vue        # 일반 설정 (언어 선택)
│   │   ├── TenantInfo.vue             # 테넌트 정보
│   │   ├── ApiInfo.vue                # API 키 정보
│   │   └── SystemInfo.vue             # 시스템 정보
│   ├── platform/
│   │   └── index.vue                  # 플랫폼 소개 페이지
│   └── creatChat/
│       └── creatChat.vue              # 대화 생성
│
├── components/             # 재사용 컴포넌트
│   ├── menu.vue                    # 사이드바 메뉴 (KB 선택, 네비게이션)
│   ├── UserMenu.vue                # 사용자 메뉴
│   ├── TenantSelector.vue          # 테넌트 선택기
│   ├── KnowledgeBaseSelector.vue   # 지식베이스 선택기
│   ├── ModelSelector.vue           # 모델 선택기
│   ├── ModelEditorDialog.vue       # 모델 편집 다이얼로그
│   ├── manual-knowledge-editor.vue # 마크다운 에디터
│   ├── doc-content.vue             # 문서 콘텐츠 표시
│   ├── Input-field.vue             # 다중 입력 필드
│   ├── upload-mask.vue             # 파일 업로드 마스크
│   └── empty-knowledge.vue         # 빈 상태 표시
│
├── router/index.ts         # Vue Router 라우팅
├── i18n/                   # 다국어 지원 (ko-KR 기본, en-US)
├── hooks/                  # Vue Composables
├── utils/                  # 유틸리티
├── types/                  # TypeScript 타입
└── main.ts                 # 앱 진입점
```

### 8.2 Pinia 스토어: `auth.ts`

```typescript
// 가장 핵심적인 스토어
state = {
    user: UserInfo | null,
    tenant: TenantInfo | null,
    token: string,
    refreshToken: string,
    knowledgeBases: KnowledgeBaseInfo[],
    currentKnowledgeBase: KnowledgeBaseInfo | null,
    selectedTenantId: number | null,  // 크로스 테넌트 선택
}

getters = {
    isLoggedIn: boolean,
    canAccessAllTenants: boolean,
    effectiveTenantId: number | null,   // selectedTenantId || user.tenantId
    currentTenantId: string,
}
```

### 8.3 SSE 스트리밍 클라이언트

```typescript
// chat/stream.ts
class StreamHandler {
    private eventSource: EventSource

    connect(url: string, options: RequestInit): void
    on(event: string, callback: (data: any) => void): void
    disconnect(): void
}

// 구독하는 이벤트 타입
"chat:chunk"          → 채팅 응답 청크
"agent:thought"       → 에이전트 추론
"agent:tool_use"      → 도구 호출
"agent:tool_result"   → 도구 결과
"agent:final_answer"  → 최종 답변
"error"               → 오류
"done"                → 완료
```

### 8.4 라우팅

```
/                   → 플랫폼 소개 (platform/index.vue)
/auth/login         → 로그인 (Login.vue)
/knowledge          → 지식베이스 목록 (KnowledgeBaseList.vue)
/knowledge/:id      → 지식 목록 (KnowledgeBase.vue)
/chat               → 채팅 (chat/index.vue)
/chat/:sessionId    → 특정 세션 채팅
/settings           → 설정 (Settings.vue)
/settings/model     → 모델 설정
/settings/agent     → 에이전트 설정
/settings/mcp       → MCP 설정
/settings/general   → 일반 설정
```

---

## 9. 문서 파서 서비스 (docreader)

### 9.1 gRPC 프로토콜

```protobuf
service DocReader {
    rpc ReadFromFile(ReadFromFileRequest) returns (ReadResponse);
    rpc ReadFromURL(ReadFromURLRequest) returns (ReadResponse);
}

message ReadFromFileRequest {
    bytes file_content = 1;
    string file_name   = 2;
    string file_type   = 3;
    ReadConfig config  = 4;
    string request_id  = 5;
}

message ReadFromURLRequest {
    string url        = 1;
    string title      = 2;
    ReadConfig config = 3;
    string request_id = 4;
}

message ReadResponse {
    repeated ParsedChunk chunks = 1;
    string error                = 2;
}

message ParsedChunk {
    string content       = 1;
    int32 seq_number     = 2;
    int32 start_pos      = 3;
    int32 end_pos        = 4;
    repeated Image images = 5;
}

message Image {
    string url      = 1;
    string caption  = 2;
    string ocr_text = 3;
}
```

### 9.2 지원 파서 (18종)

| 파서 | 지원 포맷 |
|------|---------|
| pdf_parser.py | PDF |
| docx_parser.py / docx2_parser.py | DOCX, DOC |
| excel_parser.py | XLSX, XLS |
| csv_parser.py | CSV |
| markdown_parser.py | MD |
| text_parser.py | TXT |
| image_parser.py | PNG, JPG, GIF 등 |
| web_parser.py | HTML (URL) |
| mineru_parser.py | PDF (MinerU 고급 처리) |
| markitdown_parser.py | 다양한 포맷 → Markdown 변환 |
| chain_parser.py | 여러 파서 조합 |

### 9.3 특수 기능

- **OCR**: 이미지/PDF 텍스트 인식 (`ocr_engine.py`)
- **이미지 캡션**: VLM 모델로 이미지 설명 생성 (`caption.py`)
- **청크 분할**: 문단, 섹션, 테이블 단위 분할 (`splitter/`)
- **스토리지 연동**: MinIO/COS 파일 업로드 (`storage.py`)

---

## 10. 인프라 및 배포

### 10.1 Docker Compose 서비스

```yaml
# 필수 서비스
services:
  app:        # Go 백엔드 :8080
  frontend:   # Vue 프론트엔드 :80
  docreader:  # Python gRPC 파서 :50051
  postgres:   # ParadeDB (PostgreSQL + pgvector) :5432
  redis:      # Redis 7.0 :6379

# 선택적 서비스 (프로파일)
  minio:      # 오브젝트 스토리지 :9000, :9001  (profile: minio)
  jaeger:     # 분산 추적 :16686, :4317          (profile: jaeger)
  neo4j:      # 그래프 DB :7474, :7687           (profile: neo4j)
  qdrant:     # 벡터 DB :6333, :6334             (profile: qdrant)

# 전체 프로파일: --profile full
```

### 10.2 개발 환경 (`docker-compose.dev.yml`)

```bash
make dev-start      # 인프라만 시작 (PostgreSQL, Redis)
make dev-app        # 백엔드 핫 리로드 (air 사용) :8080
make dev-frontend   # 프론트엔드 핫 리로드 (Vite) :5173
make dev-stop       # 개발 환경 중지
```

### 10.3 서비스 URL (개발 환경)

| 서비스 | URL |
|--------|-----|
| 웹 UI (개발) | http://localhost:5173 |
| 웹 UI (프로덕션) | http://localhost |
| 백엔드 API | http://localhost:8080 |
| Swagger 문서 | http://localhost:8080/swagger/index.html |
| Jaeger 추적 | http://localhost:16686 |
| Neo4j 브라우저 | http://localhost:7474 |
| MinIO 콘솔 | http://localhost:9001 |
| Qdrant 대시보드 | http://localhost:6333/dashboard |

### 10.4 주요 환경 변수

```bash
# 필수
DB_DRIVER=postgres
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=weknora

REDIS_ADDR=redis:6379
JWT_SECRET=your-secret-key

DOCREADER_ADDR=docreader:50051

# 검색 엔진
RETRIEVE_DRIVER=elasticsearch  # elasticsearch | qdrant | postgres

# Elasticsearch
ES_ADDR=http://elasticsearch:9200

# Qdrant
QDRANT_HOST=qdrant
QDRANT_PORT=6334

# 파일 저장소
STORAGE_TYPE=local             # local | minio | cos
LOCAL_STORAGE_BASE_DIR=/data/files

# MinIO (선택)
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY_ID=minioadmin
MINIO_SECRET_ACCESS_KEY=minioadmin
MINIO_USE_SSL=false

# Ollama (선택)
OLLAMA_BASE_URL=http://host.docker.internal:11434

# Neo4j (선택)
NEO4J_ENABLE=false
NEO4J_URI=bolt://neo4j:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password

# 분산 추적 (선택)
OTEL_EXPORTER_OTLP_ENDPOINT=jaeger:4317
OTEL_SERVICE_NAME=WeKnora
```

---

## 11. 핵심 기능 흐름

### 11.1 문서 업로드 및 처리

```
사용자 파일 업로드
    ↓
[KnowledgeHandler.CreateKnowledgeFromFile]
    ├─ 파일 타입/크기 검증
    ├─ 파일 스토리지 저장 (로컬/MinIO/COS)
    └─ Knowledge 레코드 생성 (status: pending)
    ↓
[Asynq 큐] 비동기 처리 작업 등록
    ↓
[비동기 워커]
    ├─ docreader gRPC 호출 (ReadFromFile)
    │   ├─ 파일 파싱 (파서 선택)
    │   ├─ 텍스트 추출
    │   ├─ OCR (이미지인 경우)
    │   └─ Chunks 반환
    │
    ├─ 각 Chunk 처리:
    │   ├─ 임베딩 생성 (EmbeddingModel API 호출)
    │   ├─ 벡터 저장 (Qdrant 또는 pgvector)
    │   ├─ Chunk 레코드 저장 (PostgreSQL)
    │   └─ 검색 인덱스 생성 (Elasticsearch)
    │
    └─ Knowledge 상태 업데이트 (status: completed)
    ↓
완료 (사용자에게 폴링 또는 WebSocket 알림)
```

### 11.2 하이브리드 검색

```
사용자 질문 입력
    ↓
[채팅 파이프라인] PluginSearch
    ↓
병렬 검색 실행:
    ├─ BM25 검색 (Elasticsearch)
    │   └─ 키워드 매칭 점수
    │
    ├─ 벡터 검색 (Qdrant / pgvector)
    │   ├─ 질문 임베딩 생성
    │   └─ 코사인 유사도 계산
    │
    └─ 그래프 검색 (Neo4j, 선택적)
        └─ 엔티티 관계 탐색
    ↓
점수 결합 (가중치 합산)
    ↓
[PluginRerank]
    └─ Rerank 모델로 최종 순위 결정
    ↓
[PluginFilterTopK]
    └─ 상위 K개 선택 (기본: 5개)
    ↓
LLM에 컨텍스트로 전달
```

### 11.3 사용자 인증 흐름

```
POST /auth/login {email, password}
    ↓
[AuthHandler.Login]
    ├─ UserService.Authenticate(email, password)
    │   ├─ 사용자 조회
    │   └─ bcrypt 비밀번호 검증
    ↓
JWT 토큰 생성
    ├─ access_token  (24시간)
    └─ refresh_token (7일)
    ↓
응답: {access_token, refresh_token, user, tenant}
    ↓
이후 요청: Authorization: Bearer {access_token}
    ↓
[Auth Middleware]
    ├─ 토큰 검증 (서명, 만료 여부)
    ├─ 사용자 정보 추출
    ├─ TenantID → gin.Context 저장
    └─ 요청 핸들러로 전달
```

---

## 12. 보안 구조

### 12.1 인증 및 권한

| 항목 | 구현 |
|------|------|
| 인증 방식 | JWT (Bearer Token) |
| 비밀번호 해싱 | bcrypt |
| 토큰 갱신 | Refresh Token (7일) |
| API 키 인증 | 외부 클라이언트용 API Key |
| 테넌트 격리 | 모든 쿼리에 tenant_id 필터 |
| 크로스 테넌트 | `can_access_all_tenants` 플래그 |

### 12.2 인증 화이트리스트 (미들웨어 제외)

```
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
GET  /health
GET  /swagger/*
```

### 12.3 프론트엔드 보안

- Vue 자동 XSS 이스케이프 (템플릿 바인딩)
- DOMPurify로 마크다운 렌더링 시 살균
- Axios 인터셉터로 토큰 자동 갱신

---

## 13. 설정 관리

### 13.1 설정 계층 (우선순위 낮음 → 높음)

```
기본값 (코드 내 하드코딩)
    ↓
config/config.yaml
    ↓
환경 변수
    ↓
최종 설정 (Viper 자동 병합)
```

### 13.2 주요 설정 섹션

```yaml
server:
  port: 8080
  shutdown_timeout: 30s

conversation:
  max_rounds: 5               # 대화 최대 라운드
  embedding_top_k: 10         # 벡터 검색 결과 수
  vector_threshold: 0.5       # 벡터 유사도 임계값
  rerank_threshold: 0.5       # 리랭크 임계값
  rerank_top_k: 5             # 최종 결과 수
  enable_rewrite: true        # 질문 재작성 활성화
  enable_rerank: true         # 리랭크 활성화
  fallback_strategy: "fixed"  # 폴백 전략

database:
  driver: postgres
  max_open_connections: 25
  max_idle_connections: 5

cache:
  type: redis                 # memory | redis

search:
  engine: elasticsearch       # elasticsearch | qdrant | postgres
  top_k: 10

storage:
  type: local                 # local | minio | cos

auth:
  token_expiry: 24h
  refresh_token_expiry: 168h

tracing:
  enabled: true
  exporter: otlp
```

### 13.3 평가 메트릭

`internal/application/service/metric/`에 구현:

| 메트릭 | 설명 |
|--------|------|
| BLEU | 번역 품질 평가 지표 |
| ROUGE | 요약 품질 평가 (ROUGE-1, ROUGE-2, ROUGE-L) |
| MRR | 평균 역순위 (검색 정확도) |
| NDCG | 정규화 할인 누적 이득 (순위 품질) |
| MAP | 평균 정밀도 (검색 전반적 품질) |
| Precision / Recall | 정밀도 / 재현율 |

---

*이 문서는 WeKnora v0.2.2 기준으로 작성되었습니다.*
