#!/usr/bin/env bash
# ============================================================
#  WeKnora 서버 전송 + 자동 설치 스크립트
#
#  사용법:
#    bash scripts/transfer.sh                       # .server-config 설정 사용
#    bash scripts/transfer.sh user@host             # 직접 지정
#    bash scripts/transfer.sh user@host:port        # 포트 지정
#    bash scripts/transfer.sh user@host --key=~/.ssh/mykey.pem
#    bash scripts/transfer.sh user@host --no-install  # 전송만
#
#  사전 준비:
#    cp deploy/.server-config.example deploy/.server-config
#    vi deploy/.server-config   # 서버 정보 입력
# ============================================================
set -euo pipefail

# ────────────────────────────────────────────
# 색상 / 로그
# ────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'
log()    { printf "%b\n" "  ${BLUE}▸${NC} $*"; }
ok()     { printf "%b\n" "  ${GREEN}✔${NC} $*"; }
warn()   { printf "%b\n" "  ${YELLOW}⚠${NC}  $*"; }
fail()   { printf "%b\n" "  ${RED}✘${NC}  $*" >&2; exit 1; }
step()   { echo ""; printf "%b\n" "${BOLD}${BLUE}[$1]${NC} $2"; }
banner() { echo ""; printf "%b\n" "${BOLD}${GREEN}$*${NC}"; echo ""; }

# ────────────────────────────────────────────
# 프로젝트 루트
# ────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_FILE="${PROJECT_ROOT}/deploy/.server-config"
VERSION=$(cat "${PROJECT_ROOT}/VERSION" 2>/dev/null | tr -d '[:space:]' || echo "unknown")

# ────────────────────────────────────────────
# 기본값
# ────────────────────────────────────────────
SERVER_HOST=""
SERVER_PORT=22
SERVER_USER="deploy"
SERVER_SSH_KEY="~/.ssh/id_rsa"
SERVER_DEPLOY_DIR="/opt/weknora"
PACKAGE_TYPE="online"
AUTO_INSTALL=true
SKIP_BUILD=false
KEEP_PACKAGE=false

# ────────────────────────────────────────────
# .server-config 파일 로드
# ────────────────────────────────────────────
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        log ".server-config 로드 중..."
        # shellcheck disable=SC1090
        set -a; . "$CONFIG_FILE"; set +a
        ok ".server-config 로드 완료"
    fi
}

# ────────────────────────────────────────────
# 인수 파싱
# ────────────────────────────────────────────
parse_args() {
    for arg in "$@"; do
        case $arg in
            # user@host 또는 user@host:port 형식
            *@*:*)
                SERVER_USER="${arg%%@*}"
                local hostport="${arg##*@}"
                SERVER_HOST="${hostport%%:*}"
                SERVER_PORT="${hostport##*:}"
                ;;
            *@*)
                SERVER_USER="${arg%%@*}"
                SERVER_HOST="${arg##*@}"
                ;;
            --key=*)
                SERVER_SSH_KEY="${arg#*=}"
                ;;
            --port=*)
                SERVER_PORT="${arg#*=}"
                ;;
            --dir=*)
                SERVER_DEPLOY_DIR="${arg#*=}"
                ;;
            --offline)
                PACKAGE_TYPE="offline"
                ;;
            --no-install)
                AUTO_INSTALL=false
                ;;
            --skip-build)
                SKIP_BUILD=true
                ;;
            --keep-package)
                KEEP_PACKAGE=true
                ;;
            -h|--help)
                _print_help
                exit 0
                ;;
        esac
    done
}

_print_help() {
    echo ""
    echo "사용법: bash scripts/transfer.sh [접속정보] [옵션]"
    echo ""
    echo "접속 정보:"
    echo "  user@host             서버 사용자@호스트"
    echo "  user@host:port        서버 포트 지정 (기본: 22)"
    echo ""
    echo "옵션:"
    echo "  --key=<경로>          SSH 개인키 경로 (기본: ~/.ssh/id_rsa)"
    echo "  --dir=<경로>          서버 설치 경로 (기본: /opt/weknora)"
    echo "  --offline             오프라인 패키지 생성 (이미지 포함)"
    echo "  --no-install          전송만 (원격 설치 스킵)"
    echo "  --skip-build          패키지 빌드 스킵 (기존 파일 사용)"
    echo "  --keep-package        전송 후 로컬 패키지 파일 유지"
    echo ""
    echo "설정 파일:"
    echo "  deploy/.server-config  서버 접속 정보 저장"
    echo "  cp deploy/.server-config.example deploy/.server-config"
    echo ""
    echo "예시:"
    echo "  bash scripts/transfer.sh deploy@192.168.1.100"
    echo "  bash scripts/transfer.sh deploy@server.example.com:2222 --key=~/.ssh/prod.pem"
    echo "  bash scripts/transfer.sh --offline    # .server-config 사용"
}

# ────────────────────────────────────────────
# 입력값 검증
# ────────────────────────────────────────────
validate() {
    if [ -z "$SERVER_HOST" ]; then
        echo ""
        warn "서버 호스트가 설정되지 않았습니다."
        echo ""
        echo "  방법 1) 직접 지정:"
        echo "    bash scripts/transfer.sh deploy@192.168.1.100"
        echo ""
        echo "  방법 2) 설정 파일 사용:"
        echo "    cp deploy/.server-config.example deploy/.server-config"
        echo "    vi deploy/.server-config"
        echo "    bash scripts/transfer.sh"
        echo ""
        exit 1
    fi

    # SSH 키 파일 경로 확장 (~/ 처리)
    SERVER_SSH_KEY="${SERVER_SSH_KEY/#\~/$HOME}"

    if [ ! -f "$SERVER_SSH_KEY" ]; then
        fail "SSH 키 파일을 찾을 수 없습니다: ${SERVER_SSH_KEY}\n       --key=<경로> 옵션으로 지정하세요."
    fi

    command -v ssh  &>/dev/null || fail "ssh 명령을 찾을 수 없습니다."
    command -v scp  &>/dev/null || fail "scp 명령을 찾을 수 없습니다."
    command -v rsync &>/dev/null || true  # rsync 옵션용 (없어도 scp 사용)
}

# ────────────────────────────────────────────
# SSH / SCP 공통 옵션
# ────────────────────────────────────────────
SSH_OPTS="-i ${SERVER_SSH_KEY} -p ${SERVER_PORT} -o StrictHostKeyChecking=no -o ConnectTimeout=15 -o ServerAliveInterval=60"
SCP_OPTS="-i ${SERVER_SSH_KEY} -P ${SERVER_PORT} -o StrictHostKeyChecking=no -o ConnectTimeout=15"

_ssh() {
    # SSH_OPTS은 이 함수 호출 전에 최종 설정됨
    # shellcheck disable=SC2086
    ssh $SSH_OPTS "${SERVER_USER}@${SERVER_HOST}" "$@"
}

_scp() {
    # shellcheck disable=SC2086
    scp $SCP_OPTS "$@"
}

# ────────────────────────────────────────────
# 1단계: SSH 연결 테스트
# ────────────────────────────────────────────
test_connection() {
    step "1/4" "SSH 연결 확인"
    log "접속 중: ${SERVER_USER}@${SERVER_HOST}:${SERVER_PORT}"
    log "SSH 키: ${SERVER_SSH_KEY}"

    if _ssh "echo OK" 2>/dev/null | grep -q "OK"; then
        ok "SSH 연결 성공"
    else
        fail "SSH 연결 실패\n\n       확인 사항:\n       - 서버 IP/포트 정확한지 확인\n       - SSH 키 경로 확인: ${SERVER_SSH_KEY}\n       - 서버에 공개키 등록 여부 확인 (~/.ssh/authorized_keys)\n\n       수동 테스트:\n       ssh -i ${SERVER_SSH_KEY} -p ${SERVER_PORT} ${SERVER_USER}@${SERVER_HOST} echo OK"
    fi

    # Docker 설치 여부 확인
    if _ssh "command -v docker && docker info" &>/dev/null 2>&1; then
        local docker_ver
        docker_ver=$(_ssh "docker --version 2>/dev/null" | awk '{print $3}' | tr -d ',')
        ok "서버 Docker: ${docker_ver}"
    else
        warn "서버에 Docker가 설치되어 있지 않거나 실행 중이 아닙니다."
        warn "설치 방법: ssh ${SERVER_USER}@${SERVER_HOST} 'sudo bash setup-server.sh'"
    fi
}

# ────────────────────────────────────────────
# 2단계: 패키지 빌드
# ────────────────────────────────────────────
build_package() {
    step "2/4" "배포 패키지 생성"

    local PACKAGE_FILE="${PROJECT_ROOT}/weknora-deploy-${VERSION}.tar.gz"

    if [ "$SKIP_BUILD" = "true" ] && [ -f "$PACKAGE_FILE" ]; then
        ok "기존 패키지 사용: $(basename "$PACKAGE_FILE") ($(du -sh "$PACKAGE_FILE" | cut -f1))"
        echo "$PACKAGE_FILE"
        return
    fi

    log "패키지 빌드 중..."
    cd "$PROJECT_ROOT"

    if [ "$PACKAGE_TYPE" = "offline" ]; then
        bash scripts/package.sh --offline
    else
        bash scripts/package.sh
    fi

    [ -f "$PACKAGE_FILE" ] || fail "패키지 파일 생성 실패: $PACKAGE_FILE"
    ok "패키지 생성 완료: $(basename "$PACKAGE_FILE") ($(du -sh "$PACKAGE_FILE" | cut -f1))"
    echo "$PACKAGE_FILE"
}

# ────────────────────────────────────────────
# 3단계: 서버로 전송
# ────────────────────────────────────────────
transfer_package() {
    local package_file="$1"
    local package_name
    package_name=$(basename "$package_file")

    step "3/4" "서버로 전송"

    # 배포 디렉토리 생성
    log "서버 디렉토리 준비: ${SERVER_DEPLOY_DIR}"
    _ssh "mkdir -p '${SERVER_DEPLOY_DIR}'"

    # 진행률 표시하며 전송
    log "전송 중: ${package_name} → ${SERVER_USER}@${SERVER_HOST}:${SERVER_DEPLOY_DIR}/"

    if command -v rsync &>/dev/null; then
        # rsync: 진행률 + 재시작 가능
        # shellcheck disable=SC2086
        rsync -avz --progress \
            -e "ssh $SSH_OPTS" \
            "$package_file" \
            "${SERVER_USER}@${SERVER_HOST}:${SERVER_DEPLOY_DIR}/" 2>&1 \
            | grep -E "(sending|sent|bytes|%|error)" || true
    else
        # scp 폴백
        _scp "$package_file" "${SERVER_USER}@${SERVER_HOST}:${SERVER_DEPLOY_DIR}/"
    fi

    # 전송 검증
    local remote_size
    remote_size=$(_ssh "du -sh '${SERVER_DEPLOY_DIR}/${package_name}' 2>/dev/null | cut -f1" || echo "")
    if [ -n "$remote_size" ]; then
        ok "전송 완료: ${package_name} (서버 측 크기: ${remote_size})"
    else
        fail "파일 전송 후 서버에서 확인 실패"
    fi
}

# ────────────────────────────────────────────
# 4단계: 원격 설치
# ────────────────────────────────────────────
remote_install() {
    local package_name="weknora-deploy-${VERSION}.tar.gz"

    step "4/4" "원격 설치 실행"
    log "서버에서 압축 해제 및 설치 시작..."

    # SSH pseudo-TTY 할당(-t)으로 대화형 install.sh 실행
    # shellcheck disable=SC2086
    ssh -t $SSH_OPTS "${SERVER_USER}@${SERVER_HOST}" bash -s << REMOTE_SCRIPT
set -euo pipefail

DEPLOY_DIR="${SERVER_DEPLOY_DIR}"
PACKAGE="${package_name}"

echo ""
echo "=== WeKnora 원격 설치 ==="
echo "배포 경로: \$DEPLOY_DIR"
echo ""

cd "\$DEPLOY_DIR"

# 압축 해제
echo "압축 해제 중..."
tar xzf "\$PACKAGE" --strip-components=1 --overwrite
echo "압축 해제 완료"

# install.sh 실행
bash install.sh

REMOTE_SCRIPT
}

# ────────────────────────────────────────────
# 로컬 패키지 정리
# ────────────────────────────────────────────
cleanup_local() {
    local package_file="$1"
    if [ "$KEEP_PACKAGE" = "false" ] && [ -f "$package_file" ]; then
        rm -f "$package_file"
        log "로컬 패키지 파일 삭제: $(basename "$package_file")"
    fi
}

# ────────────────────────────────────────────
# 완료 메시지
# ────────────────────────────────────────────
print_finish() {
    echo ""
    printf "%b\n" "${BOLD}${GREEN}"
    echo "  ╔════════════════════════════════════════════╗"
    echo "  ║                                            ║"
    echo "  ║   전송 및 설치 완료!                      ║"
    echo "  ║                                            ║"
    printf "  ║   %-43s║\n" "서버: ${SERVER_USER}@${SERVER_HOST}"
    printf "  ║   %-43s║\n" "경로: ${SERVER_DEPLOY_DIR}"
    echo "  ║                                            ║"
    echo "  ╚════════════════════════════════════════════╝"
    printf "%b\n" "${NC}"
    echo ""
    printf "%b\n" "${BOLD}서비스 관리 (서버에서):${NC}"
    echo ""
    echo "  ssh ${SERVER_USER}@${SERVER_HOST}"
    echo "  cd ${SERVER_DEPLOY_DIR}"
    echo "  bash weknora.sh status    # 상태 확인"
    echo "  bash weknora.sh logs app  # 로그 확인"
    echo ""
}

# ────────────────────────────────────────────
# 메인
# ────────────────────────────────────────────
main() {
    banner "WeKnora 서버 전송 + 설치 v${VERSION}"

    cd "$PROJECT_ROOT"

    # 설정 로드 → 인수 파싱 (인수가 설정 파일보다 우선)
    load_config
    parse_args "$@"

    # SSH 옵션 갱신 (parse_args 이후 최종 값으로)
    SSH_OPTS="-i ${SERVER_SSH_KEY/#\~/$HOME} -p ${SERVER_PORT} -o StrictHostKeyChecking=no -o ConnectTimeout=15 -o ServerAliveInterval=60"
    SCP_OPTS="-i ${SERVER_SSH_KEY/#\~/$HOME} -P ${SERVER_PORT} -o StrictHostKeyChecking=no -o ConnectTimeout=15"

    validate

    printf "  %-18s %s\n" "서버:"    "${SERVER_USER}@${SERVER_HOST}:${SERVER_PORT}"
    printf "  %-18s %s\n" "SSH 키:"  "${SERVER_SSH_KEY}"
    printf "  %-18s %s\n" "설치 경로:" "${SERVER_DEPLOY_DIR}"
    printf "  %-18s %s\n" "패키지:"  "${PACKAGE_TYPE}"
    printf "  %-18s %s\n" "자동 설치:" "$([ "$AUTO_INSTALL" = "true" ] && echo '예' || echo '아니오')"

    test_connection

    local package_file
    package_file=$(build_package | tail -1)

    transfer_package "$package_file"

    if [ "$AUTO_INSTALL" = "true" ]; then
        remote_install
    else
        echo ""
        ok "전송 완료. 서버에서 수동 설치:"
        echo "    ssh ${SERVER_USER}@${SERVER_HOST}"
        echo "    cd ${SERVER_DEPLOY_DIR}"
        echo "    tar xzf weknora-deploy-${VERSION}.tar.gz --strip-components=1"
        echo "    bash install.sh"
    fi

    cleanup_local "$package_file"
    print_finish
}

main "$@"
