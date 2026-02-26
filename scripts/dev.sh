#!/bin/bash
# 개발 환경 시작 스크립트 - 인프라만 시작, app과 frontend는 로컬에서 수동 실행 필요

# 색상 설정
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 색상 없음

# 프로젝트 루트 디렉토리 가져오기
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# 로그 함수
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

# 사용 가능한 Docker Compose 명령 선택
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_BIN="docker"
        DOCKER_COMPOSE_SUBCMD="compose"
        return 0
    fi
    if command -v docker-compose &> /dev/null; then
        if docker-compose version &> /dev/null; then
            DOCKER_COMPOSE_BIN="docker-compose"
            DOCKER_COMPOSE_SUBCMD=""
            return 0
        fi
    fi
    return 1
}

# 도움말 표시
show_help() {
    printf "%b\n" "${GREEN}WeKnora 개발 환경 스크립트${NC}"
    echo "사용법: $0 [명령] [옵션]"
    echo ""
    echo "명령:"
    echo "  start      인프라 서비스 시작 (postgres, redis, docreader)"
    echo "  stop       모든 서비스 중지"
    echo "  restart    모든 서비스 재시작"
    echo "  logs       서비스 로그 확인"
    echo "  status     서비스 상태 확인"
    echo "  app        백엔드 애플리케이션 시작 (로컬 실행)"
    echo "  frontend   프론트엔드 개발 서버 시작 (로컬 실행)"
    echo "  help       이 도움말 표시"
    echo ""
    echo "선택 Profile (start 명령용):"
    echo "  --minio    MinIO 오브젝트 스토리지 시작"
    echo "  --qdrant   Qdrant 벡터 데이터베이스 시작"
    echo "  --neo4j    Neo4j 그래프 데이터베이스 시작"
    echo "  --jaeger   Jaeger 분산 추적 시작"
    echo "  --full     모든 선택 서비스 시작"
    echo ""
    echo "예시:"
    echo "  $0 start                    # 기본 서비스 시작"
    echo "  $0 start --qdrant           # 기본 서비스 + Qdrant 시작"
    echo "  $0 start --qdrant --jaeger  # 기본 서비스 + Qdrant + Jaeger 시작"
    echo "  $0 start --full             # 모든 서비스 시작"
    echo "  $0 app                      # 다른 터미널에서 백엔드 시작"
    echo "  $0 frontend                 # 다른 터미널에서 프론트엔드 시작"
}

# Docker 확인
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker가 설치되지 않았습니다. 먼저 Docker를 설치하세요"
        return 1
    fi

    if ! detect_compose_cmd; then
        log_error "Docker Compose를 찾을 수 없습니다"
        return 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker 서비스가 실행 중이 아닙니다"
        return 1
    fi

    return 0
}

# 인프라 서비스 시작
start_services() {
    log_info "개발 환경 인프라 서비스 시작 중..."

    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"

    # .env 파일 확인
    if [ ! -f ".env" ]; then
        log_error ".env 파일이 없습니다. 먼저 생성하세요"
        return 1
    fi

    # profile 파라미터 파싱
    shift  # "start" 명령 자체 제거
    PROFILES="--profile full"
    ENABLED_SERVICES=""

    while [ $# -gt 0 ]; do
        case "$1" in
            --minio)
                PROFILES="$PROFILES --profile minio"
                ENABLED_SERVICES="$ENABLED_SERVICES minio"
                ;;
            --qdrant)
                PROFILES="$PROFILES --profile qdrant"
                ENABLED_SERVICES="$ENABLED_SERVICES qdrant"
                ;;
            --neo4j)
                PROFILES="$PROFILES --profile neo4j"
                ENABLED_SERVICES="$ENABLED_SERVICES neo4j"
                ;;
            --jaeger)
                PROFILES="$PROFILES --profile jaeger"
                ENABLED_SERVICES="$ENABLED_SERVICES jaeger"
                ;;
            --full)
                PROFILES="--profile full"
                ENABLED_SERVICES="minio qdrant neo4j jaeger"
                break
                ;;
            *)
                log_warning "알 수 없는 파라미터: $1"
                ;;
        esac
        shift
    done

    # 서비스 시작
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml $PROFILES up -d

    if [ $? -eq 0 ]; then
        log_success "인프라 서비스가 시작되었습니다"
        echo ""
        log_info "서비스 접속 주소:"
        echo "  - PostgreSQL:    localhost:5432"
        echo "  - Redis:         localhost:6379"
        echo "  - DocReader:     localhost:50051"

        # 활성화된 profile에 따라 추가 서비스 표시
        if [[ "$ENABLED_SERVICES" == *"minio"* ]]; then
            echo "  - MinIO:         localhost:9000 (Console: localhost:9001)"
        fi
        if [[ "$ENABLED_SERVICES" == *"qdrant"* ]]; then
            echo "  - Qdrant:        localhost:6333 (gRPC: localhost:6334)"
        fi
        if [[ "$ENABLED_SERVICES" == *"neo4j"* ]]; then
            echo "  - Neo4j:         localhost:7474 (Bolt: localhost:7687)"
        fi
        if [[ "$ENABLED_SERVICES" == *"jaeger"* ]]; then
            echo "  - Jaeger:        localhost:16686"
        fi

        echo ""
        log_info "다음 단계:"
        printf "%b\n" "${YELLOW}1. 새 터미널에서 백엔드 실행:${NC} make dev-app"
        printf "%b\n" "${YELLOW}2. 새 터미널에서 프론트엔드 실행:${NC} make dev-frontend"
        return 0
    else
        log_error "서비스 시작 실패"
        return 1
    fi
}

# 서비스 중지
stop_services() {
    log_info "개발 환경 서비스 중지 중..."

    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml down

    if [ $? -eq 0 ]; then
        log_success "모든 서비스가 중지되었습니다"
        return 0
    else
        log_error "서비스 중지 실패"
        return 1
    fi
}

# 서비스 재시작
restart_services() {
    stop_services
    sleep 2
    start_services
}

# 로그 확인
show_logs() {
    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml logs -f
}

# 상태 확인
show_status() {
    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml ps
}

# 백엔드 애플리케이션 시작 (로컬)
start_app() {
    log_info "백엔드 애플리케이션 시작 (로컬 개발 모드)..."

    cd "$PROJECT_ROOT"

    # Go 설치 확인
    if ! command -v go &> /dev/null; then
        log_error "Go가 설치되지 않았습니다"
        return 1
    fi

    # 환경 변수 로드 (set -a로 모든 변수 export 보장)
    if [ -f ".env" ]; then
        log_info ".env 파일 로드 중..."
        set -a
        source .env
        set +a
    else
        log_error ".env 파일이 없습니다. 먼저 설정 파일을 생성하세요"
        return 1
    fi

    # 로컬 개발 환경 변수 설정 (Docker 컨테이너 주소 덮어쓰기)
    export DB_HOST=localhost
    export DOCREADER_ADDR=localhost:50051
    export MINIO_ENDPOINT=localhost:9000
    export REDIS_ADDR=localhost:6379
    export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
    export NEO4J_URI=bolt://localhost:7687
    export QDRANT_HOST=localhost

    # 필수 환경 변수 설정 확인
    if [ -z "$DB_DRIVER" ]; then
        log_error "DB_DRIVER 환경 변수가 설정되지 않았습니다. .env 파일을 확인하세요"
        return 1
    fi

    log_info "환경 변수 설정 완료, 애플리케이션 시작 중..."
    log_info "데이터베이스 주소: $DB_HOST:${DB_PORT:-5432}"

    # Air 설치 확인 (핫 리로드 도구)
    if command -v air &> /dev/null; then
        log_success "Air 감지됨, 핫 리로드 모드로 시작..."
        log_info "Go 코드 수정 시 자동으로 다시 컴파일되고 재시작됩니다"
        air
    else
        log_info "Air가 감지되지 않음, 일반 모드로 시작"
        log_warning "팁: Air를 설치하면 코드 수정 후 자동 재시작 가능"
        log_info "설치 명령: go install github.com/air-verse/air@latest"
        # 애플리케이션 실행
        go run cmd/server/main.go
    fi
}

# 프론트엔드 시작 (로컬)
start_frontend() {
    log_info "프론트엔드 개발 서버 시작 중..."

    cd "$PROJECT_ROOT/frontend"

    # npm 설치 확인
    if ! command -v npm &> /dev/null; then
        log_error "npm이 설치되지 않았습니다"
        return 1
    fi

    # 의존성 설치 확인
    if [ ! -d "node_modules" ]; then
        log_warning "node_modules가 없습니다. 의존성 설치 중..."
        npm install
    fi

    log_info "Vite 개발 서버 시작 중..."
    log_info "프론트엔드 접속 주소: http://localhost:5173"

    # 개발 서버 실행
    npm run dev
}

# 명령 파싱
CMD="${1:-help}"
case "$CMD" in
    start)
        start_services "$@"
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    logs)
        show_logs
        ;;
    status)
        show_status
        ;;
    app)
        start_app
        ;;
    frontend)
        start_frontend
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "알 수 없는 명령: $CMD"
        show_help
        exit 1
        ;;
esac

exit 0
