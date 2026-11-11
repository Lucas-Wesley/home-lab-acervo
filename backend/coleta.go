package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// A coleta do Pesquisador (D27) não usa a IA: o servidor busca nas fontes com o que o esboço
// escolheu e guarda o material em fontes/, um arquivo por fonte, com de onde veio no topo. A IA lê
// esses arquivos ao escrever, e a conferência confere o texto contra eles.
//
// D49: cada busca tem uma chave (o que buscar), para "Buscar de novo o que faltou" refazer só as
// que falharam; as consultas tentam de novo quando o site pede para esperar; a Wikipédia é lida
// pelo nome exato do artigo (com a em inglês de reserva); os capítulos que o esboço escolhe nos
// livros marcados (D38) vêm inteiros, com as marcações do Lucas; e as páginas que o esboço escolhe
// na internet são lidas pelo servidor, com o texto de verdade.

const (
	maxArtigo       = 60_000 // caracteres de um artigo da Wikipédia, de uma página ou de um capítulo
	minArtigoBom    = 3_000  // um artigo em português menor que isso busca também o em inglês
	minDuracaoVideo = 4 * 60
	maxDuracaoVideo = 90 * 60
	tempoDeConsulta = 30 * time.Second

	fonteCapitulo = "capitulo" // um capítulo inteiro de um livro (não é uma escolha do Lucas: vem com "livros")
	fonteSite     = "site"     // uma página escolhida no esboço (vem com "sites")
)

var clienteDaColeta = &http.Client{Timeout: tempoDeConsulta}

// Quando o site responde "muitas requisições" (429) ou falha por um momento, a consulta espera e
// tenta de novo. Os testes encurtam as esperas.
var (
	esperasDaColeta   = []time.Duration{2 * time.Second, 5 * time.Second, 12 * time.Second}
	maxEsperaDaColeta = 30 * time.Second
	intervaloWiki     = 300 * time.Millisecond
)

// A Wikipédia pede que cada programa se identifique com um contato e não dispare consultas em rajada.
const agenteDaColeta = "Acervo/1.0 (https://github.com/Lucas-Wesley/home-lab-acervo; app pessoal de estudo)"

var ritmo = struct {
	sync.Mutex
	ultimo map[string]time.Time
}{ultimo: map[string]time.Time{}}

// esperarAVez espaça as consultas à Wikipédia.
func esperarAVez(ctx context.Context, host string) {
	if !strings.HasSuffix(host, "wikipedia.org") {
		return
	}
	ritmo.Lock()
	vez := ritmo.ultimo[host].Add(intervaloWiki)
	if agora := time.Now(); vez.Before(agora) {
		vez = agora
	}
	ritmo.ultimo[host] = vez
	ritmo.Unlock()
	select {
	case <-ctx.Done():
	case <-time.After(time.Until(vez)):
	}
}

type erroDeConsulta struct {
	msg      string
	repetir  bool
	aguardar time.Duration
}

func (e *erroDeConsulta) Error() string { return e.msg }

// buscarJSON faz um GET e lê a resposta, tentando de novo quando vale a pena.
func buscarJSON(ctx context.Context, endereco string, destino any) error {
	for tentativa := 0; ; tentativa++ {
		err := buscarJSONUmaVez(ctx, endereco, destino)
		var e *erroDeConsulta
		if err == nil || !errors.As(err, &e) || !e.repetir || tentativa >= len(esperasDaColeta) || ctx.Err() != nil {
			return err
		}
		espera := esperasDaColeta[tentativa]
		if e.aguardar > 0 {
			espera = min(e.aguardar, maxEsperaDaColeta)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(espera):
		}
	}
}

func buscarJSONUmaVez(ctx context.Context, endereco string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", agenteDaColeta)
	esperarAVez(ctx, req.URL.Hostname())
	resp, err := clienteDaColeta.Do(req)
	if err != nil {
		return &erroDeConsulta{msg: err.Error(), repetir: ctx.Err() == nil}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		e := &erroDeConsulta{msg: fmt.Sprintf("%s respondeu %d: %s", req.URL.Host, resp.StatusCode, primeiraLinha(string(corpo), "")),
			repetir: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
		if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && s >= 0 {
			e.aguardar = time.Duration(s) * time.Second
		}
		return e
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}

// ---------- a coleta ----------

// coletor guarda o que uma coleta já viu e quantos vídeos já pegou.
type coletor struct {
	a      *app
	p      pedidoPesquisa
	pasta  string
	al     alcance
	vistos map[string]bool
	videos int
}

func (a *app) novoColetor(p pedidoPesquisa, pasta string, anteriores []coletado) *coletor {
	c := &coletor{a: a, p: p, pasta: pasta, al: p.alcance(), vistos: map[string]bool{}}
	for _, x := range anteriores {
		if x.Erro != "" {
			continue
		}
		c.vistos[chaveDoVisto(x)] = true
		if x.Fonte == fonteYoutube && x.rodada() == p.rodada() {
			c.videos++
		}
	}
	return c
}

// chaveDoVisto: o que já foi guardado não é guardado de novo (a mesma página, artigo ou vídeo).
func chaveDoVisto(c coletado) string {
	switch c.Fonte {
	case fonteCapitulo, fonteSite:
		return c.Chave
	}
	return c.Fonte + ":" + c.URL
}

// chavesDaColeta: tudo o que a coleta busca, na ordem.
func chavesDaColeta(p pedidoPesquisa) []string {
	var chaves []string
	for i, q := range p.Esboco {
		if q.Livros != "" {
			chaves = append(chaves, fmt.Sprintf("%s:%d", fonteLivros, i+1))
		}
		if q.Wikipedia != "" {
			chaves = append(chaves, fmt.Sprintf("%s:%d", fonteWikipedia, i+1))
		}
		if q.Youtube != "" {
			chaves = append(chaves, fmt.Sprintf("%s:%d", fonteYoutube, i+1))
		}
	}
	for _, c := range p.Capitulos {
		chaves = append(chaves, c.chave())
	}
	for _, pg := range p.Paginas {
		chaves = append(chaves, pg.chave())
	}
	return chaves
}

// buscar faz a busca de uma chave e marca o resultado com ela.
func (c *coletor) buscar(ctx context.Context, chave string) []coletado {
	var achados []coletado
	pergunta := 0
	tipo, resto, _ := strings.Cut(chave, ":")
	switch tipo {
	case fonteLivros, fonteWikipedia, fonteYoutube:
		n, err := strconv.Atoi(resto)
		if err != nil || n < 1 || n > len(c.p.Esboco) {
			return nil
		}
		pergunta = n
		q := c.p.Esboco[n-1]
		switch tipo {
		case fonteLivros:
			nome := fmt.Sprintf("%s/livros-%d.md", pastaFontes, n)
			if c.p.rodada() > 1 {
				nome = fmt.Sprintf("%s/livros-r%d-%d.md", pastaFontes, c.p.rodada(), n)
			}
			achados = []coletado{c.a.coletarLivros(ctx, c.pasta, nome, q, c.al.trechos)}
		case fonteWikipedia:
			achados = c.a.coletarWikipedia(ctx, c.pasta, q.Wikipedia, c.al.artigos, c.vistos)
		case fonteYoutube:
			if c.videos >= c.al.maxVideos {
				return nil
			}
			achados = c.a.coletarVideos(ctx, c.pasta, q.Youtube, min(c.al.videos, c.al.maxVideos-c.videos), c.vistos)
			for _, x := range achados {
				if x.Erro == "" {
					c.videos++
				}
			}
		}
	case fonteCapitulo:
		for _, cap := range c.p.Capitulos {
			if cap.chave() == chave && !c.vistos[chave] {
				pergunta = cap.Pergunta
				achados = []coletado{c.a.coletarCapitulo(ctx, c.pasta, cap)}
			}
		}
	case fonteSite:
		for _, pg := range c.p.Paginas {
			if pg.chave() == chave && !c.vistos[chave] {
				pergunta = pg.Pergunta
				achados = []coletado{c.a.coletarPagina(ctx, c.pasta, pg)}
			}
		}
	}
	for i := range achados {
		achados[i].Chave, achados[i].Rodada = chave, c.p.rodada()
		if pergunta > 0 {
			achados[i].Pergunta = pergunta
		}
		if achados[i].Erro == "" {
			c.vistos[chaveDoVisto(achados[i])] = true
		}
	}
	return achados
}

func (a *app) coletar(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return pronto, err
	}
	var params struct {
		Faltou bool `json:"faltou"`
	}
	if len(t.Params) > 0 {
		json.Unmarshal(t.Params, &params)
	}
	if params.Faltou {
		return a.coletarOQueFaltou(ctx, ref, p)
	}
	pasta := a.pastaReferencia(ref.ID)
	rodada := p.rodada()
	var anteriores []coletado
	if rodada == 1 {
		// Coletar de novo começa do zero; o que a IA guardou dos sites também sai.
		os.RemoveAll(filepath.Join(pasta, pastaFontes))
	} else {
		// Ao aprofundar, o material das rodadas anteriores fica e não é buscado de novo; o desta
		// rodada (de uma tentativa que falhou) sai.
		for _, c := range p.Coleta {
			if c.rodada() == rodada {
				if c.Arquivo != "" {
					os.Remove(filepath.Join(pasta, filepath.FromSlash(c.Arquivo)))
				}
				continue
			}
			anteriores = append(anteriores, c)
		}
	}
	os.Remove(filepath.Join(pasta, "cru.md"))
	os.Remove(filepath.Join(pasta, "organizado.md"))
	if err := os.MkdirAll(filepath.Join(pasta, pastaFontes), 0o755); err != nil {
		return pronto, err
	}

	col := a.novoColetor(p, pasta, anteriores)
	var coleta []coletado
	for _, chave := range chavesDaColeta(p) {
		if ctx.Err() != nil {
			return pronto, ctx.Err()
		}
		coleta = append(coleta, col.buscar(ctx, chave)...)
	}
	if ctx.Err() != nil {
		return pronto, ctx.Err()
	}

	p.Coleta = append(anteriores, coleta...)
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return pronto, err
	}
	if guardados(coleta) == 0 && !p.usa(fonteSites) {
		if rodada > 1 {
			return pronto, errors.New("nenhuma fonte trouxe material novo para aprofundar: " + resumoDosErros(coleta))
		}
		return pronto, errors.New("nenhuma fonte trouxe material para a pesquisa: " + resumoDosErros(coleta))
	}
	if err := os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(materialColetado(p)), 0o644); err != nil {
		return pronto, err
	}
	if err := a.mudarEstado(ctx, ref.ID, "cru"); err != nil {
		return pronto, err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "redigir", nil)
	return pronto, err
}

// coletarOQueFaltou refaz só as buscas desta rodada que falharam (D49). Se alguma trouxer
// material, o texto é escrito de novo com tudo; o de antes fica como Versão anterior.
func (a *app) coletarOQueFaltou(ctx context.Context, ref *referencia, p pedidoPesquisa) (resultado, error) {
	pasta := a.pastaReferencia(ref.ID)
	var manter []coletado
	var refazer []string
	for _, c := range p.Coleta {
		if c.rodada() == p.rodada() && c.Erro != "" && c.Chave != "" {
			if !slices.Contains(refazer, c.Chave) {
				refazer = append(refazer, c.Chave)
			}
			continue
		}
		manter = append(manter, c)
	}
	if len(refazer) == 0 {
		return pronto, nil
	}
	col := a.novoColetor(p, pasta, manter)
	var novos []coletado
	for _, chave := range refazer {
		if ctx.Err() != nil {
			return pronto, ctx.Err()
		}
		novos = append(novos, col.buscar(ctx, chave)...)
	}
	p.Coleta = append(manter, novos...)
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return pronto, err
	}
	if guardados(novos) == 0 {
		return pronto, nil // nada novo: o texto continua, e a página mostra o que ainda falta
	}
	if err := os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(materialColetado(p)), 0o644); err != nil {
		return pronto, err
	}
	if p.rodada() == 1 {
		// Na 1ª rodada, anterior.md está livre: guarda o texto de antes para comparar. Nas outras,
		// ele é o texto da rodada anterior, que a escrita precisa.
		if b, err := os.ReadFile(filepath.Join(pasta, "organizado.md")); err == nil {
			os.WriteFile(filepath.Join(pasta, "anterior.md"), b, 0o644)
		}
	}
	_, err := a.fila.enfileirar(ctx, ref.ID, "redigir", map[string]bool{"faltou": true})
	return pronto, err
}

// buscasQueFalharam: quantas buscas desta rodada não trouxeram nada.
func (p pedidoPesquisa) buscasQueFalharam() int {
	n := 0
	for _, c := range p.Coleta {
		if c.rodada() == p.rodada() && c.Erro != "" && c.Chave != "" {
			n++
		}
	}
	return n
}

func guardados(coleta []coletado) int {
	n := 0
	for _, c := range coleta {
		if c.Erro == "" {
			n++
		}
	}
	return n
}

func resumoDosErros(coleta []coletado) string {
	var erros []string
	for _, c := range coleta {
		if c.Erro != "" {
			erros = append(erros, c.Titulo+": "+c.Erro)
		}
	}
	if len(erros) == 0 {
		return "nada encontrado"
	}
	return strings.Join(erros, "; ")
}

var nomesCurtos = map[string]string{fonteLivros: "Livros", fonteCapitulo: "Capítulo de livro", fonteWikipedia: "Wikipédia",
	fonteYoutube: "YouTube", fonteSite: "Site"}

// materialColetado é o cru.md da pesquisa: o que foi buscado e o que veio de cada fonte.
func materialColetado(p pedidoPesquisa) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# Material coletado\n\n> Pesquisa: %s\n\n## Perguntas\n\n", p.Tema)
	for i, q := range p.Esboco {
		fmt.Fprintf(&s, "%d. %s\n", i+1, q.Pergunta)
	}
	if len(p.Anteriores) > 0 {
		s.WriteString("\nDas rodadas anteriores:\n\n")
		for _, q := range p.Anteriores {
			fmt.Fprintf(&s, "- %s\n", q.Pergunta)
		}
	}
	s.WriteString("\n## Fontes guardadas\n\n")
	for _, c := range p.Coleta {
		if c.Erro != "" {
			continue
		}
		titulo := c.Titulo
		if c.URL != "" {
			titulo = fmt.Sprintf("[%s](%s)", c.Titulo, c.URL)
		}
		extra := ""
		if c.Pergunta > 0 {
			extra = fmt.Sprintf(" (pergunta %d)", c.Pergunta)
		}
		if p.rodada() > 1 {
			extra += fmt.Sprintf(" (rodada %d)", c.rodada())
		}
		fmt.Fprintf(&s, "- **%s:** %s — `%s`%s\n", nomesCurtos[c.Fonte], titulo, c.Arquivo, extra)
	}
	if p.usa(fonteSites) {
		s.WriteString("- **Sites:** as páginas escolhidas no esboço estão em `fontes/site-*.md`; o que a IA ainda pesquisar ao escrever vai para `fontes/web-*.md`.\n")
	}
	var semNada []string
	for _, c := range p.Coleta {
		if c.Erro != "" && c.rodada() == p.rodada() {
			semNada = append(semNada, fmt.Sprintf("- %s, %s: %s", nomesCurtos[c.Fonte], c.Titulo, c.Erro))
		}
	}
	if len(semNada) > 0 {
		s.WriteString("\n## Sem resultado\n\n" + strings.Join(semNada, "\n") + "\n")
	}
	return s.String()
}

// ---------- Livros: trechos da busca (R32 da Biblioteca) ----------

type trechoDeLivro struct {
	BookID  string `json:"book_id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	Page    int    `json:"page"`
	Chapter string `json:"chapter"`
	Text    string `json:"text"`
	Anchor  string `json:"anchor"`
}

func (a *app) linkDoLivro(t trechoDeLivro) string {
	return fmt.Sprintf("%s/#/livro/%s?em=%s", a.cfg.bibliotecaLink, url.PathEscape(t.BookID), url.QueryEscape(t.Anchor))
}

// mesmoLivro: dois exemplares do mesmo livro (mesmo título e autor) contam como um.
func mesmoLivro(titulo, autor string) string { return slug(titulo) + "|" + slug(autor) }

func (a *app) coletarLivros(ctx context.Context, pasta, arquivo string, q pergunta, limite int) coletado {
	c := coletado{Fonte: fonteLivros, Titulo: fmt.Sprintf("busca %q", q.Livros)}
	if a.cfg.biblioteca == "" {
		c.Erro = "a busca nos livros não está configurada (BIBLIOTECA_URL)"
		return c
	}
	var resposta struct {
		Results []trechoDeLivro `json:"results"`
	}
	// Pede a mais, porque os trechos de um exemplar repetido saem.
	endereco := fmt.Sprintf("%s/api/search?limit=%d&q=%s", a.cfg.biblioteca, limite*2, url.QueryEscape(q.Livros))
	if err := buscarJSON(ctx, endereco, &resposta); err != nil {
		c.Erro = "a Biblioteca não respondeu: " + err.Error()
		return c
	}
	exemplar := map[string]string{}
	var trechos []trechoDeLivro
	for _, t := range resposta.Results {
		chave := mesmoLivro(t.Title, t.Author)
		if id, ok := exemplar[chave]; ok && id != t.BookID {
			continue
		}
		exemplar[chave] = t.BookID
		if len(trechos) < limite {
			trechos = append(trechos, t)
		}
	}
	if len(trechos) == 0 {
		c.Erro = "nenhum trecho nos livros"
		return c
	}
	var s strings.Builder
	fmt.Fprintf(&s, "# Trechos dos livros da Biblioteca\n\n- Pergunta: %s\n- Busca: %s\n\n", q.Pergunta, q.Livros)
	livros := map[string]bool{}
	for _, t := range trechos {
		livros[t.Title] = true
		autor := ""
		if t.Author != "" {
			autor = ", de " + t.Author
		}
		capitulo := ""
		if t.Chapter != "" {
			capitulo = " (" + t.Chapter + ")"
		}
		fmt.Fprintf(&s, "## %s%s — p. %d%s\n\nLink: %s\n\n%s\n\n", t.Title, autor, t.Page, capitulo, a.linkDoLivro(t), t.Text)
	}
	c.Arquivo = arquivo
	c.Titulo = fmt.Sprintf("%d trechos de %d livro(s) para %q", len(trechos), len(livros), q.Livros)
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), []byte(s.String()), 0o644); err != nil {
		c.Erro = err.Error()
	}
	return c
}

// ---------- Livros: capítulos inteiros (R33), com as marcações (R34) ----------

func (a *app) coletarCapitulo(ctx context.Context, pasta string, cap capituloPesquisa) coletado {
	c := coletado{Fonte: fonteCapitulo, Titulo: fmt.Sprintf("“%s” (p. %d–%d), de %s", cap.Capitulo, cap.De, cap.Ate, cap.Titulo),
		URL: a.linkDaBiblioteca(cap.Livro, "")}
	if a.cfg.biblioteca == "" {
		c.Erro = "a Biblioteca não está configurada (BIBLIOTECA_URL)"
		return c
	}
	var texto textoDoLivro
	endereco := fmt.Sprintf("%s/api/books/%s/text?from=%d&to=%d", a.cfg.biblioteca, url.PathEscape(cap.Livro), cap.De, cap.Ate)
	if err := buscarJSON(ctx, endereco, &texto); err != nil {
		c.Erro = "a Biblioteca não devolveu o capítulo: " + err.Error()
		return c
	}
	livro := pedidoLivro{Livro: cap.Livro, Titulo: cap.Titulo, Autor: cap.Autor}
	md, caracteres := a.textoDoLivroEmMarkdown(livro, texto)
	if caracteres < 200 {
		c.Erro = "o capítulo quase não tem texto"
		return c
	}
	md = cortar(md, maxArtigo, false)
	if marcacoes := a.marcacoesEmMarkdown(livro, texto); marcacoes != "" {
		md += "\n\n" + marcacoes
	}
	c.Arquivo = fmt.Sprintf("%s/livro-%s-p%d.md", pastaFontes, cortar(slug(cap.Titulo), 40, false), cap.De)
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), []byte(md), 0o644); err != nil {
		c.Erro = err.Error()
	}
	return c
}

// ---------- Sites: as páginas que o esboço escolheu, lidas pelo servidor ----------

func (a *app) coletarPagina(ctx context.Context, pasta string, pg paginaPesquisa) coletado {
	c := coletado{Fonte: fonteSite, Titulo: primeiro(pg.Titulo, pg.URL), URL: pg.URL}
	pagina, err := a.baixarPagina(ctx, pg.URL)
	if err != nil {
		c.Erro = err.Error()
		return c
	}
	if pagina.Titulo != "" {
		c.Titulo = pagina.Titulo
	}
	if pagina.Site != "" {
		c.Titulo += " (" + pagina.Site + ")"
	}
	pagina.Texto = cortar(pagina.Texto, maxArtigo, false)
	soma := sha1.Sum([]byte(pg.URL))
	c.Arquivo = fmt.Sprintf("%s/site-%s.md", pastaFontes, hex.EncodeToString(soma[:])[:10])
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), []byte(paginaParaCru(pagina)), 0o644); err != nil {
		c.Erro = err.Error()
	}
	return c
}

// ---------- Wikipédia ----------

type artigoWiki struct {
	titulo, link, texto, ingles string
}

// lerArtigo lê um artigo pelo nome exato (seguindo os redirecionamentos). Sem artigo, texto vazio.
func (a *app) lerArtigo(ctx context.Context, base, titulo string) (artigoWiki, error) {
	var resposta struct {
		Query struct {
			Pages map[string]struct {
				Title     string `json:"title"`
				Extract   string `json:"extract"`
				Langlinks []struct {
					Lang   string `json:"lang"`
					Titulo string `json:"*"`
				} `json:"langlinks"`
			} `json:"pages"`
		} `json:"query"`
	}
	endereco := base + "/w/api.php?action=query&prop=extracts%7Clanglinks&explaintext=1&redirects=1&lllang=en&format=json&titles=" + url.QueryEscape(titulo)
	if err := buscarJSON(ctx, endereco, &resposta); err != nil {
		return artigoWiki{}, err
	}
	var art artigoWiki
	for _, pg := range resposta.Query.Pages {
		if strings.TrimSpace(pg.Extract) == "" {
			continue
		}
		art = artigoWiki{titulo: pg.Title, texto: pg.Extract, link: base + "/wiki/" + url.PathEscape(strings.ReplaceAll(pg.Title, " ", "_"))}
		for _, l := range pg.Langlinks {
			if l.Lang == "en" {
				art.ingles = l.Titulo
			}
		}
	}
	return art, nil
}

func (a *app) buscarTitulos(ctx context.Context, base, busca string) ([]string, error) {
	var achados struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	endereco := base + "/w/api.php?action=query&list=search&format=json&srnamespace=0&srlimit=5&srsearch=" + url.QueryEscape(busca)
	if err := buscarJSON(ctx, endereco, &achados); err != nil {
		return nil, err
	}
	var titulos []string
	for _, r := range achados.Query.Search {
		titulos = append(titulos, r.Title)
	}
	return titulos, nil
}

// palavrasSemSentido não contam para dizer se um título combina com o nome pedido.
var palavrasSemSentido = map[string]bool{"das": true, "dos": true, "com": true, "para": true, "por": true, "uma": true,
	"sobre": true, "como": true, "que": true, "the": true, "and": true, "nas": true, "nos": true, "pela": true, "pelo": true}

func palavras(s string) []string {
	var lista []string
	for _, p := range strings.Split(slug(s), "-") {
		if utf8.RuneCountInString(p) >= 3 && !palavrasSemSentido[p] {
			lista = append(lista, p)
		}
	}
	return lista
}

// tituloCombina: o resultado da busca só serve se tiver mais da metade das palavras do nome pedido
// (assim "Zé Ramalho" não vale por "João Ramalho", nem "Plantae" por "plantas cultivadas").
func tituloCombina(nome, titulo string) bool {
	pedidas := palavras(nome)
	if len(pedidas) == 0 {
		return false
	}
	tem := palavras(titulo)
	n := 0
	for _, p := range pedidas {
		if slices.Contains(tem, p) {
			n++
		}
	}
	return n*2 > len(pedidas)
}

// nomesDeArtigo: o esboço pode pedir mais de um artigo, separados por "|".
func nomesDeArtigo(nomes string, limite int) []string {
	var lista []string
	for _, n := range strings.Split(nomes, "|") {
		if n = strings.TrimSpace(n); n != "" && !slices.Contains(lista, n) && len(lista) < limite {
			lista = append(lista, n)
		}
	}
	return lista
}

// artigoPeloNome: o artigo com esse nome ou, se não houver, o da busca cujo título combine.
func (a *app) artigoPeloNome(ctx context.Context, base, nome string, vistos map[string]bool) (artigoWiki, error) {
	art, err := a.lerArtigo(ctx, base, nome)
	if err != nil || art.texto != "" {
		return art, err
	}
	titulos, err := a.buscarTitulos(ctx, base, nome)
	if err != nil {
		return art, err
	}
	for _, t := range titulos {
		link := base + "/wiki/" + url.PathEscape(strings.ReplaceAll(t, " ", "_"))
		if tituloCombina(nome, t) && !vistos[fonteWikipedia+":"+link] {
			return a.lerArtigo(ctx, base, t)
		}
	}
	return art, nil
}

func (a *app) coletarWikipedia(ctx context.Context, pasta, nomes string, limite int, vistos map[string]bool) []coletado {
	var coleta []coletado
	for _, nome := range nomesDeArtigo(nomes, limite) {
		art, err := a.artigoPeloNome(ctx, a.cfg.wikipedia, nome, vistos)
		if err != nil {
			coleta = append(coleta, coletado{Fonte: fonteWikipedia, Titulo: fmt.Sprintf("artigo %q", nome), Erro: err.Error()})
			continue
		}
		achou := art.texto != ""
		if achou && !vistos[fonteWikipedia+":"+art.link] {
			coleta = append(coleta, a.guardarArtigo(pasta, art, false))
			vistos[fonteWikipedia+":"+art.link] = true
		}
		// Artigo em português curto ou inexistente: o em inglês completa.
		if a.cfg.wikipediaEN != "" && utf8.RuneCountInString(art.texto) < minArtigoBom {
			var en artigoWiki
			if art.ingles != "" {
				en, err = a.lerArtigo(ctx, a.cfg.wikipediaEN, art.ingles)
			} else {
				en, err = a.artigoPeloNome(ctx, a.cfg.wikipediaEN, nome, vistos)
			}
			if err == nil && en.texto != "" {
				achou = true
				if !vistos[fonteWikipedia+":"+en.link] {
					coleta = append(coleta, a.guardarArtigo(pasta, en, true))
					vistos[fonteWikipedia+":"+en.link] = true
				}
			}
		}
		if !achou {
			coleta = append(coleta, coletado{Fonte: fonteWikipedia, Titulo: fmt.Sprintf("artigo %q", nome), Erro: "nenhum artigo com esse nome"})
		}
	}
	return coleta
}

func (a *app) guardarArtigo(pasta string, art artigoWiki, ingles bool) coletado {
	c := coletado{Fonte: fonteWikipedia, Titulo: art.titulo, URL: art.link}
	nome, rotulo := "wikipedia-"+slug(art.titulo), "Wikipédia"
	if ingles {
		nome, rotulo = "wikipedia-en-"+slug(art.titulo), "Wikipédia em inglês"
		c.Titulo += " (em inglês)"
	}
	c.Arquivo = fmt.Sprintf("%s/%s.md", pastaFontes, cortar(nome, 80, false))
	conteudo := fmt.Sprintf("# %s (%s)\n\nURL: %s\n\n%s\n", art.titulo, rotulo, art.link, cortar(art.texto, maxArtigo, false))
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), []byte(conteudo), 0o644); err != nil {
		c.Erro = err.Error()
	}
	return c
}

// ---------- YouTube ----------

type videoAchado struct {
	ID      string  `json:"id"`
	Titulo  string  `json:"title"`
	Canal   string  `json:"channel"`
	Duracao float64 `json:"duration"`
}

// Vídeos que não explicam nada: trailer, teaser, clipe, corte curto.
var videoSemConteudo = regexp.MustCompile(`(?i)\b(trailer|teaser|shorts|clipe oficial|official (music )?video)\b|#shorts`)

// coletarVideos busca no YouTube pelo yt-dlp e guarda a transcrição dos primeiros vídeos com
// legenda e de 4 a 90 minutos. O vídeo serve só de fonte: não vira Referência própria.
func (a *app) coletarVideos(ctx context.Context, pasta, busca string, limite int, vistos map[string]bool) []coletado {
	falhou := func(msg string) []coletado {
		return []coletado{{Fonte: fonteYoutube, Titulo: fmt.Sprintf("busca %q", busca), Erro: msg}}
	}
	bruto, err := a.ytdlp(ctx, "--flat-playlist", "-J", fmt.Sprintf("ytsearch%d:%s", limite+6, busca))
	if err != nil {
		return falhou(err.Error())
	}
	var lista struct {
		Entries []videoAchado `json:"entries"`
	}
	if err := json.Unmarshal(bruto, &lista); err != nil {
		return falhou("resposta do yt-dlp ilegível")
	}
	var coleta []coletado
	for _, v := range lista.Entries {
		if len(coleta) == limite || ctx.Err() != nil {
			break
		}
		link := "https://www.youtube.com/watch?v=" + v.ID
		if !idDeVideo.MatchString(v.ID) || vistos[fonteYoutube+":"+link] || videoSemConteudo.MatchString(v.Titulo) ||
			(v.Duracao > 0 && (v.Duracao < minDuracaoVideo || v.Duracao > maxDuracaoVideo)) {
			continue
		}
		vistos[fonteYoutube+":"+link] = true
		if c, ok := a.guardarTranscricao(ctx, pasta, v); ok {
			coleta = append(coleta, c)
		}
	}
	if len(coleta) == 0 {
		return falhou("nenhum vídeo com legenda")
	}
	return coleta
}

// guardarTranscricao usa a mesma extração das Referências de vídeo, numa pasta temporária.
func (a *app) guardarTranscricao(ctx context.Context, pasta string, v videoAchado) (coletado, bool) {
	link := "https://www.youtube.com/watch?v=" + v.ID
	temp, err := os.MkdirTemp(filepath.Join(pasta, pastaFontes), "video-")
	if err != nil {
		return coletado{}, false
	}
	defer os.RemoveAll(temp)
	info, _, err := a.baixarLegenda(ctx, link, temp)
	if err != nil {
		return coletado{}, false
	}
	cru, err := os.ReadFile(filepath.Join(temp, "cru.md"))
	if err != nil {
		return coletado{}, false
	}
	titulo := info.Titulo
	if titulo == "" {
		titulo = v.Titulo
	}
	if videoSemConteudo.MatchString(titulo) {
		return coletado{}, false
	}
	if canal := info.Canal; canal != "" {
		titulo += " (" + canal + ")"
	}
	c := coletado{Fonte: fonteYoutube, Titulo: titulo, URL: link, Arquivo: fmt.Sprintf("%s/video-%s.md", pastaFontes, v.ID)}
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), cru, 0o644); err != nil {
		return coletado{}, false
	}
	return c, true
}
