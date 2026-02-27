#!/bin/bash
# uiscloud_weknora 시스템 상태 확인 스크립트
#
# 모든 서비스의 상태를 빠르게 확인합니다.
#
# 사용법:
#   ./scripts/tests/check_status.sh

echo "=========================================="
echo "  uiscloud_weknora 시스템 상태 확인"
echo "=========================================="
echo ""

# 서버 상태
curl -s http://localhost:8080/health | grep -q ok && echo "✓ 백엔드 정상 (포트 8080)" || echo "✗ 백엔드 오류"
curl -s http://localhost:5173 | grep -q uiscloud_weknora && echo "✓ 프론트엔드 정상 (포트 5173)" || echo "✗ 프론트엔드 오류"
curl -s http://localhost:11434/api/version | grep -q version && echo "✓ Ollama 정상 (포트 11434)" || echo "✗ Ollama 오류"

echo ""
echo "Docker 인프라 서비스:"
docker ps --format "✓ {{.Names}}: {{.Status}}" | grep uiscloud_weknora

echo ""
echo "=========================================="
echo "  상태 확인 완료"
echo "=========================================="
