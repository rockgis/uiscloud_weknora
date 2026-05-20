# WeKnora 서버 배포 가이드

WeKnora를 서버에 배포하는 방법을 단계별로 설명합니다.

---

## 목차

1. [시스템 요구사항](#1-시스템-요구사항)
2. [서버 초기 설정](#2-서버-초기-설정)
3. [배포 패키지 전송](#3-배포-패키지-전송)
4. [최초 설치](#4-최초-설치)
5. [서비스 관리](#5-서비스-관리)
6. [환경 설정 상세](#6-환경-설정-상세)
7. [업데이트](#7-업데이트)
8. [백업 & 복원](#8-백업--복원)
9. [선택적 서비스](#9-선택적-서비스)
10. [문제 해결](#10-문제-해결)

---

## 1. 시스템 요구사항

| 항목 | 최소 | 권장 |
|------|------|------|
| CPU | 2코어 | 4코어+ |
| RAM | 4GB | 8GB+ |
| 디스크 | 40GB | 100GB+ |
| OS | Ubuntu 20.04+ / CentOS 8+ | Ubuntu 22.04 LTS |
| Docker | 24.0+ | 최신 |
| 인바운드 포트 | 80, 8080 | 80, 8080 |

---

## 2. 서버 초기 설정

**서버에서 1회만 실행합니다** (Docker 설치, 사용자 설정 등):

```bash
# 배포 패키지를 서버에 복사 (개발 머신에서)
scp weknora-deploy-*.tar.gz user@<서버IP>:~/

# 서버에서
ssh user@<서버IP>
tar xzf weknora-deploy-*.tar.gz
cd weknora-deploy-*/

# 서버 초기화 (root 필요)
sudo bash setup-server.sh
```

### 옵션

```bash
sudo bash setup-server.sh --skip-docker    # Docker 이미 설치된 경우
sudo bash setup-server.sh --no-firewall    # 방화벽 설정 건너뜀
sudo bash setup-server.sh --dir=/srv/weknora  # 설치 경로 변경
```

---

## 3. 배포 패키지 전송

### 방법 A: make package 사용 (프로젝트 루트에서)

```bash
# 개발 머신에서 배포 패키지 생성
make package

# 생성된 파일을 서버로 전송
scp weknora-deploy-0.2.6.tar.gz deploy@<서버IP>:/opt/weknora/
```

### 방법 B: 수동 복사

```bash
scp docker-compose.yml .env.example deploy/weknora.sh \
    deploy/docker-compose.prod.yml \
    deploy@<서버IP>:/opt/weknora/
```

---

## 4. 최초 설치

배포 사용자(`deploy`)로 설치 디렉토리에서 실행합니다:

```bash
# 서버에서
sudo -u deploy bash
cd /opt/weknora

# deploy/ 내용을 루트로 이동 (패키지를 풀었을 경우)
tar xzf weknora-deploy-*.tar.gz --strip-components=1

# 최초 설치 실행
bash weknora.sh install
```

**설치 과정:**
1. `.env` 파일 생성 (보안 키 자동 생성)
2. `.env` 편집 안내 출력
3. **`.env` 편집 완료 후 다시 실행**하면 서비스 자동 시작

```bash
# LLM 설정 후 재실행
vi .env
bash weknora.sh install
```

---

## 5. 서비스 관리

```bash
bash weknora.sh status           # 서비스 상태 + 헬스체크
bash weknora.sh start            # 서비스 시작
bash weknora.sh stop             # 서비스 중지
bash weknora.sh restart          # 서비스 재시작
bash weknora.sh logs             # 전체 로그
bash weknora.sh logs app         # 백엔드 로그만
bash weknora.sh logs docreader   # Docreader 로그
bash weknora.sh health           # API 헬스체크
bash weknora.sh clean            # 미사용 이미지 정리
```

---

## 6. 환경 설정 상세

`.env` 파일의 주요 설정 항목:

### LLM 설정 (필수)

```env
# 기본 LLM 모델 (OpenAI 호환 API)
INIT_LLM_MODEL_NAME=gpt-4o-mini
INIT_LLM_MODEL_BASE_URL=https://api.openai.com/v1
INIT_LLM_MODEL_API_KEY=sk-...

# 임베딩 모델
INIT_EMBEDDING_MODEL_NAME=text-embedding-3-small
INIT_EMBEDDING_MODEL_BASE_URL=https://api.openai.com/v1
INIT_EMBEDDING_MODEL_API_KEY=sk-...
INIT_EMBEDDING_MODEL_DIMENSION=1536

# 리랭크 모델 (선택사항)
# INIT_RERANK_MODEL_NAME=...
```

### Ollama 사용 시

```env
OLLAMA_BASE_URL=http://host.docker.internal:11434
INIT_LLM_MODEL_BASE_URL=http://host.docker.internal:11434/v1
INIT_LLM_MODEL_API_KEY=ollama
INIT_EMBEDDING_MODEL_BASE_URL=http://host.docker.internal:11434/v1
INIT_EMBEDDING_MODEL_API_KEY=ollama
```

### 검색 엔진 선택 (RETRIEVE_DRIVER)

| 값 | 설명 |
|----|------|
| `postgres` | pgvector 사용 (기본, 추가 설치 불필요) |
| `qdrant` | Qdrant 벡터 DB (profiles: qdrant 활성화 필요) |
| `elasticsearch_v8` | Elasticsearch 8.x |

```env
RETRIEVE_DRIVER=postgres
```

### 파일 저장소 (STORAGE_TYPE)

```env
# 로컬 파일시스템 (기본)
STORAGE_TYPE=local
LOCAL_STORAGE_BASE_DIR=/opt/weknora/data/files

# MinIO S3 호환 스토리지
# STORAGE_TYPE=minio
# MINIO_ACCESS_KEY_ID=minioadmin
# MINIO_SECRET_ACCESS_KEY=minioadmin
# MINIO_BUCKET_NAME=weknora
```

---

## 7. 업데이트

```bash
# 최신 버전으로 업데이트
bash weknora.sh update

# 특정 버전으로 업데이트
bash weknora.sh update 0.2.7
```

업데이트 순서: 이미지 Pull → Docreader 재시작 → App 재시작 → Frontend 재시작

> 인프라(postgres, redis)는 재시작하지 않아 다운타임을 최소화합니다.

---

## 8. 백업 & 복원

### 백업

```bash
bash weknora.sh backup
# 생성 위치: /opt/weknora/backups/weknora_backup_YYYYMMDD_HHMMSS.tar.gz
```

백업 내용:
- PostgreSQL 전체 덤프 (gzip 압축)
- 업로드 파일 (`data/files/`)
- `.env` 파일

30일 이상된 백업은 자동 삭제됩니다.

### 복원

```bash
bash weknora.sh restore backups/weknora_backup_20260408_120000.tar.gz
```

---

## 9. 선택적 서비스

### MinIO (파일 저장소)

```env
# .env에 추가
STORAGE_TYPE=minio
MINIO_ACCESS_KEY_ID=minioadmin
MINIO_SECRET_ACCESS_KEY=minioadmin
MINIO_BUCKET_NAME=weknora
```

```bash
docker compose --profile minio up -d
```

### Qdrant (벡터 DB)

```env
RETRIEVE_DRIVER=qdrant
QDRANT_COLLECTION=weknora_embeddings
```

```bash
docker compose --profile qdrant up -d
```

### Neo4j (지식 그래프)

```env
NEO4J_ENABLE=true
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your_secure_password
ENABLE_GRAPH_RAG=true
```

`weknora.sh start` 시 `NEO4J_ENABLE=true`이면 자동으로 Neo4j도 시작됩니다.

### Jaeger (분산 추적)

```bash
docker compose --profile jaeger up -d
# UI: http://<서버IP>:16686
```

---

## 10. 문제 해결

### 서비스가 시작되지 않을 때

```bash
# 전체 로그 확인
bash weknora.sh logs

# 특정 서비스 로그
bash weknora.sh logs app
bash weknora.sh logs postgres

# 컨테이너 상태 확인
docker compose ps
docker compose ps app
```

### PostgreSQL 헬스체크 실패

```bash
# DB 컨테이너 직접 확인
docker exec uiscloud_weknora-postgres pg_isready -U postgres
docker logs uiscloud_weknora-postgres --tail=50
```

### API 응답 없음

```bash
bash weknora.sh health
curl -v http://localhost:8080/health
bash weknora.sh logs app
```

### 포트 충돌

```env
# .env에서 포트 변경
APP_PORT=18080
FRONTEND_PORT=18000
```

### 디스크 공간 부족

```bash
bash weknora.sh clean
docker system df
```

### 전체 재설치

```bash
bash weknora.sh stop
docker volume rm $(docker volume ls -q | grep weknora) 2>/dev/null || true
bash weknora.sh install
```

---

## 디렉토리 구조 (설치 후)

```
/opt/weknora/
├── docker-compose.yml          # 서비스 정의
├── docker-compose.prod.yml     # 프로덕션 오버라이드 (자동 적용)
├── weknora.sh                  # 관리 스크립트
├── setup-server.sh             # 서버 초기화 스크립트
├── .env                        # 환경 설정 (비밀번호 포함, 보안 주의!)
├── .env.example                # 설정 템플릿
├── data/
│   └── files/                  # 업로드 파일 (로컬 스토리지)
└── backups/                    # 백업 파일
```

---

## 서비스 URL

| 서비스 | URL |
|--------|-----|
| 웹 UI | `http://<서버IP>:80` |
| 백엔드 API | `http://<서버IP>:8080` |
| API 문서 (Swagger) | `http://<서버IP>:8080/swagger/index.html` |
| MinIO 콘솔 | `http://<서버IP>:9001` (프로필 활성화 시) |
| Neo4j 브라우저 | `http://<서버IP>:7474` (프로필 활성화 시) |
| Jaeger UI | `http://<서버IP>:16686` (프로필 활성화 시) |
