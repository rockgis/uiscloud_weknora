/**
 * Tool Icons Utility
 * Maps tool names and match types to icons for better UI display
 */

// Tool name to icon mapping
export const toolIcons: Record<string, string> = {
    multi_kb_search: '🔍',
    knowledge_search: '📚',
    grep_chunks: '🔎',
    get_chunk_detail: '📄',
    list_knowledge_bases: '📂',
    list_knowledge_chunks: '🧩',
    get_document_info: 'ℹ️',
    query_knowledge_graph: '🕸️',
    think: '💭',
    todo_write: '📋',
};

// Match type to icon mapping
export const matchTypeIcons: Record<string, string> = {
    '벡터 매칭': '🎯',
    '키워드 매칭': '🔤',
    '인접 블록 매칭': '📌',
    '히스토리 매칭': '📜',
    '부모 블록 매칭': '⬆️',
    '관계 블록 매칭': '🔗',
    '그래프 매칭': '🕸️',
};

// Get icon for a tool name
export function getToolIcon(toolName: string): string {
    return toolIcons[toolName] || '🛠️';
}

// Get icon for a match type
export function getMatchTypeIcon(matchType: string): string {
    return matchTypeIcons[matchType] || '📍';
}

// Get tool display name (user-friendly)
export function getToolDisplayName(toolName: string): string {
    const displayNames: Record<string, string> = {
        multi_kb_search: '교차 지식베이스 검색',
        knowledge_search: '지식베이스 검색',
        grep_chunks: '텍스트 패턴 검색',
        get_chunk_detail: '청크 상세 조회',
        list_knowledge_chunks: '지식 청크 보기',
        list_knowledge_bases: '지식베이스 목록',
        get_document_info: '문서 정보 조회',
        query_knowledge_graph: '지식 그래프 조회',
        think: '심층 사고',
        todo_write: '계획 수립',
    };
    return displayNames[toolName] || toolName;
}

