# WeKnora 서버 배포 구성 문서

> **대상 서버:** 192.168.0.210 (gx10-ai-210)
> **배포 버전:** v0.2.6
> **최종 업데이트:** 2026-04-08

---

## 1. 서버 환경

| 항목 | 내용 |
|------|------|
| OS | Ubuntu (Linux 6.17.0-1014-nvidia) |
| 아키텍처 | aarch64 (ARM64) |
| GPU | NVIDIA GB10 (통합 메모리 121.6 GiB) |
| CUDA | 13.0 / CUDA Compute 12.1 |
| NVIDIA 드라이버 | 580.142 |
| 디스크 | 916G 중 266G 사용 (31%) |
| 접속 사용자 | `hypermakina` |
| 배포 경로 | `~/weknora/` |

---

## 2. 서비스 구성

### 실행 중인 컨테이너

| 컨테이너 이름 | 이미지 | 호스트 포트 | 역할 |
|--------------|--------|------------|------|
| `weknora-ollama` | `ollama/ollama:latest` | 내부 전용 (11434) | LLM/임베딩 추론 (GPU) |
| `uiscloud_weknora-app` | `weknora-app:0.2.6` | `0.0.0.0:8083→8080` | 백엔드 API |
| `uiscloud_weknora-frontend` | `weknora-ui:0.2.6` | `0.0.0.0:8084→80` | 프론트엔드 UI |
| `uiscloud_weknora-docreader` | `weknora-docreader:0.2.6` | `0.0.0.0:50051→50051` | 문서 파싱 (gRPC) |
| `uiscloud_weknora-postgres` | `paradedb/paradedb:v0.18.9-pg17` | 내부 전용 (5432) | DB (pgvector/ParadeDB) |
| `uiscloud_weknora-redis` | `redis:7.0-alpine` | 내부 전용 (6379) | 큐 / 캐시 |

> **포트 변경 이유:** 서버에 이미 `open-webui`(8080), `hymakina-ai-rag`(8081), `onyx`(8082)가 기동 중이어서 8083/8084로 변경.

### 서비스 접속 URL

| 서비스 | URL |
|--------|-----|
| WeKnora UI | http://192.168.0.210:8084 |
| WeKnora API | http://192.168.0.210:8083 |
| API 헬스체크 | http://192.168.0.210:8083/health |
| Swagger 문서 | http://192.168.0.210:8083/swagger/index.html |

---

## 3. Docker 네트워크

```
네트워크명: weknora_uiscloud_weknora-network
드라이버:   bridge
서브넷:     172.21.0.0/16
게이트웨이: 172.21.0.1
```

> **배경:** 컨테이너에서 호스트의 Ollama(systemd)로 직접 연결 시도 시 UFW 방화벽의 FORWARD 차단으로 실패함.  
> **해결:** Ollama를 동일 Docker 네트워크에 컨테이너로 추가하여 서비스 이름(`ollama`)으로 내부 통신.

---

## 4. NVIDIA GPU 설정

### Docker NVIDIA Runtime (`/etc/docker/daemon.json`)

```json
{
  "runtimes": {
    "nvidia": {
      "args": [],
      "path": "nvidia-container-runtime"
    }
  },
  "default-runtime": "runc"
}
```

> `nvidia-ctk`는 `/usr/bin/nvidia-ctk`에 설치됨 (`libnvidia-container-tools 1.19.0-1`).

### Ollama GPU 사용 확인

```
GPU: NVIDIA GB10
VRAM: 121.6 GiB (통합 메모리)
CUDA Compute: 12.1
프로세스: /usr/bin/ollama — GPU 메모리 28,270 MiB 점유 (모델 로드 시)
```

---

## 5. Ollama 모델 목록

모델 파일 경로: `/usr/share/ollama/.ollama/models/` (컨테이너에 `/root/.ollama`로 마운트)

| 모델 | 크기 | 용도 |
|------|------|------|
| `gemma4:31b` | 19 GB | **기본 LLM** (WeKnora 초기 설정) |
| `exaone3.5:32b` | 19 GB | 대형 LLM |
| `qwen3:32b` | 20 GB | 대형 LLM |
| `gpt-oss:latest` | 13 GB | LLM |
| `deepseek-r1:14b` | 9.0 GB | 추론 특화 LLM |
| `exaone3.5:7.8b` | 4.8 GB | 소형 LLM |
| `qwen2.5:7b` | 4.7 GB | 소형 LLM |
| `llama3.1:8b` | 4.9 GB | 소형 LLM |
| `qwen3:4b` | 2.5 GB | 소형 LLM |
| `bge-m3:latest` | 1.2 GB | **임베딩 모델** (WeKnora 초기 설정) |
| `qwen3-embedding:latest` | 4.7 GB | 임베딩 모델 |

---

## 6. 환경 설정 파일 (`~/weknora/.env` 주요 항목)

```dotenv
# 포트 (기본값 8080/80에서 변경)
APP_PORT=8083
FRONTEND_PORT=8084

# Ollama 연결 (컨테이너 서비스명 사용)
OLLAMA_BASE_URL=http://ollama:11434

# 초기 LLM 모델
INIT_LLM_MODEL_NAME=gemma4:31b
INIT_LLM_MODEL_BASE_URL=http://ollama:11434/v1
INIT_LLM_MODEL_API_KEY=ollama

# 초기 임베딩 모델
INIT_EMBEDDING_MODEL_NAME=bge-m3
INIT_EMBEDDING_MODEL_BASE_URL=http://ollama:11434/v1
INIT_EMBEDDING_MODEL_API_KEY=ollama

# 스토리지
STORAGE_TYPE=local
LOCAL_STORAGE_BASE_DIR=./data/files

# Neo4j / 지식 그래프 (비활성화 — 컨테이너 미설치)
NEO4J_ENABLE=false
ENABLE_GRAPH_RAG=false
```

---

## 7. Docker Compose 파일 구조

```
~/weknora/
├── docker-compose.yml          # 기본 서비스 정의 (원본)
├── docker-compose.local.yml    # 로컬 빌드 이미지 + Ollama 오버라이드 ★
├── docker-compose.prod.yml     # → docker-compose.local.yml 심볼릭 링크
├── .env                        # 실제 운영 환경변수
├── .env.example                # 환경변수 템플릿
├── config/
│   └── config.yaml             # 앱 상세 설정 (RAG 파라미터, 프롬프트 등)
├── install.sh                  # 최초 설치 스크립트
├── weknora.sh                  # 서비스 관리 CLI
└── setup-server.sh             # 서버 OS 초기화 스크립트
```

### `docker-compose.local.yml` 전체 내용

```yaml
# 로컬 빌드 이미지 사용 오버라이드
services:
  app:
    image: weknora-app:0.2.6
    build: ~
    restart: always
    environment:
      - GIN_MODE=release
    logging:
      driver: json-file
      options:
        max-size: 50m
        max-file: '5'

  frontend:
    image: weknora-ui:0.2.6
    build: ~
    restart: always
    logging:
      driver: json-file
      options:
        max-size: 10m
        max-file: '3'

  docreader:
    image: weknora-docreader:0.2.6
    build: ~
    restart: always
    logging:
      driver: json-file
      options:
        max-size: 50m
        max-file: '5'

  postgres:
    restart: always

  redis:
    restart: always

  ollama:
    image: ollama/ollama:latest
    container_name: weknora-ollama
    restart: always
    runtime: nvidia                          # NVIDIA GPU 런타임
    networks:
      - uiscloud_weknora-network
    volumes:
      - /usr/share/ollama/.ollama:/root/.ollama   # 호스트 모델 공유
    environment:
      - OLLAMA_HOST=0.0.0.0:11434
      - OLLAMA_MODELS=/root/.ollama/models
      - NVIDIA_VISIBLE_DEVICES=all
      - NVIDIA_DRIVER_CAPABILITIES=compute,utility
    logging:
      driver: json-file
      options:
        max-size: 50m
        max-file: '3'

networks:
  uiscloud_weknora-network:
    external: true
    name: weknora_uiscloud_weknora-network   # 기존 네트워크 참조
```

---

## 8. Docker 볼륨

| 볼륨명 | 용도 |
|--------|------|
| `weknora_postgres-data` | PostgreSQL 데이터 영속화 |
| `weknora_data-files` | 업로드 문서 파일 저장 |

---

## 9. 서비스 관리 명령

모든 명령은 `~/weknora/` 경로에서 실행합니다.

### `weknora.sh` CLI (권장)

```bash
bash weknora.sh status          # 전체 서비스 상태 확인
bash weknora.sh start           # 서비스 시작
bash weknora.sh stop            # 서비스 중지
bash weknora.sh restart         # 서비스 재시작
bash weknora.sh logs app        # 앱 로그 확인
bash weknora.sh logs ollama     # Ollama 로그 확인
bash weknora.sh health          # API 헬스체크
bash weknora.sh backup          # 데이터 백업
bash weknora.sh update [tag]    # 이미지 업데이트
bash weknora.sh clean           # 미사용 이미지 정리
```

### Docker Compose 직접 명령

```bash
# 서비스 상태 확인
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'

# 전체 서비스 재시작
docker compose -f docker-compose.yml -f docker-compose.local.yml up -d

# 특정 서비스만 재시작 (환경변수 변경 시 반드시 up -d 사용)
docker compose -f docker-compose.yml -f docker-compose.local.yml up -d app

# 로그 스트리밍
docker logs -f weknora-ollama
docker logs -f uiscloud_weknora-app

# Ollama GPU 상태 확인
docker exec weknora-ollama nvidia-smi

# Ollama 모델 목록 확인
docker exec weknora-ollama ollama list
```

### 환경변수 변경 후 적용 방법

> `docker restart`는 `.env` 변경을 반영하지 않습니다. 반드시 `up -d`를 사용하세요.

```bash
vi ~/weknora/.env
cd ~/weknora
docker compose -f docker-compose.yml -f docker-compose.local.yml up -d <서비스명>
```

---

## 10. 배포 과정 이력

### 초기 배포 (2026-04-08)

1. **소스 전송**: `rsync`로 175MB / 579 파일을 `~/weknora-src/`로 전송
2. **이미지 빌드**: 서버에서 arm64 플랫폼으로 직접 빌드
   ```bash
   # 빌드된 이미지
   weknora-app:0.2.6
   weknora-ui:0.2.6
   weknora-docreader:0.2.6
   ```
3. **서비스 기동**: docker-compose로 6개 서비스 시작
4. **포트 충돌 해결**: 기존 open-webui(8080) 충돌 → `APP_PORT=8083`, `FRONTEND_PORT=8084`로 변경
5. **Neo4j 비활성화**: neo4j 컨테이너 미설치 → `NEO4J_ENABLE=false`

### Ollama 연결 설정 (2026-04-08)

**문제:** 호스트 systemd Ollama(11434)에 컨테이너에서 접근 불가
- `host.docker.internal` → `172.17.0.1` (기본 브리지 게이트웨이)
- 실제 weknora 컨테이너 네트워크: `172.21.0.0/16`
- UFW 방화벽이 FORWARD/INPUT 트래픽 차단

**해결:** Ollama를 Docker 컨테이너로 weknora 네트워크에 추가
- 호스트 모델 경로 `/usr/share/ollama/.ollama` 볼륨 마운트로 모델 재다운로드 불필요
- NVIDIA runtime 설정 후 GPU 모드 활성화

---

## 11. 트러블슈팅

### Ollama GPU 미인식 시

```bash
# daemon.json 확인
cat /etc/docker/daemon.json

# nvidia runtime 등록 확인
docker info | grep -i runtime

# 재설정 필요 시 (sudo 필요)
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker

# ollama 컨테이너 재생성
cd ~/weknora
docker compose -f docker-compose.yml -f docker-compose.local.yml up -d ollama
```

### 앱 시작 실패 시

```bash
# 로그 확인
docker logs uiscloud_weknora-app --tail=50

# config.yaml 마운트 확인
docker exec uiscloud_weknora-app cat /app/config/config.yaml | head -5

# DB 연결 확인
docker exec uiscloud_weknora-postgres pg_isready -U postgres
```

### 포트 충돌 확인

```bash
ss -tlnp | grep -E '808[0-9]'
docker ps --format '{{.Names}}\t{{.Ports}}'
```
