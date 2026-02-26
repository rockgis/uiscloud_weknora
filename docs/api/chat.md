# 채팅 기능 API

[목차로 돌아가기](./README.md)

| 방법 | 경로                          | 설명                     |
| ---- | ----------------------------- | ------------------------ |
| POST | `/knowledge-chat/:session_id` | 지식베이스 기반 질의응답         |
| POST | `/agent-chat/:session_id`     | Agent 기반 지능형 질의응답    |
| POST | `/knowledge-search`           | 지식베이스 기반 지식 검색     |

## POST `/knowledge-chat/:session_id` - 지식베이스 기반 질의응답

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-chat/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query": "혜성 꼬리의 형태"
}'
```

**응답 형식**:
서버 전송 이벤트 스트림 (Server-Sent Events, Content-Type: text/event-stream)

**응답**:

```
event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"references","content":"","done":false,"knowledge_references":[{"id":"c8347bef-127f-4a22-b962-edf5a75386ec","content":"혜성 xxx.","knowledge_id":"a6790b93-4700-4676-bd48-0d4804e1456b","chunk_index":0,"knowledge_title":"comet.txt","start_at":0,"end_at":2760,"seq":0,"score":4.038836479187012,"match_type":3,"sub_chunk_id":["688821f0-40bf-428e-8cb6-541531ebeb76","c1e9903e-2b4d-4281-be15-0149288d45c2","7d955251-3f79-4fd5-a6aa-02f81e044091"],"metadata":{},"chunk_type":"text","parent_chunk_id":"","image_info":"","knowledge_filename":"comet.txt","knowledge_source":""},{"id":"fa3aadee-cadb-4a84-9941-c839edc3e626","content":"# 문서 이름\ncomet.txt\n\n# 요약\n혜성은 얼음과 먼지로 이루어진 태양계의 소천체로, 태양에 가까워지면 기체를 방출하여 혜성핵과 혜성 꼬리를 형성합니다. 궤도 주기는 다양하며, 카이퍼 벨트와 오르트 구름에서 기원합니다. 혜성과 소행성의 구분은 점차 모호해지고 있으며, 일부 혜성은 휘발성 물질을 잃어 소행성과 유사해집니다. 현재까지 알려진 혜성 수는 많고, 외계 혜성도 존재합니다. 혜성은 고대에 흉조로 여겨졌으나, 현대 연구는 그 복잡한 구조와 기원을 밝혀냈습니다.","knowledge_id":"a6790b93-4700-4676-bd48-0d4804e1456b","chunk_index":6,"knowledge_title":"comet.txt","start_at":0,"end_at":0,"seq":6,"score":0.6131043121858466,"match_type":3,"sub_chunk_id":null,"metadata":{},"chunk_type":"summary","parent_chunk_id":"c8347bef-127f-4a22-b962-edf5a75386ec","image_info":"","knowledge_filename":"comet.txt","knowledge_source":""}]}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"으로 나타납니다","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"구조","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":".","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"","done":true,"knowledge_references":null}
```

## POST `/agent-chat/:session_id` - Agent 기반 지능형 질의응답

Agent 모드는 도구 호출, 웹 검색, 다중 지식베이스 검색 등 더욱 지능적인 질의응답 기능을 지원합니다.

**요청 파라미터**:
- `query`: 쿼리 텍스트 (필수)
- `knowledge_base_ids`: 지식베이스 ID 배열, 이번 쿼리에 사용할 지식베이스를 동적으로 지정 (선택)
- `agent_enabled`: Agent 모드 활성화 여부 (선택, 기본값 false)
- `web_search_enabled`: 웹 검색 활성화 여부 (선택, 기본값 false)
- `summary_model_id`: 세션 기본 요약 모델 ID 재정의 (선택)
- `mcp_service_ids`: MCP 서비스 허용 목록 (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/agent-chat/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query": "오늘 날씨를 알려줘",
    "agent_enabled": true,
    "web_search_enabled": true,
    "knowledge_base_ids": ["kb-00000001"]
}'
```

**응답 형식**:
서버 전송 이벤트 스트림 (Server-Sent Events, Content-Type: text/event-stream)

**응답 유형 설명**:

| response_type | 설명 |
|---------------|------|
| `thinking` | Agent 사고 과정 |
| `tool_call` | 도구 호출 정보 |
| `tool_result` | 도구 호출 결과 |
| `references` | 지식베이스 검색 참조 |
| `answer` | 최종 응답 내용 |
| `reflection` | Agent 반성 내용 |
| `error` | 오류 정보 |

**응답 예시**:

```
event: message
data: {"id":"agent-001","response_type":"thinking","content":"사용자가 날씨를 조회하고 싶어합니다. 웹 검색 도구를 사용해야겠습니다...","done":false,"knowledge_references":null}

event: message
data: {"id":"agent-001","response_type":"tool_call","content":"","done":false,"knowledge_references":null,"data":{"tool_name":"web_search","arguments":{"query":"오늘 날씨"}}}

event: message
data: {"id":"agent-001","response_type":"tool_result","content":"검색 결과: 오늘 맑음, 기온 25°C...","done":false,"knowledge_references":null}

event: message
data: {"id":"agent-001","response_type":"answer","content":"조회 결과에 따르면 오늘 날씨는 맑고 기온은 약 25°C입니다.","done":false,"knowledge_references":null}

event: message
data: {"id":"agent-001","response_type":"answer","content":"","done":true,"knowledge_references":null}
```
