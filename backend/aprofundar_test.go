package main

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAprofundarUmaPesquisa(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)
	// O Acervo já tem um Tema sobre o assunto: a IA o aponta no esboço e ele vai para ja-escrito/.
	tema := filepath.Join(e.app.conteudo.raiz, "temas", "laranja")
	os.MkdirAll(tema, 0o755)
	os.WriteFile(filepath.Join(tema, "indice.md"), []byte("---\ntitulo: Laranja\n---\n\n# Laranja\n\nO que já está escrito.\n"), 0o644)

	id := e.pesquisar(map[string]any{"tema": "Qual é a origem da laranja?", "fontes": []string{"wikipedia"}, "profundidade": "rapida"})
	pronta := func() detalhe {
		var d detalhe
		e.esperar("pesquisa pronta", func() bool {
			d = e.referencia(id)
			return d.Estado == "organizado" && string(d.Sugestao) != "null" && d.Tarefa == nil
		})
		return d
	}
	d := pronta()
	p := pedidoDaResposta(t, d)
	if p.Rodada != 1 || !slices.Equal(p.Relacionados, []string{"temas/laranja"}) {
		t.Fatalf("rodada 1: %+v", p)
	}
	if !strings.Contains(d.Textos["organizado"], "já escrito: temas-laranja--indice.md") || strings.Contains(d.Textos["organizado"], "anterior.md") {
		t.Errorf("rodada 1 sem o já escrito:\n%s", d.Textos["organizado"])
	}
	textoDaRodada1 := d.Textos["organizado"]

	// Aprofundar: mais o YouTube, e o que ele quer a mais.
	if s := e.pedir("POST", "/api/referencias/"+id+"/aprofundar", map[string]any{
		"fontes": []string{"youtube", "wikipedia"}, "profundidade": "rapida", "foco": "as variedades",
	}, nil); s != http.StatusOK {
		t.Fatalf("aprofundar: %d", s)
	}
	if s := e.pedir("POST", "/api/referencias/"+id+"/aprofundar", map[string]any{"fontes": []string{}, "profundidade": "rapida"}, nil); s != http.StatusConflict {
		t.Errorf("aprofundar com a rodada andando: %d", s)
	}
	d = pronta()
	p = pedidoDaResposta(t, d)
	if p.Rodada != 2 || p.Foco != "as variedades" || len(p.Anteriores) != 2 || !slices.Equal(p.Fontes, []string{"wikipedia", "youtube"}) {
		t.Fatalf("rodada 2: %+v", p)
	}
	if d.Textos["anterior"] != textoDaRodada1 {
		t.Errorf("anterior.md não é o texto da rodada 1:\n%s", d.Textos["anterior"])
	}
	org := d.Textos["organizado"]
	for _, quer := range []string{"Partiu de anterior.md.", "Foco recebido.", "já escrito: temas-laranja--indice.md", "wikipedia-laranja-fruta.md", "video-"} {
		if !strings.Contains(org, quer) {
			t.Errorf("rodada 2 sem %q:\n%s", quer, org)
		}
	}
	// O material da rodada 1 ficou e não foi buscado de novo: os artigos já vistos não voltam.
	por := map[int]int{}
	for _, c := range p.Coleta {
		if c.Erro == "" {
			por[c.rodada()]++
		}
		if c.rodada() == 2 && c.Fonte == fonteWikipedia && c.Erro == "" {
			t.Errorf("artigo coletado de novo: %+v", c)
		}
	}
	if por[1] != 2 || por[2] == 0 {
		t.Errorf("coleta por rodada: %v (%+v)", por, p.Coleta)
	}
	if !strings.Contains(d.Textos["cru"], "Das rodadas anteriores:") || !strings.Contains(d.Textos["cru"], "(rodada 1)") {
		t.Errorf("material coletado:\n%s", d.Textos["cru"])
	}
	if d.Titulo != "A origem da laranja" {
		t.Errorf("o título fica o da primeira rodada: %q", d.Titulo)
	}
}

func TestAprofundarSoComTexto(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)
	id := e.pesquisar(map[string]any{"tema": "Laranja", "fontes": []string{"wikipedia"}, "profundidade": "rapida", "revisar_esboco": true})
	e.esperarEstado(id, "esboco")
	if s := e.pedir("POST", "/api/referencias/"+id+"/aprofundar", map[string]any{"fontes": []string{}, "profundidade": "rapida"}, nil); s != http.StatusConflict {
		t.Errorf("aprofundar sem texto: %d", s)
	}
	video := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	e.esperarEstado(video, "organizado")
	if s := e.pedir("POST", "/api/referencias/"+video+"/aprofundar", map[string]any{"fontes": []string{}, "profundidade": "rapida"}, nil); s != http.StatusConflict {
		t.Errorf("aprofundar um vídeo: %d", s)
	}
}
