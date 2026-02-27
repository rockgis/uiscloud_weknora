# 변경 이력

모든 중요한 프로젝트 변경 사항이 이 파일에 기록됩니다.

형식은 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)를 기반으로 하며,
이 프로젝트는 [시맨틱 버전](https://semver.org/lang/zh-CN/)을 따릅니다.

## [1.0.0] - 2024-01-XX

### 추가
- 초기 버전 출시
- uiscloud_weknora MCP Server 핵심 기능
- 완전한 uiscloud_weknora API 통합
- 테넌트 관리 도구
- 지식 베이스 관리 도구
- 지식 관리 도구
- 모델 관리 도구
- 세션 관리 도구
- 채팅 기능 도구
- 청크 관리 도구
- 다양한 시작 방법 지원
- 명령줄 인수 지원
- 환경 변수 설정
- 완전한 패키지 설치 지원
- 개발 및 프로덕션 모드
- 상세한 문서 및 설치 가이드

### 도구 목록
- `create_tenant` - 새 테넌트 생성
- `list_tenants` - 모든 테넌트 목록 조회
- `create_knowledge_base` - 지식 베이스 생성
- `list_knowledge_bases` - 지식 베이스 목록 조회
- `get_knowledge_base` - 지식 베이스 상세 정보 조회
- `delete_knowledge_base` - 지식 베이스 삭제
- `hybrid_search` - 하이브리드 검색
- `create_knowledge_from_url` - URL에서 지식 생성
- `list_knowledge` - 지식 목록 조회
- `get_knowledge` - 지식 상세 정보 조회
- `delete_knowledge` - 지식 삭제
- `create_model` - 모델 생성
- `list_models` - 모델 목록 조회
- `get_model` - 모델 상세 정보 조회
- `create_session` - 채팅 세션 생성
- `get_session` - 세션 상세 정보 조회
- `list_sessions` - 세션 목록 조회
- `delete_session` - 세션 삭제
- `chat` - 채팅 메시지 전송
- `list_chunks` - 지식 청크 목록 조회
- `delete_chunk` - 지식 청크 삭제

### 파일 구조
```
uiscloud_weknora/
├── __init__.py              # 패키지 초기화 파일
├── main.py                  # 주 진입점 (권장)
├── run.py                   # 편의 시작 스크립트
├── run_server.py           # 원본 시작 스크립트
├── weknora_mcp_server.py   # MCP 서버 구현
├── test_module.py          # 모듈 테스트 스크립트
├── requirements.txt        # 의존성 목록
├── setup.py               # 설치 스크립트 (전통 방식)
├── pyproject.toml         # 모던 프로젝트 설정
├── MANIFEST.in            # 포함 파일 목록
├── LICENSE                # MIT 라이선스
├── README.md              # 프로젝트 설명
├── INSTALL.md             # 상세 설치 가이드
└── CHANGELOG.md           # 변경 이력
```

### 시작 방법
1. `python main.py` - 주 진입점 (권장)
2. `python run_server.py` - 원본 시작 스크립트
3. `python run.py` - 편의 시작 스크립트
4. `python weknora_mcp_server.py` - 직접 실행
5. `python -m weknora_mcp_server` - 모듈 실행
6. `weknora-mcp-server` - 설치 후 명령줄 도구
7. `weknora-server` - 설치 후 명령줄 도구 (별칭)

### 기술 특성
- Model Context Protocol (MCP) 1.0.0+ 기반
- 비동기 I/O 지원
- 완전한 오류 처리
- 상세한 로깅
- 환경 변수 설정
- 명령줄 인수 지원
- 다양한 설치 방법
- 개발 및 프로덕션 모드
- 완전한 테스트 커버리지

### 의존성
- Python 3.10+
- mcp >= 1.0.0
- requests >= 2.31.0

### 호환성
- Windows, macOS, Linux 지원
- Python 3.10-3.12 지원
- 모던 Python 패키지 관리 도구 호환
