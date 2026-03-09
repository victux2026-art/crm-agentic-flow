#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$ROOT_DIR/cmd/api"
WORKER_DIR="$ROOT_DIR/cmd/event-processor"

API_URL="${API_URL:-http://127.0.0.1:8080}"
DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@localhost:5440/crm_agentic_flow}"
RECEIVER_PORT="${RECEIVER_PORT:-18081}"
RECEIVER_LOG="${RECEIVER_LOG:-/tmp/crm_webhook_receiver.log}"
API_LOG="${API_LOG:-/tmp/crm_api_smoke.log}"
WORKER_LOG="${WORKER_LOG:-/tmp/crm_worker_smoke.log}"

cleanup() {
  for pid in "${API_PID:-}" "${WORKER_PID:-}" "${RECEIVER_PID:-}"; do
    if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
      kill "${pid}" 2>/dev/null || true
      wait "${pid}" 2>/dev/null || true
    fi
  done
}

trap cleanup EXIT

rm -f "$RECEIVER_LOG" "$API_LOG" "$WORKER_LOG"

cd "$ROOT_DIR"
docker compose up -d db >/dev/null

(
  cd "$API_DIR"
  DATABASE_URL="$DATABASE_URL" go run . >"$API_LOG" 2>&1
) &
API_PID=$!

(
  cd "$WORKER_DIR"
  DATABASE_URL="$DATABASE_URL" go run . >"$WORKER_LOG" 2>&1
) &
WORKER_PID=$!

WEBHOOK_RECEIVER_PORT="$RECEIVER_PORT" WEBHOOK_RECEIVER_LOG="$RECEIVER_LOG" \
  python3 "$ROOT_DIR/scripts/webhook_receiver.py" &
RECEIVER_PID=$!

for _ in {1..30}; do
  if curl -sf "$API_URL/health" >/dev/null; then
    break
  fi
  sleep 1
done

curl -sf "$API_URL/health" >/dev/null

LOGIN_RESPONSE="$(curl -sf "$API_URL/login" \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@crmflow.local","password":"admin123","tenant_slug":"demo"}')"

TOKEN="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])' <<<"$LOGIN_RESPONSE")"

ENDPOINT_RESPONSE="$(curl -sf -X POST "$API_URL/webhook-endpoints" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Smoke Endpoint\",\"target_url\":\"http://127.0.0.1:${RECEIVER_PORT}/webhook\",\"signing_secret\":\"smoke-secret\"}")"

ENDPOINT_ID="$(python3 -c 'import json,sys; print(int(json.load(sys.stdin)["id"]))' <<<"$ENDPOINT_RESPONSE")"

curl -sf -X POST "$API_URL/webhook-subscriptions" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"webhook_endpoint_id\":${ENDPOINT_ID},\"event_type\":\"organization.created\"}" >/dev/null

ORG_RESPONSE="$(curl -sf -X POST "$API_URL/organizations" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Smoke Test Org","domain":"smoke.test","industry":"QA"}')"

ORG_ID="$(python3 -c 'import json,sys; print(int(json.load(sys.stdin)["id"]))' <<<"$ORG_RESPONSE")"

for _ in {1..30}; do
  if [[ -f "$RECEIVER_LOG" ]] && [[ "$(wc -l <"$RECEIVER_LOG")" -ge 1 ]]; then
    break
  fi
  sleep 1
done

if [[ ! -f "$RECEIVER_LOG" ]]; then
  echo "receiver log not created"
  exit 1
fi

DELIVERIES_RESPONSE="$(curl -sf "$API_URL/webhook-deliveries?event_type=organization.created&webhook_endpoint_id=${ENDPOINT_ID}" \
  -H "Authorization: Bearer $TOKEN")"

OUTBOX_RESPONSE="$(curl -sf "$API_URL/outbox-events?event_type=organization.created&limit=10" \
  -H "Authorization: Bearer $TOKEN")"

python3 - "$RECEIVER_LOG" "$DELIVERIES_RESPONSE" "$OUTBOX_RESPONSE" "$ORG_ID" <<'PY'
import json
import sys

log_path, deliveries_raw, outbox_raw, org_id_raw = sys.argv[1:5]
org_id = int(org_id_raw)

with open(log_path, encoding="utf-8") as fh:
    rows = [json.loads(line) for line in fh if line.strip()]

if not rows:
    raise SystemExit("receiver log is empty")

payload = rows[-1]["body"]
if payload["event_type"] != "organization.created":
    raise SystemExit(f"unexpected event_type: {payload['event_type']}")
if int(payload["payload"]["id"]) != org_id:
    raise SystemExit(f"unexpected organization id in webhook payload: {payload['payload']['id']}")

deliveries = json.loads(deliveries_raw)
if len(deliveries) != 1:
    raise SystemExit(f"expected 1 delivery, got {len(deliveries)}")
if deliveries[0]["status"] != "delivered":
    raise SystemExit(f"unexpected delivery status: {deliveries[0]['status']}")

outbox = json.loads(outbox_raw)
matching = [item for item in outbox if item["event_type"] == "organization.created"]
if not matching:
    raise SystemExit("organization.created outbox event not found")
if matching[0]["status"] != "processed":
    raise SystemExit(f"unexpected outbox status: {matching[0]['status']}")
PY

echo "Smoke test OK"
echo "Endpoint ID: $ENDPOINT_ID"
echo "Organization ID: $ORG_ID"
echo "Receiver log: $RECEIVER_LOG"
