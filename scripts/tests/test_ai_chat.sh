#!/bin/bash
# uiscloud_weknora AI 대화 기능 테스트 스크립트
#
# 이 스크립트는 AI 대화 관련 기능들을 테스트합니다.
# - 대화 세션 생성
# - Ollama LLM 응답 테스트
# - Embedding 모델 테스트
#
# 사용법:
#   ./scripts/tests/test_ai_chat.sh
#
# 요구사항:
#   - 백엔드 서버 실행 중
#   - Ollama 실행 중
#   - LLM 및 Embedding 모델 등록됨

echo "=========================================="
echo "  AI 대화 기능 테스트"
echo "=========================================="
echo ""

# 로그인
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)

# 1. 세션 생성 테스트
echo "1. 대화 세션 생성 테스트"
SESSION_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/sessions \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "테스트 대화",
    "mode": "normal"
  }')

if echo "$SESSION_RESPONSE" | grep -q "success"; then
    SESSION_ID=$(echo "$SESSION_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    echo "   ✓ 세션 생성 성공"
    echo "   ✓ 세션 ID: $SESSION_ID"
else
    echo "   ✗ 세션 생성 실패"
    exit 1
fi
echo ""

# 2. Ollama 모델 테스트 (간단한 응답)
echo "2. Ollama LLM 모델 응답 테스트"
echo "   질문: 안녕하세요?"
OLLAMA_TEST=$(curl -s -X POST http://localhost:11434/api/generate \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "llama3.1:8b",
    "prompt": "안녕하세요? 간단히 인사해주세요.",
    "stream": false
  }')

if echo "$OLLAMA_TEST" | grep -q "response"; then
    RESPONSE=$(echo "$OLLAMA_TEST" | python3 -c "import json,sys; print(json.load(sys.stdin).get('response','')[:100])" 2>/dev/null)
    echo "   ✓ Ollama 모델 응답 성공"
    echo "   ✓ 응답: $RESPONSE..."
else
    echo "   ⚠ Ollama 직접 호출 테스트 실패"
fi
echo ""

# 3. Embedding 모델 테스트
echo "3. Embedding 모델 테스트"
EMBED_TEST=$(curl -s -X POST http://localhost:11434/api/embeddings \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "nomic-embed-text",
    "prompt": "테스트 텍스트"
  }')

if echo "$EMBED_TEST" | grep -q "embedding"; then
    EMBED_DIM=$(echo "$EMBED_TEST" | python3 -c "import json,sys; print(len(json.load(sys.stdin).get('embedding',[])))" 2>/dev/null)
    echo "   ✓ Embedding 모델 응답 성공"
    echo "   ✓ 벡터 차원: $EMBED_DIM"
else
    echo "   ⚠ Embedding 모델 테스트 실패"
fi
echo ""

# 세션 삭제
echo "4. 테스트 세션 정리"
curl -s -X DELETE "http://localhost:8080/api/v1/sessions/$SESSION_ID" \
  -H "Authorization: Bearer $TOKEN" >/dev/null
echo "   ✓ 테스트 세션 삭제 완료"
echo ""

echo "=========================================="
echo "  AI 대화 기능 테스트 완료!"
echo "=========================================="
echo ""
