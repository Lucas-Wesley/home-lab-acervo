package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (e *ambiente) conversaPronta(local string, corpo map[string]string) perguntaFeita {
	e.t.Helper()
	corpo["local"] = local
	var p perguntaFeita
	if s := e.pedir("POST", "/api/perguntas", corpo, &p); s != http.StatusCreated || p.Modo != modoConversa {
		e.t.Fatalf("conversar: %d %+v", s, p)
	}
	e.esperar("conversa pronta", func() bool {
		var l []perguntaFeita
		e.pedir("GET", "/api/perguntas?local="+url.QueryEscape(local), nil, &l)
		for _, q := range l {
			if q.ID == p.ID {
				p = q
			}
		}
		return p.Estado == "pronto"
	})
	return p
}

func TestConversarSobreUmTema(t *testing.T) {
	e := novoAmbiente(t)
	pasta := filepath.Join(e.app.conteudo.raiz, "temas", "clube")
	os.MkdirAll(pasta, 0o755)
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n\n## Fundação\n\nFundado em 1930.\n"), 0o644)
	os.WriteFile(filepath.Join(pasta, "titulos.md"), []byte("# Títulos\n"), 0o644)
	e.app.conteudo.commit(t.Context(), "Tema de teste", "temas/clube")

	primeira := e.conversaPronta("temas/clube", map[string]string{"titulo": "Clube", "pergunta": "PRIMEIRA MENSAGEM"})
	p := e.conversaPronta("temas/clube", map[string]string{"titulo": "Clube", "pergunta": "Onde foi fundado?",
		"trecho": "Fundado em 1930.", "arquivo": "indice.md"})
	c := p.Conversa
	if c == nil || !strings.Contains(c.Resposta, "Leu: indice.md titulos.md.") || !strings.Contains(c.Resposta, "Ferramentas: Read,Glob,Grep,WebSearch,WebFetch.") ||
		!strings.Contains(c.Resposta, "Lembrou da conversa.") || !strings.Contains(c.Resposta, "Recebeu o trecho.") ||
		!strings.Contains(c.Resposta, "Recebeu o mapa do Acervo.") || p.Trecho != "Fundado em 1930." || p.Arquivo != "indice.md" {
		t.Fatalf("resposta: %+v / %+v", p, c)
	}
	// Fica a alteração que se acha no texto, e o lugar do Acervo que existe; o resto vira nota.
	if len(c.Alteracoes) != 1 || c.Alteracoes[0].Estado != "proposta" || c.Alteracoes[0].Arquivo != "indice.md" ||
		!strings.Contains(c.Resposta, "1 alteração proposta não pôde ser localizada") {
		t.Fatalf("alterações: %+v", c.Alteracoes)
	}
	if len(c.NoAcervo) != 1 || c.NoAcervo[0].Caminho != "temas/clube" || !c.Pesquisou || len(c.Fontes) != 1 {
		t.Fatalf("no Acervo: %+v", c)
	}
	if primeira.Conversa == nil || strings.Contains(primeira.Conversa.Resposta, "Lembrou") {
		t.Errorf("a primeira não tinha memória: %+v", primeira.Conversa)
	}

	// Aplicar muda o arquivo e faz commit; não se aplica duas vezes.
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/alteracoes/0/aplicar", nil, &p); s != http.StatusOK || p.Conversa.Alteracoes[0].Estado != "aplicada" {
		t.Fatalf("aplicar: %d %+v", s, p.Conversa)
	}
	texto, _ := os.ReadFile(filepath.Join(pasta, "indice.md"))
	if !strings.Contains(string(texto), "Fundado em 1930, no Recife.") {
		t.Fatalf("texto: %s", texto)
	}
	if log, _ := e.app.conteudo.git(t.Context(), e.app.conteudo.raiz, "log", "-1", "--format=%s"); !strings.Contains(log, "Conversar: Cidade da fundação") {
		t.Errorf("commit: %q", log)
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/alteracoes/0/aplicar", nil, nil); s != http.StatusConflict {
		t.Errorf("aplicar de novo: %d", s)
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/alteracoes/5/aplicar", nil, nil); s != http.StatusNotFound {
		t.Errorf("alteração que não existe: %d", s)
	}

	// Outra resposta, e o texto muda antes de aplicar: aplicar recusa; descartar vale.
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n\n## Fundação\n\nFundado em 1930.\n"), 0o644)
	q := e.conversaPronta("temas/clube", map[string]string{"pergunta": "Mais uma"})
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n\nMudou tudo.\n"), 0o644)
	if s := e.pedir("POST", "/api/perguntas/"+q.ID+"/alteracoes/0/aplicar", nil, nil); s != http.StatusConflict {
		t.Errorf("trecho que mudou: %d", s)
	}
	if s := e.pedir("POST", "/api/perguntas/"+q.ID+"/alteracoes/0/descartar", nil, &q); s != http.StatusOK || q.Conversa.Alteracoes[0].Estado != "descartada" {
		t.Errorf("descartar: %d %+v", s, q.Conversa)
	}
}

// Numa Referência, aplicar troca o trecho no texto, e o de antes fica como Versão anterior.
func TestConversarSobreUmaReferencia(t *testing.T) {
	e := novoAmbiente(t)
	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	arquivo := filepath.Join(e.app.pastaReferencia(id), "organizado.md")
	os.WriteFile(arquivo, []byte("# Vídeo sobre a história do clube\n\nFundado em 1930.\n"), 0o644)

	p := e.conversaPronta("referencias/"+id, map[string]string{"pergunta": "Onde?", "arquivo": "ignorado.md"})
	if p.Arquivo != "" || len(p.Conversa.Alteracoes) != 1 || p.Conversa.Alteracoes[0].Arquivo != "texto.md" {
		t.Fatalf("conversa: %+v / %+v", p, p.Conversa)
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/alteracoes/0/aplicar", nil, &p); s != http.StatusOK {
		t.Fatalf("aplicar: %d", s)
	}
	texto, _ := os.ReadFile(arquivo)
	anterior, _ := os.ReadFile(filepath.Join(e.app.pastaReferencia(id), "anterior.md"))
	if string(texto) != "# Vídeo sobre a história do clube\n\nFundado em 1930, no Recife.\n" || string(anterior) != "# Vídeo sobre a história do clube\n\nFundado em 1930.\n" {
		t.Fatalf("texto: %q, anterior: %q", texto, anterior)
	}
}
