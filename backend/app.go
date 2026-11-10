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
	// Pesquisador (D27): de onde vêm os livros e a Wikipédia.
	biblioteca     string // API da Biblioteca, ex.: http://biblioteca:8080 no servidor
	bibliotecaLink string // endereço que o Lucas abre, para os links de volta ao livro
	wikipedia      string
	// Flashcards de um texto (D28): a API do app Flashcards, na rede interna, e os endereços que o
	// Lucas abre (o do Flashcards, para revisar; o do Acervo, para o link de volta).
	flashcards     string
	flashcardsLink string
	acervoLink     string
	// Podcasts (D39): de onde vêm o episódio e o feed, e quem transcreve o áudio (o Whisper do Mac
	// mini; com o Mac fora, o do servidor).
	spotify        string
	itunes         string
	whisper        string
	whisperReserva string
	ffmpeg         string
	// Reels (D40): os cookies da conta secundária do Instagram, quando ele pede login.
	instagramCookies string
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

		biblioteca:     strings.TrimSuffix(os.Getenv("BIBLIOTECA_URL"), "/"),
		bibliotecaLink: strings.TrimSuffix(env("BIBLIOTECA_LINK", "https://192.168.1.20:8443"), "/"),
		wikipedia:      strings.TrimSuffix(env("WIKIPEDIA_URL", "https://pt.wikipedia.org"), "/"),

		flashcards:     strings.TrimSuffix(os.Getenv("FLASHCARDS_URL"), "/"),
		flashcardsLink: strings.TrimSuffix(env("FLASHCARDS_LINK", "https://192.168.1.20:8445"), "/"),
		acervoLink:     strings.TrimSuffix(env("ACERVO_LINK", "https://192.168.1.20:8444"), "/"),

		spotify:        strings.TrimSuffix(env("SPOTIFY_URL", "https://open.spotify.com"), "/"),
		itunes:         strings.TrimSuffix(env("ITUNES_URL", "https://itunes.apple.com"), "/"),
		whisper:        strings.TrimSuffix(os.Getenv("WHISPER_URL"), "/"),
		whisperReserva: strings.TrimSuffix(os.Getenv("WHISPER_RESERVA_URL"), "/"),
		ffmpeg:         env("FFMPEG_BIN", "ffmpeg"),

		instagramCookies: env("INSTAGRAM_COOKIES", filepath.Join(absoluto(dados), "instagram-cookies.txt")),
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
