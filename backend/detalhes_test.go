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
