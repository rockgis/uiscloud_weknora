# uiscloud_weknora 아키텍처 상세 분석

이 문서는 uiscloud_weknora 프로젝트의 전체 아키텍처를 상세하게 분석한 기술 문서입니다.

## 목차

1. [개요](#1-개요)
2. [백엔드 아키텍처 (Go)](#2-백엔드-아키텍처-go)
3. [핵심 기능 구현](#3-핵심-기능-구현)
4. [프론트엔드 아키텍처 (Vue 3)](#4-프론트엔드-아키텍처-vue-3)
5. [문서 파서 서비스 (docreader)](#5-문서-파서-서비스-docreader)
6. [데이터 흐름](#6-데이터-흐름)
7. [외부 통합](#7-외부-통합)
8. [설정 및 배포](#8-설정-및-배포)
9. [주요 아키텍처 패턴](#9-주요-아키텍처-패턴)

---

## 1. 개요

### 1.1 시스템 아키텍처 다이어그램

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              클라이언트 계층                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    Vue 3 + TypeScript + TDesign UI                   │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │   │
│  │  │  Views  │  │ Stores  │  │   API   │  │  Router │  │  i18n   │   │   │
│  │  │ (Pinia) │  │ (State) │  │ (Axios) │  │(VueRtr) │  │ (다국어)│   │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │ HTTP/SSE
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Gateway 계층                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Gin HTTP Server                              │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐            │   │
│  │  │  CORS  │→│RequestID│→│ Logger │→│Recovery│→│  Auth  │→ Routes   │   │
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              비즈니스 로직 계층                               │
│  ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐       │
│  │     Handlers      │  │     Services      │  │   Repositories    │       │
│  │  ┌─────────────┐  │  │  ┌─────────────┐  │  │  ┌─────────────┐  │       │
│  │  │   Session   │  │  │  │   Agent     │  │  │  │   Tenant    │  │       │
│  │  │  Knowledge  │  │  │  │   Chat      │  │  │  │  Knowledge  │  │       │
│  │  │    Model    │  │  │  │  Pipeline   │  │  │  │   Chunk     │  │       │
│  │  │    Auth     │  │  │  │  Retriever  │  │  │  │   Session   │  │       │
│  │  └─────────────┘  │  │  └─────────────┘  │  │  └─────────────┘  │       │
│  └───────────────────┘  └───────────────────┘  └───────────────────┘       │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌─────────────────┐      ┌─────────────────┐      ┌─────────────────┐
│    PostgreSQL   │      │      Redis      │      │   Vector DB     │
│   + pgvector    │      │   (Cache/MQ)    │      │ ES/Qdrant/Neo4j │
└─────────────────┘      └─────────────────┘      └─────────────────┘
          │
          │ gRPC
          ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           문서 처리 계층 (Python)                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         docreader Service                            │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │   │
│  │  │   PDF   │  │  Word   │  │  Image  │  │Markdown │  │   Web   │   │   │
│  │  │ Parser  │  │ Parser  │  │ (OCR)   │  │ Parser  │  │ Parser  │   │   │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 기술 스택 요약

| 계층 | 기술 | 용도 |
|------|------|------|
| 프론트엔드 | Vue 3, TypeScript, Pinia, TDesign | 사용자 인터페이스 |
| API Gateway | Gin Framework | HTTP 라우팅, 미들웨어 |
| 비즈니스 로직 | Go 1.24+ | 핵심 서비스 로직 |
| 데이터 저장 | PostgreSQL + pgvector | 관계형 데이터 + 벡터 검색 |
| 캐시/큐 | Redis | 세션 캐시, 스트림 관리 |
| 벡터 DB | Elasticsearch, Qdrant, Neo4j | 고급 벡터 검색 |
| 문서 처리 | Python + gRPC | 문서 파싱, OCR |
| 컨테이너 | Docker, Docker Compose | 배포, 오케스트레이션 |

---

## 2. 백엔드 아키텍처 (Go)

### 2.1 진입점 및 초기화 흐름

**파일 위치:** `cmd/server/main.go`

```
main()
    │
    ├── 로그 설정 (RequestID, 마이크로초 정밀도)
    │
    ├── Gin 모드 설정 (debug/release)
    │
    ├── container.BuildContainer()
    │   │
    │   ├── ResourceCleaner 등록
    │   ├── 인프라 (Config, Tracer, DB, Redis)
    │   ├── 파일 서비스 (Local/MinIO/COS)
    │   ├── Retriever Registry (Postgres/ES/Qdrant/Neo4j)
    │   ├── 외부 서비스 (DocReader, Ollama)
    │   ├── Repository 계층
    │   ├── Service 계층
    │   ├── EventBus
    │   ├── Agent System
    │   ├── Chat Pipeline (플러그인 기반)
    │   └── Handler + Router
    │
    ├── HTTP 서버 시작 (host:port)
    │
    └── Graceful Shutdown (30초 타임아웃)
        └── ResourceCleaner.Cleanup()
```

### 2.2 의존성 주입 컨테이너

**파일 위치:** `internal/container/container.go`

uber/dig를 사용한 DI 컨테이너 패턴:

```go
func BuildContainer() *dig.Container {
    container := dig.New()

    // 1. 리소스 관리
    container.Provide(NewResourceCleaner)

    // 2. 인프라 계층
    container.Provide(config.Load)
    container.Provide(tracing.NewTracer)
    container.Provide(database.NewDB)
    container.Provide(redis.NewClient)

    // 3. 저장소 계층
    container.Provide(repository.NewTenantRepository)
    container.Provide(repository.NewKnowledgeRepository)
    // ... 기타 Repository

    // 4. 서비스 계층
    container.Provide(service.NewTenantService)
    container.Provide(service.NewKnowledgeService)
    // ... 기타 Service

    // 5. Handler 계층
    container.Provide(handler.NewSessionHandler)
    // ... 기타 Handler

    // 6. Router
    container.Provide(router.NewRouter)

    return container
}
```

### 2.3 HTTP 라우팅 및 미들웨어 체인

**파일 위치:** `internal/router/router.go`

#### 미들웨어 처리 순서

```
요청 → CORS → RequestID → Logger → Recovery → ErrorHandler → Auth → Tracing → Handler
```

| 미들웨어 | 위치 | 역할 |
|----------|------|------|
| CORS | `middleware/cors.go` | 모든 출처 허용, 크로스 도메인 |
| RequestID | `middleware/request_id.go` | 요청 추적 ID 생성 |
| Logger | `middleware/logger.go` | 구조화된 로깅 |
| Recovery | `middleware/recovery.go` | 패닉 복구 |
| ErrorHandler | `middleware/error.go` | 에러 응답 변환 |
| Auth | `middleware/auth.go` | JWT 검증, 테넌트 컨텍스트 |
| Tracing | `middleware/tracing.go` | OpenTelemetry 분산 추적 |

#### API 라우트 구조

```
/health                           # 헬스 체크 (인증 불필요)
/swagger/*                        # API 문서 (인증 불필요)
/api/v1
├── /auth
│   ├── POST /register            # 회원가입
│   ├── POST /login               # 로그인
│   ├── POST /refresh             # 토큰 갱신
│   └── POST /logout              # 로그아웃
├── /tenants/*                    # 테넌트 관리
├── /knowledge-bases/*            # 지식 베이스 CRUD
├── /knowledge/*                  # 지식 문서 관리
├── /chunks/*                     # 청크 관리
├── /sessions/*                   # 세션 관리
├── /knowledge-chat/:id           # 지식 기반 QA
├── /agent-chat/:id               # 에이전트 QA
├── /models/*                     # 모델 관리
├── /mcp-services/*               # MCP 서비스
└── /web-search/*                 # 웹 검색
```

### 2.4 Handler 계층 구조

**파일 위치:** `internal/handler/`

| 파일 | 역할 |
|------|------|
| `auth.go` | 인증 (회원가입, 로그인, 토큰 갱신) |
| `tenant.go` | 테넌트 관리 |
| `knowledgebase.go` | 지식 베이스 CRUD, 복사, 검색 |
| `knowledge.go` | 문서 업로드, URL 가져오기, 수동 입력 |
| `chunk.go` | 청크 목록, 삭제, 업데이트 |
| `faq.go` | FAQ 관리 |
| `tag.go` | 태그 관리 |
| `session/handler.go` | 세션 생성/관리 |
| `session/qa.go` | 지식 QA + 에이전트 QA (SSE 스트리밍) |
| `session/stream.go` | 스트리밍 응답 처리 |
| `model.go` | 모델 설정 관리 |
| `mcp_service.go` | MCP 서비스 관리 |
| `web_search.go` | 웹 검색 프로바이더 |

### 2.5 Service 계층

**파일 위치:** `internal/application/service/`

```
service/
├── agent_service.go          # 에이전트 생성, 도구 레지스트리
├── knowledgebase.go          # 지식 베이스 비즈니스 로직
├── knowledge.go              # 지식 문서 처리
├── chunk.go                  # 청크 배치 처리
├── session.go                # 세션 라이프사이클
├── message.go                # 메시지 저장/조회
├── tenant.go                 # 테넌트 관리
├── user.go                   # 사용자 인증/권한
├── model.go                  # LLM 모델 관리
├── mcp_service.go            # MCP 통합
├── web_search/               # 웹 검색 프로바이더
├── retriever/                # 검색 엔진 레지스트리
├── chat_pipline/             # 채팅 파이프라인 플러그인
│   ├── chat_pipline.go       # 이벤트 기반 플러그인 아키텍처
│   ├── search.go             # 지식 검색 플러그인
│   ├── rerank.go             # 재순위화 플러그인
│   ├── merge.go              # 결과 병합 플러그인
│   ├── into_chat_message.go  # 컨텍스트 포맷팅
│   ├── chat_completion.go    # LLM 추론
│   └── chat_completion_stream.go  # 스트리밍 응답
├── file/                     # 파일 저장소 (Local/MinIO/COS)
└── llmcontext/               # LLM 컨텍스트 관리
```

### 2.6 Repository 계층

**파일 위치:** `internal/application/repository/`

```
repository/
├── tenant.go                 # 테넌트 CRUD
├── user.go                   # 사용자 인증
├── knowledgebase.go          # 지식 베이스 메타데이터
├── knowledge.go              # 지식 문서 메타데이터
├── chunk.go                  # 청크 저장 (대용량)
├── session.go                # 세션 영속화
├── message.go                # 메시지 저장
├── model.go                  # 모델 설정
├── tag.go                    # 태그 관리
├── mcp_service.go            # MCP 서비스 설정
└── retriever/                # 검색 엔진별 구현
    ├── postgres/             # PostgreSQL + pgvector
    ├── elasticsearch/
    │   ├── v7/               # Elasticsearch 7.x
    │   └── v8/               # Elasticsearch 8.x
    ├── qdrant/               # Qdrant 벡터 DB
    └── neo4j/                # Neo4j 그래프 DB
```

### 2.7 타입 시스템

**파일 위치:** `internal/types/`

#### 주요 타입 정의

```go
// types/agent.go
type AgentConfig struct {
    MaxIterations   int
    Tools           []string
    SystemPrompt    string
}

type AgentState struct {
    RoundSteps      []AgentStep
    KnowledgeRefs   []KnowledgeRef
    IsComplete      bool
    CurrentRound    int
}

// types/session.go
type Session struct {
    ID              string
    TenantID        string
    KnowledgeBaseID string
    Title           string
    Config          ConversationConfig
}

// types/message.go
type Message struct {
    ID        string
    SessionID string
    Role      string  // user, assistant, system
    Content   string
    Metadata  map[string]interface{}
}

// types/chunk.go
type Chunk struct {
    ID          string
    KnowledgeID string
    Content     string
    Embedding   []float32
    Metadata    ChunkMetadata
}
```

#### 인터페이스 정의

**파일 위치:** `internal/types/interfaces/`

```go
// interfaces/agent.go
type AgentEngine interface {
    Execute(ctx context.Context, query string, history []Message) (*AgentState, error)
}

type AgentService interface {
    CreateAgent(ctx context.Context, config AgentConfig) (AgentEngine, error)
}

// interfaces/retriever.go
type RetrieveEngine interface {
    Search(ctx context.Context, params RetrieveParams) ([]SearchResult, error)
    Index(ctx context.Context, chunks []Chunk) error
    Delete(ctx context.Context, chunkIDs []string) error
}

type RetrieveEngineRegistry interface {
    Get(engineType string) (RetrieveEngine, error)
    Register(engineType string, engine RetrieveEngine)
}

// interfaces/stream_manager.go
type StreamManager interface {
    CreateStream(sessionID, messageID string) error
    WriteEvent(sessionID, messageID string, event Event) error
    ReadEvents(sessionID, messageID string, offset int) ([]Event, error)
    Close(sessionID, messageID string) error
}
```

---

## 3. 핵심 기능 구현

### 3.1 에이전트 시스템 (ReACT 패턴)

**파일 위치:** `internal/agent/`

#### ReACT 실행 사이클

```
┌─────────────────────────────────────────────────────────────┐
│                    ReACT Agent 실행 흐름                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │  Think  │ →  │   Act   │ →  │ Observe │ →  │  Learn  │  │
│  │ (추론)  │    │ (실행)  │    │ (관찰)  │    │ (학습)  │  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
│       │              │              │              │        │
│       │              │              │              │        │
│       ▼              ▼              ▼              ▼        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                   AgentState                         │   │
│  │  - RoundSteps: 실행 단계 기록                        │   │
│  │  - KnowledgeRefs: 참조 지식                          │   │
│  │  - IsComplete: 완료 여부                             │   │
│  │  - CurrentRound: 현재 반복 횟수                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  최대 30회 반복 또는 완료 시 종료                           │
└─────────────────────────────────────────────────────────────┘
```

#### 에이전트 엔진 구현

**파일:** `internal/agent/engine.go`

```go
type AgentEngine struct {
    llmClient    LLMClient
    toolRegistry *ToolRegistry
    eventBus     *EventBus
    config       AgentConfig
}

func (e *AgentEngine) Execute(ctx context.Context, query string, history []Message) (*AgentState, error) {
    state := &AgentState{
        RoundSteps:   make([]AgentStep, 0),
        CurrentRound: 0,
    }

    for !state.IsComplete && state.CurrentRound < e.config.MaxIterations {
        // 1. Think: LLM에게 다음 행동 요청
        action, err := e.think(ctx, query, state, history)
        if err != nil {
            return nil, err
        }

        // 2. Act: 도구 실행
        result, err := e.act(ctx, action)
        if err != nil {
            return nil, err
        }

        // 3. Observe: 결과 관찰 및 상태 업데이트
        e.observe(state, action, result)

        // 4. 이벤트 발행 (스트리밍)
        e.eventBus.Publish(AgentStepEvent{
            Step:   state.CurrentRound,
            Action: action,
            Result: result,
        })

        state.CurrentRound++
    }

    return state, nil
}
```

#### 도구 레지스트리

**파일:** `internal/agent/tools/registry.go`

```go
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func (r *ToolRegistry) Register(name string, tool Tool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.tools[name] = tool
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
    r.mu.RLock()
    tool, ok := r.tools[name]
    r.mu.RUnlock()

    if !ok {
        return nil, fmt.Errorf("tool not found: %s", name)
    }

    return tool.Execute(ctx, params)
}
```

#### 내장 도구 목록

| 도구 | 파일 | 용도 |
|------|------|------|
| knowledge_search | `tools/knowledge_search.go` | 시맨틱 지식 검색 |
| grep_chunks | `tools/grep_chunks.go` | BM25 키워드 검색 |
| list_knowledge_chunks | `tools/list_knowledge_chunks.go` | 문서 청크 목록 |
| get_document_info | `tools/get_document_info.go` | 문서 메타데이터 |
| query_knowledge_graph | `tools/query_knowledge_graph.go` | Neo4j 그래프 쿼리 |
| database_query | `tools/database_query.go` | SQL 데이터베이스 쿼리 |
| sequential_thinking | `tools/sequentialthinking.go` | 확장 사고 도구 |
| todo_write | `tools/todo_write.go` | 작업 계획 |
| mcp_tool | `tools/mcp_tool.go` | MCP 프로토콜 도구 |
| web_search | `tools/web_search.go` | 웹 검색 |
| web_fetch | `tools/web_fetch.go` | 웹 콘텐츠 가져오기 |

### 3.2 RAG 파이프라인

#### 전체 흐름

```
사용자 쿼리
    │
    ▼
┌─────────────────┐
│   Query 전처리   │ ← 쿼리 정규화, 재작성
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Embedding 생성  │ ← Ollama 또는 외부 API
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│            하이브리드 검색               │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐ │
│  │  BM25   │  │ Vector  │  │  Graph  │ │
│  │(키워드) │  │(시맨틱) │  │(지식그래프)│ │
│  └────┬────┘  └────┬────┘  └────┬────┘ │
│       └────────────┼────────────┘      │
│                    ▼                    │
│              결과 병합                   │
└────────────────────┬────────────────────┘
                     │
                     ▼
┌─────────────────┐
│    재순위화      │ ← 선택적 (Reranker 모델)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  컨텍스트 구성   │ ← 청크 + 메타데이터
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   LLM 추론      │ ← Ollama 또는 외부 API
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  스트리밍 응답   │ ← SSE 이벤트
└─────────────────┘
```

#### Chat Pipeline 플러그인 아키텍처

**파일:** `internal/application/service/chat_pipline/`

```go
type ChatPipeline struct {
    plugins []Plugin
}

type Plugin interface {
    Name() string
    Process(ctx context.Context, event *ChatEvent) error
}

// 플러그인 실행 순서
plugins := []Plugin{
    NewSearchPlugin(retrieverService),      // 1. 지식 검색
    NewRerankPlugin(rerankService),         // 2. 재순위화
    NewMergePlugin(),                       // 3. 결과 병합
    NewIntoChatMessagePlugin(),             // 4. LLM 입력 포맷팅
    NewChatCompletionStreamPlugin(llm),     // 5. 스트리밍 응답 생성
}
```

### 3.3 검색 유틸리티

**파일 위치:** `internal/searchutil/`

| 파일 | 역할 |
|------|------|
| `conversion.go` | 검색 결과 타입 변환 |
| `normalize.go` | 쿼리 정규화 |
| `textutil.go` | 텍스트 처리 유틸리티 |

#### 검색 전략 비교

| 전략 | 장점 | 단점 | 사용 사례 |
|------|------|------|----------|
| BM25 | 정확한 키워드 매칭 | 시맨틱 이해 부족 | 전문 용어 검색 |
| Vector | 시맨틱 유사성 | 키워드 정확도 낮음 | 의미 기반 검색 |
| Hybrid | 두 장점 결합 | 복잡성 증가 | 일반적 사용 |
| Graph | 관계 기반 검색 | 그래프 구축 필요 | 엔티티 관계 검색 |

### 3.4 스트림/SSE 처리

**파일 위치:** `internal/stream/`

#### 스트림 매니저 구조

```go
// factory.go
func NewStreamManager(config *StreamConfig) StreamManager {
    switch config.Type {
    case "redis":
        return NewRedisStreamManager(config.Redis)
    default:
        return NewMemoryStreamManager()
    }
}

// redis_manager.go
type RedisStreamManager struct {
    client *redis.Client
    ttl    time.Duration
}

func (m *RedisStreamManager) WriteEvent(sessionID, messageID string, event Event) error {
    key := fmt.Sprintf("stream:%s:%s", sessionID, messageID)
    data, _ := json.Marshal(event)
    return m.client.RPush(ctx, key, data).Err()
}

func (m *RedisStreamManager) ReadEvents(sessionID, messageID string, offset int) ([]Event, error) {
    key := fmt.Sprintf("stream:%s:%s", sessionID, messageID)
    results, err := m.client.LRange(ctx, key, int64(offset), -1).Result()
    // ... 결과 파싱
}
```

#### 이벤트 타입

```go
const (
    // 쿼리 이벤트
    EventQueryReceived    = "query_received"
    EventQueryValidated   = "query_validated"
    EventQueryPreprocessed = "query_preprocessed"
    EventQueryRewritten   = "query_rewritten"

    // 검색 이벤트
    EventRetrievalStart    = "retrieval_start"
    EventRetrievalVector   = "retrieval_vector"
    EventRetrievalKeyword  = "retrieval_keyword"
    EventRetrievalComplete = "retrieval_complete"

    // 에이전트 이벤트
    EventAgentThought    = "agent_thought"
    EventAgentToolCall   = "agent_tool_call"
    EventAgentToolResult = "agent_tool_result"
    EventAgentReflection = "agent_reflection"
    EventAgentFinalAnswer = "agent_final_answer"

    // 제어 이벤트
    EventStop  = "stop"
    EventError = "error"
)
```

### 3.5 MCP 도구 통합

**파일 위치:** `internal/mcp/`

#### MCP 매니저

```go
// manager.go
type MCPManager struct {
    clients map[string]*MCPClient
    mu      sync.RWMutex
}

func (m *MCPManager) GetClient(serviceID string) (*MCPClient, error) {
    m.mu.RLock()
    client, ok := m.clients[serviceID]
    m.mu.RUnlock()

    if ok && client.IsConnected() {
        return client, nil
    }

    // 새 연결 생성
    return m.createClient(serviceID)
}

// client.go
type MCPClient struct {
    transport Transport  // Stdio, SSE, HTTP
    tools     []Tool
}

func (c *MCPClient) ListTools() ([]Tool, error) {
    return c.sendRequest("tools/list", nil)
}

func (c *MCPClient) CallTool(name string, params map[string]interface{}) (interface{}, error) {
    return c.sendRequest("tools/call", map[string]interface{}{
        "name":   name,
        "params": params,
    })
}
```

#### 지원 전송 방식

| 방식 | 설명 | 캐싱 |
|------|------|------|
| Stdio | 표준 입출력 | 매번 새 프로세스 |
| SSE | Server-Sent Events | 연결 재사용 |
| HTTP | HTTP Streamable | 연결 재사용 |

### 3.6 이벤트 시스템

**파일 위치:** `internal/event/`

```go
// event_bus.go
type EventBus struct {
    handlers map[string][]EventHandler
    mu       sync.RWMutex
    async    bool
}

func (b *EventBus) Subscribe(eventType string, handler EventHandler) {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *EventBus) Publish(event Event) {
    b.mu.RLock()
    handlers := b.handlers[event.Type()]
    b.mu.RUnlock()

    for _, handler := range handlers {
        if b.async {
            go handler.Handle(event)
        } else {
            handler.Handle(event)
        }
    }
}
```

---

## 4. 프론트엔드 아키텍처 (Vue 3)

### 4.1 디렉토리 구조

```
frontend/src/
├── main.ts                    # 앱 진입점
├── App.vue                    # 루트 컴포넌트
├── router/                    # Vue Router 설정
│   └── index.ts
├── stores/                    # Pinia 상태 관리
│   ├── auth.ts               # 인증 상태
│   ├── knowledge.ts          # 지식 베이스 상태
│   ├── settings.ts           # 사용자 설정
│   ├── ui.ts                 # UI 상태
│   └── menu.ts               # 메뉴 상태
├── api/                       # API 계층
│   ├── auth/                 # 인증 API
│   ├── chat/                 # 채팅 API + SSE
│   ├── knowledge-base/       # 지식 베이스 API
│   ├── model/                # 모델 관리 API
│   └── ...
├── views/                     # 페이지 컴포넌트
│   ├── auth/                 # 로그인/회원가입
│   ├── chat/                 # 채팅 인터페이스
│   ├── knowledge/            # 지식 관리
│   └── settings/             # 설정
├── components/                # 재사용 컴포넌트
├── hooks/                     # Vue Composables
├── i18n/                      # 다국어 지원
├── types/                     # TypeScript 타입
└── utils/                     # 유틸리티
    ├── request.ts            # HTTP 요청
    └── security.ts           # 보안 유틸리티
```

### 4.2 상태 관리 (Pinia)

#### Auth Store

```typescript
// stores/auth.ts
export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as User | null,
    tenant: null as Tenant | null,
    token: '',
    refreshToken: '',
    knowledgeBases: [] as KnowledgeBase[],
    selectedTenantId: '',
  }),

  actions: {
    async login(email: string, password: string) {
      const response = await authApi.login(email, password)
      this.setToken(response.token)
      this.setUser(response.user)
    },

    async logout() {
      await authApi.logout()
      this.$reset()
    },

    setKnowledgeBases(kbs: KnowledgeBase[]) {
      this.knowledgeBases = kbs
    }
  },

  persist: true  // localStorage 영속화
})
```

#### Knowledge Store

```typescript
// stores/knowledge.ts
export const useKnowledgeStore = defineStore('knowledge', {
  state: () => ({
    currentKB: null as KnowledgeBase | null,
    documents: [] as Document[],
    searchResults: [] as SearchResult[],
    loading: false,
  }),

  actions: {
    async loadKnowledgeBase(id: string) {
      this.loading = true
      try {
        this.currentKB = await kbApi.get(id)
        this.documents = await kbApi.listDocuments(id)
      } finally {
        this.loading = false
      }
    }
  }
})
```

### 4.3 API 계층

#### 기본 요청 유틸리티

```typescript
// utils/request.ts
export async function request<T>(
  method: 'GET' | 'POST' | 'PUT' | 'DELETE',
  url: string,
  data?: any,
  options?: RequestOptions
): Promise<T> {
  const authStore = useAuthStore()

  const response = await fetch(url, {
    method,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${authStore.token}`,
      ...options?.headers
    },
    body: data ? JSON.stringify(data) : undefined
  })

  if (!response.ok) {
    throw new ApiError(response.status, await response.text())
  }

  return response.json()
}
```

#### SSE 스트리밍 처리

```typescript
// api/chat/stream.ts
export function createChatStream(
  sessionId: string,
  messageId: string,
  onEvent: (event: ChatEvent) => void
): EventSource {
  const authStore = useAuthStore()
  const url = `/api/v1/sessions/${sessionId}/continue-stream?message_id=${messageId}`

  const eventSource = new EventSource(url, {
    headers: {
      'Authorization': `Bearer ${authStore.token}`
    }
  })

  eventSource.onmessage = (e) => {
    const event = JSON.parse(e.data) as ChatEvent
    onEvent(event)
  }

  eventSource.onerror = (e) => {
    console.error('SSE error:', e)
    eventSource.close()
  }

  return eventSource
}
```

### 4.4 라우팅 구조

```typescript
// router/index.ts
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/chat'
  },
  {
    path: '/auth',
    children: [
      { path: 'login', component: () => import('@/views/auth/Login.vue') },
      { path: 'register', component: () => import('@/views/auth/Register.vue') }
    ]
  },
  {
    path: '/chat',
    component: () => import('@/views/chat/Index.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/knowledge',
    component: () => import('@/views/knowledge/Index.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/settings',
    component: () => import('@/views/settings/Index.vue'),
    meta: { requiresAuth: true }
  }
]
```

---

## 5. 문서 파서 서비스 (docreader)

### 5.1 서비스 구조

**파일 위치:** `docreader/`

```
docreader/
├── main.py                    # gRPC 서버 진입점
├── proto/
│   └── docreader.proto        # gRPC 프로토콜 정의
├── parser/
│   ├── base_parser.py         # 기본 파서 인터페이스
│   ├── pdf_parser.py          # PDF 파서
│   ├── docx_parser.py         # Word 파서
│   ├── excel_parser.py        # Excel 파서
│   ├── csv_parser.py          # CSV 파서
│   ├── markdown_parser.py     # Markdown 파서
│   ├── text_parser.py         # 텍스트 파서
│   ├── image_parser.py        # 이미지 파서 (OCR)
│   ├── web_parser.py          # 웹 콘텐츠 파서
│   ├── ocr_engine.py          # OCR 엔진
│   └── caption.py             # 이미지 캡션
├── splitter/
│   └── text_splitter.py       # 텍스트 청킹
└── models/
    └── embedding.py           # 임베딩 클라이언트
```

### 5.2 gRPC 프로토콜

```protobuf
// proto/docreader.proto
syntax = "proto3";

service DocReader {
    rpc ReadFromFile(ReadRequest) returns (ReadResponse);
    rpc ReadFromURL(ReadURLRequest) returns (ReadResponse);
}

message ReadRequest {
    bytes file_content = 1;
    string file_name = 2;
    ReadConfig config = 3;
}

message ReadConfig {
    int32 chunk_size = 1;
    int32 chunk_overlap = 2;
    repeated string separators = 3;
}

message ReadResponse {
    repeated Chunk chunks = 1;
    DocumentMetadata metadata = 2;
}

message Chunk {
    string content = 1;
    ChunkMetadata metadata = 2;
    repeated Image images = 3;
}
```

### 5.3 문서 처리 파이프라인

```
입력 (파일/URL)
    │
    ▼
┌─────────────────┐
│  포맷 감지      │ ← 확장자 또는 MIME 타입
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 적절한 파서 선택 │
│  PDF/Word/Image │
│  /Markdown/...  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  콘텐츠 추출    │ ← 텍스트, 테이블, 이미지
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  청킹 처리      │ ← 오버랩 포함 분할
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 메타데이터 추출  │ ← 제목, 저자, 날짜
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ OCR 처리 (선택) │ ← 이미지 내 텍스트
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ 이미지 저장     │ ← MinIO/COS에 업로드
└────────┬────────┘
         │
         ▼
gRPC 응답 (청크 + 참조)
```

---

## 6. 데이터 흐름

### 6.1 요청 라이프사이클

```
┌──────────────────────────────────────────────────────────────────────┐
│                         요청 라이프사이클                             │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  [Vue Component]                                                     │
│       │                                                              │
│       ▼                                                              │
│  [Pinia Store] ──── 상태 관리                                        │
│       │                                                              │
│       ▼                                                              │
│  [API Layer] ──── HTTP/SSE 요청                                      │
│       │                                                              │
│       │ ════════════════════════════════════════════════════════     │
│       │              네트워크 경계                                    │
│       │ ════════════════════════════════════════════════════════     │
│       ▼                                                              │
│  [Middleware Chain]                                                  │
│       │                                                              │
│       ├── CORS ──── 크로스 도메인 허용                               │
│       ├── RequestID ──── 요청 추적 ID                                │
│       ├── Logger ──── 구조화 로깅                                    │
│       ├── Recovery ──── 패닉 복구                                    │
│       ├── ErrorHandler ──── 에러 응답                                │
│       ├── Auth ──── JWT 검증 + 테넌트 컨텍스트                       │
│       └── Tracing ──── 분산 추적                                     │
│       │                                                              │
│       ▼                                                              │
│  [Handler] ──── 요청 검증, 파라미터 바인딩                           │
│       │                                                              │
│       ▼                                                              │
│  [Service] ──── 비즈니스 로직                                        │
│       │                                                              │
│       ▼                                                              │
│  [Repository] ──── 데이터 접근                                       │
│       │                                                              │
│       ▼                                                              │
│  [Database] ──── PostgreSQL                                          │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 6.2 문서 수집 흐름

```
파일 업로드 (Frontend)
         │
         ▼
┌─────────────────────┐
│  Knowledge Handler  │
│  (handler/knowledge)│
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Knowledge Service  │
│  (service/knowledge)│
└──────────┬──────────┘
           │
           ▼ gRPC
┌─────────────────────┐
│  docreader Service  │
│  (Python)           │
│  ┌───────────────┐  │
│  │ 문서 파싱     │  │
│  │ 청킹 처리     │  │
│  │ 메타데이터    │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Embedding Service  │
│  (Ollama/외부 API)  │
│  배치 임베딩 생성   │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Retriever Service  │
│  ┌───────────────┐  │
│  │ PostgreSQL    │  │
│  │ Elasticsearch │  │
│  │ Qdrant        │  │
│  └───────────────┘  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  Chunk Repository   │
│  DB에 저장          │
└─────────────────────┘
```

### 6.3 쿼리/검색 흐름

```
사용자 쿼리
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│                    Session Handler                           │
│                 (KnowledgeQA / AgentQA)                      │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Embedding Service                         │
│                    쿼리 임베딩 생성                          │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Retriever Service                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   BM25      │  │   Vector    │  │   Graph     │         │
│  │  (키워드)   │  │  (시맨틱)   │  │ (지식그래프)│         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         └────────────────┼────────────────┘                 │
│                          ▼                                   │
│                    결과 병합/중복 제거                        │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      Reranker (선택)                         │
│                    관련성 재순위화                           │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Chat Pipeline                            │
│  ┌─────────┐ → ┌─────────┐ → ┌─────────┐ → ┌─────────┐    │
│  │ Search  │   │ Rerank  │   │  Merge  │   │  Chat   │    │
│  │ Plugin  │   │ Plugin  │   │ Plugin  │   │Completion│    │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘    │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     LLM Models                               │
│              (Ollama / 외부 API)                             │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Stream Manager                             │
│                 (Redis / Memory)                             │
│                   이벤트 버퍼링                              │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼ SSE
┌─────────────────────────────────────────────────────────────┐
│                  Frontend Chat Component                     │
│                  스트리밍 응답 렌더링                         │
└─────────────────────────────────────────────────────────────┘
```

### 6.4 에이전트 실행 흐름

```
에이전트 쿼리
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent Handler                             │
│           (handler/session/agent_stream_handler)             │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent Service                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 1. AgentEngine 생성                                 │   │
│  │ 2. 도구 레지스트리 등록                             │   │
│  │ 3. MCP 도구 등록                                    │   │
│  └─────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Agent Engine                              │
│                   (ReACT 루프)                               │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                 반복 (최대 30회)                      │  │
│  │  ┌─────────┐    ┌─────────┐    ┌─────────┐          │  │
│  │  │  Think  │ →  │   Act   │ →  │ Observe │          │  │
│  │  │ (LLM)   │    │ (도구)  │    │ (결과)  │          │  │
│  │  └─────────┘    └─────────┘    └─────────┘          │  │
│  │       │              │              │                │  │
│  │       │              │              │                │  │
│  │       └──────────────┴──────────────┘                │  │
│  │                      │                                │  │
│  │                      ▼                                │  │
│  │               EventBus.Publish()                      │  │
│  │               (실시간 스트리밍)                       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  도구 목록:                                                  │
│  - knowledge_search (시맨틱 검색)                           │
│  - grep_chunks (키워드 검색)                                │
│  - query_knowledge_graph (그래프 쿼리)                      │
│  - web_search (웹 검색)                                     │
│  - mcp_tool (MCP 도구)                                      │
│  - ...                                                       │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    AgentState 반환                           │
│  - RoundSteps: 실행 기록                                     │
│  - KnowledgeRefs: 참조 소스                                  │
│  - FinalAnswer: 최종 답변                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## 7. 외부 통합

### 7.1 벡터 데이터베이스 통합

| 백엔드 | 환경 변수 | 특징 | 파일 위치 |
|--------|-----------|------|----------|
| PostgreSQL (pgvector) | `RETRIEVE_DRIVER=postgres` | 네이티브 벡터 타입, BM25 지원 | `repository/retriever/postgres/` |
| Elasticsearch 7.x | `RETRIEVE_DRIVER=elasticsearch_v7` | 전문 검색 + 벡터 | `repository/retriever/elasticsearch/v7/` |
| Elasticsearch 8.x | `RETRIEVE_DRIVER=elasticsearch_v8` | 타입드 클라이언트 | `repository/retriever/elasticsearch/v8/` |
| Qdrant | `RETRIEVE_DRIVER=qdrant` | 고성능 벡터 DB | `repository/retriever/qdrant/` |
| Neo4j | `NEO4J_ENABLE=true` | 그래프 데이터베이스 | `repository/retriever/neo4j/` |

### 7.2 LLM 통합

#### Ollama (로컬)

**파일:** `internal/models/utils/ollama/`

```go
type OllamaClient struct {
    baseURL string
    client  *http.Client
}

func (c *OllamaClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
func (c *OllamaClient) Embed(ctx context.Context, req EmbedRequest) (*EmbedResponse, error)
func (c *OllamaClient) ListModels(ctx context.Context) ([]Model, error)
```

#### 외부 API

**파일:** `internal/models/chat/remote_api.go`

- OpenAI 호환 엔드포인트
- 커스텀 API 통합
- 스트리밍 지원
- Tool Use 지원

### 7.3 스토리지 통합

| 타입 | 환경 변수 | 파일 |
|------|-----------|------|
| 로컬 파일시스템 | `STORAGE_TYPE=local` | `service/file/local.go` |
| MinIO | `STORAGE_TYPE=minio` | `service/file/minio.go` |
| Tencent COS | `STORAGE_TYPE=cos` | `service/file/cos.go` |
| Dummy (테스트) | `STORAGE_TYPE=dummy` | `service/file/dummy.go` |

---

## 8. 설정 및 배포

### 8.1 설정 시스템

**파일:** `internal/config/config.go`

```go
type Config struct {
    Server          *ServerConfig         // HTTP 서버 설정
    Conversation    *ConversationConfig   // RAG 설정
    KnowledgeBase   *KnowledgeBaseConfig  // 청킹 설정
    Tenant          *TenantConfig         // 멀티테넌시
    Models          []ModelConfig         // LLM 모델
    VectorDatabase  *VectorDatabaseConfig // 벡터 DB
    DocReader       *DocReaderConfig      // 문서 파서
    StreamManager   *StreamManagerConfig  // 스트리밍
    WebSearch       *WebSearchConfig      // 웹 검색
}
```

### 8.2 주요 환경 변수

```bash
# 서버
GIN_MODE=release
APP_PORT=8080

# 데이터베이스
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=weknora

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# 벡터 검색 (쉼표로 구분)
RETRIEVE_DRIVER=postgres,elasticsearch_v8

# 스토리지
STORAGE_TYPE=local

# 문서 파서
DOCREADER_ADDR=localhost:50051

# 자동 마이그레이션
AUTO_MIGRATE=true

# Neo4j (선택)
NEO4J_ENABLE=false
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=password
```

### 8.3 Docker 배포

```bash
# 최소 서비스
docker compose up -d

# 전체 기능
docker-compose --profile full up -d

# 특정 기능 조합
docker-compose --profile neo4j --profile minio up -d
```

---

## 9. 주요 아키텍처 패턴

### 9.1 사용된 디자인 패턴

| 패턴 | 적용 위치 | 설명 |
|------|----------|------|
| 의존성 주입 (DI) | `internal/container/` | uber/dig 컨테이너 |
| Repository 패턴 | `repository/` | 데이터 추상화 계층 |
| Service 패턴 | `service/` | 비즈니스 로직 계층 |
| Handler 패턴 | `handler/` | HTTP 요청 처리 |
| Event-Driven | `event/` | 비동기 통신을 위한 EventBus |
| Plugin 아키텍처 | `chat_pipline/` | 플러그인 기반 처리 |
| Factory 패턴 | `file/`, `stream/` | 객체 생성 추상화 |
| Registry 패턴 | `retriever/` | 검색 엔진 레지스트리 |
| Strategy 패턴 | 다수 | 폴백 전략, 압축 전략 |
| Adapter 패턴 | `retriever/`, `file/` | 다중 백엔드 지원 |

### 9.2 횡단 관심사

#### 미들웨어 스택
- 요청 추적 (RequestID)
- 구조화된 로깅
- 에러 처리
- 인증/인가
- 분산 추적 (OpenTelemetry)

#### 리소스 관리
- ResourceCleaner를 통한 순서 있는 정리
- 연결 풀링 (DB, Redis)
- 고루틴 풀 (Ants)
- 컨텍스트 타임아웃

### 9.3 데이터 일관성

#### 멀티테넌시
- 미들웨어를 통한 테넌트 격리
- 행 수준 테넌트 필터링
- 교차 테넌트 접근 제어

#### 세션 관리
- 세션별 상태 격리
- 메시지 순서 보장
- 컨텍스트 압축

#### 동시성
- 고루틴 풀링
- 잠금 기반 동기화 (sync.RWMutex)
- 분산 상태를 위한 Redis

---

## 10. 참조 테이블

### 주요 파일 경로

| 구성 요소 | 위치 | 용도 |
|----------|------|------|
| 진입점 | `cmd/server/main.go` | 서버 시작 |
| DI 컨테이너 | `internal/container/` | 의존성 주입 |
| 라우팅 | `internal/router/` | HTTP 라우팅 |
| 미들웨어 | `internal/middleware/` | 요청 처리 |
| 핸들러 | `internal/handler/` | HTTP 핸들러 |
| 서비스 | `internal/application/service/` | 비즈니스 로직 |
| 레포지토리 | `internal/application/repository/` | 데이터 접근 |
| 에이전트 | `internal/agent/` | ReACT 에이전트 |
| 채팅 파이프라인 | `service/chat_pipline/` | 플러그인 파이프라인 |
| 타입/모델 | `internal/types/` | 데이터 모델 |
| 인터페이스 | `internal/types/interfaces/` | 계약 정의 |
| 이벤트 | `internal/event/` | 이벤트 버스 |
| 스트림 매니저 | `internal/stream/` | SSE 처리 |
| MCP | `internal/mcp/` | MCP 프로토콜 |
| 프론트엔드 | `frontend/src/` | UI 계층 |
| Pinia 스토어 | `frontend/src/stores/` | 상태 관리 |
| API 계층 | `frontend/src/api/` | HTTP 클라이언트 |
| docreader | `docreader/` | 문서 파싱 |

---

*이 문서는 uiscloud_weknora v0.2.x 기준으로 작성되었습니다.*
