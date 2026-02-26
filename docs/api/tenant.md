# 테넌트 관리 API

[목차로 돌아가기](./README.md)

| 메서드 | 경로           | 설명                    |
| ------ | -------------- | ----------------------- |
| POST   | `/tenants`     | 새 테넌트 생성          |
| GET    | `/tenants/:id` | 특정 테넌트 정보 조회   |
| PUT    | `/tenants/:id` | 테넌트 정보 업데이트    |
| DELETE | `/tenants/:id` | 테넌트 삭제             |
| GET    | `/tenants`     | 테넌트 목록 조회        |

## POST `/tenants` - 새 테넌트 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants' \
--header 'Content-Type: application/json' \
--data '{
    "name": "weknora",
    "description": "weknora tenants",
    "business": "wechat",
    "retriever_engines": {
        "engines": [
            {
                "retriever_type": "keywords",
                "retriever_engine_type": "postgres"
            },
            {
                "retriever_type": "vector",
                "retriever_engine_type": "postgres"
            }
        ]
    }
}'
```

**응답**:

```json
{
    "data": {
        "id": 10000,
        "name": "weknora",
        "description": "weknora tenants",
        "api_key": "sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.396980093+08:00",
        "updated_at": "2025-08-11T20:37:28.396980301+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## GET `/tenants/:id` - 특정 테넌트 정보 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/10000' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG'
```

**응답**:

```json
{
    "data": {
        "id": 10000,
        "name": "weknora",
        "description": "weknora tenants",
        "api_key": "sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.39698+08:00",
        "updated_at": "2025-08-11T20:37:28.405693+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## PUT `/tenants/:id` - 테넌트 정보 업데이트

참고: API Key가 변경됩니다.

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/tenants/10000' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-KREi84yPtahKxMtIMOW-Cxx2dxb9xROpUuDSpi3vbiC1QVDe' \
--data '{
    "name": "weknora new",
    "description": "weknora tenants new",
    "status": "active",
    "retriever_engines": {
        "engines": [
            {
                "retriever_engine_type": "postgres",
                "retriever_type": "keywords"
            },
            {
                "retriever_engine_type": "postgres",
                "retriever_type": "vector"
            }
        ]
    },
    "business": "wechat",
    "storage_quota": 10737418240
}'
```

**응답**:

```json
{
    "data": {
        "id": 10000,
        "name": "weknora new",
        "description": "weknora tenants new",
        "api_key": "sk-IKtd9JGV4-aPGQ6RiL8YJu9Vzb3-ae4lgFkjFJZmhvUn2mLu",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "2025-08-11T20:49:02.13421034+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## DELETE `/tenants/:id` - 테넌트 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/tenants/10000' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-IKtd9JGV4-aPGQ6RiL8YJu9Vzb3-ae4lgFkjFJZmhvUn2mLu'
```

**응답**:

```json
{
    "message": "Tenant deleted successfully",
    "success": true
}
```

## GET `/tenants` - 테넌트 목록 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-An7_t_izCKFIJ4iht9Xjcjnj_MC48ILvwezEDki9ScfIa7KA'
```

**응답**:

```json
{
    "data": {
        "items": [
            {
                "id": 10002,
                "name": "weknora",
                "description": "weknora tenants",
                "api_key": "sk-An7_t_izCKFIJ4iht9Xjcjnj_MC48ILvwezEDki9ScfIa7KA",
                "status": "active",
                "retriever_engines": {
                    "engines": [
                        {
                            "retriever_engine_type": "postgres",
                            "retriever_type": "keywords"
                        },
                        {
                            "retriever_engine_type": "postgres",
                            "retriever_type": "vector"
                        }
                    ]
                },
                "business": "wechat",
                "storage_quota": 10737418240,
                "storage_used": 0,
                "created_at": "2025-08-11T20:52:58.05679+08:00",
                "updated_at": "2025-08-11T20:52:58.060495+08:00",
                "deleted_at": null
            }
        ]
    },
    "success": true
}
```
