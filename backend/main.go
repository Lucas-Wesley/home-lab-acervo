package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

var version = "dev"

//go:embed all:web/dist
var webDist embed.FS

func main() {
	dataDir := env("DATA_DIR", "./data")
	addr := env("ADDR", ":8080")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := abrirBanco(filepath.Join(dataDir, "acervo.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := novoApp(ctx, db, configDoAmbiente(dataDir))
	if err != nil {
		log.Fatal(err)
	}
	go app.fila.rodar(ctx)

	web, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           newServer(db, web, app, version),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("acervo %s em %s", version, addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Print(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func abrirBanco(caminho string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", caminho+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

const esquema = `
CREATE TABLE IF NOT EXISTS referencias (
	id            TEXT PRIMARY KEY,
	url           TEXT NOT NULL,
	video_id      TEXT NOT NULL UNIQUE,
	titulo        TEXT NOT NULL DEFAULT '',
	canal         TEXT NOT NULL DEFAULT '',
	duracao       TEXT NOT NULL DEFAULT '',
	idioma        TEXT NOT NULL DEFAULT '',
	estado        TEXT NOT NULL,
	erro          TEXT NOT NULL DEFAULT '',
	sugestao      TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tarefas (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	referencia_id TEXT NOT NULL,
	tipo          TEXT NOT NULL,
	params        TEXT NOT NULL DEFAULT '{}',
	estado        TEXT NOT NULL,
	erro          TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	iniciado_em   TEXT NOT NULL DEFAULT '',
	terminado_em  TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS referencia_temas (
	referencia_id TEXT NOT NULL,
	tema          TEXT NOT NULL,
	integrado_em  TEXT NOT NULL,
	PRIMARY KEY (referencia_id, tema)
);
CREATE TABLE IF NOT EXISTS sugestoes (
	id            TEXT PRIMARY KEY,
	texto         TEXT NOT NULL,
	tela          TEXT NOT NULL DEFAULT '',
	estado        TEXT NOT NULL DEFAULT 'nova',
	resposta      TEXT NOT NULL DEFAULT '',
	criada_em     TEXT NOT NULL,
	atualizada_em TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS detalhes (
	id            TEXT PRIMARY KEY,
	local         TEXT NOT NULL,
	titulo        TEXT NOT NULL DEFAULT '',
	trecho        TEXT NOT NULL,
	antes         TEXT NOT NULL DEFAULT '',
	depois        TEXT NOT NULL DEFAULT '',
	explicacao    TEXT NOT NULL DEFAULT '',
	fontes        TEXT NOT NULL DEFAULT '[]',
	tarefa_id     INTEGER NOT NULL DEFAULT 0,
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS detalhes_local ON detalhes (local);
`
