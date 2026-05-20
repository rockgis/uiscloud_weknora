#!/usr/bin/env bash
# ============================================================
#  WeKnora 설치 스크립트
#  - 최초 설치 및 업데이트 모두 지원
#  - 온라인(레지스트리 Pull) / 오프라인(로컬 이미지 로드) 자동 감지
#
#  사용법:
#    bash install.sh              # 대화형 설치
#    bash install.sh --non-interactive  # 비대화형 (.env 사전 설정 필요)
#    bash install.sh --update     # 기존 설치 업데이트
# ============================================================
set -euo pipefail

# ────────────────────────────────────────────
# 색상 및 로그 함수
# ────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

log()     { printf "%b\n" "  ${BLUE}▸${NC} $*"; }
ok()      { printf "%b\n" "  ${GREEN}✔${NC} $*"; }
warn()    { printf "%b\n" "  ${YELLOW}⚠${NC}  $*"; }
fail()    { printf "%b\n" "  ${RED}✘${NC}  $*" >&2; exit 1; }
header()  { echo ""; printf "%b\n" "${BOLD}${CYAN}━━━  $*  ━━━${NC}"; echo ""; }
prompt()  { printf "%b" "  ${YELLOW}?${NC}  $1 "; }

# ────────────────────────────────────────────
# 옵션 파싱
# ────────────────────────────────────────────
INTERACTIVE=true
UPDATE_MODE=false

for arg in "$@"; do
    case $arg in
        --non-interactive|-y) INTERACTIVE=false ;;
        --update|-u)          UPDATE_MODE=true  ;;
        -h|--help)
            echo "사용법: bash install.sh [옵션]"
            echo ""
            echo "  --non-interactive, -y  비대화형 설치 (.env 파일 사전 설정 필요)"
            echo "  --update, -u           기존 설치 업데이트 (이미지 Pull + 재시작)"
            echo "  -h, --help             도움말 출력"
            exit 0
            ;;
    esac
done

# ────────────────────────────────────────────
# 경로 설정
# ────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_BASE="${SCRIPT_DIR}/docker-compose.yml"
COMPOSE_PROD="${SCRIPT_DIR}/docker-compose.prod.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
ENV_EXAMPLE="${SCRIPT_DIR}/.env.example"
IMAGES_DIR="${SCRIPT_DIR}/images"
WEKNORA_SH="${SCRIPT_DIR}/weknora.sh"
VERSION_FILE="${SCRIPT_DIR}/VERSION"
VERSION=$(cat "$VERSION_FILE" 2>/dev/null || echo "unknown")

# docker compose 명령 선택
if docker compose version &>/dev/null 2>&1; then
    DC="docker compose"
elif command -v docker-compose &>/dev/null; then
    DC="docker-compose"
else
    fail "Docker Compose를 찾을 수 없습니다. setup-server.sh를 먼저 실행하세요."
fi

if [ -f "$COMPOSE_PROD" ]; then
    DC_CMD="$DC -f $COMPOSE_BASE -f $COMPOSE_PROD"
else
    DC_CMD="$DC -f $COMPOSE_BASE"
fi

# ────────────────────────────────────────────
# 배너
# ────────────────────────────────────────────
print_banner() {
    echo ""
    printf "%b\n" "${BOLD}${GREEN}"
    echo "  ╔══════════════════════════════════════╗"
    echo "  ║                                      ║"
    echo "  ║      WeKnora RAG Platform            ║"
    printf "  ║      %-35s║\n" "설치 스크립트 v${VERSION}"
    echo "  ║                                      ║"
    echo "  ╚══════════════════════════════════════╝"
    printf "%b\n" "${NC}"
}

# ────────────────────────────────────────────
# 1단계: 사전 요구사항 확인
# ────────────────────────────────────────────
check_requirements() {
    header "1단계: 사전 요구사항 확인"

    # Docker
    if ! command -v docker &>/dev/null; then
        fail "Docker가 설치되어 있지 않습니다.\n       먼저 setup-server.sh를 실행하세요: sudo bash setup-server.sh"
    fi
    ok "Docker: $(docker --version | awk '{print $3}' | tr -d ',')"

    # Docker 데몬
    if ! docker info &>/dev/null 2>&1; then
        fail "Docker 데몬이 실행 중이 아닙니다.\n       실행: sudo systemctl start docker"
    fi
    ok "Docker 데몬: 실행 중"

    # docker compose
    ok "Docker Compose: $(${DC} version --short 2>/dev/null || echo '감지됨')"

    # docker-compose.yml
    [ -f "$COMPOSE_BASE" ] || fail "docker-compose.yml 파일이 없습니다: $COMPOSE_BASE"
    ok "docker-compose.yml: 확인"

    # 디스크 여유 공간 (최소 10GB)
    local avail_kb
    avail_kb=$(df -k "$SCRIPT_DIR" | awk 'NR==2{print $4}')
    local avail_gb=$((avail_kb / 1024 / 1024))
    if [ "$avail_gb" -lt 10 ]; then
        warn "디스크 여유 공간이 부족합니다 (현재: ${avail_gb}GB, 권장: 10GB+)"
    else
        ok "디스크 여유 공간: ${avail_gb}GB"
    fi

    # 메모리 (최소 4GB)
    if [ -f /proc/meminfo ]; then
        local mem_kb
        mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
        local mem_gb=$((mem_kb / 1024 / 1024))
        if [ "$mem_gb" -lt 4 ]; then
            warn "메모리가 부족합니다 (현재: ${mem_gb}GB, 권장: 4GB+)"
        else
            ok "메모리: ${mem_gb}GB"
        fi
    fi
}

# ────────────────────────────────────────────
# 2단계: 환경 설정
# ────────────────────────────────────────────
setup_env() {
    header "2단계: 환경 설정"

    # .env가 이미 있고 업데이트 모드면 유지
    if [ -f "$ENV_FILE" ] && [ "$UPDATE_MODE" = "true" ]; then
        ok ".env 파일 유지 (업데이트 모드)"
        # shellcheck disable=SC1090
        set -a; . "$ENV_FILE"; set +a
        return 0
    fi

    if [ -f "$ENV_FILE" ] && [ "$INTERACTIVE" = "false" ]; then
        ok ".env 파일 사용 (비대화형 모드)"
        # shellcheck disable=SC1090
        set -a; . "$ENV_FILE"; set +a
        return 0
    fi

    # .env 파일 생성 (처음이거나 대화형 모드)
    if [ ! -f "$ENV_FILE" ]; then
        [ -f "$ENV_EXAMPLE" ] || fail ".env.example 파일이 없습니다."
        cp "$ENV_EXAMPLE" "$ENV_FILE"

        # 보안 키 자동 생성
        _gen_secret() { LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c "${1:-32}" || date +%s%N | sha256sum | head -c "${1:-32}"; }
        local db_pass redis_pass jwt aes
        db_pass=$(_gen_secret 24)
        redis_pass=$(_gen_secret 24)
        jwt=$(_gen_secret 48)
        aes=$(_gen_secret 32)

        _sed "s|^GIN_MODE=.*|GIN_MODE=release|" "$ENV_FILE"
        _sed "s|^DB_PASSWORD=.*|DB_PASSWORD=${db_pass}|" "$ENV_FILE"
        _sed "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=${redis_pass}|" "$ENV_FILE"
        _sed "s|^JWT_SECRET=.*|JWT_SECRET=${jwt}|" "$ENV_FILE"
        _sed "s|^TENANT_AES_KEY=.*|TENANT_AES_KEY=${aes}|" "$ENV_FILE"

        ok ".env 생성 완료 (보안 키 자동 생성)"
    else
        ok ".env 파일 존재 — 기존 설정 유지"
    fi

    # 대화형 설정 (LLM 필수 항목)
    if [ "$INTERACTIVE" = "true" ]; then
        _configure_interactive
    fi

    # .env 로드
    # shellcheck disable=SC1090
    set -a; . "$ENV_FILE"; set +a
}

# ────────────────────────────────────────────
# 대화형 LLM 설정
# ────────────────────────────────────────────
_configure_interactive() {
    echo ""
    printf "%b\n" "  ${CYAN}LLM 모델 설정${NC} (Enter 입력 시 기존 값 유지)"
    echo ""

    # 현재 값 로드
    # shellcheck disable=SC1090
    [ -f "$ENV_FILE" ] && set -a && . "$ENV_FILE" && set +a || true

    # LLM API 주소
    local default_url="${INIT_LLM_MODEL_BASE_URL:-https://api.openai.com/v1}"
    prompt "LLM API 주소 [${default_url}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_LLM_MODEL_BASE_URL" "$input"

    # LLM API 키
    local default_key_display="${INIT_LLM_MODEL_API_KEY:+설정됨}"
    prompt "LLM API 키 [${default_key_display:-입력 필요}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_LLM_MODEL_API_KEY" "$input"

    # LLM 모델명
    local default_model="${INIT_LLM_MODEL_NAME:-gpt-4o-mini}"
    prompt "LLM 모델명 [${default_model}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_LLM_MODEL_NAME" "$input"

    echo ""
    printf "%b\n" "  ${CYAN}임베딩 모델 설정${NC}"
    echo ""

    # 임베딩 API 주소 (LLM 주소와 같으면 Enter)
    local current_url; current_url=$(grep "^INIT_LLM_MODEL_BASE_URL=" "$ENV_FILE" | cut -d= -f2- | tr -d '"')
    local default_emb_url="${INIT_EMBEDDING_MODEL_BASE_URL:-$current_url}"
    prompt "임베딩 API 주소 [${default_emb_url}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_EMBEDDING_MODEL_BASE_URL" "$input"

    # 임베딩 API 키 (LLM 키와 같으면 Enter)
    local default_emb_key_display="${INIT_EMBEDDING_MODEL_API_KEY:+설정됨}"
    prompt "임베딩 API 키 [${default_emb_key_display:-LLM 키와 동일하면 Enter}]:"
    read -r input
    if [ -n "$input" ]; then
        _set_env "INIT_EMBEDDING_MODEL_API_KEY" "$input"
    else
        # LLM 키를 임베딩 키로 복사
        local llm_key; llm_key=$(grep "^INIT_LLM_MODEL_API_KEY=" "$ENV_FILE" | cut -d= -f2- | tr -d '"')
        [ -n "$llm_key" ] && _set_env "INIT_EMBEDDING_MODEL_API_KEY" "$llm_key"
    fi

    # 임베딩 모델명
    local default_emb="${INIT_EMBEDDING_MODEL_NAME:-text-embedding-3-small}"
    prompt "임베딩 모델명 [${default_emb}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_EMBEDDING_MODEL_NAME" "$input"

    # 임베딩 차원
    local default_dim="${INIT_EMBEDDING_MODEL_DIMENSION:-1536}"
    prompt "임베딩 차원 수 [${default_dim}]:"
    read -r input
    [ -n "$input" ] && _set_env "INIT_EMBEDDING_MODEL_DIMENSION" "$input"

    echo ""
    printf "%b\n" "  ${CYAN}스토리지 설정${NC}"
    echo ""

    # 파일 저장소 타입
    local current_storage="${STORAGE_TYPE:-local}"
    printf "  ${YELLOW}?${NC}  파일 저장소 타입 [1-3]:\n"
    printf "     ${CYAN}1)${NC} local   - 서버 로컬 디렉토리 (기본)\n"
    printf "     ${CYAN}2)${NC} minio   - MinIO S3 호환 스토리지\n"
    printf "     ${CYAN}3)${NC} cos     - Tencent Cloud COS\n"
    prompt "(현재: ${current_storage}) 선택 [Enter=변경안함]:"
    read -r input
    case "$input" in
        1) _set_env "STORAGE_TYPE" "local"  ;;
        2) _set_env "STORAGE_TYPE" "minio"  ;;
        3) _set_env "STORAGE_TYPE" "cos"    ;;
    esac

    echo ""
    ok "환경 설정 완료"
    echo ""
    printf "%b\n" "  설정 파일: ${CYAN}${ENV_FILE}${NC}"
    printf "%b\n" "  추가 설정이 필요하면 파일을 직접 편집하세요."
    echo ""

    if [ "$INTERACTIVE" = "true" ]; then
        prompt "설치를 계속하시겠습니까? [Y/n]:"
        read -r confirm
        [[ "$confirm" =~ ^[Nn]$ ]] && { log "설치 취소됨. 설정 후 다시 실행하세요."; exit 0; }
    fi
}

# ────────────────────────────────────────────
# 3단계: Docker 이미지 준비
# ────────────────────────────────────────────
prepare_images() {
    header "3단계: Docker 이미지 준비"

    # 오프라인 패키지: images/ 디렉토리에 .tar 파일 있는 경우
    if [ -d "$IMAGES_DIR" ] && ls "${IMAGES_DIR}"/*.tar 2>/dev/null | head -1 &>/dev/null; then
        log "오프라인 패키지 감지 — 로컬 이미지 로드 중..."
        local loaded=0
        for tar_file in "${IMAGES_DIR}"/*.tar; do
            local img_name
            img_name=$(basename "$tar_file" .tar)
            log "  로드 중: ${img_name}..."
            docker load -i "$tar_file" 2>&1 | grep -v "^Loaded image" || true
            ok "  로드 완료: ${img_name}"
            loaded=$((loaded + 1))
        done
        ok "총 ${loaded}개 이미지 로드 완료"
    else
        log "온라인 모드 — 최신 이미지 Pull 중..."
        log "  (최초 실행 시 수 분이 소요될 수 있습니다)"
        echo ""
        # shellcheck disable=SC2086
        $DC_CMD pull
        echo ""
        ok "이미지 Pull 완료"
    fi
}

# ────────────────────────────────────────────
# 4단계: 서비스 시작
# ────────────────────────────────────────────
start_services() {
    header "4단계: 서비스 시작"

    # shellcheck disable=SC1090
    [ -f "$ENV_FILE" ] && set -a && . "$ENV_FILE" && set +a || true

    # 데이터 디렉토리 생성
    mkdir -p "${SCRIPT_DIR}/data/files" "${SCRIPT_DIR}/backups"

    # 인프라 (postgres, redis) 먼저 시작
    log "인프라 서비스 시작 중 (postgres, redis)..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --no-recreate postgres redis
    _wait_healthy postgres 60

    # Neo4j (설정된 경우)
    if [ "${NEO4J_ENABLE:-false}" = "true" ]; then
        log "Neo4j 시작 중..."
        # shellcheck disable=SC2086
        $DC_CMD --profile neo4j up -d --no-recreate neo4j
        ok "Neo4j 시작됨"
    fi

    # Docreader 시작
    log "Docreader(문서 파서) 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate docreader
    _wait_healthy docreader 120

    # 백엔드 App 시작
    log "백엔드 앱 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate app
    _wait_healthy app 90

    # 프론트엔드 시작
    log "프론트엔드 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate frontend

    ok "모든 서비스 시작 완료"
}

# ────────────────────────────────────────────
# 5단계: 헬스체크 및 완료 메시지
# ────────────────────────────────────────────
verify_and_finish() {
    header "5단계: 설치 확인"

    # shellcheck disable=SC1090
    [ -f "$ENV_FILE" ] && set -a && . "$ENV_FILE" && set +a || true
    local app_port="${APP_PORT:-8080}"
    local frontend_port="${FRONTEND_PORT:-80}"

    log "API 헬스체크 대기 중 (최대 30초)..."
    local elapsed=0
    while [ $elapsed -lt 30 ]; do
        if curl -sf --max-time 3 "http://localhost:${app_port}/health" &>/dev/null; then
            ok "백엔드 API 정상 응답"
            break
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    [ $elapsed -ge 30 ] && warn "API 헬스체크 타임아웃 — 로그 확인: bash weknora.sh logs app"

    echo ""
    printf "%b\n" "${BOLD}${GREEN}"
    echo "  ╔══════════════════════════════════════════════╗"
    echo "  ║                                              ║"
    echo "  ║      WeKnora 설치 완료!                     ║"
    echo "  ║                                              ║"
    printf "  ║  %-45s║\n" "버전: v${VERSION}"
    echo "  ║                                              ║"
    printf "  ║  %-45s║\n" "웹 UI:   http://<서버IP>:${frontend_port}"
    printf "  ║  %-45s║\n" "API:     http://<서버IP>:${app_port}"
    printf "  ║  %-45s║\n" "Swagger: http://<서버IP>:${app_port}/swagger/index.html"
    echo "  ║                                              ║"
    echo "  ╚══════════════════════════════════════════════╝"
    printf "%b\n" "${NC}"

    echo ""
    printf "%b\n" "${CYAN}관리 명령어 (${SCRIPT_DIR} 에서 실행):${NC}"
    echo ""
    printf "  %-40s %s\n" "bash weknora.sh status"   "서비스 상태 확인"
    printf "  %-40s %s\n" "bash weknora.sh logs app" "백엔드 로그"
    printf "  %-40s %s\n" "bash weknora.sh update"   "최신 버전으로 업데이트"
    printf "  %-40s %s\n" "bash weknora.sh backup"   "데이터 백업"
    printf "  %-40s %s\n" "bash weknora.sh stop"     "서비스 중지"
    echo ""
}

# ────────────────────────────────────────────
# 업데이트 모드
# ────────────────────────────────────────────
run_update() {
    header "WeKnora 업데이트"

    # shellcheck disable=SC1090
    [ -f "$ENV_FILE" ] && set -a && . "$ENV_FILE" && set +a || true

    log "최신 이미지 Pull 중..."
    # shellcheck disable=SC2086
    $DC_CMD pull

    log "인프라 서비스 유지..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --no-recreate postgres redis

    log "Docreader 재시작..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate docreader
    _wait_healthy docreader 120

    log "백엔드 앱 재시작..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate app
    _wait_healthy app 90

    log "프론트엔드 재시작..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate frontend

    docker image prune -f &>/dev/null || true
    ok "업데이트 완료"

    echo ""
    # shellcheck disable=SC2086
    $DC_CMD ps
}

# ────────────────────────────────────────────
# 내부 헬퍼 함수
# ────────────────────────────────────────────
_wait_healthy() {
    local service="$1"
    local timeout="${2:-90}"
    local elapsed=0

    printf "  ${BLUE}▸${NC} ${service} 준비 대기 중"
    while [ $elapsed -lt $timeout ]; do
        local cid
        cid=$($DC_CMD ps -q "$service" 2>/dev/null || echo "")
        if [ -n "$cid" ]; then
            local health
            health=$(docker inspect --format='{{.State.Health.Status}}' "$cid" 2>/dev/null || echo "")
            if [ "$health" = "healthy" ]; then
                printf " ${GREEN}✔${NC}\n"
                return 0
            fi
            [ "$health" = "unhealthy" ] && printf "\n" && fail "${service} 비정상 — 로그 확인: bash weknora.sh logs ${service}"
        fi
        printf "."
        sleep 3
        elapsed=$((elapsed + 3))
    done
    printf "\n"
    fail "${service} 헬스체크 타임아웃 (${timeout}초)"
}

_sed() {
    local pattern="$1"; local file="$2"
    if sed --version &>/dev/null 2>&1; then
        sed -i "$pattern" "$file"
    else
        sed -i '' "$pattern" "$file"
    fi
}

_set_env() {
    local key="$1"; local val="$2"
    if grep -q "^${key}=" "$ENV_FILE" 2>/dev/null; then
        _sed "s|^${key}=.*|${key}=${val}|" "$ENV_FILE"
    else
        echo "${key}=${val}" >> "$ENV_FILE"
    fi
}

# ────────────────────────────────────────────
# 메인
# ────────────────────────────────────────────
main() {
    print_banner

    if [ "$UPDATE_MODE" = "true" ]; then
        check_requirements
        run_update
        exit 0
    fi

    check_requirements
    setup_env
    prepare_images
    start_services
    verify_and_finish
}

main "$@"
