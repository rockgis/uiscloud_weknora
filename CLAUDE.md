# CLAUDE.md

이 파일은 이 저장소에서 작업할 때 Claude Code (claude.ai/code)에게 가이드를 제공합니다.

## 프로젝트 개요

WeKnora는 문서 이해 및 시맨틱 검색을 위한 LLM 기반 RAG (Retrieval-Augmented Generation) 프레임워크입니다. 멀티모달 문서 전처리, 시맨틱 벡터 인덱싱, 하이브리드 검색, LLM 추론을 결합합니다.

## 기술 스택

- **백엔드**: Go 1.24+ (Gin 웹 프레임워크, GORM ORM)
- **프론트엔드**: Vue 3 + Vite + TypeScript + TDesign UI
- **문서 파서**: Python 서비스 (docreader), gRPC 통신
- **데이터베이스**: PostgreSQL (pgvector 포함), Elasticsearch, Qdrant (벡터 저장소)
- **큐**: Asynq (Redis/Memory 백엔드)
- **스토리지**: 로컬 파일시스템, MinIO, 또는 Tencent COS

## 주요 명령어

### 개발 모드 (권장)
```bash
make dev-start      # 인프라 시작 (PostgreSQL, Redis 등)
make dev-app        # 핫 리로드로 백엔드 시작 (dev-start 먼저 실행 필요)
make dev-frontend   # 핫 리로드로 프론트엔드 시작 (dev-start 먼저 실행 필요)
make dev-stop       # 개발 환경 중지
```

### 빌드 & 테스트
```bash
make build          # Go 애플리케이션 빌드
make test           # 전체 테스트 실행
go test -v ./path/to/package  # 특정 패키지 테스트 실행
make lint           # golangci-lint 실행
make fmt            # Go 코드 포맷팅
```

### Docker
```bash
make start-all              # docker-compose로 모든 서비스 시작
make stop-all               # 모든 서비스 중지
docker-compose --profile full up -d  # 전체 기능으로 시작
```

### 데이터베이스 마이그레이션
```bash
make migrate-up             # 대기 중인 마이그레이션 실행
make migrate-down           # 마이그레이션 롤백
make migrate-create name=migration_name  # 새 마이그레이션 생성
```

### API 문서
```bash
make install-swagger  # swag 도구 설치 (최초 1회)
make docs             # Swagger 문서 생성
# http://localhost:8080/swagger/index.html 에서 접근
```

## 아키텍처

```
cmd/server/          # 애플리케이션 진입점
internal/
├── agent/           # 멀티턴 추론을 위한 ReACT 에이전트 구현
├── application/     # 비즈니스 로직 서비스
├── handler/         # HTTP 요청 핸들러 (17개 이상의 핸들러 파일)
├── types/           # 데이터 모델 및 인터페이스 (~30개 타입 파일)
├── middleware/      # 인증 (JWT), CORS, 트레이싱 미들웨어
├── searchutil/      # 하이브리드 검색 (BM25, 벡터 검색, 지식 그래프)
├── stream/          # SSE 스트리밍 응답
├── mcp/             # Model Context Protocol 도구 통합
├── database/        # GORM 데이터베이스 추상화
└── config/          # Viper를 통한 설정 관리

frontend/src/
├── api/             # Axios API 클라이언트
├── views/           # 페이지 컴포넌트
├── components/      # 재사용 가능한 UI 컴포넌트
├── stores/          # Pinia 상태 관리
└── i18n/            # 다국어 지원

docreader/           # Python 문서 파싱 서비스 (gRPC)
client/              # Go SDK 클라이언트 라이브러리
migrations/          # 데이터베이스 마이그레이션 파일
```

## 주요 패턴

- **핸들러 레이어**: `internal/handler/`의 HTTP 핸들러가 REST 엔드포인트에 매핑 (Swagger 어노테이션 포함)
- **서비스 레이어**: `internal/application/`의 비즈니스 로직이 데이터베이스 및 외부 서비스 호출 조율
- **타입 시스템**: 모든 요청/응답 타입이 `internal/types/`에 정의
- **의존성 주입**: `internal/container/`의 컨테이너 패턴
- **이벤트 기반**: `internal/event/`의 이벤트 버스로 비동기 통신

## 코드 스타일

- 최대 줄 길이: 120자
- 린터: govet, revive, lll
- 포맷터: gofmt, gofumpt
- [Go 코드 리뷰 코멘트](https://github.com/golang/go/wiki/CodeReviewComments) 준수

## 커밋 규칙

[Conventional Commits](https://www.conventionalcommits.org/) 준수:
```
feat: 새 기능 추가
fix: 버그 수정
docs: 문서 업데이트
test: 테스트 추가
refactor: 코드 리팩토링
```

## 서비스 URL (개발 환경)

- 웹 UI: http://localhost (프론트엔드) 또는 http://localhost:5173 (Vite 개발 서버)
- 백엔드 API: http://localhost:8080
- Swagger: http://localhost:8080/swagger/index.html
