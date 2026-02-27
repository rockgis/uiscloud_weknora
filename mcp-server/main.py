#!/usr/bin/env python3
"""
uiscloud_weknora MCP Server 메인 진입점

이 파일은 uiscloud_weknora MCP 서버를 시작하기 위한 통합 진입점을 제공합니다.
다음과 같은 방법으로 실행할 수 있습니다:
1. python main.py
2. python -m weknora_mcp_server
3. weknora-mcp-server (설치 후)
"""

import argparse
import asyncio
import os
import sys
from pathlib import Path


def setup_environment():
    """환경 및 경로 설정"""
    # 현재 디렉토리가 Python 경로에 있는지 확인
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))


def check_dependencies():
    """의존성 설치 여부 확인"""
    try:
        import mcp
        import requests

        return True
    except ImportError as e:
        print(f"의존성 누락: {e}")
        print("다음 명령어를 실행하세요: pip install -r requirements.txt")
        return False


def check_environment_variables():
    """환경 변수 설정 확인"""
    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    print("=== uiscloud_weknora MCP Server 환경 확인 ===")
    print(f"Base URL: {base_url or 'http://localhost:8080/api/v1 (기본값)'}")
    print(f"API Key: {'설정됨' if api_key else '설정되지 않음 (경고)'}")

    if not base_url:
        print("팁: WEKNORA_BASE_URL 환경 변수를 설정할 수 있습니다")

    if not api_key:
        print("경고: WEKNORA_API_KEY 환경 변수 설정을 권장합니다")

    print("=" * 40)
    return True


def parse_arguments():
    """명령줄 인자 파싱"""
    parser = argparse.ArgumentParser(
        description="uiscloud_weknora MCP Server - uiscloud_weknora API용 Model Context Protocol 서버",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
예시:
  python main.py                    # 기본 설정으로 시작
  python main.py --check-only       # 환경만 확인, 서버 시작 안 함
  python main.py --verbose          # 상세 로그 활성화

환경 변수:
  WEKNORA_BASE_URL    uiscloud_weknora API 기본 URL (기본값: http://localhost:8080/api/v1)
  WEKNORA_API_KEY     uiscloud_weknora API 키
        """,
    )

    parser.add_argument(
        "--check-only", action="store_true", help="환경 설정만 확인, 서버 시작 안 함"
    )

    parser.add_argument("--verbose", "-v", action="store_true", help="상세 로그 출력 활성화")

    parser.add_argument(
        "--version", action="version", version="uiscloud_weknora MCP Server 1.0.0"
    )

    return parser.parse_args()


async def main():
    """메인 함수"""
    args = parse_arguments()

    # 환경 설정
    setup_environment()

    # 의존성 확인
    if not check_dependencies():
        sys.exit(1)

    # 환경 변수 확인
    check_environment_variables()

    # 환경 확인만 하는 경우 종료
    if args.check_only:
        print("환경 확인 완료.")
        return

    # 로그 레벨 설정
    if args.verbose:
        import logging

        logging.basicConfig(level=logging.DEBUG)
        print("상세 로그 모드 활성화됨")

    try:
        print("uiscloud_weknora MCP Server 시작 중...")

        # 서버 가져오기 및 실행
        from weknora_mcp_server import run

        await run()

    except ImportError as e:
        print(f"임포트 오류: {e}")
        print("모든 파일이 올바른 위치에 있는지 확인하세요")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n서버가 중지되었습니다")
    except Exception as e:
        print(f"서버 실행 오류: {e}")
        if args.verbose:
            import traceback

            traceback.print_exc()
        sys.exit(1)


def sync_main():
    """entry_points용 동기 버전 메인 함수"""
    asyncio.run(main())


if __name__ == "__main__":
    asyncio.run(main())
