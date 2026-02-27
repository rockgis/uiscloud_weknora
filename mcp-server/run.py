#!/usr/bin/env python3
"""
uiscloud_weknora MCP Server 간편 시작 스크립트

기본적인 기능만 제공하는 간소화된 시작 스크립트입니다.
더 많은 옵션이 필요하면 main.py를 사용하세요.
"""

import os
import sys
from pathlib import Path


def main():
    """간단한 시작 함수"""
    # 현재 디렉토리를 Python 경로에 추가
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))

    # 환경 변수 확인
    base_url = os.getenv("WEKNORA_BASE_URL", "http://localhost:8080/api/v1")
    api_key = os.getenv("WEKNORA_API_KEY", "")

    print("uiscloud_weknora MCP Server")
    print(f"Base URL: {base_url}")
    print(f"API Key: {'설정됨' if api_key else '설정되지 않음'}")
    print("-" * 40)

    try:
        # 가져오기 및 실행
        from main import sync_main

        sync_main()
    except ImportError:
        print("오류: 필요한 모듈을 가져올 수 없습니다")
        print("다음 명령어를 실행하세요: pip install -r requirements.txt")
        sys.exit(1)
    except KeyboardInterrupt:
        print("\n서버가 중지되었습니다")
    except Exception as e:
        print(f"오류: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
