package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Virar Referência (D54): o que um "Mais informações" achou (a explicação e as fontes) vira uma
// Referência própria, do tipo "pesquisa", já organizada: segue o caminho de sempre (sugerir o Tema,
// criar ou integrar, revisão pelo diff) e pode ser aprofundada como qualquer pesquisa. A Referência
// guarda de onde veio (o texto do Acervo ou o livro da Biblioteca, com o trecho), com o link de volta.
// Sem IA: o texto é montado aqui, com o que a pesquisa já trouxe.

type origemDePesquisa struct {
	Tipo     string `json:"tipo"` // referencia, tema ou livro
	Titulo   string `json:"titulo"`
	Autor    string `json:"autor,omitempty"`
	Capitulo string `json:"capitulo,omitempty"`
	Link     string `json:"link"` // #/referencia/<id>, #/ler/<arquivo> ou o endereço da Biblioteca
	Trecho   string `json:"trecho"`
	Detalhe  string `json:"detalhe"` // a marcação de onde veio
}

const maxTituloDeReferencia = 90

// tituloDoTrecho: o trecho pesquisado, numa linha, sem aspas nem pontuação nas pontas, até ~90
// caracteres (cortado entre palavras), com a primeira letra maiúscula.
func tituloDoTrecho(trecho string) string {
	t := strings.TrimSpace(espacos.ReplaceAllString(trecho, " "))
	t = strings.TrimFunc(t, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(`"'“”‘’«».,;:!?…()[]—–-`, r)
	})
	if utf8.RuneCountInString(t) > maxTituloDeReferencia {
		r := []rune(t)[:maxTituloDeReferencia]
		if i := strings.LastIndex(string(r), " "); i > maxTituloDeReferencia/2 {
			t = string(r)[:i]
		} else {
			t = string(r)
		}
		t = strings.TrimRight(t, " ,.;:—–-") + "…"
	}
	if t == "" {
		return "Mais informações"
	}
	r, n := utf8.DecodeRuneInString(t)
	return string(unicode.ToUpper(r)) + t[n:]
}

var tituloMarkdown = regexp.MustCompile(`(?m)^#\s`)
var secaoFontes = regexp.MustCompile(`(?mi)^#{1,3}\s*(fontes|referências)\b`)

// textoDeDetalhe monta o organizado.md da Referência nova.
func textoDeDetalhe(titulo string, d marcacao, o origemDePesquisa) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", titulo)
	de := o.Titulo
	if o.Autor != "" {
		de += ", de " + o.Autor
	}
	if o.Capitulo != "" {
		de += " (" + o.Capitulo + ")"
	}
	trecho := strings.TrimSpace(espacos.ReplaceAllString(d.Trecho, " "))
	if o.Link != "" {
		fmt.Fprintf(&b, "> Mais informações sobre o trecho “%s”, de [%s](%s).\n\n", trecho, de, o.Link)
	} else {
		fmt.Fprintf(&b, "> Mais informações sobre o trecho “%s”, de %s.\n\n", trecho, de)
	}
	// Um título de nível 1 na explicação vira seção: o da Referência é o único.
	b.WriteString(tituloMarkdown.ReplaceAllString(strings.TrimSpace(d.Explicacao), "## "))
	b.WriteString("\n")
	if len(d.Fontes) > 0 && !secaoFontes.MatchString(d.Explicacao) {
		b.WriteString("\n## Fontes\n\n")
		for i, f := range d.Fontes {
			nome := strings.TrimSpace(f.Titulo)
			if nome == "" {
				nome = f.URL
			}
			fmt.Fprintf(&b, "%d. [%s](%s)\n", i+1, strings.ReplaceAll(nome, "]", ")"), f.URL)
		}
	}
	return b.String()
}

// referenciasDeDetalhes: marcação → a Referência em que ela virou.
func (a *app) referenciasDeDetalhes(ctx context.Context) (map[string]string, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT json_extract(pesquisa, '$.origem.detalhe'), id FROM referencias
		WHERE tipo = 'pesquisa' AND json_extract(pesquisa, '$.origem.detalhe') IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var d, id string
		if err := rows.Scan(&d, &id); err != nil {
			return nil, err
		}
		m[d] = id
	}
	return m, rows.Err()
}

// handleDetalhe: GET /api/detalhes/{id}, uma marcação (a Biblioteca acompanha a pesquisa dela).
func (a *app) handleDetalhe(w http.ResponseWriter, r *http.Request) {
	d, err := a.detalhe(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Marcação não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	if viraram, err := a.referenciasDeDetalhes(r.Context()); err == nil {
		d.Referencia = viraram[d.ID]
	}
	writeJSON(w, http.StatusOK, d)
}

func linkDaWebValido(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

// handleVirarReferencia: POST /api/detalhes/{id}/referencia {titulo?, origem?}. A origem do corpo
// vale só para o livro da Biblioteca (título, autor, capítulo e o link de volta); no Acervo, ela sai
// da própria marcação. Pedir de novo devolve a mesma Referência.
func (a *app) handleVirarReferencia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d, err := a.detalhe(ctx, r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Marcação não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	if d.Estado != "pronto" {
		erroJSON(w, http.StatusConflict, "A pesquisa deste trecho ainda não terminou.")
		return
	}
	var corpo struct {
		Titulo string           `json:"titulo"`
		Origem origemDePesquisa `json:"origem"`
	}
	if r.ContentLength != 0 && !lerCorpo(w, r, &corpo) {
		return
	}
	viraram, err := a.referenciasDeDetalhes(ctx)
	if err != nil {
		falha(w, err)
		return
	}
	if id, ok := viraram[d.ID]; ok {
		if ref, err := a.referencia(ctx, id); err == nil {
			a.completar(ctx, []*referencia{ref})
			writeJSON(w, http.StatusOK, map[string]any{"referencia": ref, "link": a.cfg.acervoLink + "/#/referencia/" + id, "ja_existia": true})
			return
		}
	}

	o := origemDePesquisa{Titulo: d.Titulo, Trecho: d.Trecho, Detalhe: d.ID}
	partes := strings.Split(d.Local, "/")
	switch partes[0] {
	case "referencias":
		o.Tipo, o.Link = "referencia", "#/referencia/"+partes[1]
	case "temas":
		o.Tipo, o.Link = "tema", "#/ler/"+d.Local
	case "livros":
		o.Tipo, o.Link = "livro", a.cfg.bibliotecaLink+"/#/livro/"+partes[1]
		c := corpo.Origem
		if t := strings.TrimSpace(c.Titulo); t != "" {
			o.Titulo = cortar(t, 200, false)
		}
		o.Autor = cortar(strings.TrimSpace(c.Autor), 200, false)
		o.Capitulo = cortar(strings.TrimSpace(c.Capitulo), 200, false)
		if linkDaWebValido(c.Link) {
			o.Link = c.Link
		}
	}
	if o.Titulo == "" {
		o.Titulo = "o texto"
	}
	titulo := strings.TrimSpace(corpo.Titulo)
	if titulo == "" {
		titulo = tituloDoTrecho(d.Trecho)
	}
	titulo = cortar(titulo, 120, false)

	p := pedidoPesquisa{
		Tema:         d.Trecho,
		Fontes:       []string{fonteSites},
		Profundidade: profRapida,
		Esboco:       []pergunta{},
		Capitulos:    []capituloPesquisa{},
		Paginas:      []paginaPesquisa{},
		Coleta:       []coletado{},
		Rodada:       1,
		Anteriores:   []pergunta{},
		Relacionados: []string{},
		Origem:       &o,
	}
	dados, _ := json.Marshal(p)
	id := novoID()
	pasta := a.pastaReferencia(id)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		falha(w, err)
		return
	}
	if err := os.WriteFile(filepath.Join(pasta, "organizado.md"), []byte(textoDeDetalhe(titulo, d, o)), 0o644); err != nil {
		falha(w, err)
		return
	}
	_, err = a.db.ExecContext(ctx,
		`INSERT INTO referencias (id, url, titulo, idioma, estado, tipo, pesquisa, criado_em, atualizado_em)
		 VALUES (?, '', ?, 'pt', 'organizado', 'pesquisa', ?, ?, ?)`,
		id, titulo, string(dados), agora(), agora())
	if err != nil {
		os.RemoveAll(pasta)
		falha(w, err)
		return
	}
	ref, _ := a.referencia(ctx, id)
	a.completar(ctx, []*referencia{ref})
	writeJSON(w, http.StatusCreated, map[string]any{"referencia": ref, "link": a.cfg.acervoLink + "/#/referencia/" + id, "ja_existia": false})
}
