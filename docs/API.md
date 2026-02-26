# WeKnora API 문서

## 목차

- [개요](#개요)
- [기본 정보](#기본-정보)
- [인증 방식](#인증-방식)
- [에러 처리](#에러-처리)
- [API 개요](#api-개요)
- [API 상세 설명](#api-상세-설명)
  - [테넌트 관리 API](#테넌트-관리api)
  - [지식베이스 관리 API](#지식베이스-관리api)
  - [지식 관리 API](#지식-관리api)
  - [모델 관리 API](#모델-관리api)
  - [청크 관리 API](#청크-관리api)
  - [태그 관리 API](#태그-관리api)
  - [FAQ 관리 API](#faq-관리api)
  - [세션 관리 API](#세션-관리api)
  - [채팅 기능 API](#채팅-기능api)
  - [메시지 관리 API](#메시지-관리api)
  - [평가 기능 API](#평가-기능api)

## 개요

WeKnora는 지식베이스 생성 및 관리, 지식 검색, 지식 기반 질의응답을 위한 RESTful API를 제공합니다. 본 문서는 이러한 API의 사용 방법을 상세히 설명합니다.

## 기본 정보

- **기본 URL**: `/api/v1`
- **응답 형식**: JSON
- **인증 방식**: API Key

## 인증 방식

모든 API 요청은 HTTP 요청 헤더에 `X-API-Key`를 포함하여 인증해야 합니다:

```
X-API-Key: your_api_key
```

문제 추적 및 디버깅을 용이하게 하기 위해, 각 요청의 HTTP 요청 헤더에 `X-Request-ID`를 추가하는 것을 권장합니다:

```
X-Request-ID: unique_request_id
```

### API Key 발급

웹 페이지에서 계정 등록을 완료한 후, 계정 정보 페이지에서 API Key를 발급받으세요.

API Key는 안전하게 보관하고 유출되지 않도록 주의하세요. API Key는 계정의 신원을 나타내며, 전체 API 접근 권한을 가집니다.

## 에러 처리

모든 API는 표준 HTTP 상태 코드를 사용하여 요청 상태를 나타내며, 통일된 에러 응답 형식을 반환합니다:

```json
{
  "success": false,
  "error": {
    "code": "에러 코드",
    "message": "에러 메시지",
    "details": "에러 상세 정보"
  }
}
```

## API 개요

WeKnora API는 기능에 따라 다음과 같이 분류됩니다:

1. **테넌트 관리**: 테넌트 계정 생성 및 관리
2. **지식베이스 관리**: 지식베이스 생성, 조회 및 관리
3. **지식 관리**: 지식 콘텐츠 업로드, 검색 및 관리
4. **모델 관리**: 다양한 AI 모델 구성 및 관리
5. **청크 관리**: 지식의 청크 콘텐츠 관리
6. **태그 관리**: 지식베이스의 태그 분류 관리
7. **FAQ 관리**: FAQ 질의응답 쌍 관리
8. **세션 관리**: 대화 세션 생성 및 관리
9. **채팅 기능**: 지식베이스 기반 질의응답
10. **메시지 관리**: 대화 메시지 조회 및 관리
11. **평가 기능**: 모델 성능 평가

## API 상세 설명

각 API의 상세 설명과 예제는 다음과 같습니다.

### 테넌트 관리API

| 방법   | 경로           | 설명                  |
| ------ | -------------- | --------------------- |
| POST   | `/tenants`     | 새 테넌트 생성        |
| GET    | `/tenants/:id` | 지정 테넌트 정보 조회 |
| PUT    | `/tenants/:id` | 테넌트 정보 업데이트  |
| DELETE | `/tenants/:id` | 테넌트 삭제           |
| GET    | `/tenants`     | 테넌트 목록 조회      |

#### POST `/tenants` - 새 테넌트 생성

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

#### GET `/tenants/:id` - 지정 테넌트 정보 조회

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

#### PUT `/tenants/:id` - 테넌트 정보 업데이트

주의: API Key가 변경됩니다.

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

#### DELETE `/tenants/:id` - 테넌트 삭제

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

#### GET `/tenants` - 테넌트 목록 조회

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

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 지식베이스 관리API

| 방법   | 경로                                 | 설명                         |
| ------ | ------------------------------------ | ---------------------------- |
| POST   | `/knowledge-bases`                   | 지식베이스 생성              |
| GET    | `/knowledge-bases`                   | 지식베이스 목록 조회         |
| GET    | `/knowledge-bases/:id`               | 지식베이스 상세 조회         |
| PUT    | `/knowledge-bases/:id`               | 지식베이스 업데이트          |
| DELETE | `/knowledge-bases/:id`               | 지식베이스 삭제              |
| POST   | `/knowledge-bases/copy`              | 지식베이스 복사              |
| GET    | `/knowledge-bases/:id/hybrid-search` | 하이브리드 검색(벡터+키워드) |

#### POST `/knowledge-bases` - 지식베이스 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "weknora",
    "description": "weknora description",
    "chunking_config": {
        "chunk_size": 1000,
        "chunk_overlap": 200,
        "separators": [
            "."
        ],
        "enable_multimodal": true
    },
    "image_processing_config": {
        "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
    },
    "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
    "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
    "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
    "vlm_config": {
        "enabled": true,
        "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
    },
    "cos_config": {
        "secret_id": "",
        "secret_key": "",
        "region": "",
        "bucket_name": "",
        "app_id": "",
        "path_prefix": ""
    }
}'
```

**응답**:

```json
{
    "data": {
        "id": "b5829e4a-3845-4624-a7fb-ea3b35e843b0",
        "name": "weknora",
        "description": "weknora description",
        "tenant_id": 1,
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "."
            ],
            "enable_multimodal": true
        },
        "image_processing_config": {
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
        "vlm_config": {
            "enabled": true,
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "cos_config": {
            "secret_id": "",
            "secret_key": "",
            "region": "",
            "bucket_name": "",
            "app_id": "",
            "path_prefix": ""
        },
        "created_at": "2025-08-12T11:30:09.206238645+08:00",
        "updated_at": "2025-08-12T11:30:09.206238854+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### GET `/knowledge-bases` - 지식베이스 목록 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "data": [
        {
            "id": "kb-00000001",
            "name": "Default Knowledge Base",
            "description": "System Default Knowledge Base",
            "tenant_id": 1,
            "chunking_config": {
                "chunk_size": 1000,
                "chunk_overlap": 200,
                "separators": [
                    "\n\n",
                    "\n",
                    "。",
                    "！",
                    "？",
                    ";",
                    "；"
                ],
                "enable_multimodal": true
            },
            "image_processing_config": {
                "model_id": ""
            },
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
            "vlm_config": {
                "enabled": true,
                "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
            },
            "cos_config": {
                "secret_id": "",
                "secret_key": "",
                "region": "",
                "bucket_name": "",
                "app_id": "",
                "path_prefix": ""
            },
            "created_at": "2025-08-11T20:10:41.817794+08:00",
            "updated_at": "2025-08-12T11:23:00.593097+08:00",
            "deleted_at": null
        }
    ],
    "success": true
}
```

#### GET `/knowledge-bases/:id` - 지식베이스 상세 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "data": {
        "id": "kb-00000001",
        "name": "Default Knowledge Base",
        "description": "System Default Knowledge Base",
        "tenant_id": 1,
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "\n\n",
                "\n",
                "。",
                "！",
                "？",
                ";",
                "；"
            ],
            "enable_multimodal": true
        },
        "image_processing_config": {
            "model_id": ""
        },
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
        "vlm_config": {
            "enabled": true,
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "cos_config": {
            "secret_id": "",
            "secret_key": "",
            "region": "",
            "bucket_name": "",
            "app_id": "",
            "path_prefix": ""
        },
        "created_at": "2025-08-11T20:10:41.817794+08:00",
        "updated_at": "2025-08-12T11:23:00.593097+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### PUT `/knowledge-bases/:id` - 지식베이스 업데이트

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/b5829e4a-3845-4624-a7fb-ea3b35e843b0' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--data '{
    "name": "weknora new",
    "description": "weknora description new",
    "config": {
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "\n\n",
                "\n",
                "。",
                "！",
                "？",
                ";",
                "；"
            ],
            "enable_multimodal": true
        },
        "image_processing_config": {
            "model_id": ""
        }
    }
}'
```

**응답**:

```json
{
    "data": {
        "id": "b5829e4a-3845-4624-a7fb-ea3b35e843b0",
        "name": "weknora new",
        "description": "weknora description new",
        "tenant_id": 1,
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "\n\n",
                "\n",
                "。",
                "！",
                "？",
                ";",
                "；"
            ],
            "enable_multimodal": true
        },
        "image_processing_config": {
            "model_id": ""
        },
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
        "vlm_config": {
            "enabled": true,
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "cos_config": {
            "secret_id": "",
            "secret_key": "",
            "region": "",
            "bucket_name": "",
            "app_id": "",
            "path_prefix": ""
        },
        "created_at": "2025-08-12T11:30:09.206238+08:00",
        "updated_at": "2025-08-12T11:36:09.083577609+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### DELETE `/knowledge-bases/:id` - 지식베이스 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/b5829e4a-3845-4624-a7fb-ea3b35e843b0' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ'
```

**응답**:

```json
{
    "message": "Knowledge base deleted successfully",
    "success": true
}
```

#### GET `/knowledge-bases/:id/hybrid-search` - 하이브리드 검색

벡터 검색과 키워드 검색을 결합한 하이브리드 검색을 실행합니다.

**참고**: 이 인터페이스는 GET 방법을 사용하지만 JSON 요청 본문이 필요합니다.

**요청 파라미터**:
- `query_text`: 검색 쿼리 텍스트 (필수)
- `vector_threshold`: 벡터 유사도 임계값 (0-1, 선택)
- `keyword_threshold`: 키워드 매칭 임계값 (선택)
- `match_count`: 반환 결과 수 (선택)
- `disable_keywords_match`: 키워드 매칭 비활성화 여부 (선택)
- `disable_vector_match`: 벡터 매칭 비활성화 여부 (선택)

**요청**:

```curl
curl --location --request GET 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/hybrid-search' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query_text": "如何使用知识库",
    "vector_threshold": 0.5,
    "match_count": 10
}'
```

**응답**:

```json
{
    "data": [
        {
            "id": "chunk-00000001",
            "content": "知识库是用于存储和检索知识的系统...",
            "knowledge_id": "knowledge-00000001",
            "chunk_index": 0,
            "knowledge_title": "知识库使用指南",
            "start_at": 0,
            "end_at": 500,
            "seq": 1,
            "score": 0.95,
            "chunk_type": "text",
            "image_info": "",
            "metadata": {},
            "knowledge_filename": "guide.pdf",
            "knowledge_source": "file"
        }
    ],
    "success": true
}
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 지식 관리API

| 방법   | 경로                                    | 설명                           |
| ------ | --------------------------------------- | ------------------------------ |
| POST   | `/knowledge-bases/:id/knowledge/file`   | 파일에서 지식 생성             |
| POST   | `/knowledge-bases/:id/knowledge/url`    | URL에서 지식 생성              |
| POST   | `/knowledge-bases/:id/knowledge/manual` | 수동 Markdown 지식 생성        |
| GET    | `/knowledge-bases/:id/knowledge`        | 지식베이스 하위 지식 목록 조회 |
| GET    | `/knowledge/:id`                        | 지식 상세 조회                 |
| DELETE | `/knowledge/:id`                        | 지식 삭제                      |
| GET    | `/knowledge/:id/download`               | 지식 파일 다운로드             |
| PUT    | `/knowledge/:id`                        | 지식 업데이트                  |
| PUT    | `/knowledge/manual/:id`                 | 수동 Markdown 지식 업데이트    |
| PUT    | `/knowledge/image/:id/:chunk_id`        | 이미지 청크 정보 업데이트      |
| PUT    | `/knowledge/tags`                       | 지식 태그 일괄 업데이트        |
| GET    | `/knowledge/batch`                      | 지식 일괄 조회                 |

#### POST `/knowledge-bases/:id/knowledge/file` - 파일에서 지식 생성

**폼 파라미터**:
- `file`: 업로드할 파일 (필수)
- `metadata`: JSON 형식의 메타데이터 (선택)
- `enable_multimodel`: 멀티모달 처리 활성화 여부 (선택, true/false)
- `fileName`: 사용자 정의 파일명, 폴더 업로드 시 경로 유지에 사용 (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/file' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--form 'file=@"/Users/xxxx/tests/彗星.txt"' \
--form 'enable_multimodel="true"'
```

**응답**:

```json
{
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "description": "",
        "source": "",
        "parse_status": "processing",
        "enable_status": "disabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "彗星.txt",
        "file_type": "txt",
        "file_size": 7710,
        "file_hash": "d69476ddbba45223a5e97e786539952c",
        "file_path": "data/files/1/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/1754970756171067621.txt",
        "storage_size": 0,
        "metadata": null,
        "created_at": "2025-08-12T11:52:36.168632288+08:00",
        "updated_at": "2025-08-12T11:52:36.173612121+08:00",
        "processed_at": null,
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

#### POST `/knowledge-bases/:id/knowledge/url` - URL에서 지식 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/url' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "url":"https://github.com/Tencent/WeKnora",
    "enable_multimodel":true
}'
```

**응답**:

```json
{
    "data": {
        "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "url",
        "title": "",
        "description": "",
        "source": "https://github.com/Tencent/WeKnora",
        "parse_status": "processing",
        "enable_status": "disabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "",
        "file_type": "",
        "file_size": 0,
        "file_hash": "",
        "file_path": "",
        "storage_size": 0,
        "metadata": null,
        "created_at": "2025-08-12T11:55:05.709266776+08:00",
        "updated_at": "2025-08-12T11:55:05.712918234+08:00",
        "processed_at": null,
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

#### GET `/knowledge-bases/:id/knowledge` - 지식베이스 하위 지식 목록 조회

**쿼리 파라미터**:
- `page`: 페이지 번호 (기본값 1)
- `page_size`: 페이지당 항목 수 (기본값 20)
- `tag_id`: 태그 ID로 필터링 (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge?page_size=1&page=1&tag_id=tag-00000001' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": [
        {
            "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "url",
            "title": "",
            "description": "",
            "source": "https://github.com/Tencent/WeKnora",
            "parse_status": "pending",
            "enable_status": "disabled",
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "file_name": "",
            "file_type": "",
            "file_size": 0,
            "file_hash": "",
            "file_path": "",
            "storage_size": 0,
            "metadata": null,
            "created_at": "2025-08-12T11:55:05.709266+08:00",
            "updated_at": "2025-08-12T11:55:05.709266+08:00",
            "processed_at": null,
            "error_message": "",
            "deleted_at": null
        }
    ],
    "page": 1,
    "page_size": 1,
    "success": true,
    "total": 2
}
```

참고: parse_status는 `pending/processing/failed/completed` 네 가지 상태를 포함합니다.

#### GET `/knowledge/:id` - 지식 상세 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "description": "彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。截至2019年，已知彗星超6600颗，数量庞大。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
        "source": "",
        "parse_status": "completed",
        "enable_status": "enabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "彗星.txt",
        "file_type": "txt",
        "file_size": 7710,
        "file_hash": "d69476ddbba45223a5e97e786539952c",
        "file_path": "data/files/1/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/1754970756171067621.txt",
        "storage_size": 33689,
        "metadata": null,
        "created_at": "2025-08-12T11:52:36.168632+08:00",
        "updated_at": "2025-08-12T11:52:53.376871+08:00",
        "processed_at": "2025-08-12T11:52:53.376573+08:00",
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

#### GET `/knowledge/batch` - 지식 일괄 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/batch?ids=9c8af585-ae15-44ce-8f73-45ad18394651&ids=4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": [
        {
            "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "url",
            "title": "",
            "description": "",
            "source": "https://github.com/Tencent/WeKnora",
            "parse_status": "pending",
            "enable_status": "disabled",
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "file_name": "",
            "file_type": "",
            "file_size": 0,
            "file_hash": "",
            "file_path": "",
            "storage_size": 0,
            "metadata": null,
            "created_at": "2025-08-12T11:55:05.709266+08:00",
            "updated_at": "2025-08-12T11:55:05.709266+08:00",
            "processed_at": null,
            "error_message": "",
            "deleted_at": null
        },
        {
            "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "file",
            "title": "彗星.txt",
            "description": "彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。截至2019年，已知彗星超6600颗，数量庞大。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
            "source": "",
            "parse_status": "completed",
            "enable_status": "enabled",
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "file_name": "彗星.txt",
            "file_type": "txt",
            "file_size": 7710,
            "file_hash": "d69476ddbba45223a5e97e786539952c",
            "file_path": "data/files/1/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/1754970756171067621.txt",
            "storage_size": 33689,
            "metadata": null,
            "created_at": "2025-08-12T11:52:36.168632+08:00",
            "updated_at": "2025-08-12T11:52:53.376871+08:00",
            "processed_at": "2025-08-12T11:52:53.376573+08:00",
            "error_message": "",
            "deleted_at": null
        }
    ],
    "success": true
}
```

#### DELETE `/knowledge/:id` - 지식 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge/9c8af585-ae15-44ce-8f73-45ad18394651' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "message": "Deleted successfully",
    "success": true
}
```

#### GET `/knowledge/:id/download` - 지식 파일 다운로드

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/download' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```
attachment
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 모델 관리API

| 방법   | 경로          | 설명           |
| ------ | ------------- | -------------- |
| POST   | `/models`     | 모델 생성      |
| GET    | `/models`     | 모델 목록 조회 |
| GET    | `/models/:id` | 모델 상세 조회 |
| PUT    | `/models/:id` | 모델 업데이트  |
| DELETE | `/models/:id` | 모델 삭제      |

#### POST `/models` - 모델 생성

대화 모델(KnowledgeQA) 요청 본문:

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

임베딩 모델(Embedding) 요청 본문:

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

리랭크 모델(Rerank) 요청 본문:

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

#### GET `/models` - 모델 목록 조회

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

#### GET `/models/:id` - 모델 상세 조회

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

#### PUT `/models/:id` - 모델 업데이트

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

#### DELETE `/models/:id` - 모델 삭제

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

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 청크 관리API

| 방법   | 경로                        | 설명                     |
| ------ | --------------------------- | ------------------------ |
| GET    | `/chunks/:knowledge_id`     | 지식의 청크 목록 조회    |
| DELETE | `/chunks/:knowledge_id/:id` | 청크 삭제                |
| DELETE | `/chunks/:knowledge_id`     | 지식 하위 모든 청크 삭제 |

#### GET `/chunks/:knowledge_id?page=&page_size=` - 지식의 청크 목록 조회

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
            "content": "彗星xxxx",
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

#### DELETE `/chunks/:knowledge_id/:id` - 청크 삭제

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

#### DELETE `/chunks/:knowledge_id` - 지식 하위 모든 청크 삭제

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

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 태그 관리API

| 방법   | 경로                                  | 설명                      |
| ------ | ------------------------------------- | ------------------------- |
| GET    | `/knowledge-bases/:id/tags`           | 지식베이스 태그 목록 조회 |
| POST   | `/knowledge-bases/:id/tags`           | 태그 생성                 |
| PUT    | `/knowledge-bases/:id/tags/:tag_id`   | 태그 업데이트             |
| DELETE | `/knowledge-bases/:id/tags/:tag_id`   | 태그 삭제                 |

#### GET `/knowledge-bases/:id/tags` - 지식베이스 태그 목록 조회

**쿼리 파라미터**:
- `page`: 페이지 번호 (기본값 1)
- `page_size`: 페이지당 항목 수 (기본값 20)
- `keyword`: 태그명 키워드 검색 (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags?page=1&page_size=10' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": {
        "total": 2,
        "page": 1,
        "page_size": 10,
        "data": [
            {
                "id": "tag-00000001",
                "tenant_id": 1,
                "knowledge_base_id": "kb-00000001",
                "name": "技术文档",
                "color": "#1890ff",
                "sort_order": 1,
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00",
                "knowledge_count": 5,
                "chunk_count": 120
            },
            {
                "id": "tag-00000002",
                "tenant_id": 1,
                "knowledge_base_id": "kb-00000001",
                "name": "常见问题",
                "color": "#52c41a",
                "sort_order": 2,
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00",
                "knowledge_count": 3,
                "chunk_count": 45
            }
        ]
    },
    "success": true
}
```

#### POST `/knowledge-bases/:id/tags` - 태그 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "name": "产品手册",
    "color": "#faad14",
    "sort_order": 3
}'
```

**응답**:

```json
{
    "data": {
        "id": "tag-00000003",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "name": "产品手册",
        "color": "#faad14",
        "sort_order": 3,
        "created_at": "2025-08-12T11:00:00+08:00",
        "updated_at": "2025-08-12T11:00:00+08:00"
    },
    "success": true
}
```

#### PUT `/knowledge-bases/:id/tags/:tag_id` - 태그 업데이트

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags/tag-00000003' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "name": "产品手册更新",
    "color": "#ff4d4f"
}'
```

**응답**:

```json
{
    "data": {
        "id": "tag-00000003",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "name": "产品手册更新",
        "color": "#ff4d4f",
        "sort_order": 3,
        "created_at": "2025-08-12T11:00:00+08:00",
        "updated_at": "2025-08-12T11:30:00+08:00"
    },
    "success": true
}
```

#### DELETE `/knowledge-bases/:id/tags/:tag_id` - 태그 삭제

**쿼리 파라미터**:
- `force`: `true`로 설정하면 강제 삭제 (태그가 참조 중인 경우에도)

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags/tag-00000003?force=true' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "success": true
}
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### FAQ 관리API

| 방법   | 경로                                         | 설명                          |
| ------ | -------------------------------------------- | ----------------------------- |
| GET    | `/knowledge-bases/:id/faq/entries`           | FAQ 항목 목록 조회            |
| POST   | `/knowledge-bases/:id/faq/entries`           | FAQ 항목 일괄 가져오기        |
| POST   | `/knowledge-bases/:id/faq/entry`             | 단일 FAQ 항목 생성            |
| PUT    | `/knowledge-bases/:id/faq/entries/:entry_id` | 단일 FAQ 항목 업데이트        |
| PUT    | `/knowledge-bases/:id/faq/entries/status`    | FAQ 활성화 상태 일괄 업데이트 |
| PUT    | `/knowledge-bases/:id/faq/entries/tags`      | FAQ 태그 일괄 업데이트        |
| DELETE | `/knowledge-bases/:id/faq/entries`           | FAQ 항목 일괄 삭제            |
| POST   | `/knowledge-bases/:id/faq/search`            | FAQ 하이브리드 검색           |

#### GET `/knowledge-bases/:id/faq/entries` - FAQ 항목 목록 조회

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
                "standard_question": "如何重置密码？",
                "similar_questions": ["忘记密码怎么办", "密码找回"],
                "negative_questions": ["如何修改用户名"],
                "answers": ["您可以通过点击登录页面的'忘记密码'链接来重置密码。"],
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

#### POST `/knowledge-bases/:id/faq/entries` - FAQ 항목 일괄 가져오기

**요청 파라미터**:
- `mode`: 가져오기 모드, `append`(추가) 또는 `replace`(교체)
- `entries`: FAQ 항목 배열
- `knowledge_id`: 연결할 지식 ID (선택)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "mode": "append",
    "entries": [
        {
            "standard_question": "如何联系客服？",
            "similar_questions": ["客服电话", "在线客服"],
            "answers": ["您可以通过拨打400-xxx-xxxx联系我们的客服。"],
            "tag_id": "tag-00000001"
        },
        {
            "standard_question": "退款政策是什么？",
            "answers": ["我们提供7天无理由退款服务。"]
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

참고: 일괄 가져오기는 비동기 작업으로, 진행 상황 추적을 위한 작업 ID를 반환합니다.

#### POST `/knowledge-bases/:id/faq/entry` - 단일 FAQ 항목 생성

단일 FAQ 항목을 동기적으로 생성합니다. 단건 입력 시나리오에 적합합니다. 표준 질문과 유사 질문이 기존 FAQ와 중복되는지 자동으로 확인합니다.

**요청 파라미터**:
- `standard_question`: 표준 질문 (필수)
- `similar_questions`: 유사 질문 배열 (선택)
- `negative_questions`: 반례 질문 배열 (선택)
- `answers`: 답변 배열 (필수)
- `tag_id`: 태그 ID (선택)
- `is_enabled`: 활성화 여부 (선택, 기본값 true)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entry' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "如何联系客服？",
    "similar_questions": ["客服电话", "在线客服"],
    "answers": ["您可以通过拨打400-xxx-xxxx联系我们的客服。"],
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
        "standard_question": "如何联系客服？",
        "similar_questions": ["客服电话", "在线客服"],
        "negative_questions": [],
        "answers": ["您可以通过拨打400-xxx-xxxx联系我们的客服。"],
        "index_mode": "hybrid",
        "chunk_type": "faq",
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

**에러 응답** (표준 질문 또는 유사 질문 중복 시):

```json
{
    "success": false,
    "error": {
        "code": "BAD_REQUEST",
        "message": "标准问与已有FAQ重复"
    }
}
```

#### PUT `/knowledge-bases/:id/faq/entries/:entry_id` - 단일 FAQ 항목 업데이트

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/faq-00000001' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "如何重置账户密码？",
    "similar_questions": ["忘记密码怎么办", "密码找回", "重置密码"],
    "answers": ["您可以通过以下步骤重置密码：1. 点击登录页面的"忘记密码" 2. 输入注册邮箱 3. 查收重置邮件"],
    "is_enabled": true
}'
```

**응답**:

```json
{
    "success": true
}
```

#### PUT `/knowledge-bases/:id/faq/entries/status` - FAQ 활성화 상태 일괄 업데이트

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

#### PUT `/knowledge-bases/:id/faq/entries/tags` - FAQ 태그 일괄 업데이트

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

#### DELETE `/knowledge-bases/:id/faq/entries` - FAQ 항목 일괄 삭제

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

#### POST `/knowledge-bases/:id/faq/search` - FAQ 하이브리드 검색

**요청 파라미터**:
- `query_text`: 검색 쿼리 텍스트
- `vector_threshold`: 벡터 유사도 임계값 (0-1)
- `match_count`: 반환 결과 수 (최대 200)

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/search' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query_text": "如何重置密码",
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
            "standard_question": "如何重置密码？",
            "similar_questions": ["忘记密码怎么办", "密码找回"],
            "answers": ["您可以通过点击登录页面的'忘记密码'链接来重置密码。"],
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

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 세션 관리API

| 방법   | 경로                                    | 설명                    |
| ------ | --------------------------------------- | ----------------------- |
| POST   | `/sessions`                             | 세션 생성               |
| GET    | `/sessions/:id`                         | 세션 상세 조회          |
| GET    | `/sessions`                             | 테넌트의 세션 목록 조회 |
| PUT    | `/sessions/:id`                         | 세션 업데이트           |
| DELETE | `/sessions/:id`                         | 세션 삭제               |
| POST   | `/sessions/:session_id/generate_title`  | 세션 제목 생성          |
| GET    | `/sessions/continue-stream/:session_id` | 미완료 세션 계속하기    |

#### POST `/sessions` - 세션 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "knowledge_base_id": "kb-00000001",
    "session_strategy": {
        "max_rounds": 5,
        "enable_rewrite": true,
        "fallback_strategy": "FIXED_RESPONSE",
        "fallback_response": "对不起，我无法回答这个问题",
        "embedding_top_k": 10,
        "keyword_threshold": 0.5,
        "vector_threshold": 0.7,
        "rerank_model_id": "排序模型ID",
        "rerank_top_k": 3,
        "rerank_threshold": 0.7,
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "summary_parameters": {
            "max_tokens": 0,
            "repeat_penalty": 1,
            "top_k": 0,
            "top_p": 0,
            "frequency_penalty": 0,
            "presence_penalty": 0,
            "prompt": "这是用户和助手之间的对话。xxx",
            "context_template": "你是一个专业的智能信息检索助手xxx",
            "no_match_prefix": "<think>\n</think>\nNO_MATCH",
            "temperature": 0.3,
            "seed": 0,
            "max_completion_tokens": 2048
        },
        "no_match_prefix": "<think>\n</think>\nNO_MATCH"
    }
}'
```

**응답**:

```json
{
    "data": {
        "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
        "title": "",
        "description": "",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "max_rounds": 5,
        "enable_rewrite": true,
        "fallback_strategy": "FIXED_RESPONSE",
        "fallback_response": "对不起，我无法回答这个问题",
        "embedding_top_k": 10,
        "keyword_threshold": 0.5,
        "vector_threshold": 0.7,
        "rerank_model_id": "排序模型ID",
        "rerank_top_k": 3,
        "rerank_threshold": 0.7,
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "summary_parameters": {
            "max_tokens": 0,
            "repeat_penalty": 1,
            "top_k": 0,
            "top_p": 0,
            "frequency_penalty": 0,
            "presence_penalty": 0,
            "prompt": "这是用户和助手之间的对话。xxx",
            "context_template": "你是一个专业的智能信息检索助手xxx",
            "no_match_prefix": "<think>\n</think>\nNO_MATCH",
            "temperature": 0.3,
            "seed": 0,
            "max_completion_tokens": 2048
        },
        "agent_config": null,
        "context_config": null,
        "created_at": "2025-08-12T12:26:19.611616669+08:00",
        "updated_at": "2025-08-12T12:26:19.611616919+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### GET `/sessions/:id` - 세션 상세 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": {
        "id": "ceb9babb-1e30-41d7-817d-fd584954304b",
        "title": "模型优化策略",
        "description": "",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "max_rounds": 5,
        "enable_rewrite": true,
        "fallback_strategy": "fixed",
        "fallback_response": "抱歉，我无法回答这个问题。",
        "embedding_top_k": 10,
        "keyword_threshold": 0.3,
        "vector_threshold": 0.5,
        "rerank_model_id": "",
        "rerank_top_k": 5,
        "rerank_threshold": 0.7,
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "summary_parameters": {
            "max_tokens": 0,
            "repeat_penalty": 1,
            "top_k": 0,
            "top_p": 0,
            "frequency_penalty": 0,
            "presence_penalty": 0,
            "prompt": "这是用户和助手之间的对话",
            "context_template": "你是一个专业的智能信息检索助手",
            "no_match_prefix": "<think>\n</think>\nNO_MATCH",
            "temperature": 0.3,
            "seed": 0,
            "max_completion_tokens": 2048
        },
        "agent_config": null,
        "context_config": null,
        "created_at": "2025-08-12T10:24:38.308596+08:00",
        "updated_at": "2025-08-12T10:25:41.317761+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### GET `/sessions?page=&page_size=` - 테넌트의 세션 목록 조회

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions?page=1&page_size=1' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": [
        {
            "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
            "title": "",
            "description": "",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "max_rounds": 5,
            "enable_rewrite": true,
            "fallback_strategy": "FIXED_RESPONSE",
            "fallback_response": "对不起，我无法回答这个问题",
            "embedding_top_k": 10,
            "keyword_threshold": 0.5,
            "vector_threshold": 0.7,
            "rerank_model_id": "排序模型ID",
            "rerank_top_k": 3,
            "rerank_threshold": 0.7,
            "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "summary_parameters": {
                "max_tokens": 0,
                "repeat_penalty": 1,
                "top_k": 0,
                "top_p": 0,
                "frequency_penalty": 0,
                "presence_penalty": 0,
                "prompt": "这是用户和助手之间的对话。xxx",
                "context_template": "你是一个专业的智能信息检索助手xxx",
                "no_match_prefix": "<think>\n</think>\nNO_MATCH",
                "temperature": 0.3,
                "seed": 0,
                "max_completion_tokens": 2048
            },
            "created_at": "2025-08-12T12:26:19.611616+08:00",
            "updated_at": "2025-08-12T12:26:19.611616+08:00",
            "deleted_at": null
        }
    ],
    "page": 1,
    "page_size": 1,
    "success": true,
    "total": 2
}
```

#### PUT `/sessions/:id` - 세션 업데이트

**요청**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/sessions/411d6b70-9a85-4d03-bb74-aab0fd8bd12f' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "title": "weknora",
    "description": "weknora description",
    "knowledge_base_id": "kb-00000001",
    "max_rounds": 5,
    "enable_rewrite": true,
    "fallback_strategy": "FIXED_RESPONSE",
    "fallback_response": "对不起，我无法回答这个问题",
    "embedding_top_k": 10,
    "keyword_threshold": 0.5,
    "vector_threshold": 0.7,
    "rerank_model_id": "排序模型ID",
    "rerank_top_k": 3,
    "rerank_threshold": 0.7,
    "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
    "summary_parameters": {
        "max_tokens": 0,
        "repeat_penalty": 1,
        "top_k": 0,
        "top_p": 0,
        "frequency_penalty": 0,
        "presence_penalty": 0,
        "prompt": "这是用户和助手之间的对话。xxx",
        "context_template": "你是一个专业的智能信息检索助手xxx",
        "no_match_prefix": "<think>\n</think>\nNO_MATCH",
        "temperature": 0.3,
        "seed": 0,
        "max_completion_tokens": 2048
    }
}'
```

**응답**:

```json
{
    "data": {
        "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
        "title": "weknora",
        "description": "weknora description",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "max_rounds": 5,
        "enable_rewrite": true,
        "fallback_strategy": "FIXED_RESPONSE",
        "fallback_response": "对不起，我无法回答这个问题",
        "embedding_top_k": 10,
        "keyword_threshold": 0.5,
        "vector_threshold": 0.7,
        "rerank_model_id": "排序模型ID",
        "rerank_top_k": 3,
        "rerank_threshold": 0.7,
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "summary_parameters": {
            "max_tokens": 0,
            "repeat_penalty": 1,
            "top_k": 0,
            "top_p": 0,
            "frequency_penalty": 0,
            "presence_penalty": 0,
            "prompt": "这是用户和助手之间的对话。xxx",
            "context_template": "你是一个专业的智能信息检索助手xxx",
            "no_match_prefix": "<think>\n</think>\nNO_MATCH",
            "temperature": 0.3,
            "seed": 0,
            "max_completion_tokens": 2048
        },
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "2025-08-12T14:20:56.738424351+08:00",
        "deleted_at": null
    },
    "success": true
}
```

#### DELETE `/sessions/:id` - 세션 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/411d6b70-9a85-4d03-bb74-aab0fd8bd12f' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "message": "Session deleted successfully",
    "success": true
}
```

#### POST `/sessions/:session_id/generate_title` - 세션 제목 생성

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b/generate_title' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
  "messages": [
    {
      "role": "user",
      "content": "你好，我想了解关于人工智能的知识"
    },
    {
      "role": "assistant",
      "content": "人工智能是计算机科学的一个分支..."
    }
  ]
}'
```

**응답**:

```json
{
    "data": "模型优化策略",
    "success": true
}
```

#### GET `/sessions/continue-stream/:session_id` - 미완료 세션 계속하기

**쿼리 파라미터**:
- `message_id`: `/messages/:session_id/load` 인터페이스에서 가져온 `is_completed`가 `false`인 메시지 ID

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/continue-stream/ceb9babb-1e30-41d7-817d-fd584954304b?message_id=b8b90eeb-7dd5-4cf9-81c6-5ebcbd759451' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답 형식**:
서버-전송 이벤트 스트림(Server-Sent Events), `/knowledge-chat/:session_id` 반환 결과와 동일

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 채팅 기능API

| 방법 | 경로                          | 설명                      |
| ---- | ----------------------------- | ------------------------- |
| POST | `/knowledge-chat/:session_id` | 지식베이스 기반 질의응답  |
| POST | `/knowledge-search`           | 지식베이스 기반 지식 검색 |

#### POST `/knowledge-chat/:session_id` - 지식베이스 기반 질의응답

**요청**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-chat/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query": "彗尾的形状"
}'
```

**응답 형식**:
서버-전송 이벤트 스트림(Server-Sent Events, Content-Type: text/event-stream)

**응답**:

```
event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"references","content":"","done":false,"knowledge_references":[{"id":"c8347bef-127f-4a22-b962-edf5a75386ec","content":"彗星xxx。","knowledge_id":"a6790b93-4700-4676-bd48-0d4804e1456b","chunk_index":0,"knowledge_title":"彗星.txt","start_at":0,"end_at":2760,"seq":0,"score":4.038836479187012,"match_type":3,"sub_chunk_id":["688821f0-40bf-428e-8cb6-541531ebeb76","c1e9903e-2b4d-4281-be15-0149288d45c2","7d955251-3f79-4fd5-a6aa-02f81e044091"],"metadata":{},"chunk_type":"text","parent_chunk_id":"","image_info":"","knowledge_filename":"彗星.txt","knowledge_source":""},{"id":"fa3aadee-cadb-4a84-9941-c839edc3e626","content":"# 文档名称\n彗星.txt\n\n# 摘要\n彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会释放气体形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。目前已知彗星数量众多，且存在系外彗星。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。","knowledge_id":"a6790b93-4700-4676-bd48-0d4804e1456b","chunk_index":6,"knowledge_title":"彗星.txt","start_at":0,"end_at":0,"seq":6,"score":0.6131043121858466,"match_type":3,"sub_chunk_id":null,"metadata":{},"chunk_type":"summary","parent_chunk_id":"c8347bef-127f-4a22-b962-edf5a75386ec","image_info":"","knowledge_filename":"彗星.txt","knowledge_source":""}]}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"表现为","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"结构","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"。","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"","done":true,"knowledge_references":null}
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 메시지 관리API

| 방법   | 경로                         | 설명                       |
| ------ | ---------------------------- | -------------------------- |
| GET    | `/messages/:session_id/load` | 최근 세션 메시지 목록 조회 |
| DELETE | `/messages/:session_id/:id`  | 메시지 삭제                |

#### GET `/messages/:session_id/load?before_time=2025-04-18T11:57:31.310671+08:00&limit=20` - 최근 세션 메시지 목록 조회

**쿼리 파라미터**:

- `before_time`: 이전 조회 시 가장 오래된 메시지의 created_at 필드값, 비어 있으면 최근 메시지를 가져옴
- `limit`: 페이지당 항목 수 (기본값 20)

**요청**:

```curl
curl --location --request GET 'http://localhost:8080/api/v1/messages/ceb9babb-1e30-41d7-817d-fd584954304b/load?limit=3&before_time=2030-08-12T14%3A35%3A42.123456789Z' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "query": "彗尾的形状"
}'
```

**응답**:

```json
{
    "data": [
        {
            "id": "b8b90eeb-7dd5-4cf9-81c6-5ebcbd759451",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "hCA8SDjxcAvv",
            "content": "<think>\n好的",
            "role": "assistant",
            "knowledge_references": [
                {
                    "id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "content": "彗星xxx",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 0,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 2760,
                    "seq": 0,
                    "score": 4.038836479187012,
                    "match_type": 4,
                    "sub_chunk_id": [
                        "688821f0-40bf-428e-8cb6-541531ebeb76",
                        "c1e9903e-2b4d-4281-be15-0149288d45c2",
                        "7d955251-3f79-4fd5-a6aa-02f81e044091"
                    ],
                    "metadata": {},
                    "chunk_type": "text",
                    "parent_chunk_id": "",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                },
                {
                    "id": "fa3aadee-cadb-4a84-9941-c839edc3e626",
                    "content": "# 文档名称\n彗星.txt\n\n# 摘要\n彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会释放气体形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。目前已知彗星数量众多，且存在系外彗星。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 6,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 0,
                    "seq": 6,
                    "score": 0.6131043121858466,
                    "match_type": 0,
                    "sub_chunk_id": null,
                    "metadata": {},
                    "chunk_type": "summary",
                    "parent_chunk_id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                }
            ],
            "agent_steps": [],
            "is_completed": true,
            "created_at": "2025-08-12T10:24:38.370548+08:00",
            "updated_at": "2025-08-12T10:25:40.416382+08:00",
            "deleted_at": null
        },
        {
            "id": "7fa136ae-a045-424e-baac-52113d92ae94",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "3475c004-0ada-4306-9d30-d7f5efce50d2",
            "content": "彗尾的形状",
            "role": "user",
            "knowledge_references": [],
            "agent_steps": [],
            "is_completed": true,
            "created_at": "2025-08-12T14:30:39.732246+08:00",
            "updated_at": "2025-08-12T14:30:39.733277+08:00",
            "deleted_at": null
        },
        {
            "id": "9bcafbcf-a758-40af-a9a3-c4d8e0f49439",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "3475c004-0ada-4306-9d30-d7f5efce50d2",
            "content": "<think>\n好的",
            "role": "assistant",
            "knowledge_references": [
                {
                    "id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "content": "彗星xxx",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 0,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 2760,
                    "seq": 0,
                    "score": 4.038836479187012,
                    "match_type": 3,
                    "sub_chunk_id": [
                        "688821f0-40bf-428e-8cb6-541531ebeb76",
                        "c1e9903e-2b4d-4281-be15-0149288d45c2",
                        "7d955251-3f79-4fd5-a6aa-02f81e044091"
                    ],
                    "metadata": {},
                    "chunk_type": "text",
                    "parent_chunk_id": "",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                },
                {
                    "id": "fa3aadee-cadb-4a84-9941-c839edc3e626",
                    "content": "# 文档名称\n彗星.txt\n\n# 摘要\n彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会释放气体形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。目前已知彗星数量众多，且存在系外彗星。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 6,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 0,
                    "seq": 6,
                    "score": 0.6131043121858466,
                    "match_type": 3,
                    "sub_chunk_id": null,
                    "metadata": {},
                    "chunk_type": "summary",
                    "parent_chunk_id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                }
            ],
            "agent_steps": [],
            "is_completed": true,
            "created_at": "2025-08-12T14:30:39.735108+08:00",
            "updated_at": "2025-08-12T14:31:17.829926+08:00",
            "deleted_at": null
        }
    ],
    "success": true
}
```

#### DELETE `/messages/:session_id/:id` - 메시지 삭제

**요청**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/messages/ceb9babb-1e30-41d7-817d-fd584954304b/9bcafbcf-a758-40af-a9a3-c4d8e0f49439' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "message": "Message deleted successfully",
    "success": true
}
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>

### 평가 기능API

| 방법 | 경로          | 설명           |
| ---- | ------------- | -------------- |
| GET  | `/evaluation` | 평가 작업 조회 |
| POST | `/evaluation` | 평가 작업 생성 |

#### GET `/evaluation` - 평가 작업 조회

**요청 파라미터**:
- `task_id`: `POST /evaluation` 인터페이스에서 가져온 작업 ID
- `X-API-Key`: 사용자 API Key

**요청**:

```bash
curl --location 'http://localhost:8080/api/v1/evaluation?task_id=c34563ad-b09f-4858-b72e-e92beb80becb' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json'
```

**응답**:

```json
{
    "data": {
        "task": {
            "id": "c34563ad-b09f-4858-b72e-e92beb80becb",
            "tenant_id": 1,
            "dataset_id": "default",
            "start_time": "2025-08-12T14:54:26.221804768+08:00",
            "status": 2,
            "total": 1,
            "finished": 1
        },
        "params": {
            "session_id": "",
            "knowledge_base_id": "2ef57434-8c8d-4442-b967-2f7fc578a2fc",
            "vector_threshold": 0.5,
            "keyword_threshold": 0.3,
            "embedding_top_k": 10,
            "vector_database": "",
            "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
            "rerank_top_k": 5,
            "rerank_threshold": 0.7,
            "chat_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "summary_config": {
                "max_tokens": 0,
                "repeat_penalty": 1,
                "top_k": 0,
                "top_p": 0,
                "frequency_penalty": 0,
                "presence_penalty": 0,
                "prompt": "这是用户和助手之间的对话。",
                "context_template": "你是一个专业的智能信息检索助手",
                "no_match_prefix": "<think>\n</think>\nNO_MATCH",
                "temperature": 0.3,
                "seed": 0,
                "max_completion_tokens": 2048
            },
            "fallback_strategy": "",
            "fallback_response": "抱歉，我无法回答这个问题。"
        },
        "metric": {
            "retrieval_metrics": {
                "precision": 0,
                "recall": 0,
                "ndcg3": 0,
                "ndcg10": 0,
                "mrr": 0,
                "map": 0
            },
            "generation_metrics": {
                "bleu1": 0.037656734016532384,
                "bleu2": 0.04067392145167686,
                "bleu4": 0.048963321289052536,
                "rouge1": 0,
                "rouge2": 0,
                "rougel": 0
            }
        }
    },
    "success": true
}
```

#### POST `/evaluation` - 평가 작업 생성

**요청 파라미터**:
- `dataset_id`: 평가에 사용할 데이터셋, 현재 공식 테스트 데이터셋 `default`만 지원
- `knowledge_base_id`: 평가에 사용할 지식베이스
- `chat_id`: 평가에 사용할 대화 모델
- `rerank_id`: 평가에 사용할 재순위 모델

**요청**:

```bash
curl --location 'http://localhost:8080/api/v1/evaluation' \
--header 'X-API-Key: sk-vQHV2NZI_LK5W7wHQvH3yGYExX8YnhaHwZipUYbiZKCYJbBQ' \
--header 'Content-Type: application/json' \
--data '{
    "dataset_id": "default",
    "knowledge_base_id": "kb-00000001",
    "chat_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
    "rerank_id": "b30171a1-787b-426e-a293-735cd5ac16c0"
}'
```

**응답**:

```json
{
    "data": {
        "task": {
            "id": "c34563ad-b09f-4858-b72e-e92beb80becb",
            "tenant_id": 1,
            "dataset_id": "default",
            "start_time": "2025-08-12T14:54:26.221804768+08:00",
            "status": 1
        },
        "params": {
            "session_id": "",
            "knowledge_base_id": "2ef57434-8c8d-4442-b967-2f7fc578a2fc",
            "vector_threshold": 0.5,
            "keyword_threshold": 0.3,
            "embedding_top_k": 10,
            "vector_database": "",
            "rerank_model_id": "b30171a1-787b-426e-a293-735cd5ac16c0",
            "rerank_top_k": 5,
            "rerank_threshold": 0.7,
            "chat_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "summary_config": {
                "max_tokens": 0,
                "repeat_penalty": 1,
                "top_k": 0,
                "top_p": 0,
                "frequency_penalty": 0,
                "presence_penalty": 0,
                "prompt": "这是用户和助手之间的对话。",
                "context_template": "你是一个专业的智能信息检索助手，xxx",
                "no_match_prefix": "<think>\n</think>\nNO_MATCH",
                "temperature": 0.3,
                "seed": 0,
                "max_completion_tokens": 2048
            },
            "fallback_strategy": "",
            "fallback_response": "抱歉，我无法回答这个问题。"
        }
    },
    "success": true
}
```

<div align="right"><a href="#weknora-api-문서">맨 위로 ↑</a></div>
