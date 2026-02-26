# FAQ 관리 API

[목차로 돌아가기](./README.md)

| 방법   | 경로                                        | 설명                     |
| ------ | ------------------------------------------- | ------------------------ |
| GET    | `/knowledge-bases/:id/faq/entries`          | FAQ 항목 목록 조회          |
| POST   | `/knowledge-bases/:id/faq/entries`          | FAQ 항목 일괄 가져오기          |
| POST   | `/knowledge-bases/:id/faq/entry`            | 단일 FAQ 항목 생성          |
| PUT    | `/knowledge-bases/:id/faq/entries/:entry_id`| 단일 FAQ 항목 수정          |
| PUT    | `/knowledge-bases/:id/faq/entries/status`   | FAQ 활성화 상태 일괄 수정      |
| PUT    | `/knowledge-bases/:id/faq/entries/tags`     | FAQ 태그 일괄 수정          |
| DELETE | `/knowledge-bases/:id/faq/entries`          | FAQ 항목 일괄 삭제          |
| POST   | `/knowledge-bases/:id/faq/search`           | FAQ 하이브리드 검색              |

## GET `/knowledge-bases/:id/faq/entries` - FAQ 항목 목록 조회

**쿼리 파라미터**:
- `page`: 페이지 번호 (기본값 1)
- `page_size`: 페이지당 항목 수 (기본값 20)
- `tag_id`: 태그 ID로 필터링 (선택)
- `keyword`: 키워드 검색 (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries?page=1&page_size=10' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": {
        "total": 100,
        "page": 1,
        "page_size": 10,
        "data": [
            {
                "id": "faq-00000001",
                "chunk_id": "chunk-00000001",
                "knowledge_id": "knowledge-00000001",
                "knowledge_base_id": "kb-00000001",
                "tag_id": "tag-00000001",
                "is_enabled": true,
                "standard_question": "비밀번호를 어떻게 재설정하나요?",
                "similar_questions": ["비밀번호를 잊어버렸어요", "비밀번호 찾기"],
                "negative_questions": ["사용자 이름은 어떻게 변경하나요"],
                "answers": ["로그인 페이지의 '비밀번호 찾기' 링크를 클릭하여 비밀번호를 재설정할 수 있습니다."],
                "index_mode": "hybrid",
                "chunk_type": "faq",
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00"
            }
        ]
    },
    "success": true
}
```

## POST `/knowledge-bases/:id/faq/entries` - FAQ 항목 일괄 가져오기

**요청 파라미터**:
- `mode`: 가져오기 모드, `append`(추가) 또는 `replace`(교체)
- `entries`: FAQ 항목 배열
- `knowledge_id`: 연관된 지식 ID (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "mode": "append",
    "entries": [
        {
            "standard_question": "고객 서비스에 어떻게 연락하나요?",
            "similar_questions": ["고객 서비스 전화번호", "온라인 고객 서비스"],
            "answers": ["400-xxx-xxxx로 전화하여 고객 서비스에 연락하실 수 있습니다."],
            "tag_id": "tag-00000001"
        },
        {
            "standard_question": "환불 정책이 어떻게 되나요?",
            "answers": ["7일 이내 무조건 환불 서비스를 제공합니다."]
        }
    ]
}'
```

**응답**:

```json
{
    "data": {
        "task_id": "task-00000001"
    },
    "success": true
}
```

참고: 일괄 가져오기는 비동기 작업이며, 진행 상황 추적을 위한 작업 ID를 반환합니다.

## POST `/knowledge-bases/:id/faq/entry` - 단일 FAQ 항목 생성

단일 FAQ 항목을 동기적으로 생성합니다. 한 건씩 입력하는 시나리오에 적합합니다. 표준 질문 및 유사 질문이 기존 FAQ와 중복되는지 자동으로 확인합니다.

**요청 파라미터**:
- `standard_question`: 표준 질문 (필수)
- `similar_questions`: 유사 질문 배열 (선택)
- `negative_questions`: 부정 예시 질문 배열 (선택)
- `answers`: 답변 배열 (필수)
- `tag_id`: 태그 ID (선택)
- `is_enabled`: 활성화 여부 (선택, 기본값 true)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entry' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "고객 서비스에 어떻게 연락하나요?",
    "similar_questions": ["고객 서비스 전화번호", "온라인 고객 서비스"],
    "answers": ["400-xxx-xxxx로 전화하여 고객 서비스에 연락하실 수 있습니다."],
    "tag_id": "tag-00000001",
    "is_enabled": true
}'
```

**응답**:

```json
{
    "data": {
        "id": "faq-00000001",
        "chunk_id": "chunk-00000001",
        "knowledge_id": "knowledge-00000001",
        "knowledge_base_id": "kb-00000001",
        "tag_id": "tag-00000001",
        "is_enabled": true,
        "standard_question": "고객 서비스에 어떻게 연락하나요?",
        "similar_questions": ["고객 서비스 전화번호", "온라인 고객 서비스"],
        "negative_questions": [],
        "answers": ["400-xxx-xxxx로 전화하여 고객 서비스에 연락하실 수 있습니다."],
        "index_mode": "hybrid",
        "chunk_type": "faq",
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

**오류 응답** (표준 질문 또는 유사 질문이 중복된 경우):

```json
{
    "success": false,
    "error": {
        "code": "BAD_REQUEST",
        "message": "표준 질문이 기존 FAQ와 중복됩니다"
    }
}
```

## PUT `/knowledge-bases/:id/faq/entries/:entry_id` - 단일 FAQ 항목 수정

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/faq-00000001' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "계정 비밀번호를 어떻게 재설정하나요?",
    "similar_questions": ["비밀번호를 잊어버렸어요", "비밀번호 찾기", "비밀번호 재설정"],
    "answers": ["다음 단계를 통해 비밀번호를 재설정할 수 있습니다: 1. 로그인 페이지의 \"비밀번호 찾기\" 클릭 2. 가입 이메일 주소 입력 3. 재설정 이메일 확인"],
    "is_enabled": true
}'
```

**응답**:

```json
{
    "success": true
}
```

## PUT `/knowledge-bases/:id/faq/entries/status` - FAQ 활성화 상태 일괄 수정

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/status' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "updates": {
        "faq-00000001": true,
        "faq-00000002": false,
        "faq-00000003": true
    }
}'
```

**응답**:

```json
{
    "success": true
}
```

## PUT `/knowledge-bases/:id/faq/entries/tags` - FAQ 태그 일괄 수정

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/tags' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "updates": {
        "faq-00000001": "tag-00000001",
        "faq-00000002": "tag-00000002",
        "faq-00000003": null
    }
}'
```

참고: `null`로 설정하면 태그 연결을 해제할 수 있습니다.

**응답**:

```json
{
    "success": true
}
```

## DELETE `/knowledge-bases/:id/faq/entries` - FAQ 항목 일괄 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "ids": ["faq-00000001", "faq-00000002"]
}'
```

**응답**:

```json
{
    "success": true
}
```

## POST `/knowledge-bases/:id/faq/search` - FAQ 하이브리드 검색

**요청 파라미터**:
- `query_text`: 검색 쿼리 텍스트
- `vector_threshold`: 벡터 유사도 임계값 (0-1)
- `match_count`: 반환할 결과 수 (최대 200)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/search' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query_text": "비밀번호 재설정 방법",
    "vector_threshold": 0.5,
    "match_count": 10
}'
```

**응답**:

```json
{
    "data": [
        {
            "id": "faq-00000001",
            "chunk_id": "chunk-00000001",
            "knowledge_id": "knowledge-00000001",
            "knowledge_base_id": "kb-00000001",
            "tag_id": "tag-00000001",
            "is_enabled": true,
            "standard_question": "비밀번호를 어떻게 재설정하나요?",
            "similar_questions": ["비밀번호를 잊어버렸어요", "비밀번호 찾기"],
            "answers": ["로그인 페이지의 '비밀번호 찾기' 링크를 클릭하여 비밀번호를 재설정할 수 있습니다."],
            "chunk_type": "faq",
            "score": 0.95,
            "match_type": "vector",
            "created_at": "2025-08-12T10:00:00+08:00",
            "updated_at": "2025-08-12T10:00:00+08:00"
        }
    ],
    "success": true
}
```
