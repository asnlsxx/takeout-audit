#!/usr/bin/env bash
set -euo pipefail
if [[ "${GITHUB_ACTIONS:-}" != true ]]; then
  echo 'This bootstrap is only for a disposable GitHub Actions runner.' >&2
  exit 2
fi
umask 077
password="$(openssl rand -hex 20)"
echo "::add-mask::$password"
cat > test/immich/.env <<EOF
DB_PASSWORD=$password
DB_USERNAME=postgres
DB_DATABASE_NAME=immich
DB_HOSTNAME=database
REDIS_HOSTNAME=redis
IMMICH_TELEMETRY_INCLUDE=none
EOF
docker compose --env-file test/immich/.env -f test/immich/compose.yml up -d
ready=false
for attempt in $(seq 1 120); do
  if curl --fail --silent http://127.0.0.1:2283/api/server/ping >/dev/null; then ready=true; break; fi
  sleep 2
done
if [[ "$ready" != true ]]; then echo 'Immich did not become ready' >&2; exit 1; fi
credentials="$(jq -nc --arg password "$password" '{email:"audit-ci@example.invalid",password:$password,name:"Disposable audit CI"}')"
curl --fail --silent --show-error -H 'Content-Type: application/json' --data "$credentials" http://127.0.0.1:2283/api/auth/admin-sign-up >/dev/null
login="$(jq -nc --arg password "$password" '{email:"audit-ci@example.invalid",password:$password}')"
token="$(curl --fail --silent --show-error -H 'Content-Type: application/json' --data "$login" http://127.0.0.1:2283/api/auth/login | jq -er '.accessToken')"
echo "::add-mask::$token"
admin_key="$(curl --fail --silent --show-error -H "Authorization: Bearer $token" -H 'Content-Type: application/json' --data '{"name":"Disposable fixture setup","permissions":["all"]}' http://127.0.0.1:2283/api/api-keys | jq -er '.secret')"
query_key="$(curl --fail --silent --show-error -H "Authorization: Bearer $token" -H 'Content-Type: application/json' --data '{"name":"Audit queries only","permissions":["user.read","asset.upload"]}' http://127.0.0.1:2283/api/api-keys | jq -er '.secret')"
echo "::add-mask::$admin_key"
echo "::add-mask::$query_key"
printf 'server: http://127.0.0.1:2283\napi_key: %s\n' "$admin_key" > config.staging.yaml
printf 'TAKEOUT_AUDIT_QUERY_KEY=%s\nTAKEOUT_AUDIT_DISPOSABLE=1\n' "$query_key" >> "$GITHUB_ENV"
