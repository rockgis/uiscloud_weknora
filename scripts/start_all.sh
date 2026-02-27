#!/bin/bash
# 이 스크립트는 필요에 따라 Ollama와 docker-compose 서비스를 시작/중지합니다

# 색상 설정
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # 색상 없음

# 프로젝트 루트 디렉토리 가져오기 (스크립트 디렉토리의 상위)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# 버전 정보
VERSION="1.0.1" # 버전 업데이트
SCRIPT_NAME=$(basename "$0")

# 도움말 표시
show_help() {
    printf "%b\n" "${GREEN}uiscloud_weknora 시작 스크립트 v${VERSION}${NC}"
    printf "%b\n" "${GREEN}사용법:${NC} $0 [옵션]"
    echo "옵션:"
    echo "  -h, --help     도움말 표시"
    echo "  -o, --ollama   Ollama 서비스 시작"
    echo "  -d, --docker   Docker 컨테이너 서비스 시작"
    echo "  -a, --all      모든 서비스 시작 (기본값)"
    echo "  -s, --stop     모든 서비스 중지"
    echo "  -c, --check    환경 확인 및 문제 진단"
    echo "  -r, --restart  지정된 컨테이너 재빌드 및 재시작"
    echo "  -l, --list     실행 중인 모든 컨테이너 나열"
    echo "  -p, --pull     최신 Docker 이미지 가져오기"
    echo "  --no-pull      시작 시 이미지 가져오지 않음 (기본적으로 가져옴)"
    echo "  -v, --version  버전 정보 표시"
    exit 0
}

# 버전 정보 표시
show_version() {
    printf "%b\n" "${GREEN}uiscloud_weknora 시작 스크립트 v${VERSION}${NC}"
    exit 0
}

# 로그 함수
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

# 사용 가능한 Docker Compose 명령 선택 (docker compose 우선, 그 다음 docker-compose)
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
	# Docker Compose 플러그인 우선 사용
	if docker compose version &> /dev/null; then
		DOCKER_COMPOSE_BIN="docker"
		DOCKER_COMPOSE_SUBCMD="compose"
		return 0
	fi

	# docker-compose (v1)로 폴백
	if command -v docker-compose &> /dev/null; then
		if docker-compose version &> /dev/null; then
			DOCKER_COMPOSE_BIN="docker-compose"
			DOCKER_COMPOSE_SUBCMD=""
			return 0
		fi
	fi

	# 둘 다 사용 불가
	return 1
}

# .env 파일 확인 및 생성
check_env_file() {
    log_info "환경 변수 설정 확인 중..."
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        log_warning ".env 파일이 없습니다. 템플릿에서 생성합니다"
        if [ -f "$PROJECT_ROOT/.env.example" ]; then
            cp "$PROJECT_ROOT/.env.example" "$PROJECT_ROOT/.env"
            log_success ".env.example에서 .env 파일을 생성했습니다"
        else
            log_error ".env.example 템플릿 파일을 찾을 수 없어 .env 파일을 생성할 수 없습니다"
            return 1
        fi
    else
        log_info ".env 파일이 이미 존재합니다"
    fi

    # 필수 환경 변수 설정 확인
    source "$PROJECT_ROOT/.env"
    local missing_vars=()

    # 기본 변수 확인
    if [ -z "$DB_DRIVER" ]; then missing_vars+=("DB_DRIVER"); fi
    if [ -z "$STORAGE_TYPE" ]; then missing_vars+=("STORAGE_TYPE"); fi

    return 0
}

# Ollama 설치 (플랫폼에 따라 다른 방법 사용)
install_ollama() {
    # 원격 서비스인지 확인
    get_ollama_base_url

    if [ $IS_REMOTE -eq 1 ]; then
        log_info "원격 Ollama 서비스 설정이 감지됨, 로컬에 Ollama를 설치할 필요 없음"
        return 0
    fi

    log_info "로컬 Ollama가 설치되지 않음, 설치 중..."

    OS=$(uname)
    if [ "$OS" = "Darwin" ]; then
        # Mac 설치 방법
        log_info "Mac 시스템 감지됨, brew로 Ollama 설치 중..."
        if ! command -v brew &> /dev/null; then
            # 설치 패키지로 직접 설치
            log_info "Homebrew가 설치되지 않음, 직접 다운로드 방식 사용..."
            curl -fsSL https://ollama.com/download/Ollama-darwin.zip -o ollama.zip
            unzip ollama.zip
            mv ollama /usr/local/bin
            rm ollama.zip
        else
            brew install ollama
        fi
    else
        # Linux 설치 방법
        log_info "Linux 시스템 감지됨, 설치 스크립트 사용..."
        curl -fsSL https://ollama.com/install.sh | sh
    fi

    if [ $? -eq 0 ]; then
        log_success "로컬 Ollama 설치 완료"
        return 0
    else
        log_error "로컬 Ollama 설치 실패"
        return 1
    fi
}

# Ollama 기본 URL 가져오기, 원격 서비스인지 확인
get_ollama_base_url() {

    check_env_file

    # 환경 변수에서 Ollama 기본 URL 가져오기
    OLLAMA_URL=${OLLAMA_BASE_URL:-"http://host.docker.internal:11434"}
    # 호스트 부분 추출
    OLLAMA_HOST=$(echo "$OLLAMA_URL" | sed -E 's|^https?://||' | sed -E 's|:[0-9]+$||' | sed -E 's|/.*$||')
    # 포트 부분 추출
    OLLAMA_PORT=$(echo "$OLLAMA_URL" | grep -oE ':[0-9]+' | grep -oE '[0-9]+' || echo "11434")
    # localhost 또는 127.0.0.1인지 확인
    IS_REMOTE=0
    if [ "$OLLAMA_HOST" = "localhost" ] || [ "$OLLAMA_HOST" = "127.0.0.1" ] || [ "$OLLAMA_HOST" = "host.docker.internal" ]; then
        IS_REMOTE=0  # 로컬 서비스
    else
        IS_REMOTE=1  # 원격 서비스
    fi
}

# Ollama 서비스 시작
start_ollama() {
    log_info "Ollama 서비스 확인 중..."
    # 호스트와 포트 추출
    get_ollama_base_url
    log_info "Ollama 서비스 주소: $OLLAMA_URL"

    if [ $IS_REMOTE -eq 1 ]; then
        log_info "원격 Ollama 서비스 감지됨, 원격 서비스를 직접 사용하며 로컬 설치 및 시작하지 않음"
        # 원격 서비스 사용 가능 여부 확인
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            log_success "원격 Ollama 서비스 접근 가능"
            return 0
        else
            log_warning "원격 Ollama 서비스에 접근할 수 없음, 서비스 주소가 올바르고 시작되었는지 확인하세요"
            return 1
        fi
    fi

    # 아래는 로컬 서비스 처리
    # Ollama 설치 여부 확인
    if ! command -v ollama &> /dev/null; then
        install_ollama
        if [ $? -ne 0 ]; then
            return 1
        fi
    fi

    # Ollama 서비스 실행 중인지 확인
    if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
        log_success "로컬 Ollama 서비스가 이미 실행 중, 포트: $OLLAMA_PORT"
    else
        log_info "로컬 Ollama 서비스 시작 중..."
        # 참고: 공식적으로 systemctl 또는 launchctl로 서비스 관리를 권장, 백그라운드 직접 실행은 임시 시나리오용
        systemctl restart ollama || (ollama serve > /dev/null 2>&1 < /dev/null &)

        # 서비스 시작 대기
        MAX_RETRIES=30
        COUNT=0
        while [ $COUNT -lt $MAX_RETRIES ]; do
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                log_success "로컬 Ollama 서비스가 성공적으로 시작됨, 포트: $OLLAMA_PORT"
                break
            fi
            echo -ne "Ollama 서비스 시작 대기 중... ($COUNT/$MAX_RETRIES)\r"
            sleep 1
            COUNT=$((COUNT + 1))
        done
        echo "" # 줄바꿈

        if [ $COUNT -eq $MAX_RETRIES ]; then
            log_error "로컬 Ollama 서비스 시작 실패"
            return 1
        fi
    fi

    log_success "로컬 Ollama 서비스 주소: http://localhost:$OLLAMA_PORT"
    return 0
}

# Ollama 서비스 중지
stop_ollama() {
    log_info "Ollama 서비스 중지 중..."

    # 원격 서비스인지 확인
    get_ollama_base_url

    if [ $IS_REMOTE -eq 1 ]; then
        log_info "원격 Ollama 서비스 감지됨, 로컬에서 중지할 필요 없음"
        return 0
    fi

    # Ollama 설치 여부 확인
    if ! command -v ollama &> /dev/null; then
        log_info "로컬 Ollama가 설치되지 않음, 중지할 필요 없음"
        return 0
    fi

    # Ollama 프로세스 찾아서 종료
    if pgrep -x "ollama" > /dev/null; then
        # systemctl 우선 사용
        if command -v systemctl &> /dev/null; then
            sudo systemctl stop ollama
        else
            pkill -f "ollama serve"
        fi
        log_success "로컬 Ollama 서비스가 중지됨"
    else
        log_info "로컬 Ollama 서비스가 실행 중이 아님"
    fi

    return 0
}

# Docker 설치 여부 확인
check_docker() {
    log_info "Docker 환경 확인 중..."

    if ! command -v docker &> /dev/null; then
        log_error "Docker가 설치되지 않음, 먼저 Docker를 설치하세요"
        return 1
    fi

	# 사용 가능한 Docker Compose 명령 확인 및 선택
	if detect_compose_cmd; then
		if [ "$DOCKER_COMPOSE_BIN" = "docker" ]; then
			log_info "Docker Compose 플러그인 감지됨 (docker compose)"
		else
			log_info "docker-compose (v1) 감지됨"
		fi
	else
		log_error "Docker Compose를 감지할 수 없음 (docker compose와 docker-compose 모두 없음). 둘 중 하나를 설치하세요."
		return 1
	fi

    # Docker 서비스 실행 상태 확인
    if ! docker info &> /dev/null; then
        log_error "Docker 서비스가 실행 중이 아님, Docker 서비스를 시작하세요"
        return 1
    fi

    log_success "Docker 환경 확인 통과"
    return 0
}

check_platform() {
     # 현재 시스템 플랫폼 감지
    log_info "시스템 플랫폼 정보 감지 중..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
    else
        log_warning "인식되지 않는 플랫폼 유형: $(uname -m), 기본 플랫폼 linux/amd64 사용"
        export PLATFORM="linux/amd64"
    fi
    log_info "현재 플랫폼: $PLATFORM"
}

# Docker 컨테이너 시작
start_docker() {
    log_info "Docker 컨테이너 시작 중..."

    # Docker 환경 확인
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    # .env 파일 확인
    check_env_file

    # .env 파일 읽기
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}

    check_platform

    # 프로젝트 루트 디렉토리로 이동 후 docker-compose 명령 실행
    cd "$PROJECT_ROOT"

    # 기본 서비스 시작
    log_info "핵심 서비스 컨테이너 시작 중..."
	# 감지된 Compose 명령으로 통일하여 시작
	if [ "$NO_PULL" = true ]; then
		# 이미지 가져오지 않음, 로컬 이미지 사용
		log_info "이미지 가져오기 건너뜀, 로컬 이미지 사용..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --build -d
	else
		# 최신 이미지 가져오기
		log_info "최신 이미지 가져오는 중..."
		PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up --pull always -d
	fi
    if [ $? -ne 0 ]; then
        log_error "Docker 컨테이너 시작 실패"
        return 1
    fi

    log_success "모든 Docker 컨테이너가 성공적으로 시작됨"

    # 컨테이너 상태 표시
    log_info "현재 컨테이너 상태:"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps

    return 0
}

# Docker 컨테이너 중지
stop_docker() {
    log_info "Docker 컨테이너 중지 중..."

    # Docker 환경 확인
    check_docker
    if [ $? -ne 0 ]; then
        # 확인 실패해도 중지 시도, 만일을 대비
        log_warning "Docker 환경 확인 실패, 그래도 컨테이너 중지 시도..."
    fi

    # 프로젝트 루트 디렉토리로 이동 후 docker-compose 명령 실행
    cd "$PROJECT_ROOT"

    # 모든 컨테이너 중지
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD down --remove-orphans
    if [ $? -ne 0 ]; then
        log_error "Docker 컨테이너 중지 실패"
        return 1
    fi

    log_success "모든 Docker 컨테이너가 중지됨"
    return 0
}

# 실행 중인 모든 컨테이너 나열
list_containers() {
    log_info "실행 중인 모든 컨테이너 나열 중..."

    # Docker 환경 확인
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    # 프로젝트 루트 디렉토리로 이동 후 docker-compose 명령 실행
    cd "$PROJECT_ROOT"

    # 모든 컨테이너 나열
    printf "%b\n" "${BLUE}현재 실행 중인 컨테이너:${NC}"
	"$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | sort

    return 0
}

# 최신 Docker 이미지 가져오기
pull_images() {
    log_info "최신 Docker 이미지 가져오는 중..."

    # Docker 환경 확인
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    # .env 파일 확인
    check_env_file

    # .env 파일 읽기
    source "$PROJECT_ROOT/.env"
    storage_type=${STORAGE_TYPE:-local}

    check_platform

    # 프로젝트 루트 디렉토리로 이동 후 docker-compose 명령 실행
    cd "$PROJECT_ROOT"

    # 모든 이미지 가져오기
    log_info "모든 서비스의 최신 이미지 가져오는 중..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD pull
    if [ $? -ne 0 ]; then
        log_error "이미지 가져오기 실패"
        return 1
    fi

    log_success "모든 이미지가 최신 버전으로 성공적으로 가져와짐"

    # 가져온 이미지 정보 표시
    log_info "가져온 이미지:"
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.CreatedAt}}\t{{.Size}}" | head -10

    return 0
}

# 지정된 컨테이너 재시작
restart_container() {
    local container_name="$1"

    if [ -z "$container_name" ]; then
        log_error "컨테이너 이름이 지정되지 않음"
        echo "사용 가능한 컨테이너:"
        list_containers
        return 1
    fi

    log_info "컨테이너 재빌드 및 재시작 중: $container_name"

    # Docker 환경 확인
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    check_platform

    # 프로젝트 루트 디렉토리로 이동 후 docker-compose 명령 실행
    cd "$PROJECT_ROOT"

    # 컨테이너 존재 여부 확인
	if ! "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD ps --services | grep -q "^$container_name$"; then
        log_error "컨테이너 '$container_name'이 존재하지 않거나 실행 중이 아님"
        echo "사용 가능한 컨테이너:"
        list_containers
        return 1
    fi

    # 컨테이너 빌드 및 재시작
    log_info "컨테이너 '$container_name' 재빌드 중..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD build "$container_name"
    if [ $? -ne 0 ]; then
        log_error "컨테이너 '$container_name' 빌드 실패"
        return 1
    fi

    log_info "컨테이너 '$container_name' 재시작 중..."
	PLATFORM=$PLATFORM "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD up -d --no-deps "$container_name"
    if [ $? -ne 0 ]; then
        log_error "컨테이너 '$container_name' 재시작 실패"
        return 1
    fi

    log_success "컨테이너 '$container_name'이 성공적으로 재빌드되고 재시작됨"
    return 0
}

# 시스템 환경 확인
check_environment() {
    log_info "환경 확인 시작..."

    # 운영체제 확인
    OS=$(uname)
    log_info "운영체제: $OS"

    # Docker 확인
    check_docker

    # .env 파일 확인
    check_env_file

    get_ollama_base_url

    if [ $IS_REMOTE -eq 1 ]; then
        log_info "원격 Ollama 서비스 설정 감지됨"
        if curl -s "$OLLAMA_URL/api/tags" &> /dev/null; then
            version=$(curl -s "$OLLAMA_URL/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
            log_success "원격 Ollama 서비스 접근 가능, 버전: $version"
        else
            log_warning "원격 Ollama 서비스에 접근할 수 없음, 서비스 주소가 올바르고 시작되었는지 확인하세요"
        fi
    else
        if command -v ollama &> /dev/null; then
            log_success "로컬 Ollama 설치됨"
            if curl -s "http://localhost:$OLLAMA_PORT/api/tags" &> /dev/null; then
                version=$(curl -s "http://localhost:$OLLAMA_PORT/api/tags" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
                log_success "로컬 Ollama 서비스 실행 중, 버전: $version"
            else
                log_warning "로컬 Ollama가 설치되었지만 서비스가 실행 중이 아님"
            fi
        else
            log_warning "로컬 Ollama 설치되지 않음"
        fi
    fi

    # 디스크 공간 확인
    log_info "디스크 공간 확인 중..."
    df -h | grep -E "(Filesystem|/$)"

    # 메모리 확인
    log_info "메모리 사용량 확인 중..."
    if [ "$OS" = "Darwin" ]; then
        vm_stat | perl -ne '/page size of (\d+)/ and $size=$1; /Pages free:\s*(\d+)/ and print "Free Memory: ", $1 * $size / 1048576, " MB\n"'
    else
        free -h | grep -E "(total|Mem:)"
    fi

    # CPU 확인
    log_info "CPU 정보:"
    if [ "$OS" = "Darwin" ]; then
        sysctl -n machdep.cpu.brand_string
        echo "CPU 코어 수: $(sysctl -n hw.ncpu)"
    else
        grep "model name" /proc/cpuinfo | head -1
        echo "CPU 코어 수: $(nproc)"
    fi

    # 컨테이너 상태 확인
    log_info "컨테이너 상태 확인 중..."
    if docker info &> /dev/null; then
        docker ps -a
    else
        log_warning "컨테이너 상태를 가져올 수 없음, Docker가 실행 중이 아닐 수 있음"
    fi

    log_success "환경 확인 완료"
    return 0
}

# 명령줄 인수 파싱
START_OLLAMA=false
START_DOCKER=false
STOP_SERVICES=false
CHECK_ENVIRONMENT=false
LIST_CONTAINERS=false
RESTART_CONTAINER=false
PULL_IMAGES=false
NO_PULL=false
CONTAINER_NAME=""

# 인수가 없을 때 기본적으로 모든 서비스 시작
if [ $# -eq 0 ]; then
    START_OLLAMA=true
    START_DOCKER=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -o | --ollama )     START_OLLAMA=true
                            ;;
        -d | --docker )     START_DOCKER=true
                            ;;
        -a | --all )        START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -s | --stop )       STOP_SERVICES=true
                            ;;
        -c | --check )      CHECK_ENVIRONMENT=true
                            ;;
        -l | --list )       LIST_CONTAINERS=true
                            ;;
        -p | --pull )       PULL_IMAGES=true
                            ;;
        --no-pull )         NO_PULL=true
                            START_OLLAMA=true
                            START_DOCKER=true
                            ;;
        -r | --restart )    RESTART_CONTAINER=true
                            CONTAINER_NAME="$2"
                            shift
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "알 수 없는 옵션: $1"
                            show_help
                            ;;
    esac
    shift
done

# 환경 확인 실행
if [ "$CHECK_ENVIRONMENT" = true ]; then
    check_environment
    exit $?
fi

# 모든 컨테이너 나열
if [ "$LIST_CONTAINERS" = true ]; then
    list_containers
    exit $?
fi

# 최신 이미지 가져오기
if [ "$PULL_IMAGES" = true ]; then
    pull_images
    exit $?
fi

# 지정된 컨테이너 재시작
if [ "$RESTART_CONTAINER" = true ]; then
    restart_container "$CONTAINER_NAME"
    exit $?
fi

# 서비스 작업 실행
if [ "$STOP_SERVICES" = true ]; then
    # 서비스 중지
    stop_ollama
    OLLAMA_RESULT=$?

    stop_docker
    DOCKER_RESULT=$?

    # 요약 표시
    echo ""
    log_info "=== 중지 결과 ==="
    if [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "✓ Ollama 서비스 중지됨"
    else
        log_error "✗ Ollama 서비스 중지 실패"
    fi

    if [ $DOCKER_RESULT -eq 0 ]; then
        log_success "✓ Docker 컨테이너 중지됨"
    else
        log_error "✗ Docker 컨테이너 중지 실패"
    fi

    log_success "서비스 중지 완료."
else
    # 서비스 시작
    OLLAMA_RESULT=1
    DOCKER_RESULT=1
    if [ "$START_OLLAMA" = true ]; then
        start_ollama
        OLLAMA_RESULT=$?
    fi

    if [ "$START_DOCKER" = true ]; then
        start_docker
        DOCKER_RESULT=$?
    fi

    # 요약 표시
    echo ""
    log_info "=== 시작 결과 ==="
    if [ "$START_OLLAMA" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ]; then
            log_success "✓ Ollama 서비스 시작됨"
        else
            log_error "✗ Ollama 서비스 시작 실패"
        fi
    fi

    if [ "$START_DOCKER" = true ]; then
        if [ $DOCKER_RESULT -eq 0 ]; then
            log_success "✓ Docker 컨테이너 시작됨"
        else
            log_error "✗ Docker 컨테이너 시작 실패"
        fi
    fi

    if [ "$START_OLLAMA" = true ] && [ "$START_DOCKER" = true ]; then
        if [ $OLLAMA_RESULT -eq 0 ] && [ $DOCKER_RESULT -eq 0 ]; then
            log_success "모든 서비스 시작 완료, 다음 주소로 접속 가능:"
            printf "%b\n" "${GREEN}  - 프론트엔드 인터페이스: http://localhost:${FRONTEND_PORT:-80}${NC}"
            printf "%b\n" "${GREEN}  - API 인터페이스: http://localhost:${APP_PORT:-8080}${NC}"
            printf "%b\n" "${GREEN}  - Jaeger 분산 추적: http://localhost:16686${NC}"
            echo ""
            log_info "컨테이너 로그를 계속 출력합니다 (Ctrl+C로 로그 종료, 컨테이너는 중지되지 않음)..."
            "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
        else
            log_error "일부 서비스 시작 실패, 로그를 확인하고 문제를 해결하세요"
        fi
    elif [ "$START_OLLAMA" = true ] && [ $OLLAMA_RESULT -eq 0 ]; then
        log_success "Ollama 서비스 시작 완료, 다음 주소로 접속 가능:"
        printf "%b\n" "${GREEN}  - Ollama API: http://localhost:$OLLAMA_PORT${NC}"
    elif [ "$START_DOCKER" = true ] && [ $DOCKER_RESULT -eq 0 ]; then
        log_success "Docker 컨테이너 시작 완료, 다음 주소로 접속 가능:"
        printf "%b\n" "${GREEN}  - 프론트엔드 인터페이스: http://localhost:${FRONTEND_PORT:-80}${NC}"
        printf "%b\n" "${GREEN}  - API 인터페이스: http://localhost:${APP_PORT:-8080}${NC}"
        printf "%b\n" "${GREEN}  - Jaeger 분산 추적: http://localhost:16686${NC}"
        echo ""
        log_info "컨테이너 로그를 계속 출력합니다 (Ctrl+C로 로그 종료, 컨테이너는 중지되지 않음)..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD logs app docreader postgres --since=10s -f
    fi
fi

exit 0
