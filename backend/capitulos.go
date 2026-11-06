package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Um Tema é, por padrão, uma página só: cada "## " do indice.md é um capítulo (D23). Quando um
// capítulo cresce, o Lucas o separa num arquivo próprio; na página fica o título e um link.

type secao struct {
	Titulo   string
	ini, fim int // posição no texto: do "## " até antes do próximo "## " (ou o fim)
}

// secoes acha os capítulos (títulos "## "), ignorando o que estiver dentro de blocos de código.
func secoes(texto string) []secao {
	var lista []secao
	emCodigo := false
	pos := 0
	for _, linha := range strings.SplitAfter(texto, "\n") {
		limpa := strings.TrimRight(linha, "\r\n")
		if strings.HasPrefix(limpa, "```") {
			emCodigo = !emCodigo
		}
		if !emCodigo && strings.HasPrefix(limpa, "## ") {
			if n := len(lista); n > 0 {
				lista[n-1].fim = pos
			}
			lista = append(lista, secao{Titulo: strings.TrimSpace(limpa[3:]), ini: pos})
		}
		pos += len(linha)
	}
	if n := len(lista); n > 0 {
		lista[n-1].fim = len(texto)
	}
	return lista
}

// Seções de apoio do Tema; não contam como capítulos.
var apendices = map[string]bool{"Linha do tempo": true, "Citados": true, "Divergências": true, "Referências": true}

func contarCapitulos(texto string) int {
	n := 0
	for _, s := range secoes(texto) {
		if !apendices[s.Titulo] {
			n++
		}
	}
	return n
}

var linkDeCapitulo = regexp.MustCompile(`^\[[^\]]*\]\(([^)\s#/]+\.md)\)$`)

// separado diz se o corpo do capítulo é só o link para o arquivo dele.
func separado(corpo string) (string, bool) {
	m := linkDeCapitulo.FindStringSubmatch(strings.TrimSpace(corpo))
	if m == nil {
		return "", false
	}
	return m[1], true
}

const rodapeCapitulo = "\n\n---\n[← Voltar ao Tema](indice.md)\n"

var rodapeQualquer = regexp.MustCompile(`\n+---\n\[← Voltar[^\]]*\]\([^)]*\)\s*$`)

var (
	errSemCapitulo = errors.New("capítulo não encontrado")
	errJaSeparado  = errors.New("este capítulo já está numa página própria")
	errNaoSeparado = errors.New("este capítulo já está na página principal")
	errSoNoIndice  = errors.New("só capítulos do indice.md de um Tema podem ser separados")
)

func acharSecao(texto, titulo string) (secao, string, error) {
	for _, s := range secoes(texto) {
		if s.Titulo == titulo {
			corpo := texto[s.ini:s.fim]
			corpo = corpo[strings.Index(corpo, "\n")+1:]
			return s, corpo, nil
		}
	}
	return secao{}, "", errSemCapitulo
}

// lerParaMudar confere a versão (hash) como na edição manual e devolve o texto atual.
func (c *conteudo) lerParaMudar(rel, base string) (string, error) {
	if path.Base(rel) != "indice.md" {
		return "", errSoNoIndice
	}
	atual, err := os.ReadFile(filepath.Join(c.raiz, filepath.FromSlash(rel)))
	if err != nil {
		return "", errNaoEncontrado
	}
	if hashDe(atual) != base {
		a, _ := c.ler(rel)
		return "", &conflito{Atual: a}
	}
	return string(atual), nil
}

// separarCapitulo move o texto do capítulo para <slug>.md e deixa o título com um link.
func (c *conteudo) separarCapitulo(ctx context.Context, rel, titulo, base string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	texto, err := c.lerParaMudar(rel, base)
	if err != nil {
		return "", err
	}
	s, corpo, err := acharSecao(texto, titulo)
	if err != nil {
		return "", err
	}
	if _, ja := separado(corpo); ja {
		return "", errJaSeparado
	}
	dir := path.Dir(rel)
	nome := slug(titulo)
	if nome == "" || nome == "indice" {
		nome = "capitulo"
	}
	for i := 2; existe(filepath.Join(c.raiz, dir, nome+".md")); i++ {
		nome = fmt.Sprintf("%s-%d", slug(titulo), i)
	}
	arquivo := path.Join(dir, nome+".md")

	pagina := "# " + titulo + "\n\n" + strings.TrimSpace(corpo) + rodapeCapitulo
	novo := texto[:s.ini] + "## " + titulo + "\n\n[Ler o capítulo →](" + nome + ".md)\n\n" + texto[s.fim:]
	if err := os.WriteFile(filepath.Join(c.raiz, filepath.FromSlash(arquivo)), []byte(pagina), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(c.raiz, filepath.FromSlash(rel)), []byte(novo), 0o644); err != nil {
		return "", err
	}
	return arquivo, c.commit(ctx, "Capítulo em página própria: "+titulo, rel, arquivo)
}

// juntarCapitulo traz o texto de volta para a página principal e apaga o arquivo.
func (c *conteudo) juntarCapitulo(ctx context.Context, rel, titulo, base string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	texto, err := c.lerParaMudar(rel, base)
	if err != nil {
		return err
	}
	s, corpo, err := acharSecao(texto, titulo)
	if err != nil {
		return err
	}
	nome, ok := separado(corpo)
	if !ok {
		return errNaoSeparado
	}
	arquivo := path.Join(path.Dir(rel), nome)
	citacoes, err := c.citacoes(arquivo)
	if err != nil {
		return err
	}
	if len(citacoes) > 0 {
		return fmt.Errorf("o capítulo é citado em %s; tire esse link antes de trazê-lo de volta", citacoes[0].Titulo)
	}
	pagina, err := os.ReadFile(filepath.Join(c.raiz, filepath.FromSlash(arquivo)))
	if err != nil {
		return errNaoEncontrado
	}
	conteudo := rodapeQualquer.ReplaceAllString(string(pagina), "")
	if strings.HasPrefix(conteudo, "# ") {
		conteudo = conteudo[strings.Index(conteudo+"\n", "\n")+1:]
	}
	novo := texto[:s.ini] + "## " + titulo + "\n\n" + strings.TrimSpace(conteudo) + "\n\n" + texto[s.fim:]
	if err := os.WriteFile(filepath.Join(c.raiz, filepath.FromSlash(rel)), []byte(novo), 0o644); err != nil {
		return err
	}
	if _, err := c.git(ctx, c.raiz, "rm", "-q", "--", arquivo); err != nil {
		return err
	}
	return c.commit(ctx, "Capítulo de volta à página: "+titulo, rel)
}
