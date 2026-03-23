#!/usr/bin/env bash
# WeKnora 배포 서버 실행 스크립트
# Jenkins가 SSH를 통해 배포 서버에서 이 스크립트를 실행합니다.
#
# 환경변수 (Jenkins에서 주입):
#   IMAGE_TAG       - 배포할 이미지 태그 (예: 0.2.6 또는 latest)
#   DEPLOY_ENV      - 배포 환경 (dev | staging | prod)
#   DOCKER_REGISTRY - Docker 레지스트리 주소

set -euo pipefail

# ──────────────────────────────────────────────────
# 로그 헬퍼
# ──────────────────────────────────────────────────
log()  { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }
ok()   { echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✓ $*"; }
fail() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] ✗ $*" >&2; exit 1; }

# 컨테이너 헬스체크 대기 (서비스명, 타임아웃(초))
wait_healthy() {
    local service="$1"
    local timeout="${2:-60}"
    local elapsed=0
    log "${service} 헬스체크 대기 중 (최대 ${timeout}초)..."
    while [[ ${elapsed} -lt ${timeout} ]]; do
        local container_id
        container_id=$(${COMPOSE_CMD} ps -q "${service}" 2>/dev/null || echo "")
        if [[ -n "${container_id}" ]]; then
            local status
            status=$(docker inspect --format='{{.State.Health.Status}}' "${container_id}" 2>/dev/null || echo "")
            if [[ "${status}" == "healthy" ]]; then
                ok "${service} 정상 (healthy)"
                return 0
            fi
            [[ "${status}" == "unhealthy" ]] && { fail "${service} 비정상 상태 (unhealthy)"; }
        fi
        sleep 3
        elapsed=$((elapsed + 3))
    done
    fail "${service} 헬스체크 타임아웃 (${timeout}초)"
}

# ──────────────────────────────────────────────────
# 변수 설정
# ──────────────────────────────────────────────────
IMAGE_TAG="${IMAGE_TAG:-latest}"
DEPLOY_ENV="${DEPLOY_ENV:-prod}"
DOCKER_REGISTRY="${DOCKER_REGISTRY:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 환경별 컴포즈 오버라이드 파일 탐색
COMPOSE_BASE="${SCRIPT_DIR}/docker-compose.yml"
COMPOSE_OVERRIDE=""
if [[ -f "${SCRIPT_DIR}/docker-compose.${DEPLOY_ENV}.yml" ]]; then
    COMPOSE_OVERRIDE="${SCRIPT_DIR}/docker-compose.${DEPLOY_ENV}.yml"
elif [[ -f "${SCRIPT_DIR}/docker-compose.override.yml" ]]; then
    COMPOSE_OVERRIDE="${SCRIPT_DIR}/docker-compose.override.yml"
fi

if [[ -n "${COMPOSE_OVERRIDE}" ]]; then
    COMPOSE_CMD="docker compose -f ${COMPOSE_BASE} -f ${COMPOSE_OVERRIDE}"
    log "컴포즈 오버라이드 적용: ${COMPOSE_OVERRIDE}"
else
    COMPOSE_CMD="docker compose -f ${COMPOSE_BASE}"
fi

# ──────────────────────────────────────────────────
# 0. 사전 점검
# ──────────────────────────────────────────────────
log "WeKnora 배포 시작 — 환경: ${DEPLOY_ENV}, 태그: ${IMAGE_TAG}"

[[ ! -f "${COMPOSE_BASE}" ]] && fail "docker-compose.yml 파일이 없습니다: ${COMPOSE_BASE}"
[[ ! -f "${SCRIPT_DIR}/.env" ]] && fail ".env 파일이 없습니다. .env.example을 복사하여 설정하세요."

command -v docker &>/dev/null || fail "docker가 설치되어 있지 않습니다."
docker info &>/dev/null       || fail "Docker 데몬이 실행 중이지 않습니다."

# ──────────────────────────────────────────────────
# 1. 이미지 태그 .env 반영
# ──────────────────────────────────────────────────
log "이미지 태그 설정: ${IMAGE_TAG}"
if grep -q "^WEKNORA_IMAGE_TAG=" "${SCRIPT_DIR}/.env"; then
    sed -i "s|^WEKNORA_IMAGE_TAG=.*|WEKNORA_IMAGE_TAG=${IMAGE_TAG}|" "${SCRIPT_DIR}/.env"
else
    echo "WEKNORA_IMAGE_TAG=${IMAGE_TAG}" >> "${SCRIPT_DIR}/.env"
fi
export WEKNORA_IMAGE_TAG="${IMAGE_TAG}"

# ──────────────────────────────────────────────────
# 2. 최신 이미지 Pull
# ──────────────────────────────────────────────────
log "Docker 이미지 Pull 중..."
[[ -n "${DOCKER_REGISTRY}" ]] && log "레지스트리: ${DOCKER_REGISTRY}"
${COMPOSE_CMD} pull --quiet || fail "이미지 Pull 실패"
ok "이미지 Pull 완료"

# ──────────────────────────────────────────────────
# 3. 서비스 재시작 (의존성 순서 보장)
# ──────────────────────────────────────────────────
log "서비스 재시작 중..."

# 인프라 서비스 (실행 중이면 유지)
${COMPOSE_CMD} up -d --no-recreate postgres redis
ok "인프라 서비스(postgres, redis) 확인 완료"

# Docreader 재시작 + 헬스체크
${COMPOSE_CMD} up -d --force-recreate docreader
wait_healthy docreader 90

# 백엔드 재시작 + 헬스체크
${COMPOSE_CMD} up -d --force-recreate app
wait_healthy app 60

# 프론트엔드 재시작
${COMPOSE_CMD} up -d --force-recreate frontend
ok "모든 서비스 재시작 완료"

# ──────────────────────────────────────────────────
# 4. DB 마이그레이션
# ──────────────────────────────────────────────────
log "DB 마이그레이션 실행 중..."
APP_CONTAINER=$(${COMPOSE_CMD} ps -q app 2>/dev/null || echo "")
if [[ -n "${APP_CONTAINER}" ]]; then
    if docker exec "${APP_CONTAINER}" sh -c 'command -v migrate &>/dev/null'; then
        docker exec "${APP_CONTAINER}" \
            migrate -path /app/migrations/versioned \
                    -database "${DATABASE_URL:-}" \
                    up 2>&1 && ok "마이그레이션 완료" || log "마이그레이션 스킵 (DATABASE_URL 미설정)"
    else
        log "migrate 바이너리 없음 — 마이그레이션 스킵"
    fi
fi

# ──────────────────────────────────────────────────
# 5. 배포 결과 확인
# ──────────────────────────────────────────────────
log "실행 중인 서비스 상태:"
${COMPOSE_CMD} ps

APP_PORT="${APP_PORT:-8080}"
log "백엔드 헬스 API 확인 (http://localhost:${APP_PORT}/health)..."
if curl -sf --max-time 10 "http://localhost:${APP_PORT}/health" &>/dev/null; then
    ok "백엔드 정상 응답"
else
    log "경고: 헬스 엔드포인트 응답 없음 — app 로그:"
    ${COMPOSE_CMD} logs --tail=30 app
fi

# ──────────────────────────────────────────────────
# 6. 오래된 이미지 정리
# ──────────────────────────────────────────────────
log "사용하지 않는 이미지 정리..."
docker image prune -f &>/dev/null || true

ok "배포 완료 — 환경: ${DEPLOY_ENV}, 버전: ${IMAGE_TAG}"
