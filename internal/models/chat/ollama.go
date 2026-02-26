package chat

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/types"
	ollamaapi "github.com/ollama/ollama/api"
)

type OllamaChat struct {
	modelName     string
	modelID       string
	ollamaService *ollama.OllamaService
}

func NewOllamaChat(config *ChatConfig, ollamaService *ollama.OllamaService) (*OllamaChat, error) {
	return &OllamaChat{
		modelName:     config.ModelName,
		modelID:       config.ModelID,
		ollamaService: ollamaService,
	}, nil
}

func (c *OllamaChat) convertMessages(messages []Message) []ollamaapi.Message {
	ollamaMessages := make([]ollamaapi.Message, len(messages))
	for i, msg := range messages {
		ollamaMessages[i] = ollamaapi.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return ollamaMessages
}

func (c *OllamaChat) buildChatRequest(messages []Message, opts *ChatOptions, isStream bool) *ollamaapi.ChatRequest {
	streamFlag := isStream

	chatReq := &ollamaapi.ChatRequest{
		Model:    c.modelName,
		Messages: c.convertMessages(messages),
		Stream:   &streamFlag,
		Options:  make(map[string]interface{}),
	}

	if opts != nil {
		if opts.Temperature > 0 {
			chatReq.Options["temperature"] = opts.Temperature
		}
		if opts.TopP > 0 {
			chatReq.Options["top_p"] = opts.TopP
		}
		if opts.MaxTokens > 0 {
			chatReq.Options["num_predict"] = opts.MaxTokens
		}
		if opts.Thinking != nil {
			chatReq.Think = &ollamaapi.ThinkValue{
				Value: *opts.Thinking,
			}
		}
	}

	return chatReq
}

func (c *OllamaChat) Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error) {
	if err := c.ensureModelAvailable(ctx); err != nil {
		return nil, err
	}

	chatReq := c.buildChatRequest(messages, opts, false)

	logger.GetLogger(ctx).Infof("모델 %s 에 채팅 요청 전송", c.modelName)

	var responseContent string
	var promptTokens, completionTokens int

	err := c.ollamaService.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
		responseContent = resp.Message.Content

		if resp.EvalCount > 0 {
			promptTokens = resp.PromptEvalCount
			completionTokens = resp.EvalCount - promptTokens
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("채팅 요청 실패: %w", err)
	}

	return &types.ChatResponse{
		Content: responseContent,
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func (c *OllamaChat) ChatStream(
	ctx context.Context,
	messages []Message,
	opts *ChatOptions,
) (<-chan types.StreamResponse, error) {
	if err := c.ensureModelAvailable(ctx); err != nil {
		return nil, err
	}

	chatReq := c.buildChatRequest(messages, opts, true)

	logger.GetLogger(ctx).Infof("모델 %s 에 스트리밍 채팅 요청 전송", c.modelName)

	streamChan := make(chan types.StreamResponse)

	go func() {
		defer close(streamChan)

		err := c.ollamaService.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
			if resp.Message.Content != "" {
				streamChan <- types.StreamResponse{
					ResponseType: types.ResponseTypeAnswer,
					Content:      resp.Message.Content,
					Done:         false,
				}
			}

			if resp.Done {
				streamChan <- types.StreamResponse{
					ResponseType: types.ResponseTypeAnswer,
					Done:         true,
				}
			}

			return nil
		})
		if err != nil {
			logger.GetLogger(ctx).Errorf("스트리밍 채팅 요청 실패: %v", err)
			streamChan <- types.StreamResponse{
				ResponseType: types.ResponseTypeAnswer,
				Done:         true,
			}
		}
	}()

	return streamChan, nil
}

func (c *OllamaChat) ensureModelAvailable(ctx context.Context) error {
	logger.GetLogger(ctx).Infof("모델 %s 사용 가능 여부 확인", c.modelName)
	return c.ollamaService.EnsureModelAvailable(ctx, c.modelName)
}

func (c *OllamaChat) GetModelName() string {
	return c.modelName
}

func (c *OllamaChat) GetModelID() string {
	return c.modelID
}
