#!/usr/bin/env bash
# Roda na VM, chamado pelo deploy.sh. Se o app não ficar saudável, volta ao commit anterior.
set -euo pipefail

SHA="$1"
REPO=https://github.com/Lucas-Wesley/home-lab-acervo.git
APP_DIR=/opt/apps/acervo
STATE=/opt/apps/acervo.deployed
COMPOSE=(docker compose -f compose.yaml -f compose.prod.yaml)

[ -d "$APP_DIR/.git" ] || git clone -q "$REPO" "$APP_DIR"
cd "$APP_DIR"
git fetch -q origin
git checkout -q --detach "$SHA"

PREV=$(cat "$STATE" 2>/dev/null || true)

# O token do Claude fica em /opt/apps/acervo.env, fora da repo (D13).
[ -f /opt/apps/acervo.env ] && cp /opt/apps/acervo.env "$APP_DIR/.env"
[ -f "$APP_DIR/.env" ] || echo "AVISO: sem /opt/apps/acervo.env; as etapas com o Claude vão falhar." >&2
mkdir -p /srv/data/acervo

export VERSION="$SHA"
export YTDLP_EM="$(date +%F)"
"${COMPOSE[@]}" build --quiet

if ! "${COMPOSE[@]}" up -d --wait --wait-timeout 60; then
  echo "ERRO: ${SHA:0:7} não ficou saudável." >&2
  "${COMPOSE[@]}" logs --tail 20 >&2 || true
  if [ -n "$PREV" ]; then
    echo "==> Voltando para ${PREV:0:7}" >&2
    git checkout -q --detach "$PREV"
    VERSION="$PREV" "${COMPOSE[@]}" up -d --wait --wait-timeout 60
  fi
  exit 1
fi

echo "$SHA" > "$STATE"

# Mantém só as imagens do commit atual e do anterior
docker images acervo --format '{{.Tag}}' \
  | { grep -vxF -e "$SHA" -e "${PREV:-$SHA}" || true; } \
  | xargs -r -I{} docker rmi -f "acervo:{}" >/dev/null
docker image prune -f >/dev/null
