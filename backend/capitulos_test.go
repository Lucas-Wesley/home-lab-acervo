package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecoes(t *testing.T) {
	texto := "# T\n\nIntro.\n\n## Um\n\nA\n\n```\n## não é capítulo\n```\n\n## Dois\n\nB\n\n## Linha do tempo\n\n- x\n"
	s := secoes(texto)
	if len(s) != 3 || s[0].Titulo != "Um" || s[1].Titulo != "Dois" {
		t.Fatalf("%+v", s)
	}
	if !strings.Contains(texto[s[0].ini:s[0].fim], "## não é capítulo") || !strings.HasPrefix(texto[s[1].ini:], "## Dois") {
		t.Errorf("limites errados: %q", texto[s[0].ini:s[0].fim])
	}
	if contarCapitulos(texto) != 2 {
		t.Errorf("capítulos: %d", contarCapitulos(texto))
	}
}

// temaPronto cria um Tema pelo fluxo normal (claude falso) e devolve o ambiente.
func temaPronto(t *testing.T) *ambiente {
	e := novoAmbiente(t)
	id := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	e.pedir("POST", "/api/referencias/"+id+"/criar-tema", map[string]string{"titulo": "Tema de Teste"}, nil)
	var revisoes []tarefaComTitulo
	e.esperar("revisão", func() bool {
		e.pedir("GET", "/api/revisoes", nil, &revisoes)
		return len(revisoes) == 1
	})
	e.pedir("POST", "/api/revisoes/"+itoa(revisoes[0].ID)+"/aprovar", nil, nil)
	return e
}

func (e *ambiente) indice() arquivo {
	var a arquivo
	e.pedir("GET", "/api/arquivos/temas/tema-de-teste/indice.md", nil, &a)
	return a
}

func TestSepararEJuntarCapitulo(t *testing.T) {
	e := temaPronto(t)
	antes := e.indice()

	// Separar: o texto vai para um arquivo; na página fica o título com o link.
	var r struct{ Arquivo, Erro string }
	if s := e.pedir("POST", "/api/capitulos/separar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": "Primeiro capítulo", "base": antes.Hash}, &r); s != http.StatusOK {
		t.Fatalf("separar: %d %s", s, r.Erro)
	}
	if r.Arquivo != "temas/tema-de-teste/primeiro-capitulo.md" {
		t.Errorf("arquivo: %q", r.Arquivo)
	}
	depois := e.indice()
	if !strings.Contains(depois.Conteudo, "## Primeiro capítulo\n\n[Ler o capítulo →](primeiro-capitulo.md)\n") || strings.Contains(depois.Conteudo, "Texto do primeiro capítulo") {
		t.Errorf("página depois de separar:\n%s", depois.Conteudo)
	}
	var cap arquivo
	e.pedir("GET", "/api/arquivos/"+r.Arquivo, nil, &cap)
	if !strings.HasPrefix(cap.Conteudo, "# Primeiro capítulo\n\nTexto do primeiro capítulo.") || len(cap.Fontes) != 1 {
		t.Errorf("capítulo separado:\n%s", cap.Conteudo)
	}
	// O capítulo seguinte e os apêndices ficam como estavam.
	if !strings.Contains(depois.Conteudo, "## Segundo capítulo\n\nTexto do segundo capítulo.") || !strings.Contains(depois.Conteudo, "## Linha do tempo") {
		t.Errorf("o resto da página mudou:\n%s", depois.Conteudo)
	}
	var temas []tema
	e.pedir("GET", "/api/temas", nil, &temas)
	if temas[0].Capitulos != 2 {
		t.Errorf("um capítulo separado continua contando: %d", temas[0].Capitulos)
	}

	// Versão velha e capítulo já separado são recusados.
	if s := e.pedir("POST", "/api/capitulos/separar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": "Segundo capítulo", "base": antes.Hash}, nil); s != http.StatusConflict {
		t.Errorf("base velha: %d", s)
	}
	if s := e.pedir("POST", "/api/capitulos/separar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": "Primeiro capítulo", "base": depois.Hash}, nil); s != http.StatusConflict {
		t.Errorf("separar de novo: %d", s)
	}

	// Citado em outro Tema: não dá para juntar sem tirar o link.
	e.pedir("POST", "/api/temas", map[string]string{"titulo": "Outro"}, nil)
	var outro arquivo
	e.pedir("GET", "/api/arquivos/temas/outro/indice.md", nil, &outro)
	e.pedir("PUT", "/api/arquivos/temas/outro/indice.md", map[string]string{"conteudo": outro.Conteudo + "[cap](../tema-de-teste/primeiro-capitulo.md)\n", "base": outro.Hash}, nil)
	if s := e.pedir("POST", "/api/capitulos/juntar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": "Primeiro capítulo", "base": depois.Hash}, &r); s != http.StatusConflict || !strings.Contains(r.Erro, "citado em") {
		t.Errorf("juntar citado: %d %q", s, r.Erro)
	}
	e.pedir("GET", "/api/arquivos/temas/outro/indice.md", nil, &outro)
	e.pedir("PUT", "/api/arquivos/temas/outro/indice.md", map[string]string{"conteudo": "# Outro\n", "base": outro.Hash}, nil)

	// Juntar: volta ao texto original e o arquivo some.
	if s := e.pedir("POST", "/api/capitulos/juntar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": "Primeiro capítulo", "base": depois.Hash}, &r); s != http.StatusOK {
		t.Fatalf("juntar: %d %s", s, r.Erro)
	}
	if final := e.indice(); final.Conteudo != antes.Conteudo {
		t.Errorf("depois de separar e juntar, a página devia ser a mesma:\n%s\n---\n%s", final.Conteudo, antes.Conteudo)
	}
	if existe(filepath.Join(e.app.conteudo.raiz, "temas/tema-de-teste/primeiro-capitulo.md")) {
		t.Error("o arquivo do capítulo ficou")
	}
	if log := git(t, e.app.conteudo.raiz, "log", "--oneline"); !strings.Contains(log, "Capítulo em página própria: Primeiro capítulo") || !strings.Contains(log, "Capítulo de volta à página") {
		t.Errorf("log: %s", log)
	}
	if st := git(t, e.app.conteudo.raiz, "status", "--porcelain"); st != "" {
		t.Errorf("sobrou mudança sem commit: %s", st)
	}
}
