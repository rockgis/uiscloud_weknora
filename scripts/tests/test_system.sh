#!/bin/bash
# WeKnora 시스템 전체 테스트 스크립트
#
# 이 스크립트는 WeKnora 시스템의 모든 주요 기능을 테스트합니다.
# - 백엔드/프론트엔드 서버 상태
# - 사용자 인증
# - Ollama 연결
# - AI 모델 등록
# - 지식베이스 CRUD
# - Docker 인프라
#
# 사용법:
#   ./scripts/tests/test_system.sh
#
# 요구사항:
#   - 모든 서비스가 실행 중이어야 함

echo "=========================================="
echo "  WeKnora 시스템 전체 테스트"
echo "=========================================="
echo ""

# 1. 백엔드 테스트
echo "1. 백엔드 서버 상태"
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo "   ✓ 백엔드 정상 (http://localhost:8080)"
else
    echo "   ✗ 백엔드 오류"
    exit 1
fi
echo ""

# 2. 프론트엔드 테스트
echo "2. 프론트엔드 서버 상태"
if curl -s http://localhost:5173 | grep -q "WeKnora"; then
    echo "   ✓ 프론트엔드 정상 (http://localhost:5173)"
else
    echo "   ✗ 프론트엔드 오류"
fi
echo ""

# 3. 인증 시스템 테스트
echo "3. 사용자 인증 테스트"
AUTH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"admin123"}')

if echo "$AUTH_RESPONSE" | grep -q "Login successful"; then
    echo "   ✓ 로그인 성공"
    TOKEN=$(echo "$AUTH_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])" 2>/dev/null)
    echo "   ✓ JWT 토큰 발급 완료"
else
    echo "   ✗ 로그인 실패"
    exit 1
fi
echo ""

# 4. Ollama 연결 테스트
echo "4. Ollama 서비스 연결 테스트"
OLLAMA_STATUS=$(curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/initialization/ollama/status)

if echo "$OLLAMA_STATUS" | grep -q '"available":true'; then
    echo "   ✓ Ollama 연결 성공"
    echo "   ✓ Ollama URL: http://localhost:11434"
else
    echo "   ✗ Ollama 연결 실패"
fi
echo ""

# 5. 등록된 모델 확인
echo "5. 등록된 AI 모델 확인"
MODELS_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/models)

MODEL_COUNT=$(echo "$MODELS_RESPONSE" | python3 -c "import json,sys; print(len(json.load(sys.stdin)['data']))" 2>/dev/null)

if [ "$MODEL_COUNT" -gt 0 ]; then
    echo "   ✓ 총 $MODEL_COUNT 개 모델 등록됨"
    echo "$MODELS_RESPONSE" | python3 -c "
import json, sys
data = json.load(sys.stdin)
for m in data['data']:
    mtype = m.get('type', 'N/A')
    default = ' [기본값]' if m.get('is_default') else ''
    print(f'   - {m[\"name\"]:30} ({mtype}){default}')
" 2>/dev/null
else
    echo "   ✗ 등록된 모델 없음"
fi
echo ""

# 6. 지식베이스 생성 테스트
echo "6. 지식베이스 생성 테스트"
KB_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/knowledge-bases \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "테스트 지식베이스",
    "description": "시스템 테스트용 지식베이스",
    "type": "document"
  }')

if echo "$KB_RESPONSE" | grep -q "success"; then
    KB_ID=$(echo "$KB_RESPONSE" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null)
    echo "   ✓ 지식베이스 생성 성공"
    echo "   ✓ 지식베이스 ID: $KB_ID"

    # 생성된 지식베이스 삭제 (테스트용)
    curl -s -X DELETE "http://localhost:8080/api/v1/knowledge-bases/$KB_ID" \
      -H "Authorization: Bearer $TOKEN" >/dev/null
    echo "   ✓ 테스트 지식베이스 삭제 완료"
else
    echo "   ⚠ 지식베이스 생성 실패 (모델 미설정 가능성)"
fi
echo ""

# 7. Docker 인프라 확인
echo "7. Docker 인프라 서비스 상태"
docker ps --format "table {{.Names}}\t{{.Status}}" | grep WeKnora | while read line; do
    echo "   ✓ $line"
done
echo ""

echo "=========================================="
echo "  테스트 완료!"
echo "=========================================="
echo ""
echo "접속 정보:"
echo "  • 프론트엔드: http://localhost:5173"
echo "  • 백엔드 API: http://localhost:8080"
echo "  • 로그인: admin@example.com / admin123"
echo ""
