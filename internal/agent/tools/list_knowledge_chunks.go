package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// ListKnowledgeChunksTool retrieves chunk snapshots for a specific knowledge document.
type ListKnowledgeChunksTool struct {
	BaseTool
	tenantID         uint64
	chunkService     interfaces.ChunkService
	knowledgeService interfaces.KnowledgeService
}

// NewListKnowledgeChunksTool creates a new tool instance.
func NewListKnowledgeChunksTool(
	tenantID uint64,
	knowledgeService interfaces.KnowledgeService,
	chunkService interfaces.ChunkService,
) *ListKnowledgeChunksTool {
	description := `Retrieve full chunk content for a document by knowledge_id.

## Use After grep_chunks or knowledge_search:
1. grep_chunks(["keyword", "variant"]) → get knowledge_id
2. list_knowledge_chunks(knowledge_id) → read full content

## When to Use:
- Need full content of chunks from a known document
- Want to see context around specific chunks
- Check how many chunks a document has

## Parameters:
- knowledge_id (required): Document ID
- limit (optional): Chunks per page (default 20, max 100)
- offset (optional): Start position (default 0)

## Output:
Full chunk content with chunk_id, chunk_index, and content text.`

	return &ListKnowledgeChunksTool{
		BaseTool:         NewBaseTool("list_knowledge_chunks", description),
		tenantID:         tenantID,
		chunkService:     chunkService,
		knowledgeService: knowledgeService,
	}
}

// Parameters returns the JSON schema describing accepted arguments.
func (t *ListKnowledgeChunksTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"knowledge_id": map[string]interface{}{
				"type":        "string",
				"description": "Document ID to retrieve chunks from",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Chunks per page (default 20, max 100)",
				"default":     20,
				"minimum":     1,
				"maximum":     100,
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Start position (default 0)",
				"default":     0,
				"minimum":     0,
			},
		},
		"required": []string{"knowledge_id", "limit", "offset"},
	}
}

// Execute performs the chunk fetch against the chunk service.
func (t *ListKnowledgeChunksTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	knowledgeID, ok := args["knowledge_id"].(string)
	if !ok || strings.TrimSpace(knowledgeID) == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "knowledge_id is required",
		}, fmt.Errorf("knowledge_id is required")
	}
	knowledgeID = strings.TrimSpace(knowledgeID)

	chunkLimit := 20
	offset := 0
	if rawLimit, exists := args["limit"]; exists {
		switch v := rawLimit.(type) {
		case float64:
			chunkLimit = int(v)
		case int:
			chunkLimit = v
		}
	}
	if rawOffset, exists := args["offset"]; exists {
		switch v := rawOffset.(type) {
		case float64:
			offset = int(v)
		case int:
			offset = v
		}
	}
	if offset < 0 {
		offset = 0
	}

	pagination := &types.Pagination{
		Page:     offset/chunkLimit + 1,
		PageSize: chunkLimit,
	}

	chunks, total, err := t.chunkService.GetRepository().ListPagedChunksByKnowledgeID(ctx,
		t.tenantID, knowledgeID, pagination, []types.ChunkType{types.ChunkTypeText, types.ChunkTypeFAQ}, "", "")
	if err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to list chunks: %v", err),
		}, err
	}
	if chunks == nil {
		return &types.ToolResult{
			Success: false,
			Error:   "chunk query returned no data",
		}, fmt.Errorf("chunk query returned no data")
	}

	totalChunks := total
	fetched := len(chunks)

	knowledgeTitle := t.lookupKnowledgeTitle(ctx, knowledgeID)

	output := t.buildOutput(knowledgeID, knowledgeTitle, totalChunks, fetched, chunks)

	formattedChunks := make([]map[string]interface{}, 0, len(chunks))
	for idx, c := range chunks {
		chunkData := map[string]interface{}{
			"seq":             idx + 1,
			"chunk_id":        c.ID,
			"chunk_index":     c.ChunkIndex,
			"content":         c.Content,
			"chunk_type":      c.ChunkType,
			"knowledge_id":    c.KnowledgeID,
			"knowledge_base":  c.KnowledgeBaseID,
			"start_at":        c.StartAt,
			"end_at":          c.EndAt,
			"parent_chunk_id": c.ParentChunkID,
		}

		if c.ImageInfo != "" {
			var imageInfos []types.ImageInfo
			if err := json.Unmarshal([]byte(c.ImageInfo), &imageInfos); err == nil && len(imageInfos) > 0 {
				imageList := make([]map[string]string, 0, len(imageInfos))
				for _, img := range imageInfos {
					imgData := make(map[string]string)
					if img.URL != "" {
						imgData["url"] = img.URL
					}
					if img.Caption != "" {
						imgData["caption"] = img.Caption
					}
					if img.OCRText != "" {
						imgData["ocr_text"] = img.OCRText
					}
					if len(imgData) > 0 {
						imageList = append(imageList, imgData)
					}
				}
				if len(imageList) > 0 {
					chunkData["images"] = imageList
				}
			}
		}

		formattedChunks = append(formattedChunks, chunkData)
	}

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"knowledge_id":    knowledgeID,
			"knowledge_title": knowledgeTitle,
			"total_chunks":    totalChunks,
			"fetched_chunks":  fetched,
			"page":            pagination.Page,
			"page_size":       pagination.PageSize,
			"chunks":          formattedChunks,
		},
	}, nil
}

// lookupKnowledgeTitle looks up the title of a knowledge document
func (t *ListKnowledgeChunksTool) lookupKnowledgeTitle(ctx context.Context, knowledgeID string) string {
	if t.knowledgeService == nil {
		return ""
	}
	knowledge, err := t.knowledgeService.GetKnowledgeByID(ctx, knowledgeID)
	if err != nil || knowledge == nil {
		return ""
	}
	return strings.TrimSpace(knowledge.Title)
}

// buildOutput builds the output for the list knowledge chunks tool
func (t *ListKnowledgeChunksTool) buildOutput(
	knowledgeID string,
	knowledgeTitle string,
	total int64,
	fetched int,
	chunks []*types.Chunk,
) string {
	builder := &strings.Builder{}
	builder.WriteString("=== 지식 문서 청크 ===\n\n")

	if knowledgeTitle != "" {
		fmt.Fprintf(builder, "문서: %s (%s)\n", knowledgeTitle, knowledgeID)
	} else {
		fmt.Fprintf(builder, "문서 ID: %s\n", knowledgeID)
	}
	fmt.Fprintf(builder, "총 청크 수: %d\n", total)

	if fetched == 0 {
		builder.WriteString("청크를 찾을 수 없습니다. 문서 파싱이 완료되었는지 확인해 주세요.\n")
		if total > 0 {
			builder.WriteString("문서는 있지만 현재 페이지 데이터가 비어 있습니다. 페이지네이션 파라미터를 확인해 주세요.\n")
		}
		return builder.String()
	}
	fmt.Fprintf(
		builder,
		"이번 조회: %d 개, 검색 범위: %d - %d\n\n",
		fetched,
		chunks[0].ChunkIndex,
		chunks[len(chunks)-1].ChunkIndex,
	)

	builder.WriteString("=== 청크 내용 미리보기 ===\n\n")
	for idx, c := range chunks {
		fmt.Fprintf(builder, "Chunk #%d (Index %d)\n", idx+1, c.ChunkIndex+1)
		fmt.Fprintf(builder, "  chunk_id: %s\n", c.ID)
		fmt.Fprintf(builder, "  유형: %s\n", c.ChunkType)
		fmt.Fprintf(builder, "  내용: %s\n", summarizeContent(c.Content))

		if c.ImageInfo != "" {
			var imageInfos []types.ImageInfo
			if err := json.Unmarshal([]byte(c.ImageInfo), &imageInfos); err == nil && len(imageInfos) > 0 {
				fmt.Fprintf(builder, "  연관 이미지 (%d):\n", len(imageInfos))
				for imgIdx, img := range imageInfos {
					fmt.Fprintf(builder, "    이미지 %d:\n", imgIdx+1)
					if img.URL != "" {
						fmt.Fprintf(builder, "      URL: %s\n", img.URL)
					}
					if img.Caption != "" {
						fmt.Fprintf(builder, "      설명: %s\n", img.Caption)
					}
					if img.OCRText != "" {
						fmt.Fprintf(builder, "      OCR 텍스트: %s\n", img.OCRText)
					}
				}
			}
		}
		builder.WriteString("\n")
	}

	if int64(fetched) < total {
		builder.WriteString("힌트: 문서에 더 많은 청크가 있습니다. offset을 조정하거나 여러 번 호출하여 전체 내용을 가져오세요.\n")
	}

	return builder.String()
}

// summarizeContent summarizes the content of a chunk
func summarizeContent(content string) string {
	cleaned := strings.TrimSpace(content)
	if cleaned == "" {
		return "(빈 내용)"
	}

	return strings.TrimSpace(string(cleaned))
}
