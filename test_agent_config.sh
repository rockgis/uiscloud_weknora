#!/bin/bash

# Agent configuration feature test script

set -e

echo "========================================="
echo "Agent Configuration Feature Test"
echo "========================================="
echo ""

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_BASE_URL="http://localhost:8080"
KB_ID="kb-00000001"
TENANT_ID="1"

echo "Configuration:"
echo "  API URL: ${API_BASE_URL}"
echo "  Knowledge Base ID: ${KB_ID}"
echo "  Tenant ID: ${TENANT_ID}"
echo ""

echo -e "${YELLOW}Test 1: Get current configuration${NC}"
echo "GET ${API_BASE_URL}/api/v1/initialization/config/${KB_ID}"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/initialization/config/${KB_ID}")
echo "Response:"
echo "$RESPONSE" | jq '.data.agent' || echo "$RESPONSE"
echo ""

echo -e "${YELLOW}Test 2: Save Agent configuration${NC}"
echo "POST ${API_BASE_URL}/api/v1/initialization/initialize/${KB_ID}"

TEST_DATA='{
  "llm": {
    "source": "local",
    "modelName": "qwen3:0.6b",
    "baseUrl": "",
    "apiKey": ""
  },
  "embedding": {
    "source": "local",
    "modelName": "nomic-embed-text:latest",
    "baseUrl": "",
    "apiKey": "",
    "dimension": 768
  },
  "rerank": {
    "enabled": false
  },
  "multimodal": {
    "enabled": false
  },
  "documentSplitting": {
    "chunkSize": 512,
    "chunkOverlap": 100,
    "separators": ["\n\n", "\n", ".", "!", "?", ";"]
  },
  "nodeExtract": {
    "enabled": false
  },
  "agent": {
    "enabled": true,
    "maxIterations": 8,
    "temperature": 0.8,
    "allowedTools": ["knowledge_search", "multi_kb_search", "list_knowledge_bases"]
  }
}'

RESPONSE=$(curl -s -X POST "${API_BASE_URL}/api/v1/initialization/initialize/${KB_ID}" \
  -H "Content-Type: application/json" \
  -d "$TEST_DATA")

if echo "$RESPONSE" | grep -q '"success":true'; then
  echo -e "${GREEN}✓ Agent configuration saved successfully${NC}"
  echo "$RESPONSE" | jq '.' || echo "$RESPONSE"
else
  echo -e "${RED}✗ Agent configuration save failed${NC}"
  echo "$RESPONSE"
fi
echo ""

sleep 1

echo -e "${YELLOW}Test 3: Verify configuration was saved${NC}"
echo "GET ${API_BASE_URL}/api/v1/initialization/config/${KB_ID}"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/initialization/config/${KB_ID}")
AGENT_CONFIG=$(echo "$RESPONSE" | jq '.data.agent')

echo "Agent configuration:"
echo "$AGENT_CONFIG" | jq '.'

ENABLED=$(echo "$AGENT_CONFIG" | jq -r '.enabled')
MAX_ITER=$(echo "$AGENT_CONFIG" | jq -r '.maxIterations')
TEMP=$(echo "$AGENT_CONFIG" | jq -r '.temperature')

if [ "$ENABLED" == "true" ] && [ "$MAX_ITER" == "8" ] && [ "$TEMP" == "0.8" ]; then
  echo -e "${GREEN}✓ Configuration verification successful - all values correct${NC}"
else
  echo -e "${RED}✗ Configuration verification failed${NC}"
  echo "  enabled: $ENABLED (expected: true)"
  echo "  maxIterations: $MAX_ITER (expected: 8)"
  echo "  temperature: $TEMP (expected: 0.8)"
fi
echo ""

echo -e "${YELLOW}Test 4: Get configuration via Tenant API${NC}"
echo "GET ${API_BASE_URL}/api/v1/tenants/${TENANT_ID}/agent-config"
RESPONSE=$(curl -s -X GET "${API_BASE_URL}/api/v1/tenants/${TENANT_ID}/agent-config")
echo "Response:"
echo "$RESPONSE" | jq '.' || echo "$RESPONSE"
echo ""

echo -e "${YELLOW}Test 5: Database verification${NC}"
echo "Tip: Run the following SQL queries to verify data manually:"
echo ""
echo "MySQL:"
echo "  mysql -u root -p weknora -e \"SELECT id, agent_config FROM tenants WHERE id = ${TENANT_ID};\""
echo ""
echo "PostgreSQL:"
echo "  psql -U postgres -d weknora -c \"SELECT id, agent_config FROM tenants WHERE id = ${TENANT_ID};\""
echo ""

echo "========================================="
echo "Tests completed!"
echo "========================================="
echo ""
echo "If all tests passed, the Agent configuration feature is working correctly."
echo "If any test failed, please check:"
echo "  1. Whether the backend service is running"
echo "  2. Whether database migrations have been applied"
echo "  3. Whether the knowledge base ID is correct"
echo ""

