#!/usr/bin/env python3
"""
MCP 임포트 테스트
"""

try:
    import mcp.types as types

    print("✓ mcp.types 임포트 성공")
except ImportError as e:
    print(f"✗ mcp.types 임포트 실패: {e}")

try:
    from mcp.server import NotificationOptions, Server

    print("✓ mcp.server 임포트 성공")
except ImportError as e:
    print(f"✗ mcp.server 임포트 실패: {e}")

try:
    import mcp.server.stdio

    print("✓ mcp.server.stdio 임포트 성공")
except ImportError as e:
    print(f"✗ mcp.server.stdio 임포트 실패: {e}")

try:
    from mcp.server.models import InitializationOptions

    print("✓ InitializationOptions mcp.server.models에서 임포트 성공")
except ImportError:
    try:
        from mcp import InitializationOptions

        print("✓ InitializationOptions mcp에서 임포트 성공")
    except ImportError as e:
        print(f"✗ InitializationOptions 임포트 실패: {e}")

# MCP 패키지 구조 확인
import mcp

print(f"\nMCP 패키지 버전: {getattr(mcp, '__version__', '알 수 없음')}")
print(f"MCP 패키지 경로: {mcp.__file__}")
print(f"MCP 패키지 내용: {dir(mcp)}")
