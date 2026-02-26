# WeKnora 메인 백엔드 Spring Boot 3.5.x 마이그레이션 검토

## 1. 개요

### 1.1 검토 목적

현재 Go 1.24+ 기반의 WeKnora 메인 백엔드를 Spring Boot 3.5.x로 마이그레이션하는 것의 기술적 타당성을 분석합니다.

### 1.2 결론 요약

| 항목 | 평가 |
|------|------|
| **전체 마이그레이션 가능성** | ✅ 가능 (70-75% 직접 매핑) |
| **난이도** | 중간~높음 |
| **예상 기간** | 12-16주 (현실적 추정) |
| **주요 위험 요소** | MCP 프로토콜, 에이전트 도구 시스템 |
| **권장 여부** | 조건부 권장 (명확한 이유가 있을 경우) |

---

## 2. 기술 스택 매핑

### 2.1 핵심 프레임워크

| Go 라이브러리 | 용도 | Spring/Java 대체 | 마이그레이션 난이도 |
|--------------|------|-----------------|-------------------|
| `gin-gonic/gin` | HTTP 프레임워크 | Spring Web MVC | 🟢 낮음 |
| `gorm/gorm` | ORM | Spring Data JPA / Hibernate | 🟢 낮음 |
| `uber/dig` | 의존성 주입 | Spring Framework DI | 🟢 낮음 |
| `go-redis/v9` | Redis 클라이언트 | Spring Data Redis (Lettuce) | 🟢 낮음 |
| `hibiken/asynq` | 비동기 작업 큐 | Spring Async / RabbitMQ | 🟡 중간 |

### 2.2 데이터베이스 및 벡터 저장소

| Go 라이브러리 | 용도 | Spring/Java 대체 | 마이그레이션 난이도 |
|--------------|------|-----------------|-------------------|
| `pgvector/pgvector-go` | PostgreSQL 벡터 확장 | pgvector-java | 🟢 낮음 |
| `go-elasticsearch/v7,v8` | Elasticsearch | Spring Data Elasticsearch | 🟡 중간 |
| `qdrant/go-client` | Qdrant 벡터 DB | Qdrant Java SDK | 🟢 낮음 |
| `neo4j/neo4j-go-driver` | Neo4j 그래프 DB | Spring Data Neo4j | 🟢 낮음 |

### 2.3 외부 서비스 통합

| Go 라이브러리 | 용도 | Spring/Java 대체 | 마이그레이션 난이도 |
|--------------|------|-----------------|-------------------|
| `ollama/ollama` | Ollama LLM | RestTemplate / WebClient | 🟢 낮음 |
| `sashabaranov/go-openai` | OpenAI API | OpenAI Java SDK | 🟢 낮음 |
| `minio/minio-go` | MinIO 스토리지 | MinIO Java SDK | 🟢 낮음 |
| `tencentyun/cos-go-sdk` | Tencent COS | Tencent COS Java SDK | 🟢 낮음 |
| `mark3labs/mcp-go` | MCP 프로토콜 | ❌ 직접 구현 필요 | 🔴 높음 |
| gRPC (docreader) | 문서 파서 통신 | gRPC Java | 🟡 중간 |

### 2.4 특수 기능

| Go 라이브러리 | 용도 | Spring/Java 대체 | 마이그레이션 난이도 |
|--------------|------|-----------------|-------------------|
| `panjf2000/ants/v2` | 고루틴 풀 | ThreadPoolExecutor | 🟢 낮음 |
| `yanyiwu/gojieba` | 중국어 형태소 분석 | HanLP / IK Analyzer | 🟢 낮음 |
| OpenTelemetry | 분산 추적 | OpenTelemetry Java SDK | 🟢 낮음 |
| SSE 스트리밍 | 실시간 응답 | Spring SseEmitter / WebFlux | 🟡 중간 |

---

## 3. 아키텍처 패턴 비교

### 3.1 계층 구조 매핑

```
┌─────────────────────────────────────────────────────────────────┐
│                         현재 Go 구조                             │
├─────────────────────────────────────────────────────────────────┤
│  internal/handler/     →  HTTP 요청 처리                        │
│  internal/application/service/  →  비즈니스 로직                │
│  internal/application/repository/  →  데이터 접근              │
│  internal/types/       →  도메인 모델                           │
│  internal/container/   →  의존성 주입 (uber/dig)               │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      Spring Boot 구조                            │
├─────────────────────────────────────────────────────────────────┤
│  controller/           →  @RestController                       │
│  service/              →  @Service                              │
│  repository/           →  @Repository (Spring Data JPA)        │
│  entity/domain/        →  @Entity                               │
│  config/               →  @Configuration                        │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 의존성 주입 비교

**현재 Go (uber/dig):**
```go
func BuildContainer() *dig.Container {
    c := dig.New()
    c.Provide(config.Load)
    c.Provide(database.NewDB)
    c.Provide(repository.NewTenantRepository)
    c.Provide(service.NewTenantService)
    c.Provide(handler.NewTenantHandler)
    return c
}
```

**Spring Boot:**
```java
@Configuration
public class AppConfig {
    @Bean
    public TenantService tenantService(TenantRepository repo) {
        return new TenantServiceImpl(repo);
    }
}

// 또는 더 간단하게 (자동 주입)
@Service
public class TenantServiceImpl implements TenantService {
    @Autowired
    private TenantRepository repository;
}
```

**평가:** 🟢 Spring의 어노테이션 기반 DI가 더 간결함

### 3.3 HTTP 핸들러 비교

**현재 Go (Gin):**
```go
func (h *TenantHandler) GetTenant(c *gin.Context) {
    id := c.Param("id")
    tenant, err := h.service.GetTenantByID(c.Request.Context(), id)
    if err != nil {
        c.Error(errors.NewNotFoundError(err.Error()))
        return
    }
    c.JSON(http.StatusOK, tenant)
}
```

**Spring Boot:**
```java
@RestController
@RequestMapping("/api/v1/tenants")
public class TenantController {
    @Autowired
    private TenantService service;

    @GetMapping("/{id}")
    public ResponseEntity<Tenant> getTenant(@PathVariable String id) {
        return ResponseEntity.ok(service.getTenantById(id));
    }

    @ExceptionHandler(TenantNotFoundException.class)
    public ResponseEntity<Error> handleNotFound(TenantNotFoundException ex) {
        return ResponseEntity.notFound().build();
    }
}
```

**평가:** 🟢 거의 동일한 패턴, 직접 매핑 가능

---

## 4. 주요 마이그레이션 과제

### 4.1 🔴 고위험: MCP (Model Context Protocol)

**현재 구현:**
- `internal/mcp/` 디렉토리
- 3가지 전송 방식 지원: Stdio, SSE, HTTP
- JSON-RPC 2.0 프로토콜

**문제점:**
- Java에 직접적인 MCP 라이브러리 없음
- 커스텀 구현 필요 (약 600-800줄)

**해결 방안:**
```java
// 옵션 1: 커스텀 MCP 클라이언트 구현
@Service
public class MCPManager {
    private Map<String, MCPClient> clients = new ConcurrentHashMap<>();

    public MCPClient getOrCreateClient(MCPService service) {
        return clients.computeIfAbsent(service.getId(),
            id -> createClient(service));
    }

    private MCPClient createClient(MCPService service) {
        switch (service.getTransportType()) {
            case STDIO: return new StdioMCPClient(service);
            case SSE: return new SSEMCPClient(service);
            case HTTP: return new HTTPMCPClient(service);
        }
    }
}

// Stdio 전송 (서브프로세스)
public class StdioMCPClient implements MCPClient {
    private Process process;

    public String call(String method, Map<String, Object> params) {
        JSONRPCRequest request = new JSONRPCRequest(method, params);
        // 프로세스 stdin으로 전송, stdout에서 응답 읽기
    }
}
```

**예상 작업량:** 2-3주

---

### 4.2 🟡 중위험: 에이전트 시스템 (ReACT 패턴)

**현재 구현:**
- `internal/agent/engine.go` (29,942 bytes)
- 14개 이상의 도구 구현 (약 200KB)

**주요 도구:**
| 도구 | 크기 | 복잡도 |
|------|------|--------|
| `knowledge_search.go` | 43KB | 높음 |
| `grep_chunks.go` | 21KB | 중간 |
| `web_fetch.go` | 17KB | 중간 |
| `todo_write.go` | 14KB | 중간 |
| 기타 10개 도구 | ~80KB | 다양 |

**Spring Boot 구현:**
```java
@Service
public class ReActAgentService {
    @Autowired private ChatModel chatModel;
    @Autowired private ToolRegistry toolRegistry;
    @Autowired private EventBus eventBus;

    public AgentState execute(String query, List<ChatMessage> history) {
        AgentState state = new AgentState();

        // ReACT 루프 (최대 30회)
        for (int i = 0; i < MAX_ITERATIONS && !state.isComplete(); i++) {
            // 1. Think - LLM 추론
            ChatResponse response = chatModel.chat(buildRequest(query, history));

            // 2. Act - 도구 실행
            if (response.hasToolCalls()) {
                for (ToolCall call : response.getToolCalls()) {
                    String result = toolRegistry.execute(call);
                    history.add(new ChatMessage("tool", result));
                }
            } else {
                state.setComplete(true);
            }
        }
        return state;
    }
}

// 도구 인터페이스
public interface Tool {
    String getName();
    String getDescription();
    String execute(String arguments);
}

// 도구 레지스트리
@Service
public class ToolRegistry {
    @Autowired private List<Tool> tools;

    public String execute(ToolCall call) {
        return tools.stream()
            .filter(t -> t.getName().equals(call.getName()))
            .findFirst()
            .orElseThrow()
            .execute(call.getArguments());
    }
}
```

**예상 작업량:** 2-3주

---

### 4.3 🟡 중위험: 비동기 작업 큐 (Asynq → Spring)

**현재 작업 유형:**
```go
const (
    TypeChunkExtract       = "chunk:extract"
    TypeDocumentProcess    = "document:process"
    TypeFAQImport          = "faq:import"
    TypeQuestionGeneration = "question:generate"
    TypeSummaryGeneration  = "summary:generate"
    TypeKBClone            = "kb:clone"
)
```

**Spring Boot 옵션:**

| 옵션 | 장점 | 단점 | 권장 |
|------|------|------|------|
| **Spring @Async** | 간단, 추가 인프라 불필요 | 분산 환경 미지원 | 소규모 |
| **Spring Batch** | 대용량 처리 최적화 | 학습 곡선 | 배치 작업 |
| **RabbitMQ** | 분산, 안정적 | 추가 인프라 필요 | ✅ 권장 |
| **Kafka** | 고처리량 | 복잡성 | 대규모 |

**RabbitMQ 구현 예시:**
```java
@Configuration
public class RabbitConfig {
    @Bean
    public Queue documentProcessQueue() {
        return new Queue("document:process", true);
    }
}

@Service
public class TaskPublisher {
    @Autowired private RabbitTemplate rabbitTemplate;

    public void enqueueDocumentProcess(DocumentTask task) {
        rabbitTemplate.convertAndSend("document:process", task);
    }
}

@Service
public class TaskConsumer {
    @RabbitListener(queues = "document:process")
    public void processDocument(DocumentTask task) {
        // 문서 처리 로직
    }
}
```

**예상 작업량:** 1-2주

---

### 4.4 🟢 저위험: SSE 스트리밍

**현재 Go 구현:**
```go
func (h *Handler) ContinueStream(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")

    for {
        events, _ := h.streamManager.GetEvents(ctx, sessionID, offset)
        for _, evt := range events {
            fmt.Fprintf(c.Writer, "data: %s\n\n", json.Marshal(evt))
            c.Writer.Flush()
        }
    }
}
```

**Spring Boot 구현:**
```java
@GetMapping(value = "/stream/{sessionId}", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
public SseEmitter stream(@PathVariable String sessionId) {
    SseEmitter emitter = new SseEmitter(300000L);

    CompletableFuture.runAsync(() -> {
        while (!completed) {
            List<Event> events = streamManager.getEvents(sessionId, offset);
            for (Event event : events) {
                emitter.send(SseEmitter.event()
                    .id(event.getId())
                    .data(event));
            }
        }
        emitter.complete();
    });

    return emitter;
}
```

**또는 Spring WebFlux (반응형):**
```java
@GetMapping(value = "/stream/{sessionId}", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
public Flux<ServerSentEvent<Event>> stream(@PathVariable String sessionId) {
    return Flux.interval(Duration.ofMillis(100))
        .map(seq -> streamManager.getLatestEvent(sessionId))
        .filter(Objects::nonNull)
        .map(event -> ServerSentEvent.builder(event).build());
}
```

**예상 작업량:** 3-5일

---

## 5. 마이그레이션 단계별 계획

### Phase 1: 기반 구축 (2-3주)

| 작업 | 상세 |
|------|------|
| 프로젝트 설정 | Spring Boot 3.5.x 프로젝트 생성, Gradle/Maven 설정 |
| 데이터베이스 설정 | PostgreSQL + pgvector, Elasticsearch, Redis 연결 |
| 기본 엔티티 | Tenant, User, KnowledgeBase, Knowledge, Chunk 엔티티 |
| 기본 Repository | Spring Data JPA Repository 인터페이스 |

### Phase 2: 핵심 서비스 (3-4주)

| 작업 | 상세 |
|------|------|
| 인증 서비스 | JWT 인증, 사용자 관리 |
| 지식 베이스 서비스 | CRUD, 검색, 복사 기능 |
| 청크 서비스 | 청크 관리, 벡터 검색 |
| 세션 서비스 | 대화 세션 관리 |
| 컨트롤러 | 15개 이상의 REST 컨트롤러 구현 |

### Phase 3: 고급 기능 (3-4주)

| 작업 | 상세 |
|------|------|
| 채팅 파이프라인 | 18개 플러그인 시스템 마이그레이션 |
| 에이전트 시스템 | ReACT 엔진, 14개 도구 구현 |
| 이벤트 버스 | 23개 이벤트 타입 처리 |
| 스트림 매니저 | SSE 스트리밍 구현 |

### Phase 4: 통합 및 비동기 (2-3주)

| 작업 | 상세 |
|------|------|
| 작업 큐 | RabbitMQ 또는 Spring Batch 설정 |
| MCP 프로토콜 | 커스텀 MCP 클라이언트 구현 |
| 외부 서비스 | gRPC (docreader), Ollama, OpenAI 통합 |
| 파일 스토리지 | MinIO, Tencent COS 연동 |

### Phase 5: 마무리 (1-2주)

| 작업 | 상세 |
|------|------|
| 테스트 | 단위 테스트, 통합 테스트 |
| 성능 최적화 | 벤치마킹, 캐싱 전략 |
| 문서화 | API 문서 (SpringDoc OpenAPI) |
| 컨테이너화 | Docker 이미지, docker-compose |

---

## 6. 리소스 요구 사항

### 6.1 팀 구성 권장

| 역할 | 인원 | 역할 설명 |
|------|------|----------|
| 시니어 Java 개발자 | 2-3명 | Spring Boot 전문가, 핵심 마이그레이션 |
| DevOps 엔지니어 | 1명 | 컨테이너화, CI/CD |
| QA 엔지니어 | 1명 | 테스트, 성능 검증 |

### 6.2 예상 일정

| 시나리오 | 기간 | 조건 |
|----------|------|------|
| **최상** | 8-10주 | 숙련된 팀, 병렬 작업 |
| **현실적** | 12-16주 | 일반적인 속도 + 테스트 |
| **보수적** | 18-24주 | 리팩토링 + 최적화 포함 |

---

## 7. 위험 평가

| 위험 요소 | 영향도 | 발생 가능성 | 완화 방안 |
|----------|--------|------------|----------|
| MCP 프로토콜 호환성 | 🔴 높음 | 🔴 높음 | 조기에 커스텀 래퍼 라이브러리 개발 |
| 에이전트 도구 복잡성 | 🟡 중간 | 🟡 중간 | 단순한 도구부터 시작, 반복적 테스트 |
| 성능 저하 | 🟡 중간 | 🟡 중간 | 핵심 경로 벤치마킹, 캐싱 활용 |
| 채팅 파이프라인 순서 | 🟡 중간 | 🟢 낮음 | 플러그인 체인 종합 테스트 |
| gRPC 통합 문제 | 🟡 중간 | 🟢 낮음 | 생성된 Java 스텁 사용 |
| 분산 세션 관리 | 🟡 중간 | 🟡 중간 | Spring Session + Redis 활용 |

---

## 8. 장단점 분석

### 8.1 마이그레이션 장점

| 장점 | 설명 |
|------|------|
| ✅ 풍부한 생태계 | Spring Boot의 방대한 라이브러리 및 통합 |
| ✅ 개발자 풀 | Java/Spring 개발자 채용 용이 |
| ✅ 엔터프라이즈 지원 | VMware(Pivotal) 공식 지원, 장기 LTS |
| ✅ 성숙한 도구 | IntelliJ, 디버깅, 프로파일링 도구 |
| ✅ Spring AI | LLM 통합을 위한 Spring AI 프로젝트 활용 가능 |
| ✅ 관측성 | Spring Actuator, Micrometer 기본 제공 |

### 8.2 마이그레이션 단점

| 단점 | 설명 |
|------|------|
| ❌ 개발 기간 | 12-16주 이상 소요 |
| ❌ 메모리 사용량 | JVM 기반으로 Go 대비 메모리 사용량 증가 |
| ❌ 시작 시간 | 콜드 스타트 시간 증가 (GraalVM Native로 완화 가능) |
| ❌ MCP 호환성 | 커스텀 구현 필요 |
| ❌ 동시성 모델 | 고루틴 → 스레드 풀 (패러다임 변경) |
| ❌ 기능 동결 | 마이그레이션 중 새 기능 개발 제한 |

### 8.3 현재 Go 유지 장점

| 장점 | 설명 |
|------|------|
| ✅ 이미 작동 중 | 안정적으로 운영 중인 시스템 |
| ✅ 성능 | Go의 낮은 메모리 사용량, 빠른 시작 |
| ✅ 동시성 | 고루틴의 경량 동시성 |
| ✅ 단순성 | 적은 추상화, 명시적 코드 |
| ✅ MCP 지원 | mark3labs/mcp-go 라이브러리 사용 중 |

---

## 9. 권장 사항

### 9.1 마이그레이션 권장 조건

다음 조건에 해당하면 **마이그레이션 권장**:

1. ✅ 조직 내 Java/Spring 전문가가 Go 전문가보다 많음
2. ✅ 기존 Spring 기반 시스템과 통합 필요
3. ✅ Spring AI 등 Spring 생태계 활용 계획
4. ✅ 엔터프라이즈 지원 및 LTS가 중요
5. ✅ 충분한 마이그레이션 기간 확보 가능

### 9.2 현재 유지 권장 조건

다음 조건에 해당하면 **Go 유지 권장**:

1. ✅ 현재 시스템이 안정적으로 운영 중
2. ✅ Go 전문가가 팀에 있음
3. ✅ 빠른 기능 개발이 우선
4. ✅ 컨테이너 환경에서 메모리 효율성 중요
5. ✅ MCP 프로토콜을 적극 활용 중

### 9.3 하이브리드 접근법

**권장:** 점진적 마이그레이션

```
┌─────────────────────────────────────────────────────────────┐
│                     하이브리드 아키텍처                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌─────────────┐          ┌─────────────┐                 │
│   │   Spring    │   REST   │     Go      │                 │
│   │   Boot      │ ◄──────► │   Backend   │                 │
│   │  (신규API)  │          │  (기존API)  │                 │
│   └─────────────┘          └─────────────┘                 │
│         │                        │                          │
│         └────────────┬───────────┘                          │
│                      ▼                                      │
│              ┌─────────────┐                               │
│              │  공유 DB    │                               │
│              │  Redis      │                               │
│              │  Vector DB  │                               │
│              └─────────────┘                               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**장점:**
- 점진적 마이그레이션으로 위험 감소
- 기존 기능 유지하면서 새 기능은 Spring으로 개발
- 필요시 롤백 가능

---

## 10. 결론

### 10.1 기술적 타당성

**Spring Boot 3.5.x 마이그레이션은 기술적으로 가능합니다.**

- 70-75%의 코드가 직접 매핑 가능
- 주요 라이브러리에 Java 대체제 존재
- 아키텍처 패턴이 Spring과 호환

### 10.2 주요 고려 사항

| 항목 | 내용 |
|------|------|
| **가장 큰 과제** | MCP 프로토콜 커스텀 구현 (2-3주) |
| **두 번째 과제** | 에이전트 도구 시스템 마이그레이션 (2-3주) |
| **예상 총 기간** | 12-16주 (현실적 추정) |
| **필요 인력** | 시니어 Java 개발자 2-3명 + DevOps 1명 |

### 10.3 최종 권장

```
┌─────────────────────────────────────────────────────────────┐
│                        최종 권장                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  명확한 비즈니스 이유가 있다면 → 마이그레이션 진행          │
│                                                             │
│  단순히 "Java가 더 익숙해서"라면 → 현재 유지 권장          │
│                                                             │
│  결정 전 → 파일럿 프로젝트로 핵심 모듈 마이그레이션 테스트  │
│           (예: Tenant/KnowledgeBase 관리 모듈)              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

*이 문서는 WeKnora v0.2.x 코드베이스 분석을 기반으로 작성되었습니다.*
*Spring Boot 3.5.x는 2024년 12월 기준 최신 버전입니다.*
