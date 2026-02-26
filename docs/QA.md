# 자주 묻는 질문

## 1. 로그를 어떻게 확인하나요?
```bash
docker compose logs -f app docreader postgres
```

## 2. 서비스를 어떻게 시작하고 중지하나요?
```bash
# 서비스 시작
./scripts/start_all.sh

# 서비스 중지
./scripts/start_all.sh --stop

# 데이터베이스 초기화
./scripts/start_all.sh --stop && make clean-db
```

## 3. 서비스 시작 후 문서를 정상적으로 업로드할 수 없나요?

일반적으로 Embedding 모델과 대화 모델이 올바르게 설정되지 않아 발생합니다. 다음 단계에 따라 점검하세요.

1. `.env` 설정에서 모델 정보가 완전히 구성되었는지 확인하세요. ollama를 통해 로컬 모델에 접근하는 경우, 로컬 ollama 서비스가 정상적으로 실행 중인지 확인하고 `.env`에서 아래 환경 변수를 올바르게 설정해야 합니다:
```bash
# LLM Model
INIT_LLM_MODEL_NAME=your_llm_model
# Embedding Model
INIT_EMBEDDING_MODEL_NAME=your_embedding_model
# Embedding 모델의 벡터 차원
INIT_EMBEDDING_MODEL_DIMENSION=your_embedding_model_dimension
# Embedding 모델의 ID, 일반적으로 문자열
INIT_EMBEDDING_MODEL_ID=your_embedding_model_id
```

remote api를 통해 모델에 접근하는 경우, 추가로 해당 `BASE_URL`과 `API_KEY`를 제공해야 합니다:
```bash
# LLM 모델의 접근 주소
INIT_LLM_MODEL_BASE_URL=your_llm_model_base_url
# LLM 모델의 API 키, 인증이 필요한 경우 설정
INIT_LLM_MODEL_API_KEY=your_llm_model_api_key
# Embedding 모델의 접근 주소
INIT_EMBEDDING_MODEL_BASE_URL=your_embedding_model_base_url
# Embedding 모델의 API 키, 인증이 필요한 경우 설정
INIT_EMBEDDING_MODEL_API_KEY=your_embedding_model_api_key
```

재정렬 기능이 필요한 경우, 추가로 Rerank 모델을 설정해야 합니다. 구체적인 설정은 다음과 같습니다:
```bash
# 사용할 Rerank 모델 이름
INIT_RERANK_MODEL_NAME=your_rerank_model_name
# Rerank 모델의 접근 주소
INIT_RERANK_MODEL_BASE_URL=your_rerank_model_base_url
# Rerank 모델의 API 키, 인증이 필요한 경우 설정
INIT_RERANK_MODEL_API_KEY=your_rerank_model_api_key
```

2. 메인 서비스 로그를 확인하여 `ERROR` 로그 출력이 있는지 확인하세요.

## 4. 멀티모달 기능을 어떻게 활성화하나요?
1. `.env`에서 아래 설정이 올바르게 지정되었는지 확인하세요:
```bash
# VLM_MODEL_NAME 사용할 멀티모달 모델 이름
VLM_MODEL_NAME=your_vlm_model_name

# VLM_MODEL_BASE_URL 사용할 멀티모달 모델 접근 주소
VLM_MODEL_BASE_URL=your_vlm_model_base_url

# VLM_MODEL_API_KEY 사용할 멀티모달 모델 API 키
VLM_MODEL_API_KEY=your_vlm_model_api_key
```
참고: 멀티모달 대형 모델은 현재 remote api 접근만 지원하므로 `VLM_MODEL_BASE_URL`과 `VLM_MODEL_API_KEY`를 제공해야 합니다.

2. 파싱된 파일은 COS에 업로드해야 하므로, `.env`에서 `COS` 정보가 올바르게 설정되었는지 확인하세요:
```bash
# 텐센트 클라우드 COS의 액세스 키 ID
COS_SECRET_ID=your_cos_secret_id

# 텐센트 클라우드 COS의 시크릿 키
COS_SECRET_KEY=your_cos_secret_key

# 텐센트 클라우드 COS의 리전, 예: ap-guangzhou
COS_REGION=your_cos_region

# 텐센트 클라우드 COS의 버킷 이름
COS_BUCKET_NAME=your_cos_bucket_name

# 텐센트 클라우드 COS의 앱 ID
COS_APP_ID=your_cos_app_id

# 텐센트 클라우드 COS의 경로 접두사, 파일 저장에 사용
COS_PATH_PREFIX=your_cos_path_prefix
```
중요: COS에 있는 파일의 권한을 반드시 **공개 읽기**로 설정하세요. 그렇지 않으면 문서 파싱 모듈이 파일을 정상적으로 파싱할 수 없습니다.

3. 문서 파싱 모듈 로그를 확인하여 OCR과 Caption이 올바르게 파싱되고 출력되는지 확인하세요.


## P.S.
위의 방법으로 문제가 해결되지 않으면, issue에 문제를 설명하고 문제 해결에 도움이 되는 필요한 로그 정보를 제공해 주세요.
