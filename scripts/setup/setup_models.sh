#!/bin/bash
# WeKnora AI 모델 자동 설정 스크립트
#
# 이 스크립트는 WeKnora에 필요한 AI 모델들을 자동으로 등록합니다.
# - Embedding 모델: nomic-embed-text (벡터 검색용)
# - LLM 모델: llama3.1:8b (기본 대화 모델)
#
# 사용법:
#   ./scripts/setup/setup_models.sh
#
# 요구사항:
#   - 백엔드 서버가 http://localhost:8080 에서 실행 중이어야 함
#   - admin 계정이 생성되어 있어야 함 (admin@example.com / admin123)
#   - Ollama가 http://localhost:11434 에서 실행 중이어야 함
#   - nomic-embed-text 모델이 Ollama에 설치되어 있어야 함

set -e

echo "=== WeKnora 모델 자동 설정 시작 ==="
echo ""

# 1. 로그인하여 토큰 받기
echo "1. 로그인 중..."
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import json, sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)

if [ -z "$TOKEN" ]; then
  echo "❌ 로그인 실패"
  echo "   admin@example.com 계정이 생성되어 있는지 확인하세요."
  exit 1
fi
echo "✓ 로그인 성공"

# 2. Embedding 모델 등록
echo ""
echo "2. Embedding 모델 등록 중 (nomic-embed-text)..."
EMBED_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/models \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "nomic-embed-text:latest",
    "type": "Embedding",
    "source": "local",
    "description": "Nomic Embedding - 벡터 검색용",
    "parameters": {
      "base_url": "http://localhost:11434",
      "api_key": "",
      "interface_type": "ollama",
      "embedding_parameters": {
        "dimension": 768,
        "truncate_prompt_tokens": 512
      }
    },
    "is_default": true,
    "status": "active"
  }')

echo "$EMBED_RESPONSE" | python3 -c "import json, sys; data=json.load(sys.stdin); print('✓ Embedding 모델 등록 완료: ' + data.get('data', {}).get('id', 'N/A'))" 2>/dev/null || echo "⚠ Embedding 모델 등록 중 오류 (이미 존재할 수 있음)"

# 3. 기본 LLM 모델 기본값으로 설정
echo ""
echo "3. LLM 모델 기본값 설정 중 (llama3.1:8b)..."
MODELS=$(curl -s -X GET http://localhost:8080/api/v1/models -H "Authorization: Bearer $TOKEN")
LLAMA_ID=$(echo "$MODELS" | python3 -c "import json, sys; models=json.load(sys.stdin).get('data', []); print(next((m['id'] for m in models if 'llama3.1' in m['name']), ''))" 2>/dev/null)

if [ -n "$LLAMA_ID" ]; then
  curl -s -X PUT "http://localhost:8080/api/v1/models/$LLAMA_ID" \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{
      "is_default": true
    }' >/dev/null
  echo "✓ llama3.1:8b를 기본 LLM으로 설정 완료"
else
  echo "⚠ llama3.1:8b 모델을 찾을 수 없음"
fi

# 4. 등록된 모델 확인
echo ""
echo "4. 등록된 모델 확인:"
curl -s -X GET http://localhost:8080/api/v1/models -H "Authorization: Bearer $TOKEN" \
  | python3 -c "
import json, sys
data = json.load(sys.stdin)
models = data.get('data', [])
for m in models:
    default = ' [기본값]' if m.get('is_default') else ''
    print(f\"  - {m['name']} ({m['type']}) - {m['status']}{default}\")
" 2>/dev/null || echo "  (파싱 오류)"

echo ""
echo "=== 모델 설정 완료! ==="
echo ""
echo "이제 http://localhost:5173 에서 WeKnora를 사용할 수 있습니다!"
