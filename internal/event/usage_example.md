# 이벤트 시스템 사용 예시

## Chat Pipeline에 이벤트 시스템 통합하기

### 1. 서비스 초기화 시 이벤트 버스 설정

```go
// internal/container/container.go 또는 main.go

import (
    "github.com/Tencent/WeKnora/internal/event"
)

func InitializeEventSystem() {
    // 전역 이벤트 버스 가져오기
    bus := event.GetGlobalEventBus()

    // 모니터링 핸들러 등록
    event.NewMonitoringHandler(bus)

    // 분석 핸들러 등록
    event.NewAnalyticsHandler(bus)

    // 또는 커스텀 핸들러 등록
    bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // 커스텀 처리 로직
        return nil
    })
}
```

### 2. 쿼리 처리 서비스에서 이벤트 발송

#### 예시: search.go에 이벤트 추가

```go
// internal/application/service/chat_pipline/search.go

import (
    "github.com/Tencent/WeKnora/internal/event"
    "time"
)

func (p *PluginSearch) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // 검색 시작 이벤트 발송
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
    }).WithSessionID(chatManage.SessionID))

    // 검색 로직 실행
    results, err := p.performSearch(ctx, chatManage)
    if err != nil {
        // 에러 이벤트 발송
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "retrieval",
            SessionID: chatManage.SessionID,
            Query:     chatManage.ProcessedQuery,
        }).WithSessionID(chatManage.SessionID))
        return ErrSearch.WithError(err)
    }

    // 검색 완료 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventRetrievalComplete, event.RetrievalData{
        Query:           chatManage.ProcessedQuery,
        KnowledgeBaseID: chatManage.KnowledgeBaseID,
        TopK:            chatManage.EmbeddingTopK,
        RetrievalType:   "vector",
        ResultCount:     len(results),
        Duration:        time.Since(startTime).Milliseconds(),
        Results:         results,
    }).WithSessionID(chatManage.SessionID))

    chatManage.SearchResult = results
    return next()
}
```

#### 예시: rewrite.go에 이벤트 추가

```go
// internal/application/service/chat_pipline/rewrite.go

func (p *PluginRewriteQuery) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // 재작성 시작 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventQueryRewrite, event.QueryData{
        OriginalQuery: chatManage.Query,
        SessionID:     chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))

    // 쿼리 재작성 실행
    rewrittenQuery, err := p.rewriteQuery(ctx, chatManage)
    if err != nil {
        return ErrRewrite.WithError(err)
    }

    // 재작성 완료 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventQueryRewritten, event.QueryData{
        OriginalQuery:  chatManage.Query,
        RewrittenQuery: rewrittenQuery,
        SessionID:      chatManage.SessionID,
    }).WithSessionID(chatManage.SessionID))

    chatManage.RewriteQuery = rewrittenQuery
    return next()
}
```

#### 예시: rerank.go에 이벤트 추가

```go
// internal/application/service/chat_pipline/rerank.go

func (p *PluginRerank) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // 리랭킹 시작 이벤트 발송
    startTime := time.Now()
    inputCount := len(chatManage.SearchResult)

    event.Emit(ctx, event.NewEvent(event.EventRerankStart, event.RerankData{
        Query:      chatManage.ProcessedQuery,
        InputCount: inputCount,
        ModelID:    chatManage.RerankModelID,
    }).WithSessionID(chatManage.SessionID))

    // 리랭킹 실행
    rerankResults, err := p.performRerank(ctx, chatManage)
    if err != nil {
        return ErrRerank.WithError(err)
    }

    // 리랭킹 완료 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
        Query:       chatManage.ProcessedQuery,
        InputCount:  inputCount,
        OutputCount: len(rerankResults),
        ModelID:     chatManage.RerankModelID,
        Duration:    time.Since(startTime).Milliseconds(),
        Results:     rerankResults,
    }).WithSessionID(chatManage.SessionID))

    chatManage.RerankResult = rerankResults
    return next()
}
```

#### 예시: chat_completion.go에 이벤트 추가

```go
// internal/application/service/chat_pipline/chat_completion.go

func (p *PluginChatCompletion) OnEvent(
    ctx context.Context,
    eventType types.EventType,
    chatManage *types.ChatManage,
    next func() *PluginError,
) *PluginError {
    // 채팅 시작 이벤트 발송
    startTime := time.Now()
    event.Emit(ctx, event.NewEvent(event.EventChatStart, event.ChatData{
        Query:    chatManage.Query,
        ModelID:  chatManage.ChatModelID,
        IsStream: false,
    }).WithSessionID(chatManage.SessionID))

    // 모델 및 메시지 준비
    chatModel, opt, err := prepareChatModel(ctx, p.modelService, chatManage)
    if err != nil {
        return ErrGetChatModel.WithError(err)
    }

    chatMessages := prepareMessagesWithHistory(chatManage)

    // 모델 호출
    chatResponse, err := chatModel.Chat(ctx, chatMessages, opt)
    if err != nil {
        event.Emit(ctx, event.NewEvent(event.EventError, event.ErrorData{
            Error:     err.Error(),
            Stage:     "chat_completion",
            SessionID: chatManage.SessionID,
            Query:     chatManage.Query,
        }).WithSessionID(chatManage.SessionID))
        return ErrModelCall.WithError(err)
    }

    // 채팅 완료 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventChatComplete, event.ChatData{
        Query:      chatManage.Query,
        ModelID:    chatManage.ChatModelID,
        Response:   chatResponse.Content,
        TokenCount: chatResponse.TokenCount,
        Duration:   time.Since(startTime).Milliseconds(),
        IsStream:   false,
    }).WithSessionID(chatManage.SessionID))

    chatManage.ChatResponse = chatResponse
    return next()
}
```

### 3. Handler 레이어에서 요청 수신 이벤트 발송

```go
// internal/handler/message.go

func (h *MessageHandler) SendMessage(c *gin.Context) {
    ctx := c.Request.Context()

    // 요청 파싱
    var req types.SendMessageRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 쿼리 수신 이벤트 발송
    event.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: req.Content,
        SessionID:     req.SessionID,
        UserID:        c.GetString("user_id"),
    }).WithSessionID(req.SessionID).WithRequestID(c.GetString("request_id")))

    // 메시지 처리...
}
```

### 4. 커스텀 모니터링 핸들러

```go
// internal/monitoring/event_monitor.go

package monitoring

import (
    "context"
    "github.com/Tencent/WeKnora/internal/event"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    retrievalDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "retrieval_duration_milliseconds",
            Help: "Duration of retrieval operations",
        },
        []string{"knowledge_base_id", "retrieval_type"},
    )

    rerankDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "rerank_duration_milliseconds",
            Help: "Duration of rerank operations",
        },
        []string{"model_id"},
    )
)

func init() {
    prometheus.MustRegister(retrievalDuration)
    prometheus.MustRegister(rerankDuration)
}

func SetupEventMonitoring() {
    bus := event.GetGlobalEventBus()

    // 검색 성능 모니터링
    bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RetrievalData)
        retrievalDuration.WithLabelValues(
            data.KnowledgeBaseID,
            data.RetrievalType,
        ).Observe(float64(data.Duration))
        return nil
    })

    // 리랭킹 성능 모니터링
    bus.On(event.EventRerankComplete, func(ctx context.Context, e event.Event) error {
        data := e.Data.(event.RerankData)
        rerankDuration.WithLabelValues(data.ModelID).Observe(float64(data.Duration))
        return nil
    })
}
```

### 5. 로그 기록 핸들러

```go
// internal/logging/event_logger.go

package logging

import (
    "context"
    "encoding/json"
    "github.com/Tencent/WeKnora/internal/event"
    "github.com/Tencent/WeKnora/internal/logger"
)

func SetupEventLogging() {
    bus := event.GetGlobalEventBus()

    // 모든 이벤트에 대한 구조화 로그 기록
    logHandler := event.ApplyMiddleware(
        func(ctx context.Context, e event.Event) error {
            data, _ := json.Marshal(e.Data)
            logger.Infof(ctx, "Event: type=%s, session=%s, request=%s, data=%s",
                e.Type, e.SessionID, e.RequestID, string(data))
            return nil
        },
        event.WithTiming(),
    )

    // 모든 핵심 이벤트에 등록
    bus.On(event.EventQueryReceived, logHandler)
    bus.On(event.EventQueryRewritten, logHandler)
    bus.On(event.EventRetrievalComplete, logHandler)
    bus.On(event.EventRerankComplete, logHandler)
    bus.On(event.EventChatComplete, logHandler)
    bus.On(event.EventError, logHandler)
}
```

### 6. 완전한 초기화 흐름

```go
// cmd/server/main.go 또는 internal/container/container.go

func Initialize() {
    // 1. 이벤트 시스템 초기화
    eventBus := event.GetGlobalEventBus()

    // 2. 모니터링 설정
    event.NewMonitoringHandler(eventBus)

    // 3. 분석 설정
    event.NewAnalyticsHandler(eventBus)

    // 4. Prometheus 모니터링 설정 (필요한 경우)
    // monitoring.SetupEventMonitoring()

    // 5. 구조화 로그 설정 (필요한 경우)
    // logging.SetupEventLogging()

    // 6. 기타 초기화...
}
```

## 이벤트 시스템 테스트

```go
// 테스트에서 독립적인 이벤트 버스 사용
func TestMyService(t *testing.T) {
    ctx := context.Background()

    // 테스트 전용 이벤트 버스 생성
    testBus := event.NewEventBus()

    // 테스트 리스너 등록
    var receivedEvents []event.Event
    testBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        receivedEvents = append(receivedEvents, e)
        return nil
    })

    // 테스트 실행...
    testBus.Emit(ctx, event.NewEvent(event.EventQueryReceived, event.QueryData{
        OriginalQuery: "test",
    }))

    // 이벤트 검증
    if len(receivedEvents) != 1 {
        t.Errorf("Expected 1 event, got %d", len(receivedEvents))
    }
}
```

## 비동기 처리 예시

```go
// 메인 흐름에 영향을 주지 않는 이벤트는 비동기 모드를 사용할 수 있습니다
func SetupAsyncAnalytics() {
    asyncBus := event.NewAsyncEventBus()

    asyncBus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
        // 메인 흐름을 블로킹하지 않고 분석 플랫폼에 비동기 전송
        // sendToAnalyticsPlatform(e)
        return nil
    })

    // 비동기 버스로 이벤트 발송
    // asyncBus.Emit(ctx, event)
}
```

## 성능 최적화 권장사항

1. **핵심 경로에서 동기 이벤트 버스 사용 자제**: 비즈니스 로직에 영향을 주지 않는 모니터링, 로그 등은 비동기 모드 사용
2. **미들웨어 적절히 활용**: 필요한 곳에만 미들웨어를 사용하여 불필요한 오버헤드 방지
3. **이벤트 데이터 크기 제어**: 특히 비동기 모드에서 이벤트에 대량의 데이터를 전달하지 않도록 주의
4. **전용 리스너 사용**: 하나의 리스너에서 너무 많은 작업을 하지 않고 단일 책임 원칙 유지
