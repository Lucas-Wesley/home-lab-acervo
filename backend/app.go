package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	dados      string // DATA_DIR
	regras     string // pasta com as regras que o claude lê
	ytdlp      string
	ytdlpExtra []string // argumentos extras, ex.: --js-runtimes node no container
	claude     string
	modelo     string // vazio = o padrão do Claude Code
	autorNome  string
	autorEmail string
}

func configDoAmbiente(dados string) config {
	return config{
		dados:      absoluto(dados),
		regras:     absoluto(env("REGRAS_DIR", "../regras")),
		ytdlp:      env("YTDLP_BIN", "yt-dlp"),
		ytdlpExtra: strings.Fields(os.Getenv("YTDLP_ARGS")),
		claude:     env("CLAUDE_BIN", "claude"),
		modelo:     os.Getenv("CLAUDE_MODEL"),
		autorNome:  env("GIT_AUTOR_NOME", "Acervo"),
		autorEmail: env("GIT_AUTOR_EMAIL", "acervo@homelab.local"),
	}
}

// app junta o que os handlers e a fila usam.
type app struct {
	db       *sql.DB
	cfg      config
	fila     *fila
	conteudo *conteudo
}

func novoApp(ctx context.Context, db *sql.DB, cfg config) (*app, error) {
	for _, pasta := range []string{"referencias", "revisoes"} {
		if err := os.MkdirAll(filepath.Join(cfg.dados, pasta), 0o755); err != nil {
			return nil, err
		}
	}
	c, err := abrirConteudo(ctx, filepath.Join(cfg.dados, "conteudo"), filepath.Join(cfg.dados, "revisoes"), cfg)
	if err != nil {
		return nil, err
	}
	a := &app{db: db, cfg: cfg, conteudo: c}
	a.fila = novaFila(db, a.executores())
	if err := a.fila.recuperar(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// absoluto: o git e o claude rodam em outras pastas, e um caminho relativo apontaria errado.
func absoluto(caminho string) string {
	if abs, err := filepath.Abs(caminho); err == nil {
		return abs
	}
	return caminho
}

func (a *app) pastaReferencia(id string) string {
	return filepath.Join(a.cfg.dados, "referencias", id)
}

func agora() string {
	return time.Now().UTC().Format(time.RFC3339)
}
