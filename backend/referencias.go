package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

type referencia struct {
	ID           string          `json:"id"`
	URL          string          `json:"url"`
	VideoID      string          `json:"video_id"`
	Titulo       string          `json:"titulo"`
	Canal        string          `json:"canal"`
	Duracao      string          `json:"duracao"`
	Idioma       string          `json:"idioma"`
	Estado       string          `json:"estado"`
	Erro         string          `json:"erro"`
	Sugestao     json.RawMessage `json:"sugestao"`
	CriadoEm     string          `json:"criado_em"`
	AtualizadoEm string          `json:"atualizado_em"`
	Tarefa       *tarefaAtiva    `json:"tarefa"`
	Temas        []string        `json:"temas"`
	Tipo         string          `json:"tipo"`     // video, artigo (D34), pesquisa (D27) ou livro (D30)
	Pesquisa     json.RawMessage `json:"pesquisa"` // o pedido, o esboço e a coleta; null num vídeo
	Livro        json.RawMessage `json:"livro"`    // o livro e os capítulos escolhidos (D30); null nos outros
}

// tarefaAtiva é o que a lista mostra: o que está acontecendo com a Referência agora.
type tarefaAtiva struct {
	ID      int64  `json:"id"`
	Tipo    string `json:"tipo"`
	Estado  string `json:"estado"`
	Posicao int    `json:"posicao"` // 0 = rodando; 1 = a próxima...
}

const colunasReferencia = `id, url, COALESCE(video_id, ''), titulo, canal, duracao, idioma, estado, erro, sugestao, criado_em, atualizado_em, tipo, pesquisa, livro`

func lerReferencia(row linha) (*referencia, error) {
	var r referencia
	var sug, pesquisa, livro string
	err := row.Scan(&r.ID, &r.URL, &r.VideoID, &r.Titulo, &r.Canal, &r.Duracao, &r.Idioma, &r.Estado, &r.Erro, &sug, &r.CriadoEm, &r.AtualizadoEm, &r.Tipo, &pesquisa, &livro)
	if err != nil {
		return nil, err
	}
	if sug == "" {
		sug = "null"
	}
	if pesquisa == "" {
		pesquisa = "null"
	}
	if livro == "" {
		livro = "null"
	}
	r.Livro = json.RawMessage(livro)
	r.Sugestao = json.RawMessage(sug)
	r.Pesquisa = json.RawMessage(pesquisa)
	r.Temas = []string{}
	return &r, nil
}

func (a *app) referencia(ctx context.Context, id string) (*referencia, error) {
	r, err := lerReferencia(a.db.QueryRowContext(ctx, `SELECT `+colunasReferencia+` FROM referencias WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNaoEncontrado
	}
	return r, err
}

// completar junta a tarefa ativa e os Temas de cada Referência.
func (a *app) completar(ctx context.Context, refs []*referencia) error {
	ativas, err := a.fila.listar(ctx, `WHERE estado IN (?, ?, ?) ORDER BY id`, rodando, naFila, emRevisao)
	if err != nil {
		return err
	}
	porID := map[string]*referencia{}
	for _, r := range refs {
		porID[r.ID] = r
	}
	posicao := 1
	for _, t := range ativas {
		p := 0
		if t.Estado == naFila {
			p = posicao
			posicao++
		}
		if r, ok := porID[t.ReferenciaID]; ok && (r.Tarefa == nil || r.Tarefa.Estado == emRevisao) {
			r.Tarefa = &tarefaAtiva{ID: t.ID, Tipo: t.Tipo, Estado: t.Estado, Posicao: p}
		}
	}
	rows, err := a.db.QueryContext(ctx, `SELECT referencia_id, tema FROM referencia_temas ORDER BY integrado_em`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, tema string
		if err := rows.Scan(&id, &tema); err != nil {
			return err
		}
		if r, ok := porID[id]; ok {
			r.Temas = append(r.Temas, tema)
		}
	}
	return rows.Err()
}

func (a *app) handleListarReferencias(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT `+colunasReferencia+` FROM referencias ORDER BY criado_em DESC, rowid DESC`)
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	refs := []*referencia{}
	for rows.Next() {
		ref, err := lerReferencia(rows)
		if err != nil {
			falha(w, err)
			return
		}
		refs = append(refs, ref)
	}
	if err := a.completar(r.Context(), refs); err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, refs)
}

func novoID() string {
	b := make([]byte, 5)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// handleCriarReferencia: POST /api/referencias {url, tipo}. Um link do YouTube vira vídeo; qualquer
// outro link de página vira artigo (D34). tipo "video" recusa o que não for do YouTube.
func (a *app) handleCriarReferencia(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		URL  string `json:"url"`
		Tipo string `json:"tipo"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	ctx := r.Context()
	var id, link, tipo, video string
	if v, err := videoDoYouTube(corpo.URL); err == nil {
		tipo, video, link = "video", v, "https://www.youtube.com/watch?v="+v
	} else if corpo.Tipo == "video" {
		erroJSON(w, http.StatusBadRequest, "Cole um link de vídeo do YouTube.")
		return
	} else if l, err := linkDaWeb(corpo.URL); err == nil {
		tipo, link = "artigo", l
	} else {
		erroJSON(w, http.StatusBadRequest, "Cole um link de vídeo do YouTube ou de uma página da web.")
		return
	}
	var existente string
	if tipo == "video" {
		a.db.QueryRowContext(ctx, `SELECT id FROM referencias WHERE video_id = ?`, video).Scan(&existente)
	} else {
		a.db.QueryRowContext(ctx, `SELECT id FROM referencias WHERE tipo = 'artigo' AND url = ?`, link).Scan(&existente)
	}
	if existente != "" {
		msg := "Este vídeo já está nas Referências."
		if tipo == "artigo" {
			msg = "Esta página já está nas Referências."
		}
		writeJSON(w, http.StatusConflict, map[string]string{"erro": msg, "id": existente})
		return
	}
	id = novoID()
	var videoID any
	if video != "" {
		videoID = video
	}
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO referencias (id, url, video_id, tipo, estado, criado_em, atualizado_em) VALUES (?, ?, ?, ?, 'novo', ?, ?)`,
		id, link, videoID, tipo, agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if _, err := a.fila.enfileirar(ctx, id, "extrair", nil); err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(ctx, id)
	a.completar(ctx, []*referencia{ref})
	writeJSON(w, http.StatusCreated, ref)
}

type detalheReferencia struct {
	*referencia
	Textos  map[string]string `json:"textos"`
	Tarefas []tarefa          `json:"tarefas"`
}

func (a *app) handleReferencia(w http.ResponseWriter, r *http.Request) {
	ref, ok := a.referenciaDoPedido(w, r)
	if !ok {
		return
	}
	a.completar(r.Context(), []*referencia{ref})
	textos := map[string]string{}
	for _, nome := range []string{"cru", "organizado", "traducao", "anterior"} {
		if b, err := os.ReadFile(filepath.Join(a.pastaReferencia(ref.ID), nome+".md")); err == nil {
			textos[nome] = string(b)
		}
	}
	tarefas, err := a.fila.listar(r.Context(), `WHERE referencia_id = ? ORDER BY id DESC`, ref.ID)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detalheReferencia{ref, textos, tarefas})
}

func (a *app) referenciaDoPedido(w http.ResponseWriter, r *http.Request) (*referencia, bool) {
	ref, err := a.referencia(r.Context(), r.PathValue("id"))
	if errors.Is(err, errNaoEncontrado) {
		erroJSON(w, http.StatusNotFound, "Referência não encontrada.")
		return nil, false
	}
	if err != nil {
		falha(w, err)
		return nil, false
	}
	return ref, true
}

// ocupada diz se a Referência já tem algo na fila ou rodando: uma coisa de cada vez.
func (a *app) ocupada(ctx context.Context, id string) bool {
	var n int
	a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tarefas WHERE referencia_id = ? AND estado IN (?, ?)`, id, naFila, rodando).Scan(&n)
	return n > 0
}

// handleAcao trata os botões da tela da Referência.
func (a *app) handleAcao(w http.ResponseWriter, r *http.Request) {
	ref, ok := a.referenciaDoPedido(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	acao := r.PathValue("acao")
	if a.ocupada(ctx, ref.ID) {
		erroJSON(w, http.StatusConflict, "Já há uma tarefa em andamento para esta Referência.")
		return
	}
	pronta := ref.Estado == "organizado" || ref.Estado == "validado" || ref.Estado == "integrado"

	var err error
	switch acao {
	case "reprocessar":
		if ref.Estado == "esboco" {
			erroJSON(w, http.StatusConflict, "Aprove o esboço para a pesquisa continuar.")
			return
		}
		_, err = a.fila.enfileirar(ctx, ref.ID, a.proximaEtapa(ref), nil)
	case "pesquisar":
		var corpo struct {
			Esboco []pergunta `json:"esboco"`
		}
		if !lerCorpo(w, r, &corpo) {
			return
		}
		var msg string
		msg, err = a.aprovarEsboco(ctx, ref, corpo.Esboco)
		if msg != "" {
			erroJSON(w, http.StatusConflict, msg)
			return
		}
	case "aprofundar":
		var corpo pedidoPesquisa
		if !lerCorpo(w, r, &corpo) {
			return
		}
		var msg string
		msg, err = a.aprofundar(ctx, ref, corpo)
		if msg != "" {
			erroJSON(w, http.StatusConflict, msg)
			return
		}
	case "sugerir":
		if !pronta {
			erroJSON(w, http.StatusConflict, "Organize a Referência antes.")
			return
		}
		_, err = a.fila.enfileirar(ctx, ref.ID, "sugerir", nil)
	case "criar-tema":
		var corpo struct {
			Titulo string `json:"titulo"`
			Pai    string `json:"pai"`
		}
		if !lerCorpo(w, r, &corpo) {
			return
		}
		pasta, msg := a.pastaNova(corpo.Pai, corpo.Titulo)
		if msg != "" {
			erroJSON(w, http.StatusBadRequest, msg)
			return
		}
		if !pronta {
			erroJSON(w, http.StatusConflict, "Organize a Referência antes.")
			return
		}
		_, err = a.fila.enfileirar(ctx, ref.ID, "criar_tema", paramsTema{Pasta: pasta, Titulo: corpo.Titulo})
	case "integrar":
		var corpo paramsIntegrar
		if !lerCorpo(w, r, &corpo) {
			return
		}
		if _, e := caminhoDeTema(corpo.Tema); e != nil || !existe(filepath.Join(a.conteudo.raiz, corpo.Tema, "indice.md")) {
			erroJSON(w, http.StatusBadRequest, "Escolha um Tema existente.")
			return
		}
		if !pronta {
			erroJSON(w, http.StatusConflict, "Organize a Referência antes.")
			return
		}
		_, err = a.fila.enfileirar(ctx, ref.ID, "integrar", corpo)
	default:
		erroJSON(w, http.StatusNotFound, "Ação desconhecida.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	// Pediu de novo: o erro da tentativa anterior não vale mais.
	a.db.ExecContext(ctx, `UPDATE referencias SET erro = '' WHERE id = ?`, ref.ID)
	ref, _ = a.referencia(ctx, ref.ID)
	a.completar(ctx, []*referencia{ref})
	writeJSON(w, http.StatusOK, ref)
}

// handleExcluirReferencia: DELETE /api/referencias/{id} (D33). Apaga a Referência, os textos, o
// histórico e as propostas pendentes dela; o que já entrou nos Temas fica, porque é do Acervo.
func (a *app) handleExcluirReferencia(w http.ResponseWriter, r *http.Request) {
	ref, ok := a.referenciaDoPedido(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	if a.ocupada(ctx, ref.ID) {
		erroJSON(w, http.StatusConflict, "Há uma tarefa em andamento para esta Referência: cancele na Fila antes de excluir.")
		return
	}
	pendentes, err := a.fila.listar(ctx, `WHERE referencia_id = ? AND estado = ?`, ref.ID, emRevisao)
	if err != nil {
		falha(w, err)
		return
	}
	for _, t := range pendentes {
		a.conteudo.descartarRevisao(ctx, t.ID)
	}
	a.descartarCartoes(ctx, ref.ID)
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		falha(w, err)
		return
	}
	defer tx.Rollback()
	for _, q := range []struct {
		sql string
		arg string
	}{
		{`DELETE FROM tarefas WHERE referencia_id = ?`, ref.ID},
		{`DELETE FROM referencia_temas WHERE referencia_id = ?`, ref.ID},
		{`DELETE FROM flashcards WHERE alvo = ?`, alvoDeReferencia(ref.ID)},
		{`DELETE FROM detalhes WHERE local LIKE ?`, "referencias/" + ref.ID + "/%"},
		{`DELETE FROM referencias WHERE id = ?`, ref.ID},
	} {
		if _, err := tx.ExecContext(ctx, q.sql, q.arg); err != nil {
			falha(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		falha(w, err)
		return
	}
	os.RemoveAll(a.pastaReferencia(ref.ID))
	w.WriteHeader(http.StatusNoContent)
}

// proximaEtapa decide o que reprocessar a partir do que já existe em disco.
func (a *app) proximaEtapa(ref *referencia) string {
	if ref.Tipo == "pesquisa" {
		return a.proximaEtapaPesquisa(ref)
	}
	pasta := a.pastaReferencia(ref.ID)
	if ref.Tipo == "livro" {
		if !existe(filepath.Join(pasta, "cru.md")) {
			return "ler_livro"
		}
		return "resumir"
	}
	switch {
	case !existe(filepath.Join(pasta, "cru.md")):
		return "extrair"
	case !existe(filepath.Join(pasta, "organizado.md")) || !precisaTraducao(ref.Idioma):
		return "organizar"
	default:
		return "traduzir"
	}
}

// pastaNova monta temas/[<pai>/]<slug> e confere que dá para criar.
func (a *app) pastaNova(pai, titulo string) (string, string) {
	s := slug(titulo)
	if s == "" {
		return "", "Dê um título ao Tema."
	}
	base := pastaTemas
	if pai != "" {
		if _, err := caminhoDeTema(pai); err != nil || !existe(filepath.Join(a.conteudo.raiz, pai, "indice.md")) {
			return "", "O Tema pai não existe."
		}
		base = pai
	}
	pasta := path.Join(base, s)
	if existe(filepath.Join(a.conteudo.raiz, pasta)) {
		return "", "Já existe um Tema nessa pasta: " + pasta
	}
	return pasta, ""
}

func existe(caminho string) bool {
	_, err := os.Stat(caminho)
	return err == nil
}
