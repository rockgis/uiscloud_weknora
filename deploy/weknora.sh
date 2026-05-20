#!/usr/bin/env bash
# WeKnora 서버 관리 스크립트
#
# 사용법:
#   bash weknora.sh <명령> [옵션]
#
# 명령:
#   install          최초 설치 (.env 생성, 이미지 Pull, 서비스 시작)
#   start            서비스 시작
#   stop             서비스 중지
#   restart          서비스 재시작
#   update [tag]     이미지 업데이트 및 재시작 (tag 미지정 시 latest)
#   status           서비스 상태 확인
#   logs [서비스]    로그 출력 (서비스 미지정 시 전체)
#   backup           데이터 백업
#   restore <파일>   백업에서 복원
#   health           헬스체크 (API 응답 확인)
#   clean            중지된 컨테이너 및 미사용 이미지 정리

set -euo pipefail

# ─────────────────────────────────────────────────
# 색상 / 로그
# ─────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(printf "%b" "${BLUE}INFO${NC}")  $*"; }
ok()   { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(printf "%b" "${GREEN}OK${NC}")    $*"; }
warn() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(printf "%b" "${YELLOW}WARN${NC}")  $*"; }
fail() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $(printf "%b" "${RED}FAIL${NC}")  $*" >&2; exit 1; }

# ─────────────────────────────────────────────────
# 경로 / 설정
# ─────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_BASE="${SCRIPT_DIR}/docker-compose.yml"
COMPOSE_PROD="${SCRIPT_DIR}/docker-compose.prod.yml"
ENV_FILE="${SCRIPT_DIR}/.env"
ENV_EXAMPLE="${SCRIPT_DIR}/.env.example"
BACKUP_DIR="${SCRIPT_DIR}/backups"
APP_PORT="${APP_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-80}"

# docker compose 또는 docker-compose 선택
if docker compose version &>/dev/null 2>&1; then
    DC="docker compose"
elif command -v docker-compose &>/dev/null; then
    DC="docker-compose"
else
    fail "Docker Compose를 찾을 수 없습니다. Docker CE를 설치하세요."
fi

# 프로덕션 오버라이드 파일이 있으면 자동 적용
if [ -f "$COMPOSE_PROD" ]; then
    DC_CMD="$DC -f $COMPOSE_BASE -f $COMPOSE_PROD"
else
    DC_CMD="$DC -f $COMPOSE_BASE"
fi

# ─────────────────────────────────────────────────
# 헬퍼: .env 파일 로드
# ─────────────────────────────────────────────────
load_env() {
    [ -f "$ENV_FILE" ] && set -a && . "$ENV_FILE" && set +a
    APP_PORT="${APP_PORT:-8080}"
    FRONTEND_PORT="${FRONTEND_PORT:-80}"
}

# ─────────────────────────────────────────────────
# 헬퍼: Neo4j 프로필 인수
# ─────────────────────────────────────────────────
neo4j_profile_args() {
    load_env
    if [ "${NEO4J_ENABLE:-false}" = "true" ]; then
        echo "--profile neo4j"
    else
        echo ""
    fi
}

# ─────────────────────────────────────────────────
# 헬퍼: 컨테이너 헬스 대기
# ─────────────────────────────────────────────────
wait_healthy() {
    local service="$1"
    local timeout="${2:-90}"
    local elapsed=0

    log "${service} 헬스체크 대기 중 (최대 ${timeout}초)..."
    while [ $elapsed -lt $timeout ]; do
        local cid
        cid=$($DC_CMD ps -q "$service" 2>/dev/null || echo "")
        if [ -n "$cid" ]; then
            local health
            health=$(docker inspect --format='{{.State.Health.Status}}' "$cid" 2>/dev/null || echo "")
            [ "$health" = "healthy" ] && { ok "${service} 정상 (healthy)"; return 0; }
            [ "$health" = "unhealthy" ] && { fail "${service} 비정상 (unhealthy) — 로그 확인: bash weknora.sh logs ${service}"; }
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    fail "${service} 헬스체크 타임아웃 (${timeout}초)"
}

# ─────────────────────────────────────────────────
# 명령: install — 최초 설치
# ─────────────────────────────────────────────────
cmd_install() {
    echo ""
    printf "%b\n" "${GREEN}====================================${NC}"
    printf "%b\n" "${GREEN}  WeKnora 최초 설치${NC}"
    printf "%b\n" "${GREEN}====================================${NC}"
    echo ""

    # 사전 요구사항 확인
    command -v docker &>/dev/null || fail "Docker가 설치되어 있지 않습니다. deploy/setup-server.sh를 먼저 실행하세요."
    docker info &>/dev/null       || fail "Docker 데몬이 실행 중이 아닙니다."
    [ -f "$COMPOSE_BASE" ]        || fail "docker-compose.yml 파일이 없습니다: $COMPOSE_BASE"

    # .env 설정
    if [ -f "$ENV_FILE" ]; then
        warn ".env 파일이 이미 존재합니다. 기존 설정을 유지합니다."
    else
        [ -f "$ENV_EXAMPLE" ] || fail ".env.example 파일이 없습니다."
        cp "$ENV_EXAMPLE" "$ENV_FILE"

        # 보안 키 자동 생성
        local db_pass redis_pass jwt aes
        db_pass=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c 24 || date +%s | sha256sum | head -c 24)
        redis_pass=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c 24 || date +%s | sha256sum | head -c 24)
        jwt=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c 48 || date +%s | sha256sum | head -c 48)
        aes=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom 2>/dev/null | head -c 32 || date +%s | sha256sum | head -c 32)

        # GIN_MODE를 release로 변경
        _sed "s|^GIN_MODE=.*|GIN_MODE=release|"       "$ENV_FILE"
        _sed "s|^DB_PASSWORD=.*|DB_PASSWORD=${db_pass}|"     "$ENV_FILE"
        _sed "s|^REDIS_PASSWORD=.*|REDIS_PASSWORD=${redis_pass}|" "$ENV_FILE"
        _sed "s|^JWT_SECRET=.*|JWT_SECRET=${jwt}|"         "$ENV_FILE"
        _sed "s|^TENANT_AES_KEY=.*|TENANT_AES_KEY=${aes}|"   "$ENV_FILE"

        ok ".env 파일 생성 완료 (보안 키 자동 생성)"
        echo ""
        printf "%b\n" "${YELLOW}서비스 시작 전 .env를 검토하세요:${NC}"
        echo "    vi ${ENV_FILE}"
        echo ""
        echo "  필수 설정 항목:"
        echo "    INIT_LLM_MODEL_NAME       - LLM 모델명"
        echo "    INIT_LLM_MODEL_BASE_URL   - LLM API 주소"
        echo "    INIT_LLM_MODEL_API_KEY    - LLM API 키"
        echo "    INIT_EMBEDDING_MODEL_NAME - 임베딩 모델명"
        echo "    STORAGE_TYPE              - local / minio / cos"
        echo ""
        printf "%b\n" "${YELLOW}.env 편집 후 다시 실행:  bash weknora.sh install${NC}"
        return 0
    fi

    load_env

    # 이미지 Pull
    log "Docker 이미지 다운로드 중..."
    # shellcheck disable=SC2086
    $DC_CMD pull

    # 인프라 먼저 시작
    log "인프라 서비스 시작 중 (postgres, redis)..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --no-recreate postgres redis
    wait_healthy postgres 60

    # Neo4j (설정된 경우)
    if [ "${NEO4J_ENABLE:-false}" = "true" ]; then
        log "Neo4j 시작 중..."
        $DC_CMD --profile neo4j up -d --no-recreate neo4j
    fi

    # Docreader 시작
    log "Docreader 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate docreader
    wait_healthy docreader 120

    # App 시작
    log "백엔드 앱 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate app
    wait_healthy app 90

    # Frontend 시작
    log "프론트엔드 시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate frontend

    ok "모든 서비스 시작 완료"
    _print_access_info
}

# ─────────────────────────────────────────────────
# 명령: start
# ─────────────────────────────────────────────────
cmd_start() {
    load_env
    log "서비스 시작 중..."
    PROFILE_ARGS=$(neo4j_profile_args)
    # shellcheck disable=SC2086
    $DC_CMD $PROFILE_ARGS up -d
    ok "서비스 시작 완료"
    _print_access_info
}

# ─────────────────────────────────────────────────
# 명령: stop
# ─────────────────────────────────────────────────
cmd_stop() {
    log "서비스 중지 중..."
    # shellcheck disable=SC2086
    $DC_CMD down --remove-orphans
    ok "서비스 중지 완료"
}

# ─────────────────────────────────────────────────
# 명령: restart
# ─────────────────────────────────────────────────
cmd_restart() {
    cmd_stop
    sleep 2
    cmd_start
}

# ─────────────────────────────────────────────────
# 명령: update [tag]
# ─────────────────────────────────────────────────
cmd_update() {
    local tag="${1:-latest}"
    load_env

    log "업데이트 시작 — 태그: ${tag}"

    # WEKNORA_IMAGE_TAG 업데이트
    if grep -q "^WEKNORA_IMAGE_TAG=" "$ENV_FILE" 2>/dev/null; then
        _sed "s|^WEKNORA_IMAGE_TAG=.*|WEKNORA_IMAGE_TAG=${tag}|" "$ENV_FILE"
    else
        echo "WEKNORA_IMAGE_TAG=${tag}" >> "$ENV_FILE"
    fi
    export WEKNORA_IMAGE_TAG="$tag"

    # 이미지 Pull
    log "최신 이미지 Pull 중..."
    # shellcheck disable=SC2086
    $DC_CMD pull

    # 인프라 유지 후 앱 재시작
    log "인프라 서비스 유지 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --no-recreate postgres redis

    log "Docreader 재시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate docreader
    wait_healthy docreader 120

    log "백엔드 앱 재시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate app
    wait_healthy app 90

    log "프론트엔드 재시작 중..."
    # shellcheck disable=SC2086
    $DC_CMD up -d --force-recreate frontend

    # 이미지 정리
    docker image prune -f &>/dev/null || true

    ok "업데이트 완료 — 태그: ${tag}"
    cmd_status
}

# ─────────────────────────────────────────────────
# 명령: status
# ─────────────────────────────────────────────────
cmd_status() {
    load_env
    echo ""
    printf "%b\n" "${BLUE}=== WeKnora 서비스 상태 ===${NC}"
    echo ""
    # shellcheck disable=SC2086
    $DC_CMD ps
    echo ""
    cmd_health
}

# ─────────────────────────────────────────────────
# 명령: logs [서비스명]
# ─────────────────────────────────────────────────
cmd_logs() {
    local service="${1:-}"
    if [ -n "$service" ]; then
        # shellcheck disable=SC2086
        $DC_CMD logs --tail=100 -f "$service"
    else
        # shellcheck disable=SC2086
        $DC_CMD logs --tail=50 -f
    fi
}

# ─────────────────────────────────────────────────
# 명령: backup
# ─────────────────────────────────────────────────
cmd_backup() {
    load_env
    local timestamp
    timestamp=$(date '+%Y%m%d_%H%M%S')
    local backup_name="weknora_backup_${timestamp}"
    local backup_path="${BACKUP_DIR}/${backup_name}"

    mkdir -p "$backup_path"

    log "백업 시작: $backup_path"

    # 1. PostgreSQL 백업
    local pg_container
    pg_container=$($DC_CMD ps -q postgres 2>/dev/null || echo "")
    if [ -n "$pg_container" ]; then
        log "PostgreSQL 백업 중..."
        docker exec "$pg_container" \
            pg_dump -U "${DB_USER:-postgres}" "${DB_NAME:-uiscloud_weknora}" \
            | gzip > "${backup_path}/postgres.sql.gz"
        ok "PostgreSQL 백업 완료: postgres.sql.gz"
    else
        warn "PostgreSQL 컨테이너가 실행 중이 아님 — DB 백업 건너뜀"
    fi

    # 2. 업로드 파일 백업 (로컬 스토리지)
    local data_dir="${SCRIPT_DIR}/data/files"
    if [ -d "$data_dir" ] && [ "$(ls -A "$data_dir" 2>/dev/null)" ]; then
        log "업로드 파일 백업 중..."
        tar czf "${backup_path}/files.tar.gz" -C "${SCRIPT_DIR}/data" files/
        ok "파일 백업 완료: files.tar.gz"
    fi

    # 3. .env 백업
    cp "$ENV_FILE" "${backup_path}/.env.bak"
    ok ".env 백업 완료"

    # 4. 아카이브 생성
    local archive_file="${BACKUP_DIR}/${backup_name}.tar.gz"
    tar czf "$archive_file" -C "$BACKUP_DIR" "$backup_name/"
    rm -rf "$backup_path"

    ok "백업 완료: $archive_file"
    echo "   크기: $(du -sh "$archive_file" | cut -f1)"

    # 오래된 백업 정리 (30일 이상)
    find "$BACKUP_DIR" -name "weknora_backup_*.tar.gz" -mtime +30 -delete 2>/dev/null || true
}

# ─────────────────────────────────────────────────
# 명령: restore <백업파일>
# ─────────────────────────────────────────────────
cmd_restore() {
    local archive="${1:-}"
    [ -z "$archive" ] && fail "사용법: bash weknora.sh restore <백업파일.tar.gz>"
    [ -f "$archive" ] || fail "백업 파일을 찾을 수 없습니다: $archive"

    load_env

    warn "복원 시 현재 데이터가 덮어씌워집니다!"
    read -r -p "계속 진행하시겠습니까? [y/N] " confirm
    [[ "$confirm" =~ ^[Yy]$ ]] || { log "복원 취소됨."; return 0; }

    local restore_tmp
    restore_tmp=$(mktemp -d)
    tar xzf "$archive" -C "$restore_tmp"
    local backup_dir
    backup_dir=$(find "$restore_tmp" -maxdepth 1 -type d -name "weknora_backup_*" | head -1)
    [ -d "$backup_dir" ] || fail "백업 아카이브 형식이 올바르지 않습니다."

    # PostgreSQL 복원
    if [ -f "${backup_dir}/postgres.sql.gz" ]; then
        log "PostgreSQL 복원 중..."
        local pg_container
        pg_container=$($DC_CMD ps -q postgres 2>/dev/null || echo "")
        [ -n "$pg_container" ] || fail "PostgreSQL 컨테이너가 실행 중이 아닙니다."

        gunzip -c "${backup_dir}/postgres.sql.gz" | \
            docker exec -i "$pg_container" \
            psql -U "${DB_USER:-postgres}" "${DB_NAME:-uiscloud_weknora}"
        ok "PostgreSQL 복원 완료"
    fi

    # 파일 복원
    if [ -f "${backup_dir}/files.tar.gz" ]; then
        log "업로드 파일 복원 중..."
        tar xzf "${backup_dir}/files.tar.gz" -C "${SCRIPT_DIR}/data/"
        ok "파일 복원 완료"
    fi

    rm -rf "$restore_tmp"
    ok "복원 완료 — 서비스를 재시작하세요: bash weknora.sh restart"
}

# ─────────────────────────────────────────────────
# 명령: health
# ─────────────────────────────────────────────────
cmd_health() {
    load_env
    local api_url="http://localhost:${APP_PORT}/health"

    if curl -sf --max-time 5 "$api_url" &>/dev/null; then
        ok "백엔드 API 정상 응답: $api_url"
    else
        warn "백엔드 API 응답 없음: $api_url"
        warn "  확인: bash weknora.sh logs app"
    fi

    local frontend_url="http://localhost:${FRONTEND_PORT}/"
    if curl -sf --max-time 5 "$frontend_url" &>/dev/null; then
        ok "프론트엔드 정상 응답: $frontend_url"
    else
        warn "프론트엔드 응답 없음: $frontend_url"
    fi
}

# ─────────────────────────────────────────────────
# 명령: clean
# ─────────────────────────────────────────────────
cmd_clean() {
    log "미사용 Docker 리소스 정리 중..."
    docker image prune -f
    docker container prune -f
    ok "정리 완료"
    docker system df
}

# ─────────────────────────────────────────────────
# 내부 헬퍼
# ─────────────────────────────────────────────────
_sed() {
    local pattern="$1"; local file="$2"
    if sed --version &>/dev/null 2>&1; then
        sed -i "$pattern" "$file"  # GNU sed
    else
        sed -i '' "$pattern" "$file"  # BSD sed (macOS)
    fi
}

_print_access_info() {
    echo ""
    printf "%b\n" "${GREEN}====================================${NC}"
    printf "%b\n" "${GREEN}  WeKnora 접속 정보${NC}"
    printf "%b\n" "${GREEN}====================================${NC}"
    echo ""
    printf "%b\n" "${GREEN}  웹 UI  :  http://<서버IP>:${FRONTEND_PORT}${NC}"
    printf "%b\n" "${GREEN}  API    :  http://<서버IP>:${APP_PORT}${NC}"
    printf "%b\n" "${GREEN}  Swagger:  http://<서버IP>:${APP_PORT}/swagger/index.html${NC}"
    echo ""
}

# ─────────────────────────────────────────────────
# 도움말
# ─────────────────────────────────────────────────
cmd_help() {
    echo ""
    printf "%b\n" "${GREEN}WeKnora 서버 관리 스크립트${NC}"
    echo ""
    echo "사용법: bash weknora.sh <명령> [인수]"
    echo ""
    echo "명령:"
    printf "  %-20s %s\n" "install"          "최초 설치 (.env 생성 → 이미지 Pull → 서비스 시작)"
    printf "  %-20s %s\n" "start"            "서비스 시작"
    printf "  %-20s %s\n" "stop"             "서비스 중지"
    printf "  %-20s %s\n" "restart"          "서비스 재시작"
    printf "  %-20s %s\n" "update [tag]"     "이미지 업데이트 및 재시작 (기본: latest)"
    printf "  %-20s %s\n" "status"           "서비스 상태 + 헬스체크"
    printf "  %-20s %s\n" "logs [서비스]"    "로그 출력 (app|docreader|postgres|redis|frontend)"
    printf "  %-20s %s\n" "backup"           "PostgreSQL + 파일 백업"
    printf "  %-20s %s\n" "restore <파일>"   "백업 파일로 복원"
    printf "  %-20s %s\n" "health"           "API 헬스체크"
    printf "  %-20s %s\n" "clean"            "미사용 Docker 리소스 정리"
    echo ""
    echo "예시:"
    echo "  bash weknora.sh install             # 최초 설치"
    echo "  bash weknora.sh update 0.2.7        # 특정 버전으로 업데이트"
    echo "  bash weknora.sh logs app            # 백엔드 로그 확인"
    echo "  bash weknora.sh backup              # 데이터 백업"
    echo ""
}

# ─────────────────────────────────────────────────
# 메인
# ─────────────────────────────────────────────────
CMD="${1:-help}"
shift || true

case "$CMD" in
    install)  cmd_install ;;
    start)    cmd_start ;;
    stop)     cmd_stop ;;
    restart)  cmd_restart ;;
    update)   cmd_update "${1:-latest}" ;;
    status)   cmd_status ;;
    logs)     cmd_logs "${1:-}" ;;
    backup)   cmd_backup ;;
    restore)  cmd_restore "${1:-}" ;;
    health)   cmd_health ;;
    clean)    cmd_clean ;;
    help|-h|--help) cmd_help ;;
    *)
        warn "알 수 없는 명령: $CMD"
        cmd_help
        exit 1
        ;;
esac
