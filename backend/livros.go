package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Resumo de capítulos (D30): o Lucas escolhe um livro da Biblioteca e um ou mais capítulos (ou um
// intervalo de páginas, no PDF sem sumário). O servidor lê o texto pela API da Biblioteca (R33 de
// lá), sem a IA, e guarda em cru.md, com a página de cada trecho. A IA resume seguindo
// resumir-livro.md, e o resumo é uma Referência do tipo "livro", que segue o caminho de sempre:
// sugerir o Tema → criar ou integrar → revisão pelo diff.

const tempoDoResumo = 30 * time.Minute

type pedidoLivro struct {
	Livro     string              `json:"livro"` // id na Biblioteca
	Titulo    string              `json:"titulo"`
	Autor     string              `json:"autor"`
	Formato   string              `json:"formato"`
	Capitulos []capituloEscolhido `json:"capitulos"`
	De        int                 `json:"de"` // páginas, quando não há capítulos escolhidos
	Ate       int                 `json:"ate"`
	Foco      string              `json:"foco"` // o que interessa ao Lucas nesses capítulos (opcional)
}

type capituloEscolhido struct {
	N      int    `json:"n"`
	Titulo string `json:"titulo"`
	De     int    `json:"de"`
	Ate    int    `json:"ate"`
}

// O sumário de um livro como a Biblioteca devolve.
type sumarioDoLivro struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Author   string `json:"author"`
	Format   string `json:"format"`
	Indexed  bool   `json:"indexed"`
	Error    string `json:"error"`
	Pages    int    `json:"pages"`
	Chapters []struct {
		N         int    `json:"n"`
		Title     string `json:"title"`
		Level     int    `json:"level"`
		FirstPage int    `json:"first_page"`
		LastPage  int    `json:"last_page"`
		Chars     int    `json:"chars"`
	} `json:"chapters"`
}

func (a *app) semBiblioteca(w http.ResponseWriter) bool {
	if a.cfg.biblioteca == "" {
		erroJSON(w, http.StatusServiceUnavailable, "A Biblioteca não está configurada neste servidor (BIBLIOTECA_URL).")
		return true
	}
	return false
}

// handleLivros: GET /api/livros, os livros da Biblioteca, para escolher.
func (a *app) handleLivros(w http.ResponseWriter, r *http.Request) {
	if a.semBiblioteca(w) {
		return
	}
	var livros []struct {
		ID       string  `json:"id"`
		Title    string  `json:"title"`
		Author   string  `json:"author"`
		Format   string  `json:"format"`
		HasCover bool    `json:"has_cover"`
		Percent  float64 `json:"percent"`
		ReadAt   string  `json:"read_at,omitempty"`
	}
	if err := buscarJSON(r.Context(), a.cfg.biblioteca+"/api/books", &livros); err != nil {
		erroJSON(w, http.StatusBadGateway, "A Biblioteca não respondeu: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, livros)
}

// handleCapaDoLivro: GET /api/livros/{id}/capa, a capa que a Biblioteca guarda, repassada para a
// lista de livros (a Biblioteca fica em outro endereço, com outro certificado).
func (a *app) handleCapaDoLivro(w http.ResponseWriter, r *http.Request) {
	if a.semBiblioteca(w) {
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet,
		a.cfg.biblioteca+"/api/books/"+url.PathEscape(r.PathValue("id"))+"/cover", nil)
	if err != nil {
		falha(w, err)
		return
	}
	resp, err := clienteDaColeta.Do(req)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, "A Biblioteca não respondeu: "+err.Error())
		return
	}
	defer resp.Body.Close()
	tipo := resp.Header.Get("Content-Type")
	if resp.StatusCode != http.StatusOK || !strings.HasPrefix(tipo, "image/") {
		erroJSON(w, http.StatusNotFound, "Capa não encontrada.")
		return
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	io.Copy(w, io.LimitReader(resp.Body, 10<<20))
}

func (a *app) sumario(ctx context.Context, id string) (sumarioDoLivro, error) {
	var s sumarioDoLivro
	err := buscarJSON(ctx, a.cfg.biblioteca+"/api/books/"+url.PathEscape(id)+"/chapters", &s)
	return s, err
}

// handleCapitulos: GET /api/livros/{id}, o sumário com as páginas e o tamanho de cada capítulo.
func (a *app) handleCapitulos(w http.ResponseWriter, r *http.Request) {
	if a.semBiblioteca(w) {
		return
	}
	s, err := a.sumario(r.Context(), r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusBadGateway, "A Biblioteca não respondeu: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// handleCriarResumo: POST /api/livros {livro, capitulos: [n], de, ate, foco}.
func (a *app) handleCriarResumo(w http.ResponseWriter, r *http.Request) {
	if a.semBiblioteca(w) {
		return
	}
	var corpo struct {
		Livro     string `json:"livro"`
		Capitulos []int  `json:"capitulos"`
		De        int    `json:"de"`
		Ate       int    `json:"ate"`
		Foco      string `json:"foco"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	s, err := a.sumario(r.Context(), corpo.Livro)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Livro não encontrado na Biblioteca.")
		return
	}
	if !s.Indexed {
		erroJSON(w, http.StatusConflict, "O texto deste livro ainda não está disponível: "+s.Error)
		return
	}
	p := pedidoLivro{Livro: s.ID, Titulo: s.Title, Autor: s.Author, Formato: s.Format, Capitulos: []capituloEscolhido{},
		Foco: cortar(strings.TrimSpace(corpo.Foco), maxTema, false)}
	slices.Sort(corpo.Capitulos)
	for _, n := range slices.Compact(corpo.Capitulos) {
		if n < 1 || n > len(s.Chapters) {
			erroJSON(w, http.StatusBadRequest, "Capítulo inexistente.")
			return
		}
		c := s.Chapters[n-1]
		p.Capitulos = append(p.Capitulos, capituloEscolhido{N: c.N, Titulo: c.Title, De: c.FirstPage, Ate: c.LastPage})
	}
	if len(p.Capitulos) == 0 {
		if corpo.De < 1 || corpo.Ate < corpo.De || corpo.Ate > max(s.Pages, 1) {
			erroJSON(w, http.StatusBadRequest, "Escolha um ou mais capítulos, ou as páginas.")
			return
		}
		p.De, p.Ate = corpo.De, corpo.Ate
	}

	dados, _ := json.Marshal(p)
	id := novoID()
	_, err = a.db.ExecContext(r.Context(),
		`INSERT INTO referencias (id, url, titulo, canal, idioma, estado, tipo, livro, criado_em, atualizado_em)
		 VALUES (?, ?, ?, ?, 'pt', 'novo', 'livro', ?, ?, ?)`,
		id, a.linkDaBiblioteca(p.Livro, ""), cortar(tituloDoResumo(p), 120, false), s.Author, string(dados), agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if _, err := a.fila.enfileirar(r.Context(), id, "ler_livro", nil); err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(r.Context(), id)
	a.completar(r.Context(), []*referencia{ref})
	writeJSON(w, http.StatusCreated, ref)
}

func tituloDoResumo(p pedidoLivro) string {
	switch len(p.Capitulos) {
	case 0:
		return fmt.Sprintf("%s, p. %d a %d", p.Titulo, p.De, p.Ate)
	case 1:
		return fmt.Sprintf("%s — %s", p.Capitulos[0].Titulo, p.Titulo)
	}
	return fmt.Sprintf("%s: %d capítulos", p.Titulo, len(p.Capitulos))
}

// linkDaBiblioteca abre o livro (no ponto da âncora, se houver) na Biblioteca.
func (a *app) linkDaBiblioteca(livro, ancora string) string {
	link := fmt.Sprintf("%s/#/livro/%s", a.cfg.bibliotecaLink, url.PathEscape(livro))
	if ancora != "" {
		link += "?em=" + url.QueryEscape(ancora)
	}
	return link
}

func (a *app) pedidoLivroDe(ref *referencia) (pedidoLivro, error) {
	var p pedidoLivro
	if ref.Tipo != "livro" {
		return p, errors.New("esta Referência não é o resumo de um livro")
	}
	err := json.Unmarshal(ref.Livro, &p)
	return p, err
}

// ---------- ler_livro (sem IA) ----------

type textoDoLivro struct {
	Parts []struct {
		Title     string `json:"title"`
		FirstPage int    `json:"first_page"`
		LastPage  int    `json:"last_page"`
		Passages  []struct {
			Page   int    `json:"page"`
			Text   string `json:"text"`
			Anchor string `json:"anchor"`
		} `json:"passages"`
	} `json:"parts"`
}

func (a *app) lerLivro(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoLivroDe(ref)
	if err != nil {
		return pronto, err
	}
	if a.cfg.biblioteca == "" {
		return pronto, errors.New("a Biblioteca não está configurada neste servidor (BIBLIOTECA_URL)")
	}
	endereco := fmt.Sprintf("%s/api/books/%s/text?from=%d&to=%d", a.cfg.biblioteca, url.PathEscape(p.Livro), p.De, p.Ate)
	if len(p.Capitulos) > 0 {
		var ns []string
		for _, c := range p.Capitulos {
			ns = append(ns, fmt.Sprint(c.N))
		}
		endereco = fmt.Sprintf("%s/api/books/%s/text?chapters=%s", a.cfg.biblioteca, url.PathEscape(p.Livro), strings.Join(ns, ","))
	}
	var texto textoDoLivro
	if err := buscarJSON(ctx, endereco, &texto); err != nil {
		return pronto, fmt.Errorf("a Biblioteca não devolveu o texto: %w", err)
	}
	cru, caracteres := a.textoDoLivroEmMarkdown(p, texto)
	if caracteres < 200 {
		return pronto, errors.New("os capítulos escolhidos quase não têm texto (PDF escaneado?)")
	}
	pasta := a.pastaReferencia(ref.ID)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return pronto, err
	}
	if err := os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(cru), 0o644); err != nil {
		return pronto, err
	}
	if err := a.mudarEstado(ctx, ref.ID, "cru"); err != nil {
		return pronto, err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "resumir", nil)
	return pronto, err
}

// textoDoLivroEmMarkdown: cada capítulo é uma seção; cada página nova começa com o link dela.
func (a *app) textoDoLivroEmMarkdown(p pedidoLivro, texto textoDoLivro) (string, int) {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n", p.Titulo)
	if p.Autor != "" {
		fmt.Fprintf(&s, "> Autor: %s\n", p.Autor)
	}
	fmt.Fprintf(&s, "> Livro na Biblioteca: %s\n\n", a.linkDaBiblioteca(p.Livro, ""))
	total := 0
	for _, parte := range texto.Parts {
		fmt.Fprintf(&s, "## %s (p. %d–%d)\n\n", parte.Title, parte.FirstPage, parte.LastPage)
		pagina := 0
		for _, t := range parte.Passages {
			if t.Page != pagina {
				fmt.Fprintf(&s, "[p. %d](%s)\n\n", t.Page, a.linkDaBiblioteca(p.Livro, t.Anchor))
				pagina = t.Page
			}
			s.WriteString(strings.TrimSpace(t.Text) + "\n\n")
			total += utf8.RuneCountInString(t.Text)
		}
	}
	return s.String(), total
}

// ---------- resumir ----------

func (a *app) resumir(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoLivroDe(ref)
	if err != nil {
		return pronto, err
	}
	pasta := a.pastaReferencia(ref.ID)
	if err := exigirArquivo(filepath.Join(pasta, "cru.md")); err != nil {
		return pronto, err
	}
	os.Remove(filepath.Join(pasta, "organizado.md"))
	var instrucao strings.Builder
	fmt.Fprintf(&instrucao, "cru.md, nesta pasta, tem o texto de %s do livro %q", descreverEscolha(p), p.Titulo)
	if p.Autor != "" {
		fmt.Fprintf(&instrucao, ", de %s", p.Autor)
	}
	instrucao.WriteString(".\n")
	if p.Foco != "" {
		fmt.Fprintf(&instrucao, "O que interessa ao Lucas nesses capítulos: %q. Dê mais espaço a isso, sem deixar de fora o essencial do resto.\n", p.Foco)
	}
	instrucao.WriteString("Leia o texto inteiro e escreva organizado.md nesta pasta. Responda só 'ok' ao terminar.")

	ctx, cancelar := context.WithTimeout(ctx, tempoDoResumo)
	defer cancelar()
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		regra:       "resumir-livro.md",
		instrucao:   instrucao.String(),
		ferramentas: ferramentasEscrita,
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return pronto, fmt.Errorf("o resumo passou do limite de %d minutos", int(tempoDoResumo.Minutes()))
	}
	if err != nil {
		return pronto, err
	}
	if err := exigirArquivo(filepath.Join(pasta, "organizado.md")); err != nil {
		return pronto, err
	}
	ctx = context.WithoutCancel(ctx)
	if err := a.mudarEstado(ctx, ref.ID, "organizado"); err != nil {
		return pronto, err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "sugerir", nil)
	return pronto, err
}

func descreverEscolha(p pedidoLivro) string {
	if len(p.Capitulos) == 0 {
		return fmt.Sprintf("as páginas %d a %d", p.De, p.Ate)
	}
	var nomes []string
	for _, c := range p.Capitulos {
		nomes = append(nomes, fmt.Sprintf("%q", c.Titulo))
	}
	if len(nomes) == 1 {
		return "o capítulo " + nomes[0]
	}
	return "os capítulos " + strings.Join(nomes, ", ")
}

// notaDoLivro entra no pedido de criar ou integrar um Tema.
func notaDoLivro(ref *referencia) string {
	var p pedidoLivro
	json.Unmarshal(ref.Livro, &p)
	autor := ""
	if p.Autor != "" {
		autor = ", de " + p.Autor
	}
	return fmt.Sprintf(" Esta Referência é o resumo de %s do livro %q%s (no cabeçalho, use url: %s). "+
		"As seções terminam com <sub>Páginas: [p. N](link)</sub>: no Tema, em vez de <sub>Fonte: id [mm:ss]</sub>, "+
		"escreva <sub>Fonte: %s, p. N</sub> com o link de cada página, e ponha o livro (título e autor) em \"Citados\".",
		descreverEscolha(p), p.Titulo, autor, ref.URL, p.Titulo)
}
