package main

import (
	"net/http"
	"strings"
	"testing"
)

func TestInstrucaoEditadaNoAppValeNaProximaEtapa(t *testing.T) {
	e := novoAmbiente(t)

	var lista []instrucao
	e.pedir("GET", "/api/instrucoes", nil, &lista)
	if len(lista) != 16 || lista[0].Arquivo != "organizar.md" || lista[0].Editada {
		t.Fatalf("lista: %+v", lista)
	}

	var i instrucao
	e.pedir("GET", "/api/instrucoes/organizar.md", nil, &i)
	if !strings.Contains(i.Conteudo, "# Regras: organizar uma Referência") {
		t.Fatalf("padrão: %q", i.Conteudo)
	}

	editada := i.Conteudo + "\n## REGRA EDITADA\n\nFrase nova.\n"
	if s := e.pedir("PUT", "/api/instrucoes/organizar.md", map[string]string{"conteudo": editada}, &i); s != http.StatusOK || !i.Editada || i.Conteudo != editada {
		t.Fatalf("PUT: %d %+v", s, i)
	}
	if s := e.pedir("PUT", "/api/instrucoes/organizar.md", map[string]string{"conteudo": "  "}, nil); s != http.StatusBadRequest {
		t.Errorf("instrução vazia aceita: %d", s)
	}
	if s := e.pedir("PUT", "/api/instrucoes/..%2Fsegredo.md", map[string]string{"conteudo": "x"}, nil); s != http.StatusNotFound {
		t.Errorf("arquivo fora da lista aceito: %d", s)
	}

	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	d := e.esperarEstado(id, "organizado")
	if !strings.Contains(d.Textos["organizado"], "Seguiu a regra editada no app.") {
		t.Fatalf("o claude não recebeu a instrução editada: %q", d.Textos["organizado"])
	}

	if s := e.pedir("DELETE", "/api/instrucoes/organizar.md", nil, &i); s != http.StatusOK || i.Editada || strings.Contains(i.Conteudo, "REGRA EDITADA") {
		t.Fatalf("voltar ao padrão: %d %+v", s, i)
	}
}

func TestSugestoesDeMelhoria(t *testing.T) {
	e := novoAmbiente(t)
	const id = "3f6c1d2e-aaaa-4bbb-8ccc-000000000001"

	var s melhoria
	if c := e.pedir("PUT", "/api/sugestoes/"+id, map[string]string{"texto": " Botão maior na fila ", "tela": "#/fila"}, &s); c != http.StatusOK {
		t.Fatalf("PUT: %d", c)
	}
	if s.Texto != "Botão maior na fila" || s.Tela != "#/fila" || s.Estado != "nova" {
		t.Fatalf("criada: %+v", s)
	}
	// Reenviar (a rede caiu no meio) não duplica.
	e.pedir("PUT", "/api/sugestoes/"+id, map[string]string{"texto": "Botão maior na fila", "tela": "#/fila"}, nil)
	if c := e.pedir("PUT", "/api/sugestoes/"+id, map[string]string{"texto": ""}, nil); c != http.StatusBadRequest {
		t.Errorf("sugestão vazia aceita: %d", c)
	}

	if c := e.pedir("PATCH", "/api/sugestoes/"+id, map[string]string{"estado": "feita", "resposta": "Aumentei o botão."}, &s); c != http.StatusOK || s.Estado != "feita" {
		t.Fatalf("PATCH: %d %+v", c, s)
	}
	// Depois de feita, o texto não muda mais.
	e.pedir("PUT", "/api/sugestoes/"+id, map[string]string{"texto": "outra coisa"}, &s)
	if s.Texto != "Botão maior na fila" {
		t.Errorf("texto de sugestão feita mudou: %q", s.Texto)
	}
	if c := e.pedir("PATCH", "/api/sugestoes/"+id, map[string]string{"estado": "talvez"}, nil); c != http.StatusBadRequest {
		t.Errorf("estado inválido aceito: %d", c)
	}

	var lista []melhoria
	e.pedir("GET", "/api/sugestoes", nil, &lista)
	if len(lista) != 1 || lista[0].Resposta != "Aumentei o botão." {
		t.Fatalf("lista: %+v", lista)
	}
	if c := e.pedir("DELETE", "/api/sugestoes/"+id, nil, nil); c != http.StatusNoContent {
		t.Fatalf("DELETE: %d", c)
	}
	e.pedir("GET", "/api/sugestoes", nil, &lista)
	if len(lista) != 0 {
		t.Fatalf("não apagou: %+v", lista)
	}
}
