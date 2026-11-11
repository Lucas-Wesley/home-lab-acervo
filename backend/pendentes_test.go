package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"testing"
)

// Pedir duas vezes para integrar a mesma resposta devolve a mesma proposta (D44); refazer a pergunta
// fica bloqueado até a decisão, e depois de descartar dá para pedir de novo.
func TestIntegrarDuasVezesDevolveAMesma(t *testing.T) {
	e := novoAmbiente(t)
	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	local := "referencias/" + id
	p := e.perguntaPronta(local, "Quando?")

	pedido := map[string]any{"resposta": true}
	var r1, r2 struct {
		Tarefa   int64
		JaPedida bool `json:"ja_pedida"`
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", pedido, &r1); s != http.StatusAccepted {
		t.Fatalf("integrar: %d", s)
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", pedido, &r2); s != http.StatusOK || r2.Tarefa != r1.Tarefa || !r2.JaPedida {
		t.Fatalf("pedir de novo: %d %+v (primeira: %d)", s, r2, r1.Tarefa)
	}
	e.revisaoPronta(r1.Tarefa)
	var n int
	e.app.db.QueryRow(`SELECT COUNT(*) FROM tarefas WHERE tipo = 'integrar_texto'`).Scan(&n)
	if n != 1 {
		t.Fatalf("%d tarefas de integrar", n)
	}

	var l []perguntaFeita
	e.pedir("GET", "/api/perguntas?local="+url.QueryEscape(local), nil, &l)
	if len(l) != 1 || l[0].Integracao == nil || l[0].Integracao.TarefaID != r1.Tarefa || l[0].Integracao.Estado != emRevisao {
		t.Fatalf("a pergunta não mostra a proposta: %+v", l)
	}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/refazer", nil, nil); s != http.StatusConflict {
		t.Fatalf("refazer com proposta esperando: %d", s)
	}

	if s := e.pedir("POST", "/api/revisoes/"+strconv.FormatInt(r1.Tarefa, 10)+"/rejeitar", nil, nil); s != http.StatusOK {
		t.Fatalf("rejeitar: %d", s)
	}
	e.pedir("GET", "/api/perguntas?local="+url.QueryEscape(local), nil, &l)
	if l[0].Integracao != nil {
		t.Fatal("a proposta descartada continua aparecendo")
	}
	var r3 struct{ Tarefa int64 }
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", pedido, &r3); s != http.StatusAccepted || r3.Tarefa == r1.Tarefa {
		t.Fatalf("pedir depois de descartar: %d %d", s, r3.Tarefa)
	}
}

func TestTrechosMudados(t *testing.T) {
	antes := "# T\n\nUm.\n\nO sapo bala.\n\nTrês.\n\n## Dúvidas\n\n- \"bala\" [00:10]: incerto.\n"
	depois := "# T\n\nUm.\n\nO sapo coaxa.\n\nTrês.\n\nQuatro.\n"
	got := trechosMudados(antes, depois)
	want := []trechoMudado{{"O sapo bala.", "O sapo coaxa."}, {"", "Quatro."}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("trechos: %+v", got)
	}
}

// Duas propostas sobre o mesmo texto: a segunda, aprovada depois, junta-se à primeira; no mesmo
// trecho, é recusada.
func TestJuntarComAtual(t *testing.T) {
	base := []byte("# T\n\nUm.\n\nDois.\n\nTrês.\n\nQuatro.\n")
	atual := []byte("# T\n\nUm corrigido.\n\nDois.\n\nTrês.\n\nQuatro.\n")
	proposta := []byte("# T\n\nUm.\n\nDois.\n\nTrês.\n\nQuatro corrigido.\n")
	got, err := juntarComAtual(context.Background(), atual, base, proposta)
	if err != nil || string(got) != "# T\n\nUm corrigido.\n\nDois.\n\nTrês.\n\nQuatro corrigido.\n" {
		t.Fatalf("juntar: %v\n%s", err, got)
	}
	cruzada := []byte("# T\n\nUm de outro jeito.\n\nDois.\n\nTrês.\n\nQuatro.\n")
	if _, err := juntarComAtual(context.Background(), atual, base, cruzada); !errors.Is(err, errPropostaCruzada) {
		t.Fatalf("mesmo trecho: %v", err)
	}
}
