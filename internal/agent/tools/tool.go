package tools

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	description string
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description string) BaseTool {
	return BaseTool{
		name:        name,
		description: description,
	}
}

// Name returns the tool name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *BaseTool) Description() string {
	return t.description
}

// ToolExecutor is a helper interface for executing tools
type ToolExecutor interface {
	types.Tool

	// GetContext returns any context-specific data needed for tool execution
	GetContext() map[string]interface{}
}

// Shared helper functions for tool output formatting

// GetRelevanceLevel converts a score to a human-readable relevance level
func GetRelevanceLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "높은 관련성"
	case score >= 0.6:
		return "중간 관련성"
	case score >= 0.4:
		return "낮은 관련성"
	default:
		return "약한 관련성"
	}
}

// FormatMatchType converts MatchType to a human-readable string
func FormatMatchType(mt types.MatchType) string {
	switch mt {
	case types.MatchTypeEmbedding:
		return "벡터 매칭"
	case types.MatchTypeKeywords:
		return "키워드 매칭"
	case types.MatchTypeNearByChunk:
		return "인접 블록 매칭"
	case types.MatchTypeHistory:
		return "히스토리 매칭"
	case types.MatchTypeParentChunk:
		return "부모 블록 매칭"
	case types.MatchTypeRelationChunk:
		return "관계 블록 매칭"
	case types.MatchTypeGraph:
		return "그래프 매칭"
	default:
		return fmt.Sprintf("알 수 없는 유형(%d)", mt)
	}
}
