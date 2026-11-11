FROM node:24-alpine AS web
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /app .

# O Node fica na imagem final: o Claude Code roda nele, e o yt-dlp o usa como motor de JavaScript.
FROM node:24-alpine
# O ffmpeg corta o áudio dos podcasts em pedaços para o Whisper (D39); o poppler lê os PDFs da web (D49).
RUN apk add --no-cache git python3 py3-pip ripgrep libgcc libstdc++ ffmpeg poppler-utils \
 && npm install -g @anthropic-ai/claude-code \
 && npm cache clean --force
# O YouTube quebra o yt-dlp com frequência: o deploy.sh passa a data aqui para sempre pegar a versão nova.
ARG YTDLP_EM=nunca
RUN pip install --no-cache-dir --break-system-packages -U "yt-dlp[default]" && echo "yt-dlp de $YTDLP_EM"
COPY --from=api /app /app
COPY regras /regras
USER 1000:1000
ENV DATA_DIR=/data \
    REGRAS_DIR=/regras \
    HOME=/data/home \
    YTDLP_ARGS="--js-runtimes node" \
    USE_BUILTIN_RIPGREP=0 \
    DISABLE_AUTOUPDATER=1
ENTRYPOINT ["/app"]
