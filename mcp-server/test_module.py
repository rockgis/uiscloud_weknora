#!/usr/bin/env python3
"""
WeKnora MCP Server 모듈 테스트 스크립트

모듈의 다양한 시작 방법과 기능을 테스트합니다.
"""

import os
import subprocess
import sys
from pathlib import Path


def test_imports():
    """모듈 임포트 테스트"""
    print("=== 모듈 임포트 테스트 ===")

    try:
        # 기본 의존성 테스트
        import mcp

        print("✓ mcp 모듈 임포트 성공")

        import requests

        print("✓ requests 모듈 임포트 성공")

        # 메인 모듈 테스트
        import weknora_mcp_server

        print("✓ weknora_mcp_server 모듈 임포트 성공")

        # 패키지 임포트 테스트
        from weknora_mcp_server import WeKnoraClient, run

        print("✓ WeKnoraClient와 run 함수 임포트 성공")

        # 메인 진입점 테스트
        import main

        print("✓ main 모듈 임포트 성공")

        return True

    except ImportError as e:
        print(f"✗ 임포트 실패: {e}")
        return False


def test_environment():
    """환경 설정 테스트"""
    print("\n=== 환경 설정 테스트 ===")

    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    print(f"WEKNORA_BASE_URL: {base_url or '설정되지 않음 (기본값 사용)'}")
    print(f"WEKNORA_API_KEY: {'설정됨' if api_key else '설정되지 않음'}")

    if not base_url:
        print("팁: WEKNORA_BASE_URL 환경 변수를 설정할 수 있습니다")

    if not api_key:
        print("팁: WEKNORA_API_KEY 환경 변수 설정을 권장합니다")

    return True


def test_client_creation():
    """클라이언트 생성 테스트"""
    print("\n=== 클라이언트 생성 테스트 ===")

    try:
        from weknora_mcp_server import WeKnoraClient

        base_url = os.getenv("WEKNORA_BASE_URL", "http://localhost:8080/api/v1")
        api_key = os.getenv("WEKNORA_API_KEY", "test_key")

        client = WeKnoraClient(base_url, api_key)
        print("✓ WeKnoraClient 생성 성공")

        # 클라이언트 속성 확인
        assert client.base_url == base_url
        assert client.api_key == api_key
        print("✓ 클라이언트 설정 정확함")

        return True

    except Exception as e:
        print(f"✗ 클라이언트 생성 실패: {e}")
        return False


def test_file_structure():
    """파일 구조 테스트"""
    print("\n=== 파일 구조 테스트 ===")

    required_files = [
        "__init__.py",
        "main.py",
        "run_server.py",
        "weknora_mcp_server.py",
        "requirements.txt",
        "setup.py",
        "pyproject.toml",
        "README.md",
        "INSTALL.md",
        "LICENSE",
        "MANIFEST.in",
    ]

    missing_files = []
    for file in required_files:
        if Path(file).exists():
            print(f"✓ {file}")
        else:
            print(f"✗ {file} (누락)")
            missing_files.append(file)

    if missing_files:
        print(f"누락된 파일: {missing_files}")
        return False

    print("✓ 모든 필수 파일이 존재합니다")
    return True


def test_entry_points():
    """진입점 테스트"""
    print("\n=== 진입점 테스트 ===")

    # main.py 도움말 옵션 테스트
    try:
        result = subprocess.run(
            [sys.executable, "main.py", "--help"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode == 0:
            print("✓ main.py --help 정상 작동")
        else:
            print(f"✗ main.py --help 실패: {result.stderr}")
            return False
    except subprocess.TimeoutExpired:
        print("✗ main.py --help 시간 초과")
        return False
    except Exception as e:
        print(f"✗ main.py --help 오류: {e}")
        return False

    # 환경 확인 테스트
    try:
        result = subprocess.run(
            [sys.executable, "main.py", "--check-only"],
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode == 0:
            print("✓ main.py --check-only 정상 작동")
        else:
            print(f"✗ main.py --check-only 실패: {result.stderr}")
            return False
    except subprocess.TimeoutExpired:
        print("✗ main.py --check-only 시간 초과")
        return False
    except Exception as e:
        print(f"✗ main.py --check-only 오류: {e}")
        return False

    return True


def test_package_installation():
    """패키지 설치 테스트 (개발 모드)"""
    print("\n=== 패키지 설치 테스트 ===")

    try:
        # 개발 모드로 설치 가능한지 확인
        result = subprocess.run(
            [sys.executable, "setup.py", "check"],
            capture_output=True,
            text=True,
            timeout=30,
        )

        if result.returncode == 0:
            print("✓ setup.py 검사 통과")
        else:
            print(f"✗ setup.py 검사 실패: {result.stderr}")
            return False

    except subprocess.TimeoutExpired:
        print("✗ setup.py 검사 시간 초과")
        return False
    except Exception as e:
        print(f"✗ setup.py 검사 오류: {e}")
        return False

    return True


def main():
    """모든 테스트 실행"""
    print("WeKnora MCP Server 모듈 테스트")
    print("=" * 50)

    tests = [
        ("모듈 임포트", test_imports),
        ("환경 설정", test_environment),
        ("클라이언트 생성", test_client_creation),
        ("파일 구조", test_file_structure),
        ("진입점", test_entry_points),
        ("패키지 설치", test_package_installation),
    ]

    passed = 0
    total = len(tests)

    for test_name, test_func in tests:
        try:
            if test_func():
                passed += 1
            else:
                print(f"테스트 실패: {test_name}")
        except Exception as e:
            print(f"테스트 예외: {test_name} - {e}")

    print("\n" + "=" * 50)
    print(f"테스트 결과: {passed}/{total} 통과")

    if passed == total:
        print("✓ 모든 테스트 통과! 모듈을 정상적으로 사용할 수 있습니다.")
        return True
    else:
        print("✗ 일부 테스트 실패. 위의 오류를 확인하세요.")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
