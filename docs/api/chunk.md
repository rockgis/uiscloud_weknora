# 청크 관리 API

[목차로 돌아가기](./README.md)

| 방법   | 경로                        | 설명                     |
| ------ | --------------------------- | ------------------------ |
| GET    | `/chunks/:knowledge_id`     | 지식의 청크 목록 조회       |
| DELETE | `/chunks/:knowledge_id/:id` | 청크 삭제                 |
| DELETE | `/chunks/:knowledge_id`     | 지식의 모든 청크 삭제     |

## GET `/chunks/:knowledge_id?page=&page_size=` - 지식의 청크 목록 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5?page=1&page_size=1' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": [
        {
            "id": "df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7",
            "tenant_id": 1,
            "knowledge_id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
            "knowledge_base_id": "kb-00000001",
            "tag_id": "",
            "content": "혜성 xxxx",
            "chunk_index": 0,
            "is_enabled": true,
            "status": 2,
            "start_at": 0,
            "end_at": 964,
            "pre_chunk_id": "",
            "next_chunk_id": "",
            "chunk_type": "text",
            "parent_chunk_id": "",
            "relation_chunks": null,
            "indirect_relation_chunks": null,
            "metadata": null,
            "content_hash": "",
            "image_info": "",
            "created_at": "2025-08-12T11:52:36.168632+08:00",
            "updated_at": "2025-08-12T11:52:53.376871+08:00",
            "deleted_at": null
        }
    ],
    "page": 1,
    "page_size": 1,
    "success": true,
    "total": 5
}
```

## DELETE `/chunks/:knowledge_id/:id` - 청크 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "message": "Chunk deleted",
    "success": true
}
```

## DELETE `/chunks/:knowledge_id` - 지식의 모든 청크 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "message": "All chunks under knowledge deleted",
    "success": true
}
```
