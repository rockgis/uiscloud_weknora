#!/bin/bash

# uiscloud_weknora Docker Startup Script
# Docker를 사용하여 uiscloud_weknora를 쉽게 실행하기 위한 스크립트

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 스크립트 디렉토리로 이동
cd "$(dirname "$0")"

# 로고 출력
print_logo() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║                                                           ║"
    echo "║  ██╗    ██╗███████╗██╗  ██╗███╗   ██╗ ██████╗ ██████╗  █████╗  ║"
    echo "║  ██║    ██║██╔════╝██║ ██╔╝████╗  ██║██╔═══██╗██╔══██╗██╔══██╗ ║"
    echo "║  ██║ █╗ ██║█████╗  █████╔╝ ██╔██╗ ██║██║   ██║██████╔╝███████║ ║"
    echo "║  ██║███╗██║██╔══╝  ██╔═██╗ ██║╚██╗██║██║   ██║██╔══██╗██╔══██║ ║"
    echo "║  ╚███╔███╔╝███████╗██║  ██╗██║ ╚████║╚██████╔╝██║  ██║██║  ██║ ║"
    echo "║   ╚══╝╚══╝ ╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝ ║"
    echo "║                                                           ║"
    echo "║              LLM-Powered RAG Framework                    ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# 도움말 출력
print_help() {
    echo -e "${GREEN}사용법:${NC}"
    echo "  ./docker-start.sh [명령] [옵션]"
    echo ""
    echo -e "${GREEN}명령:${NC}"
    echo "  start       서비스 시작 (기본 명령)"
    echo "  stop        서비스 중지"
    echo "  restart     서비스 재시작"
    echo "  logs        로그 확인"
    echo "  status      서비스 상태 확인"
    echo "  build       이미지 빌드"
    echo "  clean       컨테이너 및 볼륨 정리"
    echo "  help        도움말 출력"
    echo ""
    echo -e "${GREEN}옵션:${NC}"
    echo "  --full      모든 서비스 포함 (Jaeger, MinIO, Neo4j, Qdrant)"
    echo "  --jaeger    Jaeger 트레이싱 포함"
    echo "  --minio     MinIO 파일 스토리지 포함"
    echo "  --neo4j     Neo4j 그래프 데이터베이스 포함"
    echo "  --qdrant    Qdrant 벡터 데이터베이스 포함"
    echo "  --build     시작 시 이미지 빌드"
    echo ""
    echo -e "${GREEN}예제:${NC}"
    echo "  ./docker-start.sh start             # 기본 서비스 시작"
    echo "  ./docker-start.sh start --full      # 모든 서비스 시작"
    echo "  ./docker-start.sh start --jaeger    # Jaeger 포함 시작"
    echo "  ./docker-start.sh start --build     # 빌드 후 시작"
    echo "  ./docker-start.sh logs app          # 앱 로그 확인"
    echo "  ./docker-start.sh stop              # 서비스 중지"
    echo ""
}

# .env 파일 확인
check_env() {
    if [ ! -f ".env" ]; then
        echo -e "${YELLOW}경고: .env 파일이 없습니다.${NC}"
        echo -e "기본 설정으로 .env 파일을 생성하시겠습니까? (y/n)"
        read -r response
        if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            if [ -f ".env.example" ]; then
                cp .env.example .env
                echo -e "${GREEN}.env 파일이 생성되었습니다.${NC}"
                echo -e "${YELLOW}필요에 따라 .env 파일을 수정하세요.${NC}"
            else
                echo -e "${RED}오류: .env.example 파일을 찾을 수 없습니다.${NC}"
                exit 1
            fi
        else
            echo -e "${RED}오류: .env 파일이 필요합니다.${NC}"
            exit 1
        fi
    fi
}

# config.yaml 확인
check_config() {
    if [ ! -f "config/config.yaml" ]; then
        echo -e "${RED}오류: config/config.yaml 파일이 없습니다.${NC}"
        echo -e "기본 config.yaml 파일이 필요합니다."
        exit 1
    fi
}

# Docker 및 Docker Compose 확인
check_docker() {
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}오류: Docker가 설치되어 있지 않습니다.${NC}"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        echo -e "${RED}오류: Docker 데몬이 실행 중이 아닙니다.${NC}"
        exit 1
    fi
}

# 프로파일 빌드
build_profiles() {
    local profiles=""
    for arg in "$@"; do
        case $arg in
            --full)
                profiles="--profile full"
                ;;
            --jaeger)
                profiles="$profiles --profile jaeger"
                ;;
            --minio)
                profiles="$profiles --profile minio"
                ;;
            --neo4j)
                profiles="$profiles --profile neo4j"
                ;;
            --qdrant)
                profiles="$profiles --profile qdrant"
                ;;
        esac
    done
    echo "$profiles"
}

# 서비스 시작
start_services() {
    echo -e "${GREEN}uiscloud_weknora 서비스를 시작합니다...${NC}"

    local profiles=$(build_profiles "$@")
    local build_flag=""

    for arg in "$@"; do
        if [ "$arg" == "--build" ]; then
            build_flag="--build"
        fi
    done

    docker compose $profiles up -d $build_flag

    echo ""
    echo -e "${GREEN}서비스가 시작되었습니다!${NC}"
    echo ""
    echo -e "프론트엔드:  ${BLUE}http://localhost:${FRONTEND_PORT:-80}${NC}"
    echo -e "백엔드 API:  ${BLUE}http://localhost:${APP_PORT:-8080}${NC}"
    if [[ "$profiles" == *"jaeger"* ]] || [[ "$profiles" == *"full"* ]]; then
        echo -e "Jaeger UI:   ${BLUE}http://localhost:16686${NC}"
    fi
    if [[ "$profiles" == *"minio"* ]] || [[ "$profiles" == *"full"* ]]; then
        echo -e "MinIO Console: ${BLUE}http://localhost:${MINIO_CONSOLE_PORT:-9001}${NC}"
    fi
    if [[ "$profiles" == *"neo4j"* ]] || [[ "$profiles" == *"full"* ]]; then
        echo -e "Neo4j Browser: ${BLUE}http://localhost:7474${NC}"
    fi
    echo ""
}

# 서비스 중지
stop_services() {
    echo -e "${YELLOW}uiscloud_weknora 서비스를 중지합니다...${NC}"
    docker compose --profile full down
    echo -e "${GREEN}서비스가 중지되었습니다.${NC}"
}

# 서비스 재시작
restart_services() {
    stop_services
    start_services "$@"
}

# 로그 확인
show_logs() {
    local service=$1
    if [ -z "$service" ]; then
        docker compose logs -f
    else
        docker compose logs -f "$service"
    fi
}

# 서비스 상태 확인
show_status() {
    echo -e "${GREEN}uiscloud_weknora 서비스 상태:${NC}"
    docker compose ps -a
}

# 이미지 빌드
build_images() {
    echo -e "${GREEN}Docker 이미지를 빌드합니다...${NC}"
    local profiles=$(build_profiles "$@")
    docker compose $profiles build
    echo -e "${GREEN}빌드가 완료되었습니다.${NC}"
}

# 정리
clean_all() {
    echo -e "${YELLOW}경고: 모든 컨테이너와 볼륨이 삭제됩니다.${NC}"
    echo -e "계속하시겠습니까? (y/n)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo -e "${RED}컨테이너와 볼륨을 삭제합니다...${NC}"
        docker compose --profile full down -v
        echo -e "${GREEN}정리가 완료되었습니다.${NC}"
    else
        echo "취소되었습니다."
    fi
}

# 메인 실행
main() {
    print_logo
    check_docker

    # .env 파일에서 변수 로드
    if [ -f ".env" ]; then
        set -a
        source .env
        set +a
    fi

    local command=${1:-start}
    shift || true

    case $command in
        start)
            check_env
            check_config
            start_services "$@"
            ;;
        stop)
            stop_services
            ;;
        restart)
            check_env
            check_config
            restart_services "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        status)
            show_status
            ;;
        build)
            build_images "$@"
            ;;
        clean)
            clean_all
            ;;
        help|--help|-h)
            print_help
            ;;
        *)
            echo -e "${RED}알 수 없는 명령: $command${NC}"
            print_help
            exit 1
            ;;
    esac
}

main "$@"
