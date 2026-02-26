package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// QueryKnowledgeGraphTool queries the knowledge graph for entities and relationships
type QueryKnowledgeGraphTool struct {
	BaseTool
	knowledgeService interfaces.KnowledgeBaseService
}

// NewQueryKnowledgeGraphTool creates a new query knowledge graph tool
func NewQueryKnowledgeGraphTool(knowledgeService interfaces.KnowledgeBaseService) *QueryKnowledgeGraphTool {
	description := `Query knowledge graph to explore entity relationships and knowledge networks.

## Core Function
Explores relationships between entities in knowledge bases that have graph extraction configured.

## When to Use
✅ **Use for**:
- Understanding relationships between entities (e.g., "relationship between Docker and Kubernetes")
- Exploring knowledge networks and concept associations
- Finding related information about specific entities
- Understanding technical architecture and system relationships

❌ **Don't use for**:
- General text search → use knowledge_search
- Knowledge base without graph extraction configured
- Need exact document content → use knowledge_search

## Parameters
- **knowledge_base_ids** (required): Array of knowledge base IDs (1-10). Only KBs with graph extraction configured will be effective.
- **query** (required): Query content - can be entity name, relationship query, or concept search.

## Graph Configuration
Knowledge graph must be pre-configured in knowledge bases:
- **Entity types** (Nodes): e.g., "Technology", "Tool", "Concept"
- **Relationship types** (Relations): e.g., "depends_on", "uses", "contains"

If KB is not configured with graph, tool will return regular search results.

## Workflow
1. **Relationship exploration**: query_knowledge_graph → list_knowledge_chunks (for detailed content)
2. **Network analysis**: query_knowledge_graph → knowledge_search (for comprehensive understanding)
3. **Topic research**: knowledge_search → query_knowledge_graph (for deep entity relationships)

## Notes
- Results indicate graph configuration status
- Cross-KB results are automatically deduplicated
- Results are sorted by relevance`

	return &QueryKnowledgeGraphTool{
		BaseTool:         NewBaseTool("query_knowledge_graph", description),
		knowledgeService: knowledgeService,
	}
}

// Parameters returns the JSON schema for the tool's parameters
func (t *QueryKnowledgeGraphTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"knowledge_base_ids": map[string]interface{}{
				"type":        "array",
				"description": "Array of knowledge base IDs to query",
				"items": map[string]interface{}{
					"type": "string",
				},
				"minItems": 1,
				"maxItems": 10,
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "검색 내용 (엔티티 이름 또는 검색 텍스트)",
			},
		},
		"required": []string{"knowledge_base_ids", "query"},
	}
}

// Execute performs the knowledge graph query with concurrent KB processing
func (t *QueryKnowledgeGraphTool) Execute(ctx context.Context, args map[string]interface{}) (*types.ToolResult, error) {
	// Extract knowledge_base_ids array
	kbIDsRaw, ok := args["knowledge_base_ids"].([]interface{})
	if !ok || len(kbIDsRaw) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "knowledge_base_ids is required and must be a non-empty array",
		}, fmt.Errorf("knowledge_base_ids is required")
	}

	// Convert to string slice
	var kbIDs []string
	for _, id := range kbIDsRaw {
		if idStr, ok := id.(string); ok && idStr != "" {
			kbIDs = append(kbIDs, idStr)
		}
	}

	if len(kbIDs) == 0 {
		return &types.ToolResult{
			Success: false,
			Error:   "knowledge_base_ids must contain at least one valid KB ID",
		}, fmt.Errorf("no valid KB IDs provided")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return &types.ToolResult{
			Success: false,
			Error:   "query is required",
		}, fmt.Errorf("invalid query")
	}

	// Concurrently query all knowledge bases
	type graphQueryResult struct {
		kbID    string
		kb      *types.KnowledgeBase
		results []*types.SearchResult
		err     error
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	kbResults := make(map[string]*graphQueryResult)

	searchParams := types.SearchParams{
		QueryText:  query,
		MatchCount: 10,
	}

	for _, kbID := range kbIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Get knowledge base to check graph configuration
			kb, err := t.knowledgeService.GetKnowledgeBaseByID(ctx, id)
			if err != nil {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, err: fmt.Errorf("지식베이스 조회 실패: %v", err)}
				mu.Unlock()
				return
			}

			// Check if graph extraction is enabled
			if kb.ExtractConfig == nil || (len(kb.ExtractConfig.Nodes) == 0 && len(kb.ExtractConfig.Relations) == 0) {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, err: fmt.Errorf("지식 그래프 추출이 설정되지 않았습니다")}
				mu.Unlock()
				return
			}

			// Query graph
			results, err := t.knowledgeService.HybridSearch(ctx, id, searchParams)
			if err != nil {
				mu.Lock()
				kbResults[id] = &graphQueryResult{kbID: id, kb: kb, err: fmt.Errorf("검색 실패: %v", err)}
				mu.Unlock()
				return
			}

			mu.Lock()
			kbResults[id] = &graphQueryResult{kbID: id, kb: kb, results: results}
			mu.Unlock()
		}(kbID)
	}

	wg.Wait()

	// Collect and deduplicate results
	seenChunks := make(map[string]*types.SearchResult)
	var errors []string
	graphConfigs := make(map[string]map[string]interface{})
	kbCounts := make(map[string]int)

	for _, kbID := range kbIDs {
		result := kbResults[kbID]
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("KB %s: %v", kbID, result.err))
			continue
		}

		if result.kb != nil && result.kb.ExtractConfig != nil {
			graphConfigs[kbID] = map[string]interface{}{
				"nodes":     result.kb.ExtractConfig.Nodes,
				"relations": result.kb.ExtractConfig.Relations,
			}
		}

		kbCounts[kbID] = len(result.results)
		for _, r := range result.results {
			if _, seen := seenChunks[r.ID]; !seen {
				seenChunks[r.ID] = r
			}
		}
	}

	// Convert map to slice and sort by score
	allResults := make([]*types.SearchResult, 0, len(seenChunks))
	for _, result := range seenChunks {
		allResults = append(allResults, result)
	}

	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	if len(allResults) == 0 {
		return &types.ToolResult{
			Success: true,
			Output:  "관련 그래프 정보를 찾을 수 없습니다.",
			Data: map[string]interface{}{
				"knowledge_base_ids": kbIDs,
				"query":              query,
				"results":            []interface{}{},
				"graph_configs":      graphConfigs,
				"errors":             errors,
			},
		}, nil
	}

	// Format output with enhanced graph information
	output := "=== 지식 그래프 검색 ===\n\n"
	output += fmt.Sprintf("📊 검색어: %s\n", query)
	output += fmt.Sprintf("🎯 대상 지식베이스: %v\n", kbIDs)
	output += fmt.Sprintf("✓ 관련 결과 %d 개 발견 (중복 제거)\n\n", len(allResults))

	if len(errors) > 0 {
		output += "=== ⚠️ 일부 실패 ===\n"
		for _, errMsg := range errors {
			output += fmt.Sprintf("  - %s\n", errMsg)
		}
		output += "\n"
	}

	// Display graph configuration status
	hasGraphConfig := false
	output += "=== 📈 그래프 설정 상태 ===\n\n"
	for kbID, config := range graphConfigs {
		hasGraphConfig = true
		output += fmt.Sprintf("지식베이스 【%s】:\n", kbID)

		nodes, _ := config["nodes"].([]interface{})
		relations, _ := config["relations"].([]interface{})

		if len(nodes) > 0 {
			output += fmt.Sprintf("  ✓ 엔티티 유형 (%d): ", len(nodes))
			nodeNames := make([]string, 0, len(nodes))
			for _, n := range nodes {
				if nodeMap, ok := n.(map[string]interface{}); ok {
					if name, ok := nodeMap["name"].(string); ok {
						nodeNames = append(nodeNames, name)
					}
				}
			}
			output += fmt.Sprintf("%v\n", nodeNames)
		} else {
			output += "  ⚠️ 엔티티 유형 미설정\n"
		}

		if len(relations) > 0 {
			output += fmt.Sprintf("  ✓ 관계 유형 (%d): ", len(relations))
			relNames := make([]string, 0, len(relations))
			for _, r := range relations {
				if relMap, ok := r.(map[string]interface{}); ok {
					if name, ok := relMap["name"].(string); ok {
						relNames = append(relNames, name)
					}
				}
			}
			output += fmt.Sprintf("%v\n", relNames)
		} else {
			output += "  ⚠️ 관계 유형 미설정\n"
		}
		output += "\n"
	}

	if !hasGraphConfig {
		output += "⚠️ 검색된 지식베이스에 그래프 추출이 설정되지 않았습니다\n"
		output += "💡 힌트: 지식베이스 설정에서 엔티티 및 관계 유형을 설정해야 합니다\n\n"
	}

	// Display result counts by KB
	if len(kbCounts) > 0 {
		output += "=== 📚 지식베이스 커버리지 ===\n"
		for kbID, count := range kbCounts {
			output += fmt.Sprintf("  - %s: %d 개 결과\n", kbID, count)
		}
		output += "\n"
	}

	// Display search results
	output += "=== 🔍 검색 결과 ===\n\n"
	if !hasGraphConfig {
		output += "💡 현재 관련 문서 조각을 반환합니다 (지식베이스에 그래프 미설정)\n\n"
	} else {
		output += "💡 그래프 설정 기반 관련 내용 검색\n\n"
	}

	formattedResults := make([]map[string]interface{}, 0, len(allResults))
	currentKB := ""

	for i, result := range allResults {
		// Group by knowledge base
		if result.KnowledgeID != currentKB {
			currentKB = result.KnowledgeID
			if i > 0 {
				output += "\n"
			}
			output += fmt.Sprintf("【출처 문서: %s】\n\n", result.KnowledgeTitle)
		}

		relevanceLevel := GetRelevanceLevel(result.Score)

		output += fmt.Sprintf("결과 #%d:\n", i+1)
		output += fmt.Sprintf("  📍 관련도: %.2f (%s)\n", result.Score, relevanceLevel)
		output += fmt.Sprintf("  🔗 매칭 방식: %s\n", FormatMatchType(result.MatchType))
		output += fmt.Sprintf("  📄 내용: %s\n", result.Content)
		output += fmt.Sprintf("  🆔 chunk_id: %s\n\n", result.ID)

		formattedResults = append(formattedResults, map[string]interface{}{
			"result_index":    i + 1,
			"chunk_id":        result.ID,
			"content":         result.Content,
			"score":           result.Score,
			"relevance_level": relevanceLevel,
			"knowledge_id":    result.KnowledgeID,
			"knowledge_title": result.KnowledgeTitle,
			"match_type":      FormatMatchType(result.MatchType),
		})
	}

	output += "=== 💡 사용 힌트 ===\n"
	output += "- ✓ 결과는 지식베이스 간 중복 제거 및 관련도 순으로 정렬되었습니다\n"
	output += "- ✓ get_chunk_detail을 사용하여 전체 내용을 가져오세요\n"
	output += "- ✓ list_knowledge_chunks를 사용하여 컨텍스트를 탐색하세요\n"
	if !hasGraphConfig {
		output += "- ⚠️ 그래프 추출을 설정하면 더 정확한 엔티티 관계 결과를 얻을 수 있습니다\n"
	}
	output += "- ⏳ 완전한 그래프 쿼리 언어 (Cypher) 지원 개발 중\n"

	// Build structured graph data for frontend visualization
	graphData := buildGraphVisualizationData(allResults, graphConfigs)

	return &types.ToolResult{
		Success: true,
		Output:  output,
		Data: map[string]interface{}{
			"knowledge_base_ids": kbIDs,
			"query":              query,
			"results":            formattedResults,
			"count":              len(allResults),
			"kb_counts":          kbCounts,
			"graph_configs":      graphConfigs,
			"graph_data":         graphData,
			"has_graph_config":   hasGraphConfig,
			"errors":             errors,
			"display_type":       "graph_query_results",
		},
	}, nil
}

// buildGraphVisualizationData builds structured data for graph visualization
func buildGraphVisualizationData(
	results []*types.SearchResult,
	graphConfigs map[string]map[string]interface{},
) map[string]interface{} {
	// Build a simple graph structure for frontend visualization
	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	// Create nodes from results
	seenEntities := make(map[string]bool)
	for i, result := range results {
		if !seenEntities[result.ID] {
			nodes = append(nodes, map[string]interface{}{
				"id":       result.ID,
				"label":    fmt.Sprintf("Chunk %d", i+1),
				"content":  result.Content,
				"kb_id":    result.KnowledgeID,
				"kb_title": result.KnowledgeTitle,
				"score":    result.Score,
				"type":     "chunk",
			})
			seenEntities[result.ID] = true
		}
	}

	return map[string]interface{}{
		"nodes":       nodes,
		"edges":       edges,
		"total_nodes": len(nodes),
		"total_edges": len(edges),
	}
}
