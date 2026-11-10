#!/usr/bin/env bash
# Leva ao servidor os cookies da conta secundária do Instagram (D40), logada num navegador deste Mac.
# Só os cookies do instagram.com saem daqui. Passo a passo em docs/INSTAGRAM.md.
# Uso: scripts/cookies-instagram.sh [navegador]   (padrão: firefox; ex.: chrome:"Profile 1", safari)
set -euo pipefail

cd "$(dirname "$0")/.."

NAVEGADOR="${1:-firefox}"
DEPLOY_HOST="${DEPLOY_HOST:-lucas@192.168.1.20}"
DEPLOY_KEY="${DEPLOY_KEY:-$HOME/.ssh/id_ed25519_homelab}"
DESTINO="${DESTINO:-/srv/data/acervo/instagram-cookies.txt}"
REEL_DE_TESTE="https://www.instagram.com/reel/CDUMkliABpa/"

falhar() { echo "ERRO: $*" >&2; exit 1; }

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "==> Lendo os cookies do $NAVEGADOR"
yt-dlp -q --skip-download --cookies-from-browser "$NAVEGADOR" --cookies "$tmp/todos.txt" "$REEL_DE_TESTE" \
  || falhar "o yt-dlp não leu os cookies do $NAVEGADOR (feche o navegador e tente de novo)."
{ echo "# Netscape HTTP Cookie File"; grep -E '^(#HttpOnly_)?\.?(www\.)?instagram\.com\b' "$tmp/todos.txt" || true; } > "$tmp/instagram.txt"
grep -q $'\tsessionid\t' "$tmp/instagram.txt" \
  || falhar "não há login do Instagram no $NAVEGADOR. Entre com a conta secundária em instagram.com nele e rode de novo."

echo "==> Conferindo o login com um Reel"
yt-dlp -q --skip-download --cookies "$tmp/instagram.txt" "$REEL_DE_TESTE" \
  || falhar "o Instagram recusou estes cookies. Saia e entre de novo na conta secundária e rode de novo."

echo "==> Mandando ao servidor ($DESTINO)"
ssh -i "$DEPLOY_KEY" -o IdentitiesOnly=yes -o BatchMode=yes "$DEPLOY_HOST" \
  "umask 077 && cat > '$DESTINO.novo' && mv '$DESTINO.novo' '$DESTINO'" < "$tmp/instagram.txt"
echo "Pronto: o Acervo passa a usar a conta secundária quando o Instagram pedir login."
