package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestMaisInformacoesSobreUmTrecho(t *testing.T) {
	e := novoAmbiente(t)
	const id = "9a1b2c3d-aaaa-4bbb-8ccc-000000000001"
	pedido := map[string]string{
		"local":  "temas/historia/indice.md",
		"titulo": "História",
		"trecho": "Etiópia",
		"antes":  "O café veio da ",
		"depois": ", segundo a lenda.",
	}

	var m marcacao
	if s := e.pedir("PUT", "/api/detalhes/"+id, pedido, &m); s != http.StatusCreated || m.Estado != "processando" {
		t.Fatalf("PUT: %d %+v", s, m)
	}
	e.esperar("explicação pronta", func() bool {
		var lista []marcacao
		e.pedir("GET", "/api/detalhes?local=temas%2Fhistoria%2Findice.md", nil, &lista)
		if len(lista) == 1 {
			m = lista[0]
		}
		return m.Estado == "pronto"
	})
	// Só esta tarefa pode ir à internet; as etapas continuam sem.
	if !strings.Contains(m.Explicacao, "WebSearch,WebFetch") || len(m.Fontes) != 2 || m.Fontes[0].URL != "https://exemplo.com/a" {
		t.Fatalf("pronto: %+v", m)
	}
	if proibidas(ferramentasEscrita) != ferramentasProibidas || strings.Contains(proibidas(ferramentasInternet), "Web") {
		t.Errorf("proibidas: %q %q", proibidas(ferramentasEscrita), proibidas(ferramentasInternet))
	}

	// Reenviar (a rede caiu no meio) não pesquisa de novo.
	var tarefas []tarefaComTitulo
	e.pedir("PUT", "/api/detalhes/"+id, pedido, nil)
	e.pedir("GET", "/api/tarefas", nil, &tarefas)
	if len(tarefas) != 1 || tarefas[0].Referencia != "“Etiópia”" || tarefas[0].Link != "#/ler/temas/historia/indice.md" {
		t.Fatalf("fila: %+v", tarefas)
	}

	if s := e.pedir("POST", "/api/detalhes/"+id+"/refazer", nil, &m); s != http.StatusOK || m.Estado != "processando" || m.Explicacao != "" {
		t.Fatalf("refazer: %d %+v", s, m)
	}
	e.esperar("explicação refeita", func() bool {
		var lista []marcacao
		e.pedir("GET", "/api/detalhes", nil, &lista)
		return len(lista) == 1 && lista[0].Estado == "pronto"
	})

	for _, ruim := range []map[string]string{
		{"local": "../segredo.md", "trecho": "x"},
		{"local": "temas/a/../../b.md", "trecho": "x"},
		{"local": "temas/a.md", "trecho": "  "},
		{"local": "referencias/abc/outra", "trecho": "x"},
		{"local": "temas/a.md", "trecho": strings.Repeat("a", maxTrecho+1)},
	} {
		if s := e.pedir("PUT", "/api/detalhes/9a1b2c3d-aaaa-4bbb-8ccc-00000000000f", ruim, nil); s != http.StatusBadRequest {
			t.Errorf("pedido inválido aceito (%v): %d", ruim["local"], s)
		}
	}

	if s := e.pedir("DELETE", "/api/detalhes/"+id, nil, nil); s != http.StatusNoContent {
		t.Fatalf("DELETE: %d", s)
	}
	var lista []marcacao
	e.pedir("GET", "/api/detalhes", nil, &lista)
	if len(lista) != 0 {
		t.Fatalf("não apagou: %+v", lista)
	}
}

func TestPesquisaQueFalhaApareceNaMarcacao(t *testing.T) {
	e := novoAmbiente(t)
	t.Setenv("CLAUDE_FALSO_FALHA", "1")
	const id = "9a1b2c3d-aaaa-4bbb-8ccc-000000000002"
	e.pedir("PUT", "/api/detalhes/"+id, map[string]string{"local": "referencias/abc123/organizado", "trecho": "Tordesilhas"}, nil)
	var m marcacao
	e.esperar("falha", func() bool {
		var lista []marcacao
		e.pedir("GET", "/api/detalhes?local=referencias/abc123/organizado", nil, &lista)
		if len(lista) == 1 {
			m = lista[0]
		}
		return m.Estado == "falhou"
	})
	if !strings.Contains(m.Erro, "limite de uso") {
		t.Fatalf("erro: %+v", m)
	}
}

// D54: a pesquisa de um trecho vira uma Referência própria, com o texto, as fontes e a origem.
func TestMaisInformacoesViraReferencia(t *testing.T) {
	e := novoAmbiente(t)
	const id = "9a1b2c3d-aaaa-4bbb-8ccc-0000000000a1"
	e.pedir("PUT", "/api/detalhes/"+id, map[string]string{"local": "referencias/abc123/organizado", "titulo": "Navegações", "trecho": "o Tratado de Tordesilhas"}, nil)
	if s := e.pedir("POST", "/api/detalhes/"+id+"/referencia", nil, nil); s != http.StatusConflict {
		t.Fatalf("antes de pronta: %d", s)
	}
	e.esperar("pronta", func() bool {
		var m marcacao
		e.pedir("GET", "/api/detalhes/"+id, nil, &m)
		return m.Estado == "pronto"
	})
	var criada struct {
		Referencia referencia `json:"referencia"`
		Link       string     `json:"link"`
		JaExistia  bool       `json:"ja_existia"`
	}
	if s := e.pedir("POST", "/api/detalhes/"+id+"/referencia", map[string]string{}, &criada); s != http.StatusCreated {
		t.Fatalf("virar: %d", s)
	}
	r := criada.Referencia
	if r.Tipo != "pesquisa" || r.Estado != "organizado" || r.Titulo != "O Tratado de Tordesilhas" || !strings.HasSuffix(criada.Link, "/#/referencia/"+r.ID) {
		t.Fatalf("referência: %+v %s", r, criada.Link)
	}
	d := e.referencia(r.ID)
	texto := d.Textos["organizado"]
	if !strings.HasPrefix(texto, "# O Tratado de Tordesilhas\n") || !strings.Contains(texto, "[Navegações](#/referencia/abc123)") ||
		!strings.Contains(texto, "## Fontes") || !strings.Contains(texto, "https://exemplo.com/a") {
		t.Fatalf("texto:\n%s", texto)
	}
	if !strings.Contains(string(r.Pesquisa), `"detalhe":"`+id+`"`) {
		t.Fatalf("origem: %s", r.Pesquisa)
	}
	var m marcacao
	e.pedir("GET", "/api/detalhes/"+id, nil, &m)
	if m.Referencia != r.ID {
		t.Fatalf("a marcação não aponta a Referência: %+v", m)
	}
	// De novo: a mesma Referência, sem criar outra.
	if s := e.pedir("POST", "/api/detalhes/"+id+"/referencia", map[string]string{}, &criada); s != http.StatusOK || !criada.JaExistia || criada.Referencia.ID != r.ID {
		t.Fatalf("de novo: %d %+v", s, criada)
	}
}

// D54: um trecho de livro da Biblioteca pesquisa por aqui; não integra ao texto, mas vira Referência
// com o livro, o autor e o link de volta.
func TestMaisInformacoesDeUmLivro(t *testing.T) {
	e := novoAmbiente(t)
	const id = "9a1b2c3d-aaaa-4bbb-8ccc-0000000000b1"
	if s := e.pedir("PUT", "/api/detalhes/"+id, map[string]string{"local": "livros/0123abcd", "titulo": "Raízes do Brasil", "trecho": "homem cordial"}, nil); s != http.StatusCreated {
		t.Fatalf("PUT: %d", s)
	}
	e.esperar("pronta", func() bool {
		var m marcacao
		e.pedir("GET", "/api/detalhes/"+id, nil, &m)
		return m.Estado == "pronto"
	})
	if s := e.pedir("POST", "/api/detalhes/"+id+"/integrar", map[string]string{"modo": "completo"}, nil); s != http.StatusBadRequest {
		t.Fatalf("integrar num livro: %d", s)
	}
	var criada struct {
		Referencia referencia `json:"referencia"`
	}
	origem := map[string]any{"origem": map[string]string{"titulo": "Raízes do Brasil", "autor": "Sérgio Buarque de Holanda", "capitulo": "O homem cordial", "link": "https://biblioteca.internal/#/livro/0123abcd?em=x"}}
	if s := e.pedir("POST", "/api/detalhes/"+id+"/referencia", origem, &criada); s != http.StatusCreated {
		t.Fatalf("virar: %d", s)
	}
	texto := e.referencia(criada.Referencia.ID).Textos["organizado"]
	if !strings.Contains(texto, "[Raízes do Brasil, de Sérgio Buarque de Holanda (O homem cordial)](https://biblioteca.internal/#/livro/0123abcd?em=x)") {
		t.Fatalf("texto:\n%s", texto)
	}
	if !strings.Contains(string(criada.Referencia.Pesquisa), `"tipo":"livro"`) {
		t.Fatalf("origem: %s", criada.Referencia.Pesquisa)
	}
}

func TestTituloDoTrecho(t *testing.T) {
	for trecho, quer := range map[string]string{
		"  “cinema novo”. ":            "Cinema novo",
		"Etiópia":                      "Etiópia",
		strings.Repeat("palavra ", 30): "Palavra palavra palavra palavra palavra palavra palavra palavra palavra palavra palavra…",
	} {
		if got := tituloDoTrecho(trecho); got != quer {
			t.Errorf("tituloDoTrecho(%q) = %q, quer %q", trecho, got, quer)
		}
	}
}
