#!/usr/bin/env bash
# WeKnora 서버 초기 환경 설정 스크립트
# Ubuntu 20.04+ / Debian 11+ / CentOS 8+ / RHEL 8+ 지원
#
# 사용법:
#   sudo bash setup-server.sh
#   sudo bash setup-server.sh --skip-docker   # Docker가 이미 설치된 경우
#
# 이 스크립트가 하는 일:
#   1. Docker CE + Compose 플러그인 설치
#   2. deploy 사용자 생성 및 docker 그룹 추가
#   3. 배포 디렉토리 생성
#   4. (선택) UFW/firewalld 포트 개방

set -euo pipefail

# ─────────────────────────────────────────────────
# 색상 / 로그
# ─────────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { printf "%b\n" "${BLUE}[INFO]${NC}  $*"; }
ok()   { printf "%b\n" "${GREEN}[OK]${NC}    $*"; }
warn() { printf "%b\n" "${YELLOW}[WARN]${NC}  $*"; }
fail() { printf "%b\n" "${RED}[FAIL]${NC}  $*" >&2; exit 1; }

# ─────────────────────────────────────────────────
# 옵션 파싱
# ─────────────────────────────────────────────────
SKIP_DOCKER=false
DEPLOY_USER="${DEPLOY_USER:-deploy}"
INSTALL_DIR="${INSTALL_DIR:-/opt/weknora}"
OPEN_FIREWALL="${OPEN_FIREWALL:-true}"

for arg in "$@"; do
    case $arg in
        --skip-docker)   SKIP_DOCKER=true ;;
        --no-firewall)   OPEN_FIREWALL=false ;;
        --user=*)        DEPLOY_USER="${arg#*=}" ;;
        --dir=*)         INSTALL_DIR="${arg#*=}" ;;
        -h|--help)
            echo "사용법: sudo bash setup-server.sh [옵션]"
            echo ""
            echo "옵션:"
            echo "  --skip-docker      Docker 설치 건너뜀 (이미 설치된 경우)"
            echo "  --no-firewall      방화벽 설정 건너뜀"
            echo "  --user=<이름>      배포 사용자명 (기본: deploy)"
            echo "  --dir=<경로>       설치 디렉토리 (기본: /opt/weknora)"
            exit 0
            ;;
    esac
done

# ─────────────────────────────────────────────────
# 루트 권한 확인
# ─────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && fail "이 스크립트는 root 또는 sudo 권한이 필요합니다."

# ─────────────────────────────────────────────────
# OS 감지
# ─────────────────────────────────────────────────
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="${ID:-unknown}"
        OS_VER="${VERSION_ID:-0}"
    else
        fail "지원하지 않는 OS입니다 (/etc/os-release 없음)."
    fi
    log "OS: ${PRETTY_NAME:-$OS_ID $OS_VER}"
}

# ─────────────────────────────────────────────────
# 1. Docker 설치
# ─────────────────────────────────────────────────
install_docker() {
    if command -v docker &>/dev/null && docker info &>/dev/null; then
        ok "Docker 이미 설치됨: $(docker --version)"
        return 0
    fi

    log "Docker CE 설치 중..."

    case "$OS_ID" in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y -qq ca-certificates curl gnupg lsb-release

            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
                | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            chmod a+r /etc/apt/keyrings/docker.gpg

            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
                https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
                > /etc/apt/sources.list.d/docker.list

            apt-get update -qq
            apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        centos|rhel|rocky|almalinux)
            yum install -y -q yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            yum install -y -q docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        fedora)
            dnf -y -q install dnf-plugins-core
            dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            dnf install -y -q docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        *)
            warn "지원하지 않는 OS '$OS_ID' — Docker 공식 스크립트로 설치 시도..."
            curl -fsSL https://get.docker.com | sh
            ;;
    esac

    systemctl enable --now docker
    ok "Docker 설치 완료: $(docker --version)"
}

# ─────────────────────────────────────────────────
# 2. deploy 사용자 생성
# ─────────────────────────────────────────────────
setup_deploy_user() {
    if id "$DEPLOY_USER" &>/dev/null; then
        ok "사용자 '$DEPLOY_USER' 이미 존재"
    else
        log "사용자 '$DEPLOY_USER' 생성 중..."
        useradd -m -s /bin/bash "$DEPLOY_USER"
        ok "사용자 '$DEPLOY_USER' 생성 완료"
    fi

    # docker 그룹에 추가
    if ! groups "$DEPLOY_USER" | grep -qw docker; then
        usermod -aG docker "$DEPLOY_USER"
        ok "'$DEPLOY_USER' → docker 그룹 추가 완료"
    else
        ok "'$DEPLOY_USER' 이미 docker 그룹에 속해있음"
    fi

    # sudo 권한 추가 (docker 명령에 필요한 경우만)
    SUDOERS_FILE="/etc/sudoers.d/weknora-${DEPLOY_USER}"
    if [ ! -f "$SUDOERS_FILE" ]; then
        echo "${DEPLOY_USER} ALL=(ALL) NOPASSWD: /usr/bin/docker, /usr/bin/docker-compose, /usr/local/bin/docker-compose" \
            > "$SUDOERS_FILE"
        chmod 440 "$SUDOERS_FILE"
        ok "sudoers 설정 완료: $SUDOERS_FILE"
    fi
}

# ─────────────────────────────────────────────────
# 3. 배포 디렉토리 생성
# ─────────────────────────────────────────────────
setup_directories() {
    log "배포 디렉토리 생성 중: $INSTALL_DIR"

    mkdir -p "${INSTALL_DIR}"/{data/files,backups,logs}

    chown -R "${DEPLOY_USER}:${DEPLOY_USER}" "$INSTALL_DIR"
    chmod 750 "$INSTALL_DIR"
    chmod 770 "${INSTALL_DIR}/data"
    chmod 770 "${INSTALL_DIR}/backups"

    ok "디렉토리 구조 생성 완료:"
    find "$INSTALL_DIR" -maxdepth 2 -type d | sort | sed 's/^/    /'
}

# ─────────────────────────────────────────────────
# 4. 방화벽 포트 개방
# ─────────────────────────────────────────────────
setup_firewall() {
    [[ "$OPEN_FIREWALL" != "true" ]] && return 0

    log "방화벽 포트 개방 중..."

    if command -v ufw &>/dev/null && ufw status | grep -q "Status: active"; then
        ufw allow 80/tcp    comment 'WeKnora Frontend'
        ufw allow 8080/tcp  comment 'WeKnora API'
        ok "UFW 포트 개방: 80, 8080"

    elif command -v firewall-cmd &>/dev/null; then
        firewall-cmd --permanent --add-port=80/tcp
        firewall-cmd --permanent --add-port=8080/tcp
        firewall-cmd --reload
        ok "firewalld 포트 개방: 80, 8080"

    else
        warn "방화벽 도구(ufw/firewalld)를 감지하지 못했습니다. 필요 시 수동으로 80, 8080 포트를 개방하세요."
    fi
}

# ─────────────────────────────────────────────────
# 5. 시스템 최적화 (선택)
# ─────────────────────────────────────────────────
system_tuning() {
    log "시스템 파라미터 최적화 중..."

    # vm.max_map_count: Elasticsearch 요구사항
    if ! grep -q "vm.max_map_count" /etc/sysctl.conf 2>/dev/null; then
        echo "vm.max_map_count=262144" >> /etc/sysctl.conf
        sysctl -w vm.max_map_count=262144 &>/dev/null || true
        ok "vm.max_map_count=262144 설정"
    fi

    # 파일 디스크립터 한도
    LIMITS_FILE="/etc/security/limits.d/weknora.conf"
    if [ ! -f "$LIMITS_FILE" ]; then
        cat > "$LIMITS_FILE" <<'EOF'
# WeKnora: 파일 디스크립터 한도 설정
*  soft  nofile  65536
*  hard  nofile  65536
EOF
        ok "파일 디스크립터 한도 설정: 65536"
    fi
}

# ─────────────────────────────────────────────────
# 6. 완료 안내
# ─────────────────────────────────────────────────
print_summary() {
    echo ""
    printf "%b\n" "${GREEN}============================================${NC}"
    printf "%b\n" "${GREEN}  서버 초기 설정 완료!${NC}"
    printf "%b\n" "${GREEN}============================================${NC}"
    echo ""
    echo "  배포 사용자 : $DEPLOY_USER"
    echo "  설치 디렉토리: $INSTALL_DIR"
    echo ""
    printf "%b\n" "${YELLOW}다음 단계:${NC}"
    echo ""
    echo "  1. deploy 사용자로 로그인하거나 SSH 키 설정:"
    echo "     sudo -u $DEPLOY_USER bash"
    echo ""
    echo "  2. WeKnora 배포 파일을 서버로 복사:"
    echo "     scp weknora-deploy-*.tar.gz $DEPLOY_USER@<서버IP>:$INSTALL_DIR/"
    echo ""
    echo "  3. 배포 실행:"
    echo "     cd $INSTALL_DIR"
    echo "     tar xzf weknora-deploy-*.tar.gz"
    echo "     bash weknora.sh install"
    echo ""
    printf "%b\n" "${YELLOW}참고:${NC} docker 그룹 변경사항을 적용하려면 재로그인이 필요합니다."
    echo ""
}

# ─────────────────────────────────────────────────
# 메인
# ─────────────────────────────────────────────────
main() {
    echo ""
    printf "%b\n" "${GREEN}========================================${NC}"
    printf "%b\n" "${GREEN}  WeKnora 서버 초기 환경 설정 스크립트${NC}"
    printf "%b\n" "${GREEN}========================================${NC}"
    echo ""

    detect_os

    if [[ "$SKIP_DOCKER" == "false" ]]; then
        install_docker
    else
        log "Docker 설치 건너뜀 (--skip-docker)"
        command -v docker &>/dev/null || fail "Docker가 설치되어 있지 않습니다."
    fi

    setup_deploy_user
    setup_directories
    setup_firewall
    system_tuning
    print_summary
}

main "$@"
