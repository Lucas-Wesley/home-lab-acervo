package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Flashcards de um texto (D28): o Lucas aperta "Gerar flashcards" num Tema, num capítulo ou numa
// Referência, e o Acervo manda o texto para o app Flashcards (POST /api/textos), pela rede interna do
// servidor. A IA roda na fila de lá; o Acervo não decide nada sobre os cartões. A proposta fica para
// revisar no Flashcards, e a de uma Referência também aparece na página dela, para aprovar ali junto
// com a pesquisa. Cada envio fica anotado aqui, para mostrar "enviado em" e achar a proposta.

// O Flashcards recusa acima disso; aqui a mensagem já sugere mandar por capítulo.
const maxTextoFlashcards = 150_000

// Seções que só listam fontes: não viram cartão.
var semCartoes = map[string]bool{"Citados": true, "Referências": true}

// Sem tempo fixo no cliente: aprovar baixa imagens e leva mais; cada chamada traz o seu prazo.
var clienteFlashcards = &http.Client{}

type envioFlashcards struct {
	ID        int64  `json:"id"`
	Alvo      string `json:"alvo"`
	Titulo    string `json:"titulo"`
	GeracaoID int64  `json:"geracao_id"` // o pedido no Flashcards; 0 = não chegou lá
	Erro      string `json:"erro"`
	CriadoEm  string `json:"criado_em"`
}

// textoParaCartoes é o que vai para o Flashcards.
type textoParaCartoes struct {
	Titulo   string `json:"titulo"`
	Baralho  string `json:"baralho"`
	Contexto string `json:"contexto"`
	Texto    string `json:"texto"`
	Link     string `json:"link"`
}

func alvoDeArquivo(arquivo, capitulo string) string {
	if capitulo == "" {
		return "arquivo:" + arquivo
	}
	return "arquivo:" + arquivo + "#" + capitulo
}

func alvoDeReferencia(id string) string { return "referencia:" + id }

// ---------- O texto ----------

// textoDoArquivo: a página inteira (com os capítulos separados dela, que entram no lugar do link)
// ou um capítulo só.
func (a *app) textoDoArquivo(arquivo, capitulo string) (textoParaCartoes, error) {
	var t textoParaCartoes
	f, err := a.conteudo.ler(arquivo)
	if err != nil {
		return t, err
	}
	_, corpo := lerFrente(f.Conteudo)
	titulo := tituloDe(f.Conteudo, path.Base(arquivo))
	nomes := []string{}
	for _, m := range f.Migalhas {
		nomes = append(nomes, m.Titulo)
	}
	if !strings.HasSuffix(arquivo, "/indice.md") || len(nomes) == 0 {
		nomes = append(nomes, titulo)
	}
	if capitulo != "" {
		achou := false
		for _, s := range secoes(corpo) {
			if s.Titulo == capitulo {
				corpo, achou = corpo[s.ini:s.fim], true
				break
			}
		}
		if !achou {
			return t, errNaoEncontrado
		}
		nomes = append(nomes, capitulo)
	}
	t.Texto = a.juntarCapitulos(path.Dir(arquivo), semSecoes(corpo, semCartoes))
	t.Titulo = strings.Join(nomes, " › ")
	t.Baralho = nomes[0]
	t.Link = a.cfg.acervoLink + "/#/ler/" + arquivo

	var contexto strings.Builder
	fmt.Fprintf(&contexto, "Onde fica no Acervo: %s.", t.Titulo)
	if len(f.Migalhas) > 0 && f.Migalhas[0].Resumo != "" {
		fmt.Fprintf(&contexto, " Resumo do Tema %q: %s", f.Migalhas[0].Titulo, f.Migalhas[0].Resumo)
	}
	switch {
	case capitulo != "":
		fmt.Fprintf(&contexto, " O texto é só o capítulo %q.", capitulo)
	case !strings.HasSuffix(arquivo, "/indice.md"):
		fmt.Fprintf(&contexto, " O texto é o capítulo %q, que tem página própria.", titulo)
	default:
		contexto.WriteString(" O texto é o Tema inteiro.")
	}
	t.Contexto = strings.TrimSpace(contexto.String())
	return t, nil
}

// semSecoes tira os capítulos com esses títulos.
func semSecoes(texto string, titulos map[string]bool) string {
	lista := secoes(texto)
	for i := len(lista) - 1; i >= 0; i-- {
		if titulos[lista[i].Titulo] {
			texto = texto[:lista[i].ini] + texto[lista[i].fim:]
		}
	}
	return texto
}

// juntarCapitulos põe o texto de cada capítulo separado (a linha que é só o link) no lugar do link.
func (a *app) juntarCapitulos(pasta, texto string) string {
	linhas := strings.Split(texto, "\n")
	for i, l := range linhas {
		m := linkDeCapitulo.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		rel, err := caminhoDeArquivo(path.Join(pasta, m[1]))
		if err != nil {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(a.conteudo.raiz, filepath.FromSlash(rel))); err == nil {
			_, corpo := lerFrente(string(b))
			// O título da página do capítulo já está no "## " de cima.
			if primeira, resto, ok := strings.Cut(corpo, "\n"); ok && strings.HasPrefix(primeira, "# ") {
				corpo = resto
			}
			linhas[i] = strings.TrimSpace(corpo)
		}
	}
	return strings.Join(linhas, "\n")
}

// textoDaReferencia: o texto organizado (nunca o cru), quando ela já pode virar estudo.
func (a *app) textoDaReferencia(ctx context.Context, ref *referencia) (textoParaCartoes, error) {
	var t textoParaCartoes
	if !podeVirarCartoes(ref) {
		return t, errors.New("Só uma Referência validada (ou uma pesquisa pronta) vira flashcards: o texto ainda não foi conferido.")
	}
	b, err := os.ReadFile(filepath.Join(a.pastaReferencia(ref.ID), "organizado.md"))
	if err != nil {
		return t, errors.New("Esta Referência ainda não tem o texto organizado.")
	}
	t.Texto = string(b)
	t.Titulo = ref.Titulo
	t.Baralho = ref.Titulo
	t.Link = a.cfg.acervoLink + "/#/referencia/" + ref.ID
	if ref.Tipo == "pesquisa" {
		t.Contexto = "É uma pesquisa feita a pedido do Lucas, com as fontes marcadas no texto."
	} else {
		t.Contexto = fmt.Sprintf("É o texto organizado de um vídeo (%s).", strings.TrimSpace(ref.Canal+" · "+ref.URL))
	}
	// Já integrada: os cartões vão para o baralho do Tema principal onde ela entrou.
	var tema string
	a.db.QueryRowContext(ctx, `SELECT tema FROM referencia_temas WHERE referencia_id = ? ORDER BY integrado_em LIMIT 1`, ref.ID).Scan(&tema)
	// O Tema principal: temas/<slug>, mesmo que ela tenha entrado num subtema.
	if partes := strings.Split(tema, "/"); len(partes) >= 2 {
		raiz := partes[0] + "/" + partes[1]
		if indice, err := os.ReadFile(filepath.Join(a.conteudo.raiz, filepath.FromSlash(raiz), "indice.md")); err == nil {
			t.Baralho = tituloDe(string(indice), t.Baralho)
			t.Contexto += fmt.Sprintf(" Já entrou no Tema %q.", t.Baralho)
		}
	}
	return t, nil
}

func podeVirarCartoes(ref *referencia) bool {
	switch ref.Estado {
	case "validado", "integrado":
		return true
	case "organizado":
		return ref.Tipo == "pesquisa"
	}
	return false
}

// ---------- O envio ----------

// enviarFlashcards manda o texto e anota o envio, mesmo quando o Flashcards não respondeu (aí com o erro).
func (a *app) enviarFlashcards(ctx context.Context, alvo string, t textoParaCartoes) (envioFlashcards, error) {
	e := envioFlashcards{Alvo: alvo, Titulo: t.Titulo, CriadoEm: agora()}
	var erroDoEnvio error
	if n := utf8.RuneCountInString(t.Texto); n > maxTextoFlashcards {
		return e, fmt.Errorf("O texto tem %d mil caracteres; o máximo é %d mil. Gere por capítulo.", n/1000, maxTextoFlashcards/1000)
	}
	var g struct {
		ID int64 `json:"id"`
	}
	if status, err := a.chamarFlashcards(ctx, "POST", "/api/textos", t, &g); err != nil {
		erroDoEnvio = err
		if status == http.StatusBadRequest || status == http.StatusRequestEntityTooLarge {
			return e, err // o texto é que não serve: nada a tentar de novo
		}
		e.Erro = err.Error()
	}
	e.GeracaoID = g.ID
	res, err := a.db.ExecContext(ctx, `INSERT INTO flashcards (alvo, titulo, geracao_id, erro, criado_em) VALUES (?, ?, ?, ?, ?)`,
		e.Alvo, e.Titulo, e.GeracaoID, e.Erro, e.CriadoEm)
	if err != nil {
		return e, err
	}
	e.ID, _ = res.LastInsertId()
	return e, erroDoEnvio
}

// chamarFlashcards fala com a API do Flashcards e devolve a mensagem de erro dela, se houver.
func (a *app) chamarFlashcards(ctx context.Context, metodo, caminho string, corpo, destino any) (int, error) {
	if a.cfg.flashcards == "" {
		return 0, errors.New("O Flashcards não está configurado neste servidor (FLASHCARDS_URL).")
	}
	if _, tem := ctx.Deadline(); !tem {
		var cancelar context.CancelFunc
		ctx, cancelar = context.WithTimeout(ctx, 20*time.Second)
		defer cancelar()
	}
	var leitor io.Reader
	if corpo != nil {
		b, err := json.Marshal(corpo)
		if err != nil {
			return 0, err
		}
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, a.cfg.flashcards+caminho, leitor)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := clienteFlashcards.Do(req)
	if err != nil {
		log.Printf("flashcards: %v", err)
		return 0, errors.New("O Flashcards não respondeu. Ele está no ar?")
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode >= 300 {
		var e struct {
			Erro string `json:"erro"`
		}
		if json.Unmarshal(b, &e) != nil || e.Erro == "" {
			e.Erro = fmt.Sprintf("O Flashcards respondeu %d.", res.StatusCode)
		}
		return res.StatusCode, errors.New(e.Erro)
	}
	if destino != nil {
		return res.StatusCode, json.Unmarshal(b, destino)
	}
	return res.StatusCode, nil
}

func (a *app) envios(ctx context.Context, filtro string, args ...any) ([]envioFlashcards, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT id, alvo, titulo, geracao_id, erro, criado_em FROM flashcards `+filtro, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []envioFlashcards{}
	for rows.Next() {
		var e envioFlashcards
		if err := rows.Scan(&e.ID, &e.Alvo, &e.Titulo, &e.GeracaoID, &e.Erro, &e.CriadoEm); err != nil {
			return nil, err
		}
		lista = append(lista, e)
	}
	return lista, rows.Err()
}

// cartoesDaPesquisa: a pesquisa ficou pronta e o Lucas pediu flashcards junto. Uma falha aqui não
// para a pesquisa: fica anotada, e a página da Referência oferece tentar de novo.
func (a *app) cartoesDaPesquisa(ctx context.Context, id string) {
	ref, err := a.referencia(ctx, id)
	if err != nil {
		return
	}
	if p, err := a.pedidoDe(ref); err != nil || !p.Flashcards {
		return
	}
	if ja, _ := a.envios(ctx, `WHERE alvo = ? AND geracao_id > 0`, alvoDeReferencia(id)); len(ja) > 0 {
		return // a pesquisa foi refeita: os cartões já foram pedidos
	}
	t, err := a.textoDaReferencia(ctx, ref)
	if err == nil {
		_, err = a.enviarFlashcards(ctx, alvoDeReferencia(id), t)
	}
	if err != nil {
		log.Printf("flashcards da pesquisa %s: %v", id, err)
	}
}

// descartarCartoes: a Referência foi descartada; a proposta de cartões dela, ainda em revisão, vai junto.
func (a *app) descartarCartoes(ctx context.Context, id string) {
	lista, _ := a.envios(ctx, `WHERE alvo = ? AND geracao_id > 0`, alvoDeReferencia(id))
	for _, e := range lista {
		a.chamarFlashcards(ctx, "POST", fmt.Sprintf("/api/geracoes/%d/descartar", e.GeracaoID), nil, nil)
	}
}

// ---------- Handlers ----------

// handleEnviarFlashcards: POST /api/flashcards {arquivo, capitulo} ou {referencia}.
func (a *app) handleEnviarFlashcards(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Arquivo    string `json:"arquivo"`
		Capitulo   string `json:"capitulo"`
		Referencia string `json:"referencia"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	ctx := r.Context()
	var t textoParaCartoes
	var alvo string
	var err error
	if corpo.Referencia != "" {
		ref, e := a.referencia(ctx, corpo.Referencia)
		if e != nil {
			erroJSON(w, http.StatusNotFound, "Referência não encontrada.")
			return
		}
		alvo = alvoDeReferencia(ref.ID)
		if t, err = a.textoDaReferencia(ctx, ref); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		arquivo, e := caminhoDeArquivo(corpo.Arquivo)
		if e != nil {
			erroJSON(w, http.StatusBadRequest, "Arquivo inválido.")
			return
		}
		capitulo := strings.TrimSpace(corpo.Capitulo)
		alvo = alvoDeArquivo(arquivo, capitulo)
		t, err = a.textoDoArquivo(arquivo, capitulo)
		if errors.Is(err, errNaoEncontrado) {
			erroJSON(w, http.StatusNotFound, "Texto não encontrado.")
			return
		} else if err != nil {
			falha(w, err)
			return
		}
	}
	e, err := a.enviarFlashcards(ctx, alvo, t)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// handleListarFlashcards: GET /api/flashcards?alvo= — os envios daquele texto, o mais novo primeiro,
// e o endereço do Flashcards para os links.
func (a *app) handleListarFlashcards(w http.ResponseWriter, r *http.Request) {
	filtro, args := `ORDER BY id DESC LIMIT 50`, []any{}
	if alvo := r.URL.Query().Get("alvo"); alvo != "" {
		filtro, args = `WHERE alvo = ? ORDER BY id DESC LIMIT 20`, []any{alvo}
	}
	lista, err := a.envios(r.Context(), filtro, args...)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disponivel": a.cfg.flashcards != "", "link": a.cfg.flashcardsLink, "envios": lista})
}

// O Acervo só repassa ao Flashcards o que é de um envio feito daqui.
func (a *app) geracaoDoPedido(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, _ := strconv.ParseInt(r.PathValue("geracao"), 10, 64)
	var n int
	a.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM flashcards WHERE geracao_id = ?`, id).Scan(&n)
	if id <= 0 || n == 0 {
		erroJSON(w, http.StatusNotFound, "Proposta de flashcards não encontrada.")
		return 0, false
	}
	return id, true
}

// handleProposta: GET /api/flashcards/{geracao} — a proposta, como o Flashcards a devolve.
func (a *app) handleProposta(w http.ResponseWriter, r *http.Request) {
	id, ok := a.geracaoDoPedido(w, r)
	if !ok {
		return
	}
	var g json.RawMessage
	if status, err := a.chamarFlashcards(r.Context(), "GET", fmt.Sprintf("/api/geracoes/%d", id), nil, &g); err != nil {
		erroJSON(w, statusOu(status, http.StatusBadGateway), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// handleDecidirProposta: POST /api/flashcards/{geracao}/{acao} — aprovar, descartar ou refazer, lá.
func (a *app) handleDecidirProposta(w http.ResponseWriter, r *http.Request) {
	id, ok := a.geracaoDoPedido(w, r)
	if !ok {
		return
	}
	acao := r.PathValue("acao")
	if acao != "aprovar" && acao != "descartar" && acao != "refazer" {
		erroJSON(w, http.StatusNotFound, "Ação desconhecida.")
		return
	}
	var corpo any
	if acao == "aprovar" && !lerCorpo(w, r, &corpo) {
		return
	}
	// Aprovar baixa as imagens no Flashcards: pode levar mais que uma chamada comum, e não deve
	// parar no meio se a tela fechar.
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Minute)
	defer cancelar()
	var resposta json.RawMessage
	if status, err := a.chamarFlashcards(ctx, "POST", fmt.Sprintf("/api/geracoes/%d/%s", id, acao), corpo, &resposta); err != nil {
		erroJSON(w, statusOu(status, http.StatusBadGateway), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resposta)
}

// statusOu devolve o erro do Flashcards quando ele respondeu, ou o alternativo.
func statusOu(status, alternativo int) int {
	if status >= 400 {
		return status
	}
	return alternativo
}
