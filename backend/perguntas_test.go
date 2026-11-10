package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPerguntarSobreUmTema(t *testing.T) {
	e := novoAmbiente(t)
	pasta := filepath.Join(e.app.conteudo.raiz, "temas", "clube")
	os.MkdirAll(filepath.Join(pasta, "base"), 0o755)
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n\n## Fundação\n\nFundado em 1930.\n"), 0o644)
	os.WriteFile(filepath.Join(pasta, "titulos.md"), []byte("# Títulos\n\nCapítulo separado.\n"), 0o644)
	os.WriteFile(filepath.Join(pasta, "base", "indice.md"), []byte("# Subtema\n"), 0o644)

	var p perguntaFeita
	pedido := map[string]string{"local": "temas/clube", "titulo": "Clube", "pergunta": "  Quando foi fundado?  "}
	if s := e.pedir("POST", "/api/perguntas", pedido, &p); s != http.StatusCreated || p.Estado != "processando" || p.Pergunta != "Quando foi fundado?" {
		t.Fatalf("POST: %d %+v", s, p)
	}
	lista := func() []perguntaFeita {
		var l []perguntaFeita
		e.pedir("GET", "/api/perguntas?local="+url.QueryEscape("temas/clube"), nil, &l)
		return l
	}
	e.esperar("resposta pronta", func() bool {
		l := lista()
		if len(l) == 1 {
			p = l[0]
		}
		return p.Estado == "pronto"
	})
	r := p.Resultado
	// Lê a página e o capítulo separado, sem o subtema; pesquisa na internet, sem escrever.
	if r == nil || r.Cobertura != "em_parte" || !strings.Contains(r.Resposta, "Leu: indice.md titulos.md.") ||
		!strings.Contains(r.Resposta, "Ferramentas: Read,Glob,Grep,WebSearch,WebFetch.") {
		t.Fatalf("pronto: %+v", r)
	}
	if len(r.Divergencias) != 1 || r.Divergencias[0].Tipo != "omissao" || len(r.Divergencias[0].Fontes) != 2 || len(r.Fontes) != 2 {
		t.Fatalf("divergências: %+v", r)
	}

	var tarefas []tarefaComTitulo
	e.pedir("GET", "/api/tarefas", nil, &tarefas)
	if len(tarefas) != 1 || tarefas[0].Referencia != "“Quando foi fundado?”" || tarefas[0].Link != "#/ler/temas/clube/indice.md" {
		t.Fatalf("fila: %+v", tarefas)
	}

	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/refazer", nil, &p); s != http.StatusOK || p.Estado != "processando" || p.Resultado != nil {
		t.Fatalf("refazer: %d %+v", s, p)
	}
	e.esperar("resposta refeita", func() bool { l := lista(); return len(l) == 1 && l[0].Estado == "pronto" })

	for _, ruim := range []map[string]string{
		{"local": "temas/nao-existe", "pergunta": "x"},
		{"local": "temas/../segredo", "pergunta": "x"},
		{"local": "referencias/zzz", "pergunta": "x"},
		{"local": "temas/clube", "pergunta": "   "},
		{"local": "temas/clube", "pergunta": strings.Repeat("a", maxPergunta+1)},
	} {
		if s := e.pedir("POST", "/api/perguntas", ruim, nil); s != http.StatusBadRequest {
			t.Errorf("%v: %d", ruim, s)
		}
	}

	if s := e.pedir("DELETE", "/api/perguntas/"+p.ID, nil, nil); s != http.StatusNoContent || len(lista()) != 0 {
		t.Fatalf("apagar: %d", s)
	}
}

func TestPerguntaQueFalha(t *testing.T) {
	e := novoAmbiente(t)
	pasta := filepath.Join(e.app.conteudo.raiz, "temas", "clube")
	os.MkdirAll(pasta, 0o755)
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n"), 0o644)
	t.Setenv("CLAUDE_FALSO_FALHA", "1")

	var p perguntaFeita
	e.pedir("POST", "/api/perguntas", map[string]string{"local": "temas/clube", "pergunta": "Quando?"}, &p)
	e.esperar("falha visível", func() bool {
		var l []perguntaFeita
		e.pedir("GET", "/api/perguntas?local=temas%2Fclube", nil, &l)
		if len(l) == 1 {
			p = l[0]
		}
		return p.Estado == "falhou"
	})
	if !strings.Contains(p.Erro, "limite de uso") {
		t.Errorf("erro: %q", p.Erro)
	}
}
