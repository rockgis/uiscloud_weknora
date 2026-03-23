# WeKnora Jenkins CI/CD 설정 가이드

## 1. 필수 플러그인

Jenkins 관리 > 플러그인 관리에서 아래 플러그인을 설치합니다.

| 플러그인 | 용도 |
|----------|------|
| Pipeline | Jenkinsfile 파이프라인 실행 |
| SSH Agent | SSH 키 기반 배포 서버 접속 |
| Credentials Binding | 자격증명 환경변수 바인딩 |
| Docker Pipeline | Docker 빌드/푸시 통합 |
| Git | 소스 체크아웃 |
| Timestamper | 빌드 로그 타임스탬프 |

---

## 2. Jenkins Credentials 설정

Jenkins 관리 > Credentials > System > Global credentials 에서 아래 항목을 생성합니다.

### 2-1. Docker 레지스트리 URL

| 항목 | 값 |
|------|----|
| Kind | Secret text |
| ID | `weknora-docker-registry-url` |
| Secret | 레지스트리 주소 (예: `registry.example.com` 또는 `docker.io/yourorg`) |

### 2-2. Docker 레지스트리 인증

| 항목 | 값 |
|------|----|
| Kind | Username with password |
| ID | `weknora-docker-credentials` |
| Username | Docker 레지스트리 사용자명 |
| Password | Docker 레지스트리 비밀번호 또는 토큰 |

### 2-3. 배포 서버 SSH 키 (환경별)

각 환경마다 SSH 키를 등록합니다.

| Kind | SSH Username with private key |
|------|-------------------------------|
| ID | `weknora-ssh-dev` |
| Username | `deploy` |
| Private Key | 배포 서버 SSH 개인키 |

동일 방식으로 `weknora-ssh-staging`, `weknora-ssh-prod` 도 등록합니다.

> **배포 서버 설정**: 배포 서버의 `deploy` 계정 `~/.ssh/authorized_keys`에
> 위 개인키에 대응하는 공개키를 등록해야 합니다.

---

## 3. Jenkins 시스템 환경변수 설정

Jenkins 관리 > Configure System > Global properties > Environment variables 에서
배포 대상 서버 주소를 설정합니다.

| 변수명 | 예시 값 | 설명 |
|--------|---------|------|
| `WEKNORA_DEV_HOST` | `192.168.1.10` | 개발 서버 IP/도메인 |
| `WEKNORA_STAGING_HOST` | `staging.example.com` | 스테이징 서버 |
| `WEKNORA_PROD_HOST` | `prod.example.com` | 운영 서버 |

---

## 4. Jenkins Pipeline Job 생성

1. **New Item** 클릭
2. 이름: `weknora-pipeline`, 유형: **Pipeline** 선택
3. **Pipeline** 탭 설정:
   - Definition: `Pipeline script from SCM`
   - SCM: `Git`
   - Repository URL: 프로젝트 Git URL
   - Branch: `*/main`
   - Script Path: `Jenkinsfile`
4. **저장**

---

## 5. 배포 서버 사전 준비

배포할 서버에서 아래 작업을 수행합니다.

```bash
# deploy 계정 생성 (없는 경우)
sudo useradd -m -s /bin/bash deploy
sudo usermod -aG docker deploy

# 배포 디렉토리 생성 (환경별)
sudo mkdir -p /opt/weknora
sudo mkdir -p /opt/weknora-dev
sudo mkdir -p /opt/weknora-staging
sudo chown -R deploy:deploy /opt/weknora*

# 각 배포 디렉토리에 설정 파일 배치
cd /opt/weknora
cp /path/to/project/docker-compose.yml .
cp /path/to/project/.env.example .env
# .env 파일 편집하여 실제 값 입력
nano .env
```

---

## 6. 파이프라인 단계 설명

```
Checkout → Lint → Test → Build Images (병렬) → Push Images → Deploy
```

| 단계 | 설명 | 소요시간 |
|------|------|---------|
| Checkout | 소스 체크아웃 + 버전 추출 | ~30초 |
| Lint | Go lint + Frontend 타입 체크 (병렬) | ~2분 |
| Test | Go 유닛 테스트 + 커버리지 | ~3분 |
| Build App | Go 백엔드 Docker 이미지 빌드 | ~5분 |
| Build Frontend | Vue 3 프론트엔드 이미지 빌드 | ~3분 |
| Build Docreader | Python 파서 이미지 빌드 (선택) | ~20분 |
| Push Images | 레지스트리 푸시 | ~2분 |
| Deploy | SSH 배포 + 헬스체크 | ~3분 |

> **Docreader 빌드**: Python 의존성(PaddleOCR 등)으로 인해 빌드 시간이 깁니다.
> 파서 코드 변경이 없을 때는 `BUILD_DOCREADER=false`(기본값)로 두세요.

---

## 7. 빌드 파라미터

빌드 트리거 시 아래 파라미터를 조정할 수 있습니다.

| 파라미터 | 기본값 | 설명 |
|---------|--------|------|
| `DEPLOY_ENV` | `dev` | 배포 환경 (dev/staging/prod) |
| `BUILD_DOCREADER` | `false` | Docreader 이미지 빌드 여부 |
| `SKIP_TESTS` | `false` | 테스트 건너뛰기 (긴급 배포 시) |
| `DEPLOY_ONLY` | `false` | 빌드 없이 배포만 수행 |

---

## 8. 환경별 docker-compose 오버라이드

배포 서버에 환경별 설정을 분리하려면 오버라이드 파일을 사용합니다.

```
/opt/weknora/
├── docker-compose.yml          # 공통 베이스 (프로젝트 원본)
├── docker-compose.prod.yml     # 운영 환경 오버라이드
└── .env                        # 환경변수
```

`docker-compose.prod.yml` 예시:
```yaml
services:
  app:
    restart: always
    environment:
      GIN_MODE: release
  frontend:
    restart: always
```

---

## 9. 문제 해결

### SSH 연결 실패
```bash
# Jenkins 에이전트에서 SSH 연결 테스트
ssh -i /path/to/key -o StrictHostKeyChecking=no deploy@your-server "echo OK"
```

### Docker 레지스트리 인증 실패
```bash
# 레지스트리 로그인 테스트
docker login your-registry.example.com
```

### 배포 로그 확인
```bash
# 배포 서버에서 서비스 로그 확인
cd /opt/weknora
docker compose logs --tail=100 app
docker compose logs --tail=100 docreader
```

### 서비스 상태 확인
```bash
docker compose ps
curl http://localhost:8080/health
```
