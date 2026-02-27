#!/usr/bin/env python3
"""
uiscloud_weknora MCP Server 시작 스크립트
"""

import asyncio
import os
import sys


def check_environment():
    """환경 설정 확인"""
    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    if not base_url:
        print(
            "경고: WEKNORA_BASE_URL 환경 변수가 설정되지 않았습니다. 기본값 사용: http://localhost:8080/api/v1"
        )

    if not api_key:
        print("경고: WEKNORA_API_KEY 환경 변수가 설정되지 않았습니다")

    print(f"uiscloud_weknora Base URL: {base_url or 'http://localhost:8080/api/v1'}")
    print(f"API Key: {'설정됨' if api_key else '설정되지 않음'}")


def main():
    """메인 함수"""
    print("uiscloud_weknora MCP Server 시작 중...")
    check_environment()

    try:
        from weknora_mcp_server import run

        asyncio.run(run())
    except ImportError as e:
        print(f"임포트 오류: {e}")
        print("모든 의존성이 설치되었는지 확인하세요: pip install -r requirements.txt")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n서버가 중지되었습니다")
    except Exception as e:
        print(f"서버 실행 오류: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
