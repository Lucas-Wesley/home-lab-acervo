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
	Tipo         string          `json:"tipo"`     // video ou pesquisa (D27)
	Pesquisa     json.RawMessage `json:"pesquisa"` // o pedido, o esboço e a coleta; null num vídeo
}

// tarefaAtiva é o que a lista mostra: o que está acontecendo com a Referência agora.
type tarefaAtiva struct {
	ID      int64  `json:"id"`
	Tipo    string `json:"tipo"`
	Estado  string `json:"estado"`
	Posicao int    `json:"posicao"` // 0 = rodando; 1 = a próxima...
}

const colunasReferencia = `id, url, COALESCE(video_id, ''), titulo, canal, duracao, idioma, estado, erro, sugestao, criado_em, atualizado_em, tipo, pesquisa`

func lerReferencia(row linha) (*referencia, error) {
	var r referencia
	var sug, pesquisa string
	err := row.Scan(&r.ID, &r.URL, &r.VideoID, &r.Titulo, &r.Canal, &r.Duracao, &r.Idioma, &r.Estado, &r.Erro, &sug, &r.CriadoEm, &r.AtualizadoEm, &r.Tipo, &pesquisa)
	if err != nil {
		return nil, err
	}
	if sug == "" {
		sug = "null"
	}
	if pesquisa == "" {
		pesquisa = "null"
	}
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

func (a *app) handleCriarReferencia(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		URL string `json:"url"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	video, err := videoDoYouTube(corpo.URL)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Cole um link de vídeo do YouTube.")
		return
	}
	var existente string
	if a.db.QueryRowContext(r.Context(), `SELECT id FROM referencias WHERE video_id = ?`, video).Scan(&existente) == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"erro": "Este vídeo já está nas Referências.", "id": existente})
		return
	}
	id := novoID()
	link := "https://www.youtube.com/watch?v=" + video
	_, err = a.db.ExecContext(r.Context(),
		`INSERT INTO referencias (id, url, video_id, estado, criado_em, atualizado_em) VALUES (?, ?, ?, 'novo', ?, ?)`,
		id, link, video, agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if _, err := a.fila.enfileirar(r.Context(), id, "extrair", nil); err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(r.Context(), id)
	a.completar(r.Context(), []*referencia{ref})
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
	for _, nome := range []string{"cru", "organizado", "traducao"} {
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
	if acao != "descartar" && a.ocupada(ctx, ref.ID) {
		erroJSON(w, http.StatusConflict, "Já há uma tarefa em andamento para esta Referência.")
		return
	}
	pronta := ref.Estado == "organizado" || ref.Estado == "validado" || ref.Estado == "integrado"

	var err error
	switch acao {
	case "validar":
		if ref.Estado != "organizado" {
			erroJSON(w, http.StatusConflict, "Só uma Referência organizada pode ser validada.")
			return
		}
		err = a.mudarEstado(ctx, ref.ID, "validado")
	case "descartar":
		err = a.mudarEstado(ctx, ref.ID, "descartado")
		if err == nil {
			a.descartarCartoes(ctx, ref.ID)
		}
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
	if acao != "validar" && acao != "descartar" {
		a.db.ExecContext(ctx, `UPDATE referencias SET erro = '' WHERE id = ?`, ref.ID)
	}
	ref, _ = a.referencia(ctx, ref.ID)
	a.completar(ctx, []*referencia{ref})
	writeJSON(w, http.StatusOK, ref)
}

// proximaEtapa decide o que reprocessar a partir do que já existe em disco.
func (a *app) proximaEtapa(ref *referencia) string {
	if ref.Tipo == "pesquisa" {
		return a.proximaEtapaPesquisa(ref)
	}
	pasta := a.pastaReferencia(ref.ID)
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
