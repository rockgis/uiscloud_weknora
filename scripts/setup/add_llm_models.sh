#!/bin/bash
# WeKnora 추가 LLM 모델 등록 스크립트
#
# 이 스크립트는 기본 모델 외에 추가 LLM 모델들을 등록합니다.
# - gemma2:9b (Agent 모드용 고성능 모델)
# - qwen3:8b (다국어 지원 모델)
#
# 사용법:
#   ./scripts/setup/add_llm_models.sh
#
# 요구사항:
#   - 백엔드 서버가 실행 중이어야 함
#   - admin 계정으로 로그인 가능해야 함
#   - Ollama에 해당 모델들이 설치되어 있어야 함

set -e

echo "=== 추가 LLM 모델 등록 시작 ==="
echo ""

# 로그인
echo "로그인 중..."
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import json, sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo "❌ 로그인 실패"
  exit 1
fi
echo "✓ 로그인 성공"
echo ""

# gemma2:9b 등록
echo "1. gemma2:9b 등록 중..."
curl -s -X POST http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "gemma2:9b",
    "type": "KnowledgeQA",
    "source": "local",
    "description": "Gemma2 9B - 고성능 대화 모델 (Agent 모드)",
    "parameters": {
      "base_url": "http://localhost:11434",
      "api_key": "",
      "interface_type": "ollama",
      "parameter_size": "9B"
    },
    "is_default": false,
    "status": "active"
  }' >/dev/null && echo "✓ gemma2:9b 등록 완료"

sleep 1

# qwen3:8b 등록
echo "2. qwen3:8b 등록 중..."
curl -s -X POST http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "qwen3:8b",
    "type": "KnowledgeQA",
    "source": "local",
    "description": "Qwen3 8B - 다국어 대화 모델",
    "parameters": {
      "base_url": "http://localhost:11434",
      "api_key": "",
      "interface_type": "ollama",
      "parameter_size": "8B"
    },
    "is_default": false,
    "status": "active"
  }' >/dev/null && echo "✓ qwen3:8b 등록 완료"

echo ""
echo "최종 모델 목록:"
curl -s -X GET http://localhost:8080/api/v1/models -H "Authorization: Bearer $TOKEN" \
  | python3 -c "
import json, sys
data = json.load(sys.stdin)
models = data.get('data', [])
for m in models:
    default = ' ⭐ [기본값]' if m.get('is_default') else ''
    print(f\"  • {m['name']:30} ({m['type']:12}) - {m['status']}{default}\")
" 2>/dev/null

echo ""
echo "=== 추가 모델 등록 완료! ==="
