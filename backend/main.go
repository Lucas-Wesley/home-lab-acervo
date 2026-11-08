package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	if err := migrarReferencias(db); err != nil {
		db.Close()
		return nil, err
	}
	if err := acrescentarColuna(db, "referencias", "livro", `TEXT NOT NULL DEFAULT ''`); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// migrarReferencias: a Referência que não é vídeo (a pesquisa, D27) não tem video_id. O SQLite não
// tira o NOT NULL de uma coluna, então a tabela antiga é refeita uma vez, com os mesmos dados.
func migrarReferencias(db *sql.DB) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('referencias') WHERE name = 'tipo'`).Scan(&n); err != nil || n > 0 {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	colunas := `id, url, video_id, titulo, canal, duracao, idioma, estado, erro, sugestao, criado_em, atualizado_em`
	for _, passo := range []string{
		strings.Replace(tabelaReferencias, "referencias (", "referencias_nova (", 1),
		`INSERT INTO referencias_nova (` + colunas + `) SELECT ` + colunas + ` FROM referencias`,
		`DROP TABLE referencias`,
		`ALTER TABLE referencias_nova RENAME TO referencias`,
	} {
		if _, err := tx.Exec(passo); err != nil {
			return fmt.Errorf("migrar referências: %w", err)
		}
	}
	return tx.Commit()
}

// acrescentarColuna cria a coluna num banco de antes dela.
func acrescentarColuna(db *sql.DB, tabela, coluna, definicao string) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, tabela, coluna).Scan(&n); err != nil || n > 0 {
		return err
	}
	_, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, tabela, coluna, definicao))
	return err
}

// tipo: "video", "artigo" (D34), "pesquisa" (D27) ou "livro" (D30); pesquisa guarda o pedido, o esboço e o que foi
// coletado; livro, o livro e os capítulos escolhidos.
const tabelaReferencias = `CREATE TABLE IF NOT EXISTS referencias (
	id            TEXT PRIMARY KEY,
	url           TEXT NOT NULL,
	video_id      TEXT UNIQUE,
	titulo        TEXT NOT NULL DEFAULT '',
	canal         TEXT NOT NULL DEFAULT '',
	duracao       TEXT NOT NULL DEFAULT '',
	idioma        TEXT NOT NULL DEFAULT '',
	estado        TEXT NOT NULL,
	erro          TEXT NOT NULL DEFAULT '',
	sugestao      TEXT NOT NULL DEFAULT '',
	criado_em     TEXT NOT NULL,
	atualizado_em TEXT NOT NULL,
	tipo          TEXT NOT NULL DEFAULT 'video',
	pesquisa      TEXT NOT NULL DEFAULT '',
	livro         TEXT NOT NULL DEFAULT ''
);`

const esquema = tabelaReferencias + `
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
-- Textos mandados ao Flashcards (D28): alvo = "arquivo:<caminho>[#<capítulo>]" ou "referencia:<id>".
CREATE TABLE IF NOT EXISTS flashcards (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	alvo       TEXT NOT NULL,
	titulo     TEXT NOT NULL,
	geracao_id INTEGER NOT NULL DEFAULT 0, -- o pedido no Flashcards; 0 = não chegou lá
	erro       TEXT NOT NULL DEFAULT '',
	criado_em  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS flashcards_alvo ON flashcards (alvo);
`
