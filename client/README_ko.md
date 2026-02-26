# WeKnora HTTP 클라이언트

이 패키지는 WeKnora 서비스와 상호 작용하기 위한 클라이언트 라이브러리를 제공합니다. 모든 HTTP 기반 인터페이스 호출을 지원하여 다른 모듈이 직접 HTTP 요청 코드를 작성하지 않고도 WeKnora 서비스를 쉽게 통합할 수 있습니다.

## 주요 기능

이 클라이언트에는 다음과 같은 주요 기능 모듈이 포함되어 있습니다:

1. **세션 관리**: 세션 생성, 조회, 수정 및 삭제
2. **지식베이스 관리**: 지식베이스 생성, 조회, 수정 및 삭제
3. **지식 관리**: 지식 콘텐츠 추가, 조회 및 삭제
4. **테넌트 관리**: 테넌트 CRUD 작업
5. **지식 Q&A**: 일반 Q&A 및 스트리밍 Q&A 지원
6. **Agent Q&A**: Agent 기반 지능형 Q&A 지원, 사고 과정, 도구 호출 및 반성 포함
7. **청크 관리**: 지식 청크 조회, 수정 및 삭제
8. **메시지 관리**: 세션 메시지 조회 및 삭제
9. **모델 관리**: 모델 생성, 조회, 수정 및 삭제

## 사용 방법

### 클라이언트 인스턴스 생성

```go
import (
    "context"
    "github.com/Tencent/WeKnora/internal/client"
    "time"
)

// 클라이언트 인스턴스 생성
apiClient := client.NewClient(
    "http://api.example.com",
    client.WithToken("your-auth-token"),
    client.WithTimeout(30*time.Second),
)
```

### 예시: 지식베이스 생성 및 파일 업로드

```go
// 지식베이스 생성
kb := &client.KnowledgeBase{
    Name:        "테스트 지식베이스",
    Description: "테스트 지식베이스입니다",
    ChunkingConfig: client.ChunkingConfig{
        ChunkSize:    500,
        ChunkOverlap: 50,
        Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
    },
    ImageProcessingConfig: client.ImageProcessingConfig{
        ModelID: "image_model_id",
    },
    EmbeddingModelID: "embedding_model_id",
    SummaryModelID:   "summary_model_id",
}

kb, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
if err != nil {
    // 오류 처리
}

// 지식 파일 업로드 및 메타데이터 추가
metadata := map[string]string{
    "source": "local",
    "type":   "document",
}
knowledge, err := apiClient.CreateKnowledgeFromFile(context.Background(), kb.ID, "path/to/file.pdf", metadata)
if err != nil {
    // 오류 처리
}
```

### 예시: 세션 생성 및 Q&A 수행

```go
// 세션 생성
sessionRequest := &client.CreateSessionRequest{
    KnowledgeBaseID: knowledgeBaseID,
    SessionStrategy: &client.SessionStrategy{
        MaxRounds:        10,
        EnableRewrite:    true,
        FallbackStrategy: "fixed_answer",
        FallbackResponse: "죄송합니다, 이 질문에 답변할 수 없습니다",
        EmbeddingTopK:    5,
        KeywordThreshold: 0.5,
        VectorThreshold:  0.7,
        RerankModelID:    "rerank_model_id",
        RerankTopK:       3,
        RerankThreshold:  0.8,
        SummaryModelID:   "summary_model_id",
    },
}

session, err := apiClient.CreateSession(context.Background(), sessionRequest)
if err != nil {
    // 오류 처리
}

// 일반 Q&A
answer, err := apiClient.KnowledgeQA(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query: "인공지능이란 무엇인가요?",
})
if err != nil {
    // 오류 처리
}

// 스트리밍 Q&A
err = apiClient.KnowledgeQAStream(context.Background(), session.ID, "머신러닝이란 무엇인가요?", func(response *client.StreamResponse) error {
    // 각 응답 조각 처리
    fmt.Print(response.Content)
    return nil
})
if err != nil {
    // 오류 처리
}
```

### 예시: Agent 지능형 Q&A

Agent Q&A는 도구 호출, 사고 과정 표시 및 자기 반성을 지원하는 더 강력한 지능형 대화 기능을 제공합니다.

```go
// Agent 세션 생성
agentSession := apiClient.NewAgentSession(session.ID)

// 완전한 이벤트 처리와 함께 Agent Q&A 수행
err := agentSession.Ask(context.Background(), "머신러닝 관련 지식을 검색하고 요점을 정리해주세요",
    func(resp *client.AgentStreamResponse) error {
        switch resp.ResponseType {
        case client.AgentResponseTypeThinking:
            // Agent가 생각 중
            if resp.Done {
                fmt.Printf("💭 생각: %s\n", resp.Content)
            }

        case client.AgentResponseTypeToolCall:
            // Agent가 도구 호출
            if resp.Data != nil {
                toolName := resp.Data["tool_name"]
                fmt.Printf("🔧 도구 호출: %v\n", toolName)
            }

        case client.AgentResponseTypeToolResult:
            // 도구 실행 결과
            fmt.Printf("✓ 도구 결과: %s\n", resp.Content)

        case client.AgentResponseTypeReferences:
            // 지식 참조
            if resp.KnowledgeReferences != nil {
                fmt.Printf("📚 %d개의 관련 지식 발견\n", len(resp.KnowledgeReferences))
                for _, ref := range resp.KnowledgeReferences {
                    fmt.Printf("  - [%.3f] %s\n", ref.Score, ref.KnowledgeTitle)
                }
            }

        case client.AgentResponseTypeAnswer:
            // 최종 답변 (스트리밍 출력)
            fmt.Print(resp.Content)
            if resp.Done {
                fmt.Println() // 완료 후 줄바꿈
            }

        case client.AgentResponseTypeReflection:
            // Agent의 자기 반성
            if resp.Done {
                fmt.Printf("🤔 반성: %s\n", resp.Content)
            }

        case client.AgentResponseTypeError:
            // 오류 정보
            fmt.Printf("❌ 오류: %s\n", resp.Content)
        }
        return nil
    })

if err != nil {
    // 오류 처리
}

// 간소화 버전: 최종 답변만 관심
var finalAnswer string
err = agentSession.Ask(context.Background(), "딥러닝이란 무엇인가요?",
    func(resp *client.AgentStreamResponse) error {
        if resp.ResponseType == client.AgentResponseTypeAnswer {
            finalAnswer += resp.Content
        }
        return nil
    })
```

### Agent 이벤트 타입 설명

| 이벤트 타입 | 설명 | 트리거 시점 |
|---------|------|---------|
| `AgentResponseTypeThinking` | Agent 사고 과정 | Agent가 문제를 분석하고 계획을 수립할 때 |
| `AgentResponseTypeToolCall` | 도구 호출 | Agent가 특정 도구를 사용하기로 결정했을 때 |
| `AgentResponseTypeToolResult` | 도구 실행 결과 | 도구 실행이 완료된 후 |
| `AgentResponseTypeReferences` | 지식 참조 | 관련 지식을 검색했을 때 |
| `AgentResponseTypeAnswer` | 최종 답변 | Agent가 응답을 생성할 때 (스트리밍) |
| `AgentResponseTypeReflection` | 자기 반성 | Agent가 자신의 응답을 평가할 때 |
| `AgentResponseTypeError` | 오류 | 오류가 발생했을 때 |

### Agent Q&A 테스트 도구

Agent 기능을 테스트하기 위한 대화형 명령줄 도구를 제공합니다:

```bash
cd client/cmd/agent_test
go build -o agent_test
./agent_test -url http://localhost:8080 -kb <knowledge_base_id>
```

이 도구는 다음을 지원합니다:
- 세션 생성 및 관리
- 대화형 Agent Q&A
- 모든 Agent 이벤트 실시간 표시
- 성능 통계 및 디버그 정보

자세한 사용 방법은 `client/cmd/agent_test/README.md`를 참조하세요.

### Agent Q&A 고급 사용법

더 많은 고급 사용법 예시는 `agent_example.go` 파일을 참조하세요:
- 기본 Agent Q&A
- 도구 호출 추적
- 지식 참조 캡처
- 완전한 이벤트 추적
- 사용자 정의 오류 처리
- 스트림 취소 제어
- 다중 세션 관리

```

### 예시: 모델 관리

```go
// 모델 생성
modelRequest := &client.CreateModelRequest{
    Name:        "테스트 모델",
    Type:        client.ModelTypeChat,
    Source:      client.ModelSourceInternal,
    Description: "테스트 모델입니다",
    Parameters: client.ModelParameters{
        "temperature": 0.7,
        "top_p":       0.9,
    },
    IsDefault: true,
}
model, err := apiClient.CreateModel(context.Background(), modelRequest)
if err != nil {
    // 오류 처리
}

// 모든 모델 조회
models, err := apiClient.ListModels(context.Background())
if err != nil {
    // 오류 처리
}
```

### 예시: 지식 청크 관리

```go
// 지식 청크 목록 조회
chunks, total, err := apiClient.ListKnowledgeChunks(context.Background(), knowledgeID, 1, 10)
if err != nil {
    // 오류 처리
}

// 청크 업데이트
updateRequest := &client.UpdateChunkRequest{
    Content:   "업데이트된 청크 내용",
    IsEnabled: true,
}
updatedChunk, err := apiClient.UpdateChunk(context.Background(), knowledgeID, chunkID, updateRequest)
if err != nil {
    // 오류 처리
}
```

### 예시: 세션 메시지 조회

```go
// 최근 메시지 조회
messages, err := apiClient.GetRecentMessages(context.Background(), sessionID, 10)
if err != nil {
    // 오류 처리
}

// 특정 시간 이전의 메시지 조회
beforeTime := time.Now().Add(-24 * time.Hour)
olderMessages, err := apiClient.GetMessagesBefore(context.Background(), sessionID, beforeTime, 10)
if err != nil {
    // 오류 처리
}
```

## 전체 예시

클라이언트의 전체 사용 흐름을 보여주는 `example.go` 파일의 `ExampleUsage` 함수를 참조하세요.
