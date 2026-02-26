# 모델 관리 API

[목차로 돌아가기](./README.md)

| 메서드 | 경로          | 설명              |
| ------ | ------------- | ----------------- |
| POST   | `/models`     | 모델 생성         |
| GET    | `/models`     | 모델 목록 조회    |
| GET    | `/models/:id` | 모델 상세 조회    |
| PUT    | `/models/:id` | 모델 업데이트     |
| DELETE | `/models/:id` | 모델 삭제         |

## POST `/models` - 모델 생성

### 대화 모델 생성 (KnowledgeQA)

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "qwen3:8b",
    "type": "KnowledgeQA",
    "source": "local",
    "description": "LLM Model for Knowledge QA",
    "parameters": {
        "base_url": "",
        "api_key": ""
    },
    "is_default": false
}'
```

### 임베딩 모델 생성 (Embedding)

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "nomic-embed-text:latest",
    "type": "Embedding",
    "source": "local",
    "description": "Embedding Model",
    "parameters": {
        "base_url": "",
        "api_key": "",
        "embedding_parameters": {
            "dimension": 768,
            "truncate_prompt_tokens": 0
        }
    },
    "is_default": false
}'
```

### 리랭크 모델 생성 (Rerank)

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "linux6200/bge-reranker-v2-m3:latest",
    "type": "Rerank",
    "source": "local",
    "description": "Rerank Model for Knowledge QA",
    "parameters": {
        "base_url": "",
        "api_key": ""
    },
    "is_default": false
}'
```

**응답**:

```json
{
    "data": {
        "id": "09c5a1d6-ee8b-4657-9a17-d3dcbd5c70cb",
        "tenant_id": 1,
        "name": "nomic-embed-text:latest3",
        "type": "Embedding",
        "source": "local",
        "description": "Embedding Model",
        "parameters": {
            "base_url": "",
            "api_key": "",
            "embedding_parameters": {
                "dimension": 768,
                "truncate_prompt_tokens": 0
            }
        },
        "is_default": false,
        "status": "downloading",
        "created_at": "2025-08-12T10:39:01.454591766+08:00",
        "updated_at": "2025-08-12T10:39:01.454591766+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## GET `/models` - 모델 목록 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "data": [
        {
            "id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "tenant_id": 1,
            "name": "nomic-embed-text:latest",
            "type": "Embedding",
            "source": "local",
            "description": "Embedding Model",
            "parameters": {
                "base_url": "",
                "api_key": "",
                "embedding_parameters": {
                    "dimension": 768,
                    "truncate_prompt_tokens": 0
                }
            },
            "is_default": true,
            "status": "active",
            "created_at": "2025-08-11T20:10:41.813832+08:00",
            "updated_at": "2025-08-11T20:10:41.822354+08:00",
            "deleted_at": null
        },
        {
            "id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "tenant_id": 1,
            "name": "qwen3:8b",
            "type": "KnowledgeQA",
            "source": "local",
            "description": "LLM Model for Knowledge QA",
            "parameters": {
                "base_url": "",
                "api_key": "",
                "embedding_parameters": {
                    "dimension": 0,
                    "truncate_prompt_tokens": 0
                }
            },
            "is_default": true,
            "status": "active",
            "created_at": "2025-08-11T20:10:41.811761+08:00",
            "updated_at": "2025-08-11T20:10:41.825381+08:00",
            "deleted_at": null
        }
    ],
    "success": true
}
```

## GET `/models/:id` - 모델 상세 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/models/dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "data": {
        "id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "tenant_id": 1,
        "name": "nomic-embed-text:latest",
        "type": "Embedding",
        "source": "local",
        "description": "Embedding Model",
        "parameters": {
            "base_url": "",
            "api_key": "",
            "embedding_parameters": {
                "dimension": 768,
                "truncate_prompt_tokens": 0
            }
        },
        "is_default": true,
        "status": "active",
        "created_at": "2025-08-11T20:10:41.813832+08:00",
        "updated_at": "2025-08-11T20:10:41.822354+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## PUT `/models/:id` - 모델 업데이트

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/models/8fdc464d-8eaa-44d4-a85b-094b28af5330' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "linux6200/bge-reranker-v2-m3:latest",
    "description": "Rerank Model for Knowledge QA new",
    "parameters": {
        "base_url": "",
        "api_key": ""
    },
    "is_default": false
}'
```

**응답**:

```json
{
    "data": {
        "id": "8fdc464d-8eaa-44d4-a85b-094b28af5330",
        "tenant_id": 1,
        "name": "linux6200/bge-reranker-v2-m3:latest",
        "type": "Rerank",
        "source": "local",
        "description": "Rerank Model for Knowledge QA new",
        "parameters": {
            "base_url": "",
            "api_key": "",
            "embedding_parameters": {
                "dimension": 0,
                "truncate_prompt_tokens": 0
            }
        },
        "is_default": false,
        "status": "active",
        "created_at": "2025-08-12T10:57:39.512681+08:00",
        "updated_at": "2025-08-12T11:00:27.271678+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## DELETE `/models/:id` - 모델 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/models/8fdc464d-8eaa-44d4-a85b-094b28af5330' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "message": "Model deleted",
    "success": true
}
```
