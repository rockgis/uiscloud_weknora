#!/usr/bin/env bash
# ============================================================
#  WeKnora 배포 패키지 생성 스크립트
#
#  사용법:
#    bash scripts/package.sh              # 온라인 패키지 (스크립트+설정만)
#    bash scripts/package.sh --offline    # 오프라인 패키지 (Docker 이미지 포함)
#    bash scripts/package.sh --offline --skip-docreader  # docreader 제외
#
#  온라인 패키지 (~15KB):
#    docker-compose.yml + .env.example + 관리 스크립트
#    → 서버에서 docker pull로 이미지 다운로드
#
#  오프라인 패키지 (1-5GB):
#    온라인 패키지 + Docker 이미지 tar 파일
#    → 인터넷 없는 서버(폐쇄망)에서도 설치 가능
# ============================================================
set -euo pipefail

# ────────────────────────────────────────────
# 색상 / 로그
# ────────────────────────────────────────────
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'
log()  { printf "%b\n" "  ${BLUE}▸${NC} $*"; }
ok()   { printf "%b\n" "  ${GREEN}✔${NC} $*"; }
warn() { printf "%b\n" "  ${YELLOW}⚠${NC}  $*"; }
fail() { printf "%b\n" "  ${RED}✘${NC}  $*" >&2; exit 1; }
step() { echo ""; printf "%b\n" "${BOLD}${BLUE}[$1]${NC} $2"; }

# ────────────────────────────────────────────
# 옵션 파싱
# ────────────────────────────────────────────
OFFLINE=false
SKIP_DOCREADER=false
BUILD_IMAGES=false
OUTPUT_DIR="."

for arg in "$@"; do
    case $arg in
        --offline)           OFFLINE=true ;;
        --skip-docreader)    SKIP_DOCREADER=true ;;
        --build-images)      BUILD_IMAGES=true ;;
        --output=*)          OUTPUT_DIR="${arg#*=}" ;;
        -h|--help)
            echo "사용법: bash scripts/package.sh [옵션]"
            echo ""
            echo "옵션:"
            echo "  --offline          오프라인 패키지 생성 (Docker 이미지 포함)"
            echo "  --skip-docreader   오프라인 패키지에서 docreader 이미지 제외 (빌드 시간 절약)"
            echo "  --build-images     이미지 빌드 후 패키지에 포함 (--offline과 함께 사용)"
            echo "  --output=<경로>    출력 디렉토리 (기본: 현재 디렉토리)"
            echo ""
            echo "예시:"
            echo "  bash scripts/package.sh                     # 온라인 패키지"
            echo "  bash scripts/package.sh --offline           # 오프라인 패키지 (이미지 미리 Pull 필요)"
            echo "  bash scripts/package.sh --offline --build-images  # 이미지 빌드 후 오프라인 패키지"
            exit 0
            ;;
    esac
done

# ────────────────────────────────────────────
# 프로젝트 루트 확인
# ────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "$PROJECT_ROOT"

[ -f "VERSION" ] || fail "VERSION 파일이 없습니다. 프로젝트 루트에서 실행하세요."
VERSION=$(cat VERSION | tr -d '[:space:]')
COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

PACKAGE_TYPE=$([ "$OFFLINE" = "true" ] && echo "offline" || echo "online")
PACKAGE_NAME="weknora-deploy-${VERSION}"
PACKAGE_FILE="${OUTPUT_DIR}/${PACKAGE_NAME}.tar.gz"
TMP_DIR=$(mktemp -d)
PKG_DIR="${TMP_DIR}/${PACKAGE_NAME}"

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

echo ""
printf "%b\n" "${BOLD}${GREEN}WeKnora 배포 패키지 생성${NC}"
echo ""
printf "  %-18s %s\n" "버전:"       "${VERSION}"
printf "  %-18s %s\n" "커밋:"       "${COMMIT_ID}"
printf "  %-18s %s\n" "빌드 시간:"  "${BUILD_TIME}"
printf "  %-18s %s\n" "패키지 타입:" "${PACKAGE_TYPE}"
printf "  %-18s %s\n" "출력 경로:"  "${PACKAGE_FILE}"

# ────────────────────────────────────────────
# 1단계: 이미지 빌드 (--build-images 옵션)
# ────────────────────────────────────────────
if [ "$BUILD_IMAGES" = "true" ]; then
    step "1/4" "Docker 이미지 빌드"

    # 플랫폼 감지
    case "$(uname -m)" in
        x86_64)          PLATFORM="linux/amd64" ;;
        aarch64|arm64)   PLATFORM="linux/arm64" ;;
        *)               PLATFORM="linux/amd64" ;;
    esac

    # 버전 빌드 인자
    BUILD_ARGS="--build-arg VERSION_ARG=${VERSION} --build-arg COMMIT_ID_ARG=${COMMIT_ID} --build-arg BUILD_TIME_ARG=${BUILD_TIME}"

    log "백엔드 앱 이미지 빌드 중..."
    # shellcheck disable=SC2086
    docker build --platform "$PLATFORM" $BUILD_ARGS \
        -f docker/Dockerfile.app \
        -t "weknora-app:${VERSION}" \
        -t "weknora-app:latest" \
        . 2>&1 | tail -5
    ok "weknora-app:${VERSION} 빌드 완료"

    log "프론트엔드 이미지 빌드 중..."
    # shellcheck disable=SC2086
    docker build --platform "$PLATFORM" \
        -f frontend/Dockerfile \
        -t "weknora-ui:${VERSION}" \
        -t "weknora-ui:latest" \
        frontend/ 2>&1 | tail -5
    ok "weknora-ui:${VERSION} 빌드 완료"

    if [ "$SKIP_DOCREADER" = "false" ]; then
        log "Docreader 이미지 빌드 중 (시간이 걸릴 수 있습니다)..."
        # shellcheck disable=SC2086
        docker build --platform "$PLATFORM" \
            -f docker/Dockerfile.docreader \
            -t "weknora-docreader:${VERSION}" \
            -t "weknora-docreader:latest" \
            . 2>&1 | tail -5
        ok "weknora-docreader:${VERSION} 빌드 완료"
    fi
else
    step "1/4" "Docker 이미지 빌드 건너뜀"
fi

# ────────────────────────────────────────────
# 2단계: 패키지 디렉토리 구성
# ────────────────────────────────────────────
step "2/4" "패키지 디렉토리 구성"

mkdir -p "${PKG_DIR}"

# 필수 파일 복사
log "설정 파일 복사 중..."
cp docker-compose.yml         "${PKG_DIR}/"
cp .env.example               "${PKG_DIR}/"
cp VERSION                    "${PKG_DIR}/"

# 프로덕션 오버라이드 파일
cp deploy/docker-compose.prod.yml "${PKG_DIR}/"

# 관리 스크립트
log "스크립트 파일 복사 중..."
cp deploy/install.sh          "${PKG_DIR}/"
cp deploy/weknora.sh          "${PKG_DIR}/"
cp deploy/setup-server.sh     "${PKG_DIR}/"

# README
cp deploy/README.md           "${PKG_DIR}/README.md"

# 실행 권한 부여
chmod +x "${PKG_DIR}/install.sh"
chmod +x "${PKG_DIR}/weknora.sh"
chmod +x "${PKG_DIR}/setup-server.sh"

ok "기본 파일 구성 완료"

# ────────────────────────────────────────────
# 3단계: Docker 이미지 저장 (오프라인 모드)
# ────────────────────────────────────────────
if [ "$OFFLINE" = "true" ]; then
    step "3/4" "Docker 이미지 저장 (오프라인 패키지)"

    mkdir -p "${PKG_DIR}/images"

    # 이미지 태그 결정 (빌드된 태그 우선, 없으면 레지스트리 이미지)
    _get_image_name() {
        local service="$1"
        local local_tag="${2}:${VERSION}"
        local local_latest="${2}:latest"

        if docker image inspect "${local_tag}" &>/dev/null 2>&1; then
            echo "${local_tag}"
        elif docker image inspect "${local_latest}" &>/dev/null 2>&1; then
            echo "${local_latest}"
        else
            # docker-compose.yml에서 이미지 이름 추출
            grep "image:" docker-compose.yml | grep "$service" | awk '{print $2}' | head -1
        fi
    }

    # 백엔드 앱
    APP_IMAGE=$(_get_image_name "app" "weknora-app")
    if [ -n "$APP_IMAGE" ] && docker image inspect "$APP_IMAGE" &>/dev/null 2>&1; then
        log "백엔드 이미지 저장 중: ${APP_IMAGE}"
        docker save "$APP_IMAGE" | gzip > "${PKG_DIR}/images/weknora-app.tar"
        ok "weknora-app.tar 저장 완료 ($(du -sh "${PKG_DIR}/images/weknora-app.tar" | cut -f1))"
    else
        warn "백엔드 이미지를 찾을 수 없습니다. docker pull 또는 --build-images 옵션을 사용하세요."
    fi

    # 프론트엔드
    UI_IMAGE=$(_get_image_name "frontend" "weknora-ui")
    if [ -n "$UI_IMAGE" ] && docker image inspect "$UI_IMAGE" &>/dev/null 2>&1; then
        log "프론트엔드 이미지 저장 중: ${UI_IMAGE}"
        docker save "$UI_IMAGE" | gzip > "${PKG_DIR}/images/weknora-ui.tar"
        ok "weknora-ui.tar 저장 완료 ($(du -sh "${PKG_DIR}/images/weknora-ui.tar" | cut -f1))"
    else
        warn "프론트엔드 이미지를 찾을 수 없습니다."
    fi

    # Docreader
    if [ "$SKIP_DOCREADER" = "false" ]; then
        DR_IMAGE=$(_get_image_name "docreader" "weknora-docreader")
        if [ -n "$DR_IMAGE" ] && docker image inspect "$DR_IMAGE" &>/dev/null 2>&1; then
            log "Docreader 이미지 저장 중: ${DR_IMAGE}"
            docker save "$DR_IMAGE" | gzip > "${PKG_DIR}/images/weknora-docreader.tar"
            ok "weknora-docreader.tar 저장 완료 ($(du -sh "${PKG_DIR}/images/weknora-docreader.tar" | cut -f1))"
        else
            warn "Docreader 이미지를 찾을 수 없습니다."
        fi
    else
        log "Docreader 이미지 건너뜀 (--skip-docreader)"
        # install.sh가 docreader만 Pull하도록 안내 파일 생성
        echo "docreader" > "${PKG_DIR}/images/.pull-required"
        warn "Docreader는 서버에서 별도로 Pull됩니다."
    fi

    # 이미지 목록 파일 생성
    ls "${PKG_DIR}/images/"*.tar 2>/dev/null | xargs -I{} basename {} .tar \
        > "${PKG_DIR}/images/manifest.txt" 2>/dev/null || true
else
    step "3/4" "이미지 저장 건너뜀 (온라인 패키지)"
fi

# ────────────────────────────────────────────
# 4단계: 아카이브 생성
# ────────────────────────────────────────────
step "4/4" "아카이브 생성"

mkdir -p "$OUTPUT_DIR"

log "압축 중: ${PACKAGE_FILE}"
tar czf "$PACKAGE_FILE" -C "$TMP_DIR" "${PACKAGE_NAME}/"

PACKAGE_SIZE=$(du -sh "$PACKAGE_FILE" | cut -f1)
ok "패키지 생성 완료"

# ────────────────────────────────────────────
# 완료 출력
# ────────────────────────────────────────────
echo ""
printf "%b\n" "${BOLD}${GREEN}패키지 생성 완료!${NC}"
echo ""
printf "  %-18s %s\n" "파일:"     "${PACKAGE_FILE}"
printf "  %-18s %s\n" "크기:"     "${PACKAGE_SIZE}"
printf "  %-18s %s\n" "타입:"     "${PACKAGE_TYPE}"

echo ""
log "패키지 내용:"
tar tzf "$PACKAGE_FILE" | sed "s|${PACKAGE_NAME}/||" | grep -v "^$" | sort | sed 's/^/    /'

echo ""
printf "%b\n" "${BOLD}서버 배포 방법:${NC}"
echo ""
echo "  # 1. 서버로 전송"
echo "  scp ${PACKAGE_FILE} user@<서버IP>:~/"
echo ""
echo "  # 2. 서버 접속"
echo "  ssh user@<서버IP>"
echo ""
echo "  # 3. 서버 초기 설정 (최초 1회 — root 필요)"
echo "  sudo bash setup-server.sh  # 패키지 안에 포함되지 않음, 별도 전송 필요"
echo ""
echo "  # 4. 설치"
echo "  mkdir -p /opt/weknora"
echo "  tar xzf $(basename "$PACKAGE_FILE") -C /opt/weknora --strip-components=1"
echo "  cd /opt/weknora"
echo "  bash install.sh"
echo ""
