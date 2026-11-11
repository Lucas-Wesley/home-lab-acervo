package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

// Artigos, blogs e qualquer página da web como Referência (D34). O servidor baixa a página e tira
// dela o texto principal, sem IA, e guarda em cru.md com o título, o site e o link; daí em diante o
// caminho é o do vídeo: organizar (regras/organizar-artigo.md) → traduzir, se for outra língua →
// sugerir → criar ou integrar, e gerar flashcards. Nada é específico de um site.

const (
	maxPaginaWeb  = 5 << 20 // bytes do HTML lidos
	maxTextoWeb   = 200_000 // caracteres do texto guardado em cru.md
	minTextoWeb   = 300     // menos que isso não é um artigo (login, página montada por JavaScript)
	tempoDaPagina = 30 * time.Second
)

var (
	clienteDaWeb   = &http.Client{Timeout: tempoDaPagina}
	errLinkDaWeb   = errors.New("Cole o link de uma página da web (http ou https).")
	errSemTextoWeb = errors.New("Não achei o texto desta página. Ela pode pedir login ou montar o texto no navegador (JavaScript), e aí o servidor não consegue ler.")
)

// linkDaWeb acha o primeiro link http(s) do texto (o "Compartilhar" do celular manda o título
// junto) e tira dele o que só serve para rastrear.
func linkDaWeb(bruto string) (string, error) {
	for _, parte := range strings.Fields(bruto) {
		u, err := url.Parse(strings.TrimRight(parte, ".,;)"))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			continue
		}
		u.Fragment = ""
		q := u.Query()
		for k := range q {
			if strings.HasPrefix(k, "utm_") || k == "fbclid" || k == "gclid" {
				q.Del(k)
			}
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
	return "", errLinkDaWeb
}

type paginaDaWeb struct {
	Titulo    string
	Site      string
	Autor     string
	Publicado string
	Idioma    string
	URL       string
	Texto     string
}

func (a *app) baixarPagina(ctx context.Context, link string) (paginaDaWeb, error) {
	var p paginaDaWeb
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return p, errLinkDaWeb
	}
	// Alguns sites recusam quem não parece um navegador.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 Acervo-homelab")
	req.Header.Set("Accept", "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en;q=0.8")
	res, err := clienteDaWeb.Do(req)
	if err != nil {
		return p, fmt.Errorf("A página não abriu: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return p, fmt.Errorf("A página respondeu %d.", res.StatusCode)
	}
	tipo := strings.ToLower(res.Header.Get("Content-Type"))
	if strings.Contains(tipo, "pdf") {
		return lerPDF(ctx, res.Body, res.Request.URL)
	}
	if tipo != "" && !strings.Contains(tipo, "html") {
		return p, errors.New("O link não é de uma página da web.")
	}
	doc, err := html.Parse(io.LimitReader(res.Body, maxPaginaWeb))
	if err != nil {
		return p, fmt.Errorf("Não consegui ler a página: %v", err)
	}
	p = lerPagina(doc, res.Request.URL)
	if utf8.RuneCountInString(p.Texto) < minTextoWeb {
		return p, errSemTextoWeb
	}
	return p, nil
}

const maxPDF = 30 << 20

// lerPDF tira o texto de um PDF da web com o pdftotext (D49). O título vem dos metadados do PDF; sem
// eles, fica vazio, e quem chamou usa o que tiver (o título da busca, ou o nome do arquivo).
func lerPDF(ctx context.Context, corpo io.Reader, endereco *url.URL) (paginaDaWeb, error) {
	p := paginaDaWeb{URL: endereco.String(), Site: strings.TrimPrefix(endereco.Hostname(), "www.")}
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return p, errors.New("O link é de um PDF, e este servidor não tem o leitor de PDF (pdftotext).")
	}
	temp, err := os.CreateTemp("", "pagina-*.pdf")
	if err != nil {
		return p, err
	}
	defer os.Remove(temp.Name())
	n, err := io.Copy(temp, io.LimitReader(corpo, maxPDF+1))
	temp.Close()
	if err != nil {
		return p, fmt.Errorf("O PDF não baixou: %v", err)
	}
	if n > maxPDF {
		return p, errors.New("O PDF é grande demais (mais de 30 MB).")
	}
	texto, err := exec.CommandContext(ctx, "pdftotext", "-enc", "UTF-8", "-q", temp.Name(), "-").Output()
	if err != nil {
		return p, errors.New("Não consegui ler o texto deste PDF.")
	}
	p.Texto = limparTextoDoPDF(string(texto))
	if info, err := exec.CommandContext(ctx, "pdfinfo", temp.Name()).Output(); err == nil {
		for _, linha := range strings.Split(string(info), "\n") {
			if v, ok := strings.CutPrefix(linha, "Title:"); ok {
				p.Titulo = strings.TrimSpace(v)
			}
		}
	}
	if utf8.RuneCountInString(p.Texto) < minTextoWeb {
		return p, errors.New("O PDF não tem texto (pode ser escaneado).")
	}
	return p, nil
}

// limparTextoDoPDF junta as linhas de um parágrafo (o PDF quebra no fim de cada linha) e tira as
// quebras de página.
func limparTextoDoPDF(s string) string {
	s = strings.ReplaceAll(s, "\f", "\n\n")
	var paragrafos []string
	for _, bloco := range regexp.MustCompile(`\n\s*\n`).Split(s, -1) {
		if linha := strings.Join(strings.Fields(bloco), " "); linha != "" {
			paragrafos = append(paragrafos, linha)
		}
	}
	return strings.Join(paragrafos, "\n\n")
}

// lerPagina tira os metadados e o texto principal da página.
func lerPagina(doc *html.Node, endereco *url.URL) paginaDaWeb {
	p := paginaDaWeb{URL: endereco.String()}
	meta := map[string]string{}
	var titulo, h1 string
	var andar func(*html.Node)
	andar = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				p.Idioma = atributo(n, "lang")
			case "title":
				if titulo == "" {
					titulo = limpar(textoDoNo(n))
				}
			case "h1":
				if h1 == "" {
					h1 = limpar(textoDoNo(n))
				}
			case "meta":
				chave := strings.ToLower(atributo(n, "property") + atributo(n, "name"))
				if v := limpar(atributo(n, "content")); chave != "" && v != "" && meta[chave] == "" {
					meta[chave] = v
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			andar(c)
		}
	}
	andar(doc)

	p.Site = primeiro(meta["og:site_name"], strings.TrimPrefix(endereco.Hostname(), "www."))
	p.Titulo = primeiro(meta["og:title"], semNomeDoSite(titulo), h1, p.Site)
	// "Café – Wikipédia, a enciclopédia livre", com o <h1> "Café": fica o do <h1>.
	if h1 != "" && h1 != p.Titulo && strings.HasPrefix(p.Titulo, h1) {
		p.Titulo = h1
	}
	p.Autor = primeiro(meta["author"], meta["article:author"])
	if strings.HasPrefix(p.Autor, "http") {
		p.Autor = "" // o article:author às vezes é o link do perfil
	}
	p.Publicado = primeiro(meta["article:published_time"], meta["date"])
	if len(p.Publicado) > 10 {
		p.Publicado = p.Publicado[:10]
	}
	p.Texto = textoPrincipal(doc)
	return p
}

// Partes da página que nunca são o conteúdo.
var foraDoArtigo = map[string]bool{"head": true, "script": true, "style": true, "noscript": true, "template": true,
	"svg": true, "iframe": true, "nav": true, "footer": true, "aside": true, "form": true, "button": true,
	"select": true, "input": true, "dialog": true, "menu": true}

// Pistas, entre as palavras da classe, do id ou do papel, de blocos que ficam em volta do artigo.
var pistasDeLixo = map[string]bool{"comment": true, "comments": true, "share": true, "sharing": true, "social": true,
	"related": true, "recommended": true, "newsletter": true, "subscribe": true, "sidebar": true, "cookie": true,
	"cookies": true, "ad": true, "ads": true, "advert": true, "advertisement": true, "banner": true, "promo": true,
	"popup": true, "modal": true, "breadcrumb": true, "breadcrumbs": true, "paywall": true, "navbox": true,
	"catlinks": true, "noprint": true, "navigation": true, "menu": true, "dropdown": true, "toc": true}

var separadoresDePistas = regexp.MustCompile(`[\s_-]+`)

var blocosDoArtigo = map[string]bool{"p": true, "div": true, "section": true, "article": true, "main": true, "li": true,
	"br": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "tr": true, "blockquote": true,
	"figure": true, "figcaption": true, "dd": true, "dt": true, "table": true, "ul": true, "ol": true, "header": true}

// Num <pre> (código, poema), as quebras de linha valem; no resto do HTML, são só espaço.
const quebraNoPre = "\x00"

// textoPrincipal acha a parte da página com o artigo e a escreve em Markdown simples: títulos,
// parágrafos, listas, citações e código.
func textoPrincipal(doc *html.Node) string {
	raiz := raizDoArtigo(doc)
	if raiz == nil {
		return ""
	}
	var b strings.Builder
	noPre := 0
	var andar func(*html.Node)
	andar = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			if noPre > 0 {
				b.WriteString(strings.ReplaceAll(n.Data, "\n", quebraNoPre))
			} else {
				b.WriteString(strings.ReplaceAll(n.Data, "\n", " "))
			}
			return
		case html.ElementNode:
			if foraDoArtigo[n.Data] || (n != raiz && pareceLixo(n)) {
				return
			}
			switch n.Data {
			case "h1", "h2":
				b.WriteString("\n## ")
			case "h3", "h4", "h5", "h6":
				b.WriteString("\n### ")
			case "li":
				b.WriteString("\n- ")
			case "blockquote":
				b.WriteString("\n> ")
			case "br":
				b.WriteString("\n")
			case "td", "th":
				b.WriteString(" ")
			case "pre":
				b.WriteString("\n```" + quebraNoPre)
				noPre++
				defer func() { noPre--; b.WriteString(quebraNoPre + "```\n") }()
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			andar(c)
		}
		if n.Type == html.ElementNode && blocosDoArtigo[n.Data] {
			b.WriteString("\n")
		}
	}
	andar(raiz)

	var linhas []string
	for _, l := range strings.Split(b.String(), "\n") {
		if strings.Contains(l, quebraNoPre) {
			l = strings.ReplaceAll(strings.TrimSpace(l), quebraNoPre, "\n")
		} else {
			l = limpar(l)
		}
		switch l {
		case "", "##", "###", "-", ">":
			continue
		}
		if len(linhas) > 0 && linhas[len(linhas)-1] == l {
			continue
		}
		linhas = append(linhas, l)
	}
	return strings.Join(linhas, "\n\n")
}

// raizDoArtigo: o elemento com mais texto em parágrafos próprios (os <p> filhos dele), como no
// "modo leitura" dos navegadores. Se ele tem pouco do texto da página em parágrafos (o artigo vem
// dividido em blocos, entre propagandas), sobe até juntar a maior parte.
func raizDoArtigo(doc *html.Node) *html.Node {
	corpo := buscarElemento(doc, "body")
	if corpo == nil {
		return nil
	}
	diretos := map[*html.Node]int{}
	var andar func(*html.Node)
	andar = func(n *html.Node) {
		if n.Type == html.ElementNode && (foraDoArtigo[n.Data] || (n != corpo && pareceLixo(n))) {
			return
		}
		if n.Type == html.ElementNode && n.Data == "p" && n.Parent != nil {
			diretos[n.Parent] += utf8.RuneCountInString(limpar(textoDoNo(n)))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			andar(c)
		}
	}
	andar(corpo)
	var melhor *html.Node
	maior := 0
	for n, t := range diretos {
		if t > maior || (t == maior && melhor != nil && contem(n, melhor)) {
			melhor, maior = n, t
		}
	}
	if melhor == nil {
		return corpo
	}
	total := 0
	for _, t := range diretos {
		total += t
	}
	emParagrafos := func(raiz *html.Node) int {
		soma := 0
		for n, t := range diretos {
			if contem(raiz, n) {
				soma += t
			}
		}
		return soma
	}
	for melhor != corpo && melhor.Parent != nil && emParagrafos(melhor)*10 < total*7 {
		melhor = melhor.Parent
	}
	return melhor
}

// contem diz se n está dentro de raiz (ou é ela).
func contem(raiz, n *html.Node) bool {
	for ; n != nil; n = n.Parent {
		if n == raiz {
			return true
		}
	}
	return false
}

func buscarElemento(n *html.Node, nome string) *html.Node {
	if n.Type == html.ElementNode && n.Data == nome {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if e := buscarElemento(c, nome); e != nil {
			return e
		}
	}
	return nil
}

func pareceLixo(n *html.Node) bool {
	if atributo(n, "hidden") != "" || atributo(n, "aria-hidden") == "true" {
		return true
	}
	for _, palavra := range separadoresDePistas.Split(strings.ToLower(atributo(n, "class")+" "+atributo(n, "id")+" "+atributo(n, "role")), -1) {
		if pistasDeLixo[palavra] {
			return true
		}
	}
	return false
}

func atributo(n *html.Node, nome string) string {
	for _, a := range n.Attr {
		if a.Key == nome {
			if a.Val == "" {
				return nome // atributo sem valor, como "hidden"
			}
			return a.Val
		}
	}
	return ""
}

func textoDoNo(n *html.Node) string {
	var b strings.Builder
	var andar func(*html.Node)
	andar = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode && foraDoArtigo[n.Data] {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			andar(c)
		}
	}
	andar(n)
	return b.String()
}

func limpar(s string) string {
	return strings.TrimSpace(espacos.ReplaceAllString(strings.ReplaceAll(s, " ", " "), " "))
}

func primeiro(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

// semNomeDoSite: "Como fazer pão - Blog do Pão" → "Como fazer pão".
func semNomeDoSite(titulo string) string {
	for _, sep := range []string{" | ", " - ", " – ", " — "} {
		if i := strings.LastIndex(titulo, sep); i > 0 {
			return strings.TrimSpace(titulo[:i])
		}
	}
	return titulo
}

// paginaParaCru: o cabeçalho com de onde veio e o texto da página.
func paginaParaCru(p paginaDaWeb) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Site: %s\n", p.Titulo, p.Site)
	if p.Autor != "" {
		fmt.Fprintf(&s, "- Autor: %s\n", p.Autor)
	}
	if p.Publicado != "" {
		fmt.Fprintf(&s, "- Publicado: %s\n", p.Publicado)
	}
	fmt.Fprintf(&s, "- URL: %s\n\n", p.URL)
	s.WriteString(cortar(p.Texto, maxTextoWeb, false))
	s.WriteString("\n")
	return s.String()
}

// extrairPagina é a etapa "extrair" de um artigo.
func (a *app) extrairPagina(ctx context.Context, ref *referencia, pasta string) error {
	p, err := a.baixarPagina(ctx, ref.URL)
	if p.Titulo == "" && p.Texto != "" {
		if u, e := url.Parse(ref.URL); e == nil {
			p.Titulo, _ = url.PathUnescape(path.Base(u.Path))
		}
	}
	if p.Titulo != "" {
		a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ?, canal = ?, idioma = ? WHERE id = ?`,
			cortar(p.Titulo, 200, false), p.Site, p.Idioma, ref.ID)
		ref.Idioma = p.Idioma
	}
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(paginaParaCru(p)), 0o644)
}

func notaDoArtigo(ref *referencia) string {
	return fmt.Sprintf(" Esta Referência é uma página da web (%s), não um vídeo: não há marcas de tempo. "+
		"No Tema, em vez de <sub>Fonte: id [mm:ss]</sub>, escreva <sub>Fonte: %s</sub>, e ponha a página (título, site e link) em \"Citados\".",
		strings.TrimSpace(ref.Canal+" · "+ref.URL), ref.ID)
}
