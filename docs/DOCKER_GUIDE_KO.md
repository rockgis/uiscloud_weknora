# uiscloud_weknora Docker 실행 가이드

이 문서는 Docker를 사용하여 uiscloud_weknora를 실행하는 방법을 설명합니다.

## 목차

- [사전 요구사항](#사전-요구사항)
- [빠른 시작](#빠른-시작)
- [서비스 구성](#서비스-구성)
- [환경 설정](#환경-설정)
- [실행 옵션](#실행-옵션)
- [명령어 레퍼런스](#명령어-레퍼런스)
- [문제 해결](#문제-해결)

## 사전 요구사항

### 필수 소프트웨어

- **Docker**: 20.10 이상
- **Docker Compose**: v2.0 이상 (Docker Desktop에 포함)
- **시스템 메모리**: 최소 8GB RAM 권장

### Docker 설치 확인

```bash
# Docker 버전 확인
docker --version

# Docker Compose 버전 확인
docker compose version

# Docker 데몬 실행 상태 확인
docker info
```

## 빠른 시작

### 1. 환경 설정

프로젝트 루트에 `.env` 파일이 이미 생성되어 있습니다. 필요한 경우 값을 수정하세요.

```bash
# .env 파일 확인 및 수정 (선택사항)
cat .env
```

### 2. 서비스 시작

```bash
# 스크립트를 사용한 시작 (권장)
./docker-start.sh start

# 또는 Docker Compose 직접 사용
docker compose up -d
```

### 3. 서비스 확인

```bash
# 서비스 상태 확인
./docker-start.sh status

# 또는
docker compose ps
```

### 4. 접속

- **프론트엔드 UI**: http://localhost:80
- **백엔드 API**: http://localhost:8080
- **API 문서**: http://localhost:8080/swagger/index.html

## 서비스 구성

### 기본 서비스 (항상 실행)

| 서비스 | 설명 | 포트 |
|--------|------|------|
| `frontend` | Vue.js 기반 웹 UI | 80 |
| `app` | Go 백엔드 API 서버 | 8080 |
| `docreader` | 문서 파싱 gRPC 서비스 | 50051 |
| `postgres` | PostgreSQL + pgvector | 5432 (내부) |
| `redis` | Redis 캐시/스트림 | 6379 (내부) |

### 선택적 서비스 (프로파일로 활성화)

| 서비스 | 프로파일 | 설명 | 포트 |
|--------|----------|------|------|
| `minio` | `minio` | 파일 스토리지 | 9000, 9001 |
| `jaeger` | `jaeger` | 분산 트레이싱 | 16686 |
| `neo4j` | `neo4j` | 그래프 데이터베이스 | 7474, 7687 |
| `qdrant` | `qdrant` | 벡터 데이터베이스 | 6333, 6334 |

## 환경 설정

### 주요 환경 변수

`.env` 파일에서 다음 변수들을 설정할 수 있습니다:

#### 포트 설정
```env
APP_PORT=8080           # 백엔드 API 포트
FRONTEND_PORT=80        # 프론트엔드 포트
DOCREADER_PORT=50051    # 문서 파서 gRPC 포트
```

#### 데이터베이스 설정
```env
DB_USER=postgres
DB_PASSWORD=postgres123!@#
DB_NAME=uiscloud_weknora
```

#### 저장소 설정
```env
STORAGE_TYPE=local              # local, minio, cos
LOCAL_STORAGE_BASE_DIR=/data/files
RETRIEVE_DRIVER=postgres        # postgres, elasticsearch_v8, qdrant
```

#### LLM 설정 (Ollama)
```env
OLLAMA_BASE_URL=http://host.docker.internal:11434
```

### Ollama 연결

로컬에서 Ollama를 실행 중인 경우:

```bash
# Ollama가 localhost:11434에서 실행 중이면 자동으로 연결됩니다.
# Docker 컨테이너에서는 host.docker.internal을 통해 호스트에 접근합니다.
```

## 실행 옵션

### 기본 실행 (필수 서비스만)

```bash
./docker-start.sh start
```

### 전체 서비스 실행

```bash
./docker-start.sh start --full
```

### 선택적 서비스 추가

```bash
# Jaeger 트레이싱 포함
./docker-start.sh start --jaeger

# MinIO 파일 스토리지 포함
./docker-start.sh start --minio

# Neo4j 그래프 DB 포함
./docker-start.sh start --neo4j

# Qdrant 벡터 DB 포함
./docker-start.sh start --qdrant

# 여러 프로파일 조합
./docker-start.sh start --jaeger --minio
```

### 빌드 후 시작

```bash
./docker-start.sh start --build
```

## 명령어 레퍼런스

### docker-start.sh 명령어

| 명령어 | 설명 |
|--------|------|
| `start` | 서비스 시작 |
| `stop` | 서비스 중지 |
| `restart` | 서비스 재시작 |
| `logs [서비스]` | 로그 확인 |
| `status` | 서비스 상태 확인 |
| `build` | 이미지 빌드 |
| `clean` | 컨테이너 및 볼륨 삭제 |
| `help` | 도움말 출력 |

### 사용 예제

```bash
# 서비스 시작
./docker-start.sh start

# 특정 서비스 로그 확인
./docker-start.sh logs app

# 모든 로그 확인
./docker-start.sh logs

# 서비스 상태 확인
./docker-start.sh status

# 서비스 중지
./docker-start.sh stop

# 이미지 빌드
./docker-start.sh build

# 전체 정리 (데이터 포함)
./docker-start.sh clean
```

### Docker Compose 직접 사용

```bash
# 기본 서비스 시작
docker compose up -d

# 전체 서비스 시작
docker compose --profile full up -d

# 특정 프로파일 시작
docker compose --profile jaeger --profile minio up -d

# 로그 확인
docker compose logs -f app

# 서비스 중지
docker compose down

# 볼륨 포함 중지
docker compose down -v
```

## 문제 해결

### 서비스가 시작되지 않는 경우

1. **포트 충돌 확인**
   ```bash
   # 사용 중인 포트 확인
   lsof -i :8080
   lsof -i :80
   ```

2. **로그 확인**
   ```bash
   docker compose logs app
   docker compose logs postgres
   ```

3. **컨테이너 재시작**
   ```bash
   docker compose restart app
   ```

### 데이터베이스 연결 오류

```bash
# PostgreSQL 상태 확인
docker compose logs postgres

# PostgreSQL 재시작
docker compose restart postgres

# 데이터베이스 완전 초기화
docker compose down -v
docker compose up -d
```

### 메모리 부족

Docker Desktop에서 할당된 메모리를 늘리세요:
- Docker Desktop > Settings > Resources > Memory

권장 설정:
- Memory: 8GB 이상
- CPU: 4 cores 이상

### 이미지 빌드 실패

```bash
# 캐시 없이 재빌드
docker compose build --no-cache

# 특정 서비스만 빌드
docker compose build app
```

### Ollama 연결 실패

1. Ollama가 실행 중인지 확인:
   ```bash
   curl http://localhost:11434/api/tags
   ```

2. Docker에서 호스트 접근 확인:
   ```bash
   # 컨테이너 내부에서 테스트
   docker compose exec app curl http://host.docker.internal:11434/api/tags
   ```

## 데이터 지속성

다음 Docker 볼륨이 데이터를 유지합니다:

| 볼륨 | 용도 |
|------|------|
| `postgres-data` | PostgreSQL 데이터 |
| `data-files` | 업로드된 파일 |
| `redis-data` | Redis 데이터 (설정된 경우) |
| `minio_data` | MinIO 파일 (프로파일 사용 시) |
| `neo4j-data` | Neo4j 그래프 (프로파일 사용 시) |
| `qdrant_data` | Qdrant 벡터 (프로파일 사용 시) |
| `jaeger_data` | Jaeger 트레이스 (프로파일 사용 시) |

### 데이터 백업

```bash
# PostgreSQL 데이터 백업
docker compose exec postgres pg_dump -U postgres uiscloud_weknora > backup.sql

# 볼륨 백업 (Docker 볼륨 위치)
docker run --rm -v weknora_postgres-data:/data -v $(pwd):/backup alpine \
  tar cvf /backup/postgres-data.tar /data
```

## 프로덕션 배포

프로덕션 환경에서는 다음 사항을 확인하세요:

1. **보안 설정 변경**
   ```env
   GIN_MODE=release
   JWT_SECRET=<강력한-랜덤-시크릿>
   TENANT_AES_KEY=<32자-암호화-키>
   DB_PASSWORD=<강력한-비밀번호>
   REDIS_PASSWORD=<강력한-비밀번호>
   ```

2. **HTTPS 설정** - 프론트엔드에 SSL 인증서 설정

3. **방화벽 구성** - 필요한 포트만 외부에 노출

4. **모니터링 설정** - Jaeger 프로파일 활성화 권장

5. **백업 정책** - 정기적인 데이터베이스 백업 구성
