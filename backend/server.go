package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
)

func newServer(db *sql.DB, web fs.FS, a *app, version string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth(db, version))

	mux.HandleFunc("GET /api/referencias", a.handleListarReferencias)
	mux.HandleFunc("POST /api/referencias", a.handleCriarReferencia)
	mux.HandleFunc("GET /api/referencias/{id}", a.handleReferencia)
	mux.HandleFunc("POST /api/referencias/{id}/{acao}", a.handleAcao)

	mux.HandleFunc("GET /api/temas", a.handleTemas)
	mux.HandleFunc("POST /api/temas", a.handleCriarTema)
	mux.HandleFunc("GET /api/arquivos/{caminho...}", a.handleLerArquivo)
	mux.HandleFunc("PUT /api/arquivos/{caminho...}", a.handleSalvarArquivo)
	mux.HandleFunc("GET /api/pacotes/{caminho...}", a.handlePacote)
	mux.HandleFunc("POST /api/capitulos/{acao}", a.handleCapitulo)

	mux.HandleFunc("GET /api/tarefas", a.handleTarefas)
	mux.HandleFunc("POST /api/tarefas/{id}/cancelar", a.handleCancelar)
	mux.HandleFunc("GET /api/revisoes", a.handleRevisoes)
	mux.HandleFunc("GET /api/revisoes/{id}", a.handleRevisao)
	mux.HandleFunc("POST /api/revisoes/{id}/{decisao}", a.handleDecidir)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		erroJSON(w, http.StatusNotFound, "não encontrado")
	})
	mux.Handle("/", spaHandler(web))
	return mux
}

func handleHealth(db *sql.DB, version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "erro", "version": version})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": version})
	}
}

// ---------- Temas e arquivos ----------

func (a *app) handleTemas(w http.ResponseWriter, r *http.Request) {
	temas, err := a.conteudo.listarTemas()
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, temas)
}

func (a *app) handleCriarTema(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Titulo string `json:"titulo"`
		Pai    string `json:"pai"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	pasta, msg := a.pastaNova(corpo.Pai, strings.TrimSpace(corpo.Titulo))
	if msg != "" {
		erroJSON(w, http.StatusBadRequest, msg)
		return
	}
	if err := a.conteudo.criarTemaVazio(r.Context(), pasta, strings.TrimSpace(corpo.Titulo)); err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"caminho": pasta})
}

func (a *app) handleLerArquivo(w http.ResponseWriter, r *http.Request) {
	rel, err := caminhoDeArquivo(r.PathValue("caminho"))
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Caminho inválido.")
		return
	}
	arq, err := a.conteudo.ler(rel)
	if errors.Is(err, errNaoEncontrado) {
		erroJSON(w, http.StatusNotFound, "Arquivo não encontrado.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, arq)
}

func (a *app) handleSalvarArquivo(w http.ResponseWriter, r *http.Request) {
	rel, err := caminhoDeArquivo(r.PathValue("caminho"))
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Caminho inválido.")
		return
	}
	var corpo struct {
		Conteudo string `json:"conteudo"`
		Base     string `json:"base"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	hash, err := a.conteudo.salvar(r.Context(), rel, corpo.Conteudo, corpo.Base)
	var c *conflito
	switch {
	case errors.As(err, &c):
		writeJSON(w, http.StatusConflict, map[string]any{"erro": "O arquivo mudou no servidor.", "atual": c.Atual})
	case errors.Is(err, errCaminhoInvalido):
		erroJSON(w, http.StatusBadRequest, "Arquivo novo só dentro de um Tema.")
	case err != nil:
		falha(w, err)
	default:
		writeJSON(w, http.StatusOK, map[string]string{"hash": hash})
	}
}

func (a *app) handlePacote(w http.ResponseWriter, r *http.Request) {
	pasta, err := caminhoDeTema(r.PathValue("caminho"))
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Caminho inválido.")
		return
	}
	arquivos, err := a.conteudo.pacote(pasta)
	if errors.Is(err, errNaoEncontrado) {
		erroJSON(w, http.StatusNotFound, "Tema não encontrado.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tema": pasta, "arquivos": arquivos})
}

// handleCapitulo separa um capítulo em página própria ou o traz de volta (D23).
func (a *app) handleCapitulo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Arquivo string `json:"arquivo"`
		Titulo  string `json:"titulo"`
		Base    string `json:"base"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	rel, err := caminhoDeArquivo(corpo.Arquivo)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Caminho inválido.")
		return
	}
	var arquivo string
	switch r.PathValue("acao") {
	case "separar":
		arquivo, err = a.conteudo.separarCapitulo(r.Context(), rel, corpo.Titulo, corpo.Base)
	case "juntar":
		err = a.conteudo.juntarCapitulo(r.Context(), rel, corpo.Titulo, corpo.Base)
	default:
		erroJSON(w, http.StatusNotFound, "Ação desconhecida.")
		return
	}
	var c *conflito
	switch {
	case errors.As(err, &c):
		writeJSON(w, http.StatusConflict, map[string]any{"erro": "O arquivo mudou no servidor; recarregue.", "atual": c.Atual})
	case errors.Is(err, errNaoEncontrado):
		erroJSON(w, http.StatusNotFound, "Arquivo não encontrado.")
	case err != nil && (errors.Is(err, errSemCapitulo) || errors.Is(err, errJaSeparado) || errors.Is(err, errNaoSeparado) || errors.Is(err, errSoNoIndice) || strings.Contains(err.Error(), "citado em")):
		erroJSON(w, http.StatusConflict, err.Error())
	case err != nil:
		falha(w, err)
	default:
		writeJSON(w, http.StatusOK, map[string]string{"arquivo": arquivo})
	}
}

// ---------- Fila e revisões ----------

func (a *app) handleTarefas(w http.ResponseWriter, r *http.Request) {
	lista, err := a.fila.listar(r.Context(),
		`WHERE estado IN (?, ?) OR id IN (SELECT id FROM tarefas ORDER BY id DESC LIMIT 50) ORDER BY id DESC`, naFila, rodando)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.comTitulos(r, lista))
}

type tarefaComTitulo struct {
	tarefa
	Referencia string `json:"referencia"`
}

func (a *app) comTitulos(r *http.Request, lista []tarefa) []tarefaComTitulo {
	saida := make([]tarefaComTitulo, 0, len(lista))
	titulos := map[string]string{}
	for _, t := range lista {
		titulo, ok := titulos[t.ReferenciaID]
		if !ok {
			if ref, err := a.referencia(r.Context(), t.ReferenciaID); err == nil {
				titulo = ref.Titulo
				if titulo == "" {
					titulo = ref.URL
				}
			}
			titulos[t.ReferenciaID] = titulo
		}
		saida = append(saida, tarefaComTitulo{t, titulo})
	}
	return saida
}

func idDoPedido(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "id inválido")
		return 0, false
	}
	return id, true
}

func (a *app) handleCancelar(w http.ResponseWriter, r *http.Request) {
	id, ok := idDoPedido(w, r)
	if !ok {
		return
	}
	if err := a.fila.cancelarTarefa(r.Context(), id); errors.Is(err, errNaoCancelavel) {
		erroJSON(w, http.StatusConflict, err.Error())
		return
	} else if err != nil {
		falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) handleRevisoes(w http.ResponseWriter, r *http.Request) {
	lista, err := a.fila.listar(r.Context(), `WHERE estado = ? ORDER BY id`, emRevisao)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.comTitulos(r, lista))
}

func (a *app) tarefaEmRevisao(w http.ResponseWriter, r *http.Request) (*tarefa, bool) {
	id, ok := idDoPedido(w, r)
	if !ok {
		return nil, false
	}
	t, err := a.fila.tarefa(r.Context(), id)
	if err != nil || t.Estado != emRevisao {
		erroJSON(w, http.StatusNotFound, "Revisão não encontrada.")
		return nil, false
	}
	return t, true
}

func (a *app) handleRevisao(w http.ResponseWriter, r *http.Request) {
	t, ok := a.tarefaEmRevisao(w, r)
	if !ok {
		return
	}
	diff, err := a.conteudo.diff(r.Context(), t.ID)
	if err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(r.Context(), t.ReferenciaID)
	writeJSON(w, http.StatusOK, map[string]any{"tarefa": t, "referencia": ref, "diff": diff})
}

func (a *app) handleDecidir(w http.ResponseWriter, r *http.Request) {
	t, ok := a.tarefaEmRevisao(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	ref, err := a.referencia(ctx, t.ReferenciaID)
	if err != nil {
		falha(w, err)
		return
	}
	var alvo struct {
		Pasta  string `json:"pasta"`
		Titulo string `json:"titulo"`
		Tema   string `json:"tema"`
	}
	json.Unmarshal(t.Params, &alvo)
	tema := alvo.Pasta + alvo.Tema

	switch r.PathValue("decisao") {
	case "rejeitar":
		a.conteudo.descartarRevisao(ctx, t.ID)
		a.fila.terminar(t, rejeitada, "")
	case "aprovar":
		msg := "Integração: " + ref.Titulo + " em " + tema
		if t.Tipo == "criar_tema" {
			msg = "Novo Tema: " + alvo.Titulo + " (a partir de " + ref.Titulo + ")"
		}
		if err := a.conteudo.aprovarRevisao(ctx, t.ID, msg); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
		a.fila.terminar(t, concluida, "")
		a.db.ExecContext(ctx, `INSERT OR REPLACE INTO referencia_temas (referencia_id, tema, integrado_em) VALUES (?, ?, ?)`, ref.ID, tema, agora())
		a.mudarEstado(ctx, ref.ID, "integrado")
	default:
		erroJSON(w, http.StatusNotFound, "Decisão desconhecida.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"tema": tema})
}

// ---------- Utilidades ----------

// spaHandler entrega os arquivos do frontend; caminhos desconhecidos recebem o index.html.
func spaHandler(web fs.FS) http.Handler {
	files := http.FileServerFS(web)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if _, err := fs.Stat(web, name); err != nil {
				http.ServeFileFS(w, r, web, "index.html")
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}

func lerCorpo(w http.ResponseWriter, r *http.Request, destino any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := json.NewDecoder(r.Body).Decode(destino); err != nil {
		erroJSON(w, http.StatusBadRequest, "Pedido inválido.")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func erroJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"erro": msg})
}

func falha(w http.ResponseWriter, err error) {
	log.Print(err)
	erroJSON(w, http.StatusInternalServerError, err.Error())
}
