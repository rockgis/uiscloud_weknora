package tools

// AvailableTool defines a simple tool metadata used by settings APIs.
type AvailableTool struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableToolDefinitions returns the list of tools exposed to the UI.
// Keep this in sync with registered tools in this package.
func AvailableToolDefinitions() []AvailableTool {
	return []AvailableTool{
		{Name: "thinking", Label: "사고", Description: "동적이고 반성적인 문제 해결 사고 도구"},
		{Name: "todo_write", Label: "계획 수립", Description: "구조화된 연구 계획 생성"},
		{Name: "grep_chunks", Label: "키워드 검색", Description: "특정 키워드가 포함된 문서 및 청크 신속 탐색"},
		{Name: "knowledge_search", Label: "의미 검색", Description: "질문을 이해하고 의미적으로 관련된 콘텐츠 탐색"},
		{Name: "list_knowledge_chunks", Label: "문서 청크 보기", Description: "문서의 전체 청크 내용 조회"},
		{Name: "query_knowledge_graph", Label: "지식 그래프 조회", Description: "지식 그래프에서 관계 조회"},
		{Name: "get_document_info", Label: "문서 정보 조회", Description: "문서 메타데이터 확인"},
		{Name: "database_query", Label: "데이터베이스 조회", Description: "데이터베이스에서 정보 조회"},
	}
}

// DefaultAllowedTools returns the default allowed tools list.
func DefaultAllowedTools() []string {
	return []string{
		"thinking",
		"todo_write",
		"knowledge_search",
		"grep_chunks",
		"list_knowledge_chunks",
		"query_knowledge_graph",
		"get_document_info",
		"database_query",
	}
}
