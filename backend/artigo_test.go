package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const paginaDeTeste = `<!doctype html><html lang="pt-BR"><head>
<title>A origem do café - Blog de História</title>
<meta property="og:site_name" content="Blog de História">
<meta name="author" content="Ana Souza">
<meta property="article:published_time" content="2025-03-04T10:00:00Z">
<script>var x = "não é texto";</script></head>
<body>
<nav><a href="/">Início</a> <a href="/sobre">Sobre</a></nav>
<div class="cookie-banner">Usamos cookies.</div>
<article>
<h1>A origem do café</h1>
<p>O café nasceu na Etiópia, onde, segundo a lenda, um pastor chamado Kaldi viu as cabras agitadas depois de comer os frutos vermelhos de um arbusto. Dali a bebida passou ao Iêmen, no século XV, e se espalhou pelo mundo árabe.</p>
<h2>A chegada ao Brasil</h2>
<p>Em 1727, Francisco de Melo Palheta trouxe as primeiras mudas da Guiana Francesa para o Pará. No século XIX, o Vale do Paraíba virou o centro da produção.</p>
<ul><li>Etiópia</li><li>Iêmen</li><li>Brasil</li></ul>
<div class="share-buttons">Compartilhe no Facebook</div>
<blockquote>O café é o leite dos pensadores.</blockquote>
</article>
<aside>Leia também: o chá</aside>
<section id="comments">Ótimo texto!</section>
<footer>© Blog de História</footer>
</body></html>`

func TestReferenciaDeArtigo(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/cafe":
			fmt.Fprint(w, paginaDeTeste)
		case "/vazia":
			fmt.Fprint(w, `<html><body><div id="app"></div><script>montar()</script></body></html>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()
	e := novoAmbiente(t)

	// O título do "Compartilhar" vem junto; o rastreio e o # saem do link.
	id := e.adicionar("A origem do café " + site.URL + "/cafe?utm_source=x&p=1#topo")
	d := e.esperarEstado(id, "organizado")
	if d.Tipo != "artigo" || d.URL != site.URL+"/cafe?p=1" || d.Titulo != "A origem do café" || d.Canal != "Blog de História" || d.Idioma != "pt-BR" {
		t.Errorf("metadados: %+v", d.referencia)
	}
	cru := d.Textos["cru"]
	for _, deve := range []string{"# A origem do café", "- Site: Blog de História", "- Autor: Ana Souza", "- Publicado: 2025-03-04",
		"Kaldi viu as cabras", "## A chegada ao Brasil", "- Iêmen", "> O café é o leite dos pensadores."} {
		if !strings.Contains(cru, deve) {
			t.Errorf("faltou %q no cru:\n%s", deve, cru)
		}
	}
	for _, nao := range []string{"Início", "cookies", "Compartilhe", "Leia também", "Ótimo texto", "©", "não é texto"} {
		if strings.Contains(cru, nao) {
			t.Errorf("sobra do site %q no cru:\n%s", nao, cru)
		}
	}
	if d.Tarefas[0].Tipo != "organizar" || d.Textos["organizado"] == "" {
		t.Errorf("organizar: %+v", d.Tarefas)
	}

	// A mesma página de novo é recusada, mesmo com outro rastreio.
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": site.URL + "/cafe?p=1&utm_medium=y"}, nil); s != http.StatusConflict {
		t.Errorf("duplicada: %d", s)
	}

	// Ao criar o Tema, a IA sabe que não há marcas de tempo.
	if nota := notaDoTipo(&d.referencia); !strings.Contains(nota, "<sub>Fonte: "+id+"</sub>") {
		t.Errorf("nota do artigo: %q", nota)
	}

	// Página sem texto (montada por JavaScript) e página que não existe: erro visível.
	vazia := e.adicionar(site.URL + "/vazia")
	if d := e.esperarEstado(vazia, "erro"); !strings.Contains(d.Erro, "Não achei o texto") {
		t.Errorf("página vazia: %q", d.Erro)
	}
	sumiu := e.adicionar(site.URL + "/sumiu")
	if d := e.esperarEstado(sumiu, "erro"); !strings.Contains(d.Erro, "404") {
		t.Errorf("página que não existe: %q", d.Erro)
	}
}
