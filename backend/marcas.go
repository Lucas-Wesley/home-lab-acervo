package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Livros da Biblioteca marcados por assunto (D38). Com a fila parada, o Acervo pega um livro ainda
// não marcado (os novos entram sozinhos; os antigos, aos poucos) e um modelo menor lê os capítulos
// em lotes, seguindo regras/marcar-livro.md: o assunto de cada capítulo, com as páginas, e as marcas
// (Temas do Acervo, pessoas, organizações, lugares, períodos, eventos). Cada tarefa marca um lote e
// põe o próximo no fim da fila: um livro grande nunca segura a fila. As marcas mostram, no Tema,
// "Livros da sua biblioteca que falam disto", e entram como pista no Perguntar.

const (
	caracteresPorLote = 60_000
	paginasPorBloco   = 20
	caracteresPorPag  = 2_000 // estimativa, para juntar blocos de páginas em lotes
	maxCaracteresLote = 90_000
	modeloDasMarcas   = "haiku"
	esperaDasMarcas   = 10 * time.Minute
)

type marcaDeCapitulo struct {
	N            int      `json:"n"`
	Titulo       string   `json:"titulo"`
	De           int      `json:"de"`
	Ate          int      `json:"ate"`
	Resumo       string   `json:"resumo"`
	Temas        []string `json:"temas"`
	Pessoas      []string `json:"pessoas"`
	Organizacoes []string `json:"organizacoes"`
	Lugares      []string `json:"lugares"`
	Periodos     []string `json:"periodos"`
	Eventos      []string `json:"eventos"`
	Trechos      []struct {
		Pagina  int    `json:"pagina"`
		Assunto string `json:"assunto"`
	} `json:"trechos"`
}

const esquemaMarcas = `{"type":"object","properties":{"capitulos":{"type":"array","items":{"type":"object","properties":{` +
	`"n":{"type":"integer"},"resumo":{"type":"string"},` +
	`"temas":{"type":"array","items":{"type":"string"}},"pessoas":{"type":"array","items":{"type":"string"}},` +
	`"organizacoes":{"type":"array","items":{"type":"string"}},"lugares":{"type":"array","items":{"type":"string"}},` +
	`"periodos":{"type":"array","items":{"type":"string"}},"eventos":{"type":"array","items":{"type":"string"}},` +
	`"trechos":{"type":"array","items":{"type":"object","properties":{"pagina":{"type":"integer"},"assunto":{"type":"string"}},"required":["pagina","assunto"]}}},` +
	`"required":["n","resumo","temas","pessoas","organizacoes","lugares","periodos","eventos","trechos"]}}},"required":["capitulos"]}`

// lotes junta capítulos seguidos até ~60 mil caracteres; capítulos quase vazios (o título de uma
// "Parte") vão junto com os outros.
func lotes(caps []marcaDeCapitulo, chars map[int]int) [][]int {
	var saida [][]int
	var atual []int
	soma := 0
	for _, c := range caps {
		n := chars[c.N]
		if len(atual) > 0 && soma+n > caracteresPorLote {
			saida = append(saida, atual)
			atual, soma = nil, 0
		}
		atual = append(atual, c.N)
		soma += n
	}
	if len(atual) > 0 {
		saida = append(saida, atual)
	}
	return saida
}

// blocosDePaginas: um livro sem sumário (um PDF sem índice) é lido em blocos de 20 páginas, cada
// um marcado como se fosse um capítulo ("Páginas 21 a 40").
func blocosDePaginas(paginas int) ([]marcaDeCapitulo, map[int]int) {
	var caps []marcaDeCapitulo
	chars := map[int]int{}
	for de := 1; de <= paginas; de += paginasPorBloco {
		ate := min(de+paginasPorBloco-1, paginas)
		n := len(caps) + 1
		caps = append(caps, marcaDeCapitulo{N: n, Titulo: fmt.Sprintf("Páginas %d a %d", de, ate), De: de, Ate: ate})
		chars[n] = (ate - de + 1) * caracteresPorPag
	}
	return caps, chars
}

// marcarAosPoucos: de tempos em tempos, com a fila parada, põe na fila o próximo livro.
func (a *app) marcarAosPoucos(ctx context.Context) {
	if a.cfg.biblioteca == "" {
		return
	}
	espera := time.Minute
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(espera):
		}
		espera = esperaDasMarcas
		if err := a.proximoLivro(ctx); err != nil {
			log.Printf("marcar livros: %v", err)
		}
	}
}

func (a *app) proximoLivro(ctx context.Context) error {
	var ocupada int
	a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tarefas WHERE estado IN (?, ?)`, naFila, rodando).Scan(&ocupada)
	if ocupada > 0 {
		return nil
	}
	var livros []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Author string `json:"author"`
	}
	if err := buscarJSON(ctx, a.cfg.biblioteca+"/api/books", &livros); err != nil {
		return err
	}
	for _, l := range livros {
		var estado, quando string
		err := a.db.QueryRowContext(ctx, `SELECT estado, atualizado_em FROM livros_marcados WHERE livro = ?`, l.ID).Scan(&estado, &quando)
		// Um que falhou (ainda não indexado, a Biblioteca fora) tenta de novo no dia seguinte.
		if err == nil && !(estado == "falhou" && antesDe(quando, 24*time.Hour)) {
			continue
		}
		if _, err := a.db.ExecContext(ctx, `INSERT INTO livros_marcados (livro, titulo, autor, estado, atualizado_em) VALUES (?, ?, ?, 'marcando', ?)
			ON CONFLICT (livro) DO UPDATE SET estado = 'marcando', erro = '', capitulos = '[]', pendentes = '[]', atualizado_em = excluded.atualizado_em`,
			l.ID, l.Title, l.Author, agora()); err != nil {
			return err
		}
		_, err = a.fila.enfileirar(ctx, "", "marcar_livro", map[string]string{"livro": l.ID})
		return err
	}
	return nil
}

func antesDe(iso string, d time.Duration) bool {
	t, err := time.Parse(time.RFC3339, iso)
	return err != nil || time.Since(t) > d
}

func (a *app) falhaNaMarca(livro string, err error) error {
	a.db.Exec(`UPDATE livros_marcados SET estado = 'falhou', erro = ?, atualizado_em = ? WHERE livro = ?`, err.Error(), agora(), livro)
	return err
}

// marcarLivro marca o próximo lote de capítulos do livro.
func (a *app) marcarLivro(ctx context.Context, t *tarefa) (resultado, error) {
	var p struct {
		Livro string `json:"livro"`
	}
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	if a.cfg.biblioteca == "" {
		return pronto, errors.New("a Biblioteca não está configurada (BIBLIOTECA_URL)")
	}
	var titulo, autor, feitosJSON, pendentesJSON string
	if err := a.db.QueryRowContext(ctx, `SELECT titulo, autor, capitulos, pendentes FROM livros_marcados WHERE livro = ?`, p.Livro).
		Scan(&titulo, &autor, &feitosJSON, &pendentesJSON); err != nil {
		return pronto, err
	}
	var feitos []marcaDeCapitulo
	var pendentes [][]int
	json.Unmarshal([]byte(feitosJSON), &feitos)
	json.Unmarshal([]byte(pendentesJSON), &pendentes)

	var s sumarioDoLivro
	if err := buscarJSON(ctx, a.cfg.biblioteca+"/api/books/"+url.PathEscape(p.Livro)+"/chapters", &s); err != nil {
		return pronto, a.falhaNaMarca(p.Livro, fmt.Errorf("a Biblioteca não devolveu o sumário: %w", err))
	}
	porPaginas := len(s.Chapters) == 0
	switch {
	case !s.Indexed:
		msg := "o livro ainda não foi indexado na Biblioteca"
		if s.Error != "" {
			msg += ": " + s.Error
		}
		return pronto, a.falhaNaMarca(p.Livro, errors.New(msg))
	case porPaginas && s.Pages == 0:
		return pronto, a.falhaNaMarca(p.Livro, errors.New("o livro não tem capítulos nem páginas com texto"))
	}
	info := map[int]marcaDeCapitulo{}
	chars := map[int]int{}
	var caps []marcaDeCapitulo
	if porPaginas {
		caps, chars = blocosDePaginas(s.Pages)
	}
	for _, c := range s.Chapters {
		caps = append(caps, marcaDeCapitulo{N: c.N, Titulo: c.Title, De: c.FirstPage, Ate: c.LastPage})
		chars[c.N] = c.Chars
	}
	for _, c := range caps {
		info[c.N] = c
	}
	// Primeira tarefa do livro: divide em lotes.
	if len(feitos) == 0 && len(pendentes) == 0 {
		pendentes = lotes(caps, chars)
	}
	if len(pendentes) == 0 {
		a.db.ExecContext(ctx, `UPDATE livros_marcados SET estado = 'pronto', atualizado_em = ? WHERE livro = ?`, agora(), p.Livro)
		return pronto, nil
	}
	lote := pendentes[0]

	var texto textoDoLivro
	ns := make([]string, len(lote))
	for i, n := range lote {
		ns[i] = fmt.Sprint(n)
	}
	if porPaginas {
		for _, n := range lote {
			var bloco textoDoLivro
			if err := buscarJSON(ctx, fmt.Sprintf("%s/api/books/%s/text?from=%d&to=%d", a.cfg.biblioteca, url.PathEscape(p.Livro), info[n].De, info[n].Ate), &bloco); err != nil {
				return pronto, a.falhaNaMarca(p.Livro, fmt.Errorf("a Biblioteca não devolveu o texto: %w", err))
			}
			texto.Parts = append(texto.Parts, bloco.Parts...)
		}
	} else if err := buscarJSON(ctx, fmt.Sprintf("%s/api/books/%s/text?chapters=%s", a.cfg.biblioteca, url.PathEscape(p.Livro), strings.Join(ns, ",")), &texto); err != nil {
		return pronto, a.falhaNaMarca(p.Livro, fmt.Errorf("a Biblioteca não devolveu o texto: %w", err))
	}
	var md strings.Builder
	fmt.Fprintf(&md, "# %s", titulo)
	if autor != "" {
		fmt.Fprintf(&md, " — %s", autor)
	}
	md.WriteString("\n\n")
	for i, parte := range texto.Parts {
		n := 0
		if i < len(lote) {
			n = lote[i]
		}
		fmt.Fprintf(&md, "## Capítulo %d: %s (p. %d–%d)\n\n", n, parte.Title, parte.FirstPage, parte.LastPage)
		pagina := 0
		for _, x := range parte.Passages {
			if x.Page != pagina {
				fmt.Fprintf(&md, "[p. %d] ", x.Page)
				pagina = x.Page
			}
			md.WriteString(strings.TrimSpace(x.Text) + "\n\n")
		}
	}
	tmp, err := os.MkdirTemp("", "marcar-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "capitulos.md"), []byte(cortar(md.String(), maxCaracteresLote, false)), 0o644); err != nil {
		return pronto, err
	}
	temas, _ := a.conteudo.listarTemas()
	var lista strings.Builder
	for _, tm := range temas {
		fmt.Fprintf(&lista, "- %s — %s\n", tm.Caminho, tm.Titulo)
	}
	if len(temas) == 0 {
		lista.WriteString("(o Acervo ainda não tem Temas)\n")
	}
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta: tmp,
		regra: "marcar-livro.md",
		instrucao: fmt.Sprintf("Marque os capítulos %s de capitulos.md, nesta pasta (livro %q). Os Temas que já existem no Acervo (pasta — título):\n\n%s",
			strings.Join(ns, ", "), titulo, lista.String()),
		ferramentas: ferramentasLeitura,
		esquema:     esquemaMarcas,
		modelo:      modeloDasMarcas,
	})
	if err != nil {
		return pronto, err // a fila mostra o erro; o lote fica pendente para a próxima vez
	}
	var saida struct {
		Capitulos []marcaDeCapitulo `json:"capitulos"`
	}
	if err := dadosDaResposta(r, &saida); err != nil {
		return pronto, err
	}
	for _, m := range saida.Capitulos {
		base, ok := info[m.N]
		if !ok {
			continue
		}
		m.Titulo, m.De, m.Ate = base.Titulo, base.De, base.Ate
		feitos = append(feitos, m)
	}
	sort.Slice(feitos, func(i, j int) bool { return feitos[i].N < feitos[j].N })
	pendentes = pendentes[1:]
	estado := "marcando"
	if len(pendentes) == 0 {
		estado = "pronto"
	}
	fj, _ := json.Marshal(feitos)
	pj, _ := json.Marshal(pendentes)
	if _, err := a.db.ExecContext(ctx, `UPDATE livros_marcados SET capitulos = ?, pendentes = ?, estado = ?, erro = '', atualizado_em = ? WHERE livro = ?`,
		string(fj), string(pj), estado, agora(), p.Livro); err != nil {
		return pronto, err
	}
	if len(pendentes) > 0 {
		_, err = a.fila.enfileirar(ctx, "", "marcar_livro", p)
	}
	return pronto, err
}

// capituloSobre é um capítulo de livro que trata de um Tema.
type capituloSobre struct {
	Livro    string   `json:"livro"`
	Titulo   string   `json:"titulo"`
	Autor    string   `json:"autor"`
	Capitulo string   `json:"capitulo"`
	De       int      `json:"de"`
	Ate      int      `json:"ate"`
	Resumo   string   `json:"resumo"`
	Marcas   []string `json:"marcas"`
	Link     string   `json:"link"`
	pontos   int
}

// livrosSobre: os capítulos marcados com o Tema (pela pasta) ou com o título dele.
func (a *app) livrosSobre(ctx context.Context, tema, tituloDoTema string, max int) ([]capituloSobre, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT livro, titulo, autor, capitulos FROM livros_marcados WHERE capitulos != '[]'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	alvoTitulo := slug(tituloDoTema)
	var achados []capituloSobre
	for rows.Next() {
		var livro, titulo, autor, capsJSON string
		if err := rows.Scan(&livro, &titulo, &autor, &capsJSON); err != nil {
			return nil, err
		}
		var caps []marcaDeCapitulo
		json.Unmarshal([]byte(capsJSON), &caps)
		for _, c := range caps {
			pontos := 0
			for _, t := range c.Temas {
				t = strings.TrimSpace(t)
				switch {
				case t == tema || strings.HasPrefix(t, tema+"/"):
					pontos += 3
				case alvoTitulo != "" && slug(t) == alvoTitulo:
					pontos += 2
				}
			}
			if pontos == 0 {
				continue
			}
			marcas := append(append(append([]string{}, c.Pessoas...), c.Lugares...), c.Periodos...)
			if len(marcas) > 8 {
				marcas = marcas[:8]
			}
			achados = append(achados, capituloSobre{Livro: livro, Titulo: titulo, Autor: autor, Capitulo: c.Titulo, De: c.De, Ate: c.Ate,
				Resumo: c.Resumo, Marcas: marcas, Link: a.linkDaBiblioteca(livro, ""), pontos: pontos})
		}
	}
	sort.SliceStable(achados, func(i, j int) bool { return achados[i].pontos > achados[j].pontos })
	if len(achados) > max {
		achados = achados[:max]
	}
	return achados, rows.Err()
}

// handleLivrosDoTema: GET /api/temas-livros?tema=temas/<pasta>.
func (a *app) handleLivrosDoTema(w http.ResponseWriter, r *http.Request) {
	tema := strings.TrimSuffix(r.URL.Query().Get("tema"), "/")
	if !localDePergunta.MatchString(tema) || !strings.HasPrefix(tema, "temas/") {
		erroJSON(w, http.StatusBadRequest, "Tema inválido.")
		return
	}
	titulo := ""
	if b, err := os.ReadFile(filepath.Join(a.conteudo.raiz, filepath.FromSlash(tema), "indice.md")); err == nil {
		titulo = tituloDe(string(b), "")
	}
	lista, err := a.livrosSobre(r.Context(), tema, titulo, 12)
	if err != nil {
		falha(w, err)
		return
	}
	if lista == nil {
		lista = []capituloSobre{}
	}
	writeJSON(w, http.StatusOK, lista)
}

// handleLivrosMarcados: GET /api/livros-marcados, o andamento (para Ajustes).
func (a *app) handleLivrosMarcados(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `SELECT livro, titulo, autor, estado, erro, capitulos, pendentes, atualizado_em FROM livros_marcados ORDER BY titulo`)
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	type item struct {
		Livro      string `json:"livro"`
		Titulo     string `json:"titulo"`
		Autor      string `json:"autor"`
		Estado     string `json:"estado"`
		Erro       string `json:"erro,omitempty"`
		Marcados   int    `json:"marcados"`
		Faltam     int    `json:"faltam"`
		Atualizado string `json:"atualizado_em"`
	}
	lista := []item{}
	for rows.Next() {
		var it item
		var caps, pend string
		if err := rows.Scan(&it.Livro, &it.Titulo, &it.Autor, &it.Estado, &it.Erro, &caps, &pend, &it.Atualizado); err != nil {
			falha(w, err)
			return
		}
		var c []json.RawMessage
		var p [][]int
		json.Unmarshal([]byte(caps), &c)
		json.Unmarshal([]byte(pend), &p)
		it.Marcados, it.Faltam = len(c), len(p)
		lista = append(lista, it)
	}
	writeJSON(w, http.StatusOK, lista)
}
