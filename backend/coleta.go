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
	"strings"
	"time"
)

// A coleta do Pesquisador (D27) não usa a IA: o servidor busca nas fontes com as palavras do
// esboço e guarda o material em fontes/, um arquivo por fonte, com de onde veio no topo. A IA lê
// esses arquivos ao escrever, e a conferência confere o texto contra eles.

const (
	maxArtigo       = 60_000 // caracteres de um artigo da Wikipédia
	minDuracaoVideo = 2 * 60
	maxDuracaoVideo = 90 * 60
	tempoDeConsulta = 30 * time.Second
)

var clienteDaColeta = &http.Client{Timeout: tempoDeConsulta}

// buscarJSON faz um GET e lê a resposta. A Wikipédia pede que cada programa se identifique.
func buscarJSON(ctx context.Context, endereco string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Acervo/1.0 (app pessoal no homelab do Lucas)")
	resp, err := clienteDaColeta.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("%s respondeu %d: %s", req.URL.Host, resp.StatusCode, strings.TrimSpace(string(corpo)))
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}

func (a *app) coletar(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return pronto, err
	}
	pasta := a.pastaReferencia(ref.ID)
	// Coletar de novo começa do zero; o que a IA guardou dos sites também sai.
	os.RemoveAll(filepath.Join(pasta, pastaFontes))
	os.Remove(filepath.Join(pasta, "cru.md"))
	os.Remove(filepath.Join(pasta, "organizado.md"))
	if err := os.MkdirAll(filepath.Join(pasta, pastaFontes), 0o755); err != nil {
		return pronto, err
	}

	al := p.alcance()
	var coleta []coletado
	vistos := map[string]bool{}
	videos := 0
	for i, q := range p.Esboco {
		if ctx.Err() != nil {
			return pronto, ctx.Err()
		}
		if q.Livros != "" {
			coleta = append(coleta, a.coletarLivros(ctx, pasta, i+1, q, al.trechos))
		}
		if q.Wikipedia != "" {
			coleta = append(coleta, a.coletarWikipedia(ctx, pasta, q.Wikipedia, al.artigos, vistos)...)
		}
		if q.Youtube != "" && videos < al.maxVideos {
			novos := a.coletarVideos(ctx, pasta, q.Youtube, min(al.videos, al.maxVideos-videos), vistos)
			for _, c := range novos {
				if c.Erro == "" {
					videos++
				}
			}
			coleta = append(coleta, novos...)
		}
	}
	if ctx.Err() != nil {
		return pronto, ctx.Err()
	}

	p.Coleta = coleta
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return pronto, err
	}
	guardados := 0
	for _, c := range coleta {
		if c.Erro == "" {
			guardados++
		}
	}
	if guardados == 0 && !p.usa(fonteSites) {
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

var nomesCurtos = map[string]string{fonteLivros: "Livros", fonteWikipedia: "Wikipédia", fonteYoutube: "YouTube"}

// materialColetado é o cru.md da pesquisa: o que foi buscado e o que veio de cada fonte.
func materialColetado(p pedidoPesquisa) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# Material coletado\n\n> Pesquisa: %s\n\n## Perguntas\n\n", p.Tema)
	for i, q := range p.Esboco {
		fmt.Fprintf(&s, "%d. %s\n", i+1, q.Pergunta)
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
		fmt.Fprintf(&s, "- **%s:** %s — `%s`\n", nomesCurtos[c.Fonte], titulo, c.Arquivo)
	}
	if p.usa(fonteSites) {
		s.WriteString("- **Sites:** a IA pesquisa ao escrever e guarda em `fontes/web-*.md` o que usar.\n")
	}
	var semNada []string
	for _, c := range p.Coleta {
		if c.Erro != "" {
			semNada = append(semNada, fmt.Sprintf("- %s, %s: %s", nomesCurtos[c.Fonte], c.Titulo, c.Erro))
		}
	}
	if len(semNada) > 0 {
		s.WriteString("\n## Sem resultado\n\n" + strings.Join(semNada, "\n") + "\n")
	}
	return s.String()
}

// ---------- Livros (a busca da Biblioteca, R32) ----------

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

func (a *app) coletarLivros(ctx context.Context, pasta string, n int, q pergunta, limite int) coletado {
	c := coletado{Fonte: fonteLivros, Titulo: fmt.Sprintf("busca %q", q.Livros)}
	if a.cfg.biblioteca == "" {
		c.Erro = "a busca nos livros não está configurada (BIBLIOTECA_URL)"
		return c
	}
	var resposta struct {
		Results []trechoDeLivro `json:"results"`
	}
	endereco := fmt.Sprintf("%s/api/search?limit=%d&q=%s", a.cfg.biblioteca, limite, url.QueryEscape(q.Livros))
	if err := buscarJSON(ctx, endereco, &resposta); err != nil {
		c.Erro = "a Biblioteca não respondeu: " + err.Error()
		return c
	}
	if len(resposta.Results) == 0 {
		c.Erro = "nenhum trecho nos livros"
		return c
	}
	var s strings.Builder
	fmt.Fprintf(&s, "# Trechos dos livros da Biblioteca\n\n- Pergunta: %s\n- Busca: %s\n\n", q.Pergunta, q.Livros)
	livros := map[string]bool{}
	for _, t := range resposta.Results {
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
	c.Arquivo = fmt.Sprintf("%s/livros-%d.md", pastaFontes, n)
	c.Titulo = fmt.Sprintf("%d trechos de %d livro(s) para %q", len(resposta.Results), len(livros), q.Livros)
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), []byte(s.String()), 0o644); err != nil {
		c.Erro = err.Error()
	}
	return c
}

// ---------- Wikipédia ----------

func (a *app) coletarWikipedia(ctx context.Context, pasta, busca string, limite int, vistos map[string]bool) []coletado {
	var achados struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	endereco := a.cfg.wikipedia + "/w/api.php?action=query&list=search&format=json&srnamespace=0&srlimit=" +
		fmt.Sprint(limite+2) + "&srsearch=" + url.QueryEscape(busca)
	if err := buscarJSON(ctx, endereco, &achados); err != nil {
		return []coletado{{Fonte: fonteWikipedia, Titulo: fmt.Sprintf("busca %q", busca), Erro: err.Error()}}
	}
	var coleta []coletado
	for _, r := range achados.Query.Search {
		if len(coleta) == limite {
			break
		}
		if vistos["wiki:"+r.Title] {
			continue
		}
		vistos["wiki:"+r.Title] = true
		coleta = append(coleta, a.guardarArtigo(ctx, pasta, r.Title))
	}
	if len(coleta) == 0 {
		return []coletado{{Fonte: fonteWikipedia, Titulo: fmt.Sprintf("busca %q", busca), Erro: "nenhum artigo novo"}}
	}
	return coleta
}

func (a *app) guardarArtigo(ctx context.Context, pasta, titulo string) coletado {
	link := a.cfg.wikipedia + "/wiki/" + url.PathEscape(strings.ReplaceAll(titulo, " ", "_"))
	c := coletado{Fonte: fonteWikipedia, Titulo: titulo, URL: link}
	var resposta struct {
		Query struct {
			Pages map[string]struct {
				Title   string `json:"title"`
				Extract string `json:"extract"`
			} `json:"pages"`
		} `json:"query"`
	}
	endereco := a.cfg.wikipedia + "/w/api.php?action=query&prop=extracts&explaintext=1&redirects=1&format=json&titles=" + url.QueryEscape(titulo)
	if err := buscarJSON(ctx, endereco, &resposta); err != nil {
		c.Erro = err.Error()
		return c
	}
	texto := ""
	for _, pg := range resposta.Query.Pages {
		texto = pg.Extract
	}
	if strings.TrimSpace(texto) == "" {
		c.Erro = "artigo sem texto"
		return c
	}
	c.Arquivo = fmt.Sprintf("%s/wikipedia-%s.md", pastaFontes, slug(titulo))
	conteudo := fmt.Sprintf("# %s (Wikipédia)\n\nURL: %s\n\n%s\n", titulo, link, cortar(texto, maxArtigo, false))
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

// coletarVideos busca no YouTube pelo yt-dlp e guarda a transcrição dos primeiros vídeos com
// legenda e de 2 a 90 minutos. O vídeo serve só de fonte: não vira Referência própria.
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
		if !idDeVideo.MatchString(v.ID) || vistos["yt:"+v.ID] || (v.Duracao > 0 && (v.Duracao < minDuracaoVideo || v.Duracao > maxDuracaoVideo)) {
			continue
		}
		vistos["yt:"+v.ID] = true
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
	if canal := info.Canal; canal != "" {
		titulo += " (" + canal + ")"
	}
	c := coletado{Fonte: fonteYoutube, Titulo: titulo, URL: link, Arquivo: fmt.Sprintf("%s/video-%s.md", pastaFontes, v.ID)}
	if err := os.WriteFile(filepath.Join(pasta, c.Arquivo), cru, 0o644); err != nil {
		return coletado{}, false
	}
	return c, true
}
