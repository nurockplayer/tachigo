#!/usr/bin/env bash
# validate-keys.sh — 驗證 .env 中的 3 個 key 是否有效

set -euo pipefail

ENV_FILE="$(dirname "$0")/.env"
PASS="✅"
FAIL="❌"
WARN="⚠️"

if [[ ! -f "$ENV_FILE" ]]; then
  echo "$FAIL .env 檔案不存在：$ENV_FILE"
  exit 1
fi

# 載入 .env（忽略空行與註解）
while IFS='=' read -r key value; do
  [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
  export "$key=$value"
done < "$ENV_FILE"

ERRORS=0

echo ""
echo "=== Key 驗證報告 ==="
echo ""

# ──────────────────────────────────────────
# 1. DEPLOYER_PRIVATE_KEY
# ──────────────────────────────────────────
echo "1. DEPLOYER_PRIVATE_KEY"

if [[ -z "${DEPLOYER_PRIVATE_KEY:-}" ]]; then
  echo "   $FAIL 未設定"
  ERRORS=$((ERRORS+1))
elif [[ "${DEPLOYER_PRIVATE_KEY}" =~ ^0x[0-9a-fA-F]{64}$ ]]; then
  # 遮蔽中間部分只顯示頭尾
  PK="${DEPLOYER_PRIVATE_KEY}"
  echo "   $PASS 格式正確（0x + 64 hex）：${PK:0:8}...${PK: -4}"
else
  echo "   $FAIL 格式錯誤（應為 0x 開頭 + 64 個 hex 字元）"
  ERRORS=$((ERRORS+1))
fi

echo ""

# ──────────────────────────────────────────
# 2. SEPOLIA_RPC_URL
# ──────────────────────────────────────────
echo "2. SEPOLIA_RPC_URL"

if [[ -z "${SEPOLIA_RPC_URL:-}" ]]; then
  echo "   $FAIL 未設定"
  ERRORS=$((ERRORS+1))
else
  echo "   URL：${SEPOLIA_RPC_URL}"

  RESPONSE=$(curl -s -m 10 -X POST "${SEPOLIA_RPC_URL}" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' 2>&1) || true

  if echo "$RESPONSE" | grep -q '"result"'; then
    CHAIN_ID_HEX=$(echo "$RESPONSE" | grep -o '"result":"0x[^"]*"' | grep -o '0x[^"]*')
    CHAIN_ID_DEC=$((16#${CHAIN_ID_HEX#0x}))
    if [[ "$CHAIN_ID_DEC" -eq 11155111 ]]; then
      echo "   $PASS RPC 連線成功（chainId = $CHAIN_ID_DEC，確認為 Sepolia）"
    else
      echo "   $WARN RPC 可連線，但 chainId = $CHAIN_ID_DEC（Sepolia 應為 11155111）"
      ERRORS=$((ERRORS+1))
    fi
  else
    echo "   $FAIL RPC 連線失敗或回應異常"
    echo "   回應：${RESPONSE:0:200}"
    ERRORS=$((ERRORS+1))
  fi
fi

echo ""

# ──────────────────────────────────────────
# 3. ETHERSCAN_API_KEY
# ──────────────────────────────────────────
echo "3. ETHERSCAN_API_KEY"

if [[ -z "${ETHERSCAN_API_KEY:-}" ]]; then
  echo "   $FAIL 未設定"
  ERRORS=$((ERRORS+1))
else
  # 用 V2 API 測試 key 是否有效（查 Sepolia 最新 block number，chainid=11155111）
  API_RESPONSE=$(curl -s -m 10 \
    "https://api.etherscan.io/v2/api?chainid=11155111&module=proxy&action=eth_blockNumber&apikey=${ETHERSCAN_API_KEY}" 2>&1) || true

  if echo "$API_RESPONSE" | grep -q '"result":"0x'; then
    BLOCK_HEX=$(echo "$API_RESPONSE" | grep -o '"result":"0x[^"]*"' | grep -o '0x[^"]*')
    BLOCK_DEC=$((16#${BLOCK_HEX#0x}))
    echo "   $PASS API key 有效（目前 Sepolia block = $BLOCK_DEC）"
  elif echo "$API_RESPONSE" | grep -q 'Invalid API Key'; then
    echo "   $FAIL API key 無效"
    ERRORS=$((ERRORS+1))
  elif echo "$API_RESPONSE" | grep -q 'Max rate limit'; then
    echo "   $WARN API key 格式看起來正確，但已達速率限制（稍後再試）"
  else
    echo "   $FAIL 無法驗證（回應異常）"
    echo "   回應：${API_RESPONSE:0:200}"
    ERRORS=$((ERRORS+1))
  fi
fi

echo ""
echo "=============================="
if [[ "$ERRORS" -eq 0 ]]; then
  echo "$PASS 所有 key 驗證通過"
else
  echo "$FAIL 共 $ERRORS 個 key 驗證失敗"
  exit 1
fi
