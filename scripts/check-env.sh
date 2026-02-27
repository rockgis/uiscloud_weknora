#!/bin/bash
# 개발 환경 설정 확인

# 색상 설정
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 색상 없음

# 프로젝트 루트 디렉토리 가져오기
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[✓]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[✗]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[!]${NC} $1"
}

echo ""
printf "%b\n" "${GREEN}========================================${NC}"
printf "%b\n" "${GREEN}  uiscloud_weknora 개발 환경 설정 확인${NC}"
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

cd "$PROJECT_ROOT"

# .env 파일 확인
log_info ".env 파일 확인 중..."
if [ -f ".env" ]; then
    log_success ".env 파일 존재"
else
    log_error ".env 파일이 존재하지 않습니다"
    echo ""
    log_info "해결 방법:"
    echo "  1. 예시 파일 복사: cp .env.example .env"
    echo "  2. .env 파일을 편집하여 필요한 환경 변수 설정"
    exit 1
fi

echo ""
log_info "필수 환경 변수 확인 중..."

# .env 파일 로드
set -a
source .env
set +a

# 필수 환경 변수 확인
errors=0

check_var() {
    local var_name=$1
    local var_value="${!var_name}"

    if [ -z "$var_value" ]; then
        log_error "$var_name 설정되지 않음"
        errors=$((errors + 1))
    else
        log_success "$var_name = $var_value"
    fi
}

# 데이터베이스 설정
log_info "데이터베이스 설정:"
check_var "DB_DRIVER"
check_var "DB_HOST"
check_var "DB_PORT"
check_var "DB_USER"
check_var "DB_PASSWORD"
check_var "DB_NAME"

echo ""
log_info "스토리지 설정:"
check_var "STORAGE_TYPE"

if [ "$STORAGE_TYPE" = "minio" ]; then
    check_var "MINIO_BUCKET_NAME"
fi

echo ""
log_info "Redis 설정:"
check_var "REDIS_ADDR"

echo ""
log_info "Ollama 설정:"
check_var "OLLAMA_BASE_URL"

echo ""
log_info "모델 설정:"
if [ -n "$INIT_LLM_MODEL_NAME" ]; then
    log_success "INIT_LLM_MODEL_NAME = $INIT_LLM_MODEL_NAME"
else
    log_warning "INIT_LLM_MODEL_NAME 설정되지 않음 (선택사항)"
fi

if [ -n "$INIT_EMBEDDING_MODEL_NAME" ]; then
    log_success "INIT_EMBEDDING_MODEL_NAME = $INIT_EMBEDDING_MODEL_NAME"
else
    log_warning "INIT_EMBEDDING_MODEL_NAME 설정되지 않음 (선택사항)"
fi

# Go 환경 확인
echo ""
log_info "Go 환경 확인 중..."
if command -v go &> /dev/null; then
    go_version=$(go version)
    log_success "Go 설치됨: $go_version"
else
    log_error "Go가 설치되지 않았습니다"
    errors=$((errors + 1))
fi

# Air 확인
if command -v air &> /dev/null; then
    log_success "Air 설치됨 (핫 리로드 지원)"
else
    log_warning "Air가 설치되지 않았습니다 (선택사항, 핫 리로드용)"
    log_info "설치 명령: go install github.com/cosmtrek/air@latest"
fi

# npm 확인
echo ""
log_info "Node.js 환경 확인 중..."
if command -v npm &> /dev/null; then
    npm_version=$(npm --version)
    log_success "npm 설치됨: $npm_version"
else
    log_error "npm이 설치되지 않았습니다"
    errors=$((errors + 1))
fi

# Docker 확인
echo ""
log_info "Docker 환경 확인 중..."
if command -v docker &> /dev/null; then
    docker_version=$(docker --version)
    log_success "Docker 설치됨: $docker_version"

    if docker info &> /dev/null; then
        log_success "Docker 서비스 실행 중"
    else
        log_error "Docker 서비스가 실행 중이 아닙니다"
        errors=$((errors + 1))
    fi
else
    log_error "Docker가 설치되지 않았습니다"
    errors=$((errors + 1))
fi

# Docker Compose 확인
if docker compose version &> /dev/null; then
    compose_version=$(docker compose version)
    log_success "Docker Compose 설치됨: $compose_version"
elif command -v docker-compose &> /dev/null; then
    compose_version=$(docker-compose --version)
    log_success "docker-compose 설치됨: $compose_version"
else
    log_error "Docker Compose가 설치되지 않았습니다"
    errors=$((errors + 1))
fi

# 요약
echo ""
printf "%b\n" "${GREEN}========================================${NC}"
if [ $errors -eq 0 ]; then
    log_success "모든 확인 통과! 환경 설정이 정상입니다"
    echo ""
    log_info "다음 단계:"
    echo "  1. 개발 환경 시작: make dev-start"
    echo "  2. 백엔드 시작: make dev-app"
    echo "  3. 프론트엔드 시작: make dev-frontend"
else
    log_error "$errors 개의 문제가 발견되었습니다. 수정 후 개발 환경을 시작하세요"
    echo ""
    log_info "일반적인 문제:"
    echo "  - .env 파일이 없으면 .env.example을 복사하세요"
    echo "  - DB_DRIVER를 'postgres' 또는 'mysql'로 설정하세요"
    echo "  - Docker 서비스가 실행 중인지 확인하세요"
fi
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

exit $errors

