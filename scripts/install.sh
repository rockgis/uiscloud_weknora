#!/bin/bash
# uiscloud_weknora 설치/업데이트 스크립트
#
# 사용법:
#   최신 버전 설치:   bash install.sh
#   특정 버전 설치:   bash install.sh 0.2.4
#   설치 경로 지정:   WEKNORA_DIR=/opt/weknora bash install.sh
#
# 원라인 설치 (최신 릴리즈):
#   curl -fsSL https://github.com/rockgis/uiscloud_weknora/releases/latest/download/install.sh | bash

set -euo pipefail

# ---- 설정 ----
GITHUB_REPO="rockgis/uiscloud_weknora"
INSTALL_DIR="${WEKNORA_DIR:-$HOME/weknora}"
# CI 빌드 시 실제 버전으로 치환됨. 플레이스홀더 상태이면 GitHub API로 최신 버전 조회
EMBEDDED_VERSION="__VERSION__"

# ---- 색상 ----
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { printf "%b\n" "${BLUE}[INFO]${NC} $1"; }
log_success() { printf "%b\n" "${GREEN}[✓]${NC} $1"; }
log_error()   { printf "%b\n" "${RED}[✗]${NC} $*" >&2; exit 1; }
log_warning() { printf "%b\n" "${YELLOW}[!]${NC} $1"; }

# ---- 버전 결정: 인수 → 내장 버전 → GitHub API 최신 버전 ----
determine_version() {
    if [ -n "${1:-}" ]; then
        echo "$1"
    elif [ "$EMBEDDED_VERSION" != "__VERSION__" ]; then
        echo "$EMBEDDED_VERSION"
    else
        log_info "GitHub에서 최신 릴리즈 버전 조회 중..."
        local ver
        ver=$(curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" \
            | grep '"tag_name"' \
            | sed 's/.*"tag_name": *"v\([^"]*\)".*/\1/')
        [ -z "$ver" ] && log_error "최신 버전을 가져올 수 없습니다. 네트워크 연결을 확인하세요."
        echo "$ver"
    fi
}

# ---- 사전 요구사항 확인 ----
check_requirements() {
    log_info "사전 요구사항 확인 중..."

    command -v docker >/dev/null 2>&1 \
        || log_error "Docker가 필요합니다. 설치 안내: https://docs.docker.com/get-docker/"

    docker info >/dev/null 2>&1 \
        || log_error "Docker 데몬이 실행 중이지 않습니다. 'sudo systemctl start docker' 또는 Docker Desktop을 시작하세요."

    docker compose version >/dev/null 2>&1 \
        || log_error "Docker Compose v2가 필요합니다. Docker Engine 최신 버전을 설치하세요."

    command -v curl >/dev/null 2>&1 \
        || log_error "curl이 필요합니다. 'apt-get install curl' 또는 'yum install curl'로 설치하세요."

    log_success "사전 요구사항 확인 완료"
}

# ---- 릴리즈 파일 다운로드 ----
download_release_files() {
    local version="$1"
    local base_url="https://github.com/${GITHUB_REPO}/releases/download/v${version}"

    log_info "릴리즈 파일 다운로드 중 (v${version})..."

    curl -fsSL "${base_url}/docker-compose.yml" -o docker-compose.yml \
        || log_error "docker-compose.yml 다운로드 실패. v${version} 릴리즈가 존재하는지 확인하세요: https://github.com/${GITHUB_REPO}/releases"

    curl -fsSL "${base_url}/.env.example" -o .env.example \
        || log_error ".env.example 다운로드 실패"

    log_success "파일 다운로드 완료"
}

# ---- .env 초기 설정 ----
# 반환값: 0 = 기존 .env 존재, 1 = 신규 생성
setup_env() {
    if [ -f ".env" ]; then
        log_success ".env 파일이 이미 존재합니다. 기존 설정을 유지합니다."
        return 0
    fi

    cp .env.example .env

    # 보안: 기본 비밀번호를 랜덤 값으로 교체
    local db_pass redis_pass jwt_secret tenant_key
    db_pass=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c 24 || echo "changeme$(date +%s)")
    redis_pass=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c 24 || echo "changeme$(date +%s)")
    jwt_secret=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c 32 || echo "changeme$(date +%s)")
    tenant_key=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom 2>/dev/null | head -c 32 || echo "changeme$(date +%s)")

    # GNU sed와 BSD sed 모두 지원
    if sed --version >/dev/null 2>&1; then
        # GNU sed (Linux)
        sed -i \
            -e "s|^DB_PASSWORD=.*|DB_PASSWORD=${db_pass}|" \
            -e "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=${redis_pass}|" \
            -e "s|^JWT_SECRET=.*|JWT_SECRET=${jwt_secret}|" \
            -e "s|^TENANT_AES_KEY=.*|TENANT_AES_KEY=${tenant_key}|" \
            .env
    else
        # BSD sed (macOS)
        sed -i '' \
            -e "s|^DB_PASSWORD=.*|DB_PASSWORD=${db_pass}|" \
            -e "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=${redis_pass}|" \
            -e "s|^JWT_SECRET=.*|JWT_SECRET=${jwt_secret}|" \
            -e "s|^TENANT_AES_KEY=.*|TENANT_AES_KEY=${tenant_key}|" \
            .env
    fi

    log_success ".env 파일을 생성했습니다 (비밀번호 자동 생성)"
    log_warning "서비스 시작 전 .env 파일을 검토하고 필요한 항목을 설정하세요:"
    echo ""
    echo "    vi ${INSTALL_DIR}/.env"
    echo ""
    echo "  주요 설정 항목:"
    echo "    GIN_MODE              - release (프로덕션) 또는 debug"
    echo "    OLLAMA_BASE_URL       - Ollama 서버 주소"
    echo "    INIT_LLM_MODEL_NAME   - 기본 LLM 모델명"
    echo "    INIT_EMBEDDING_*      - 임베딩 모델 설정"
    echo "    STORAGE_TYPE          - local / minio / cos"
    echo ""

    return 1
}

# ---- 서비스 시작 ----
start_services() {
    log_info "Docker 이미지 다운로드 중... (첫 설치 시 수 분 소요될 수 있습니다)"
    docker compose pull

    log_info "서비스 시작 중..."
    docker compose up -d

    log_success "서비스가 시작되었습니다"
}

# ---- 완료 메시지 출력 ----
print_status() {
    local version="$1"
    echo ""
    printf "%b\n" "${GREEN}==========================================${NC}"
    printf "%b\n" "${GREEN}  uiscloud_weknora v${version} 설치 완료!${NC}"
    printf "%b\n" "${GREEN}==========================================${NC}"
    echo ""
    echo "  웹 UI:        http://localhost"
    echo "  백엔드 API:   http://localhost:8080"
    echo "  설치 경로:    ${INSTALL_DIR}"
    echo ""
    log_info "유용한 명령어 (${INSTALL_DIR} 에서 실행):"
    echo ""
    echo "  cd ${INSTALL_DIR}"
    echo "  docker compose ps                          # 서비스 상태 확인"
    echo "  docker compose logs -f                     # 실시간 로그"
    echo "  docker compose logs -f app                 # 백엔드 로그만"
    echo "  docker compose down                        # 서비스 중지"
    echo "  docker compose pull && docker compose up -d  # 업데이트"
    echo ""
}

# ---- 메인 ----
main() {
    echo ""
    printf "%b\n" "${GREEN}======================================${NC}"
    printf "%b\n" "${GREEN}  uiscloud_weknora 설치 스크립트${NC}"
    printf "%b\n" "${GREEN}======================================${NC}"
    echo ""

    local version
    version=$(determine_version "${1:-}")

    log_info "설치 버전: v${version}"
    log_info "설치 경로: ${INSTALL_DIR}"
    echo ""

    check_requirements

    mkdir -p "$INSTALL_DIR"
    cd "$INSTALL_DIR"

    download_release_files "$version"

    local is_new_install=false
    setup_env || is_new_install=true

    if [ "$is_new_install" = "true" ]; then
        log_warning ".env 설정 완료 후 아래 명령어로 서비스를 시작하세요:"
        echo ""
        echo "  cd ${INSTALL_DIR} && docker compose up -d"
        echo ""
    else
        start_services
        print_status "$version"
    fi
}

main "$@"
