package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// revisaoPronta espera a proposta da tarefa e devolve o diff.
func (e *ambiente) revisaoPronta(tarefa int64) string {
	e.t.Helper()
	var r struct {
		Diff string `json:"diff"`
	}
	e.esperar("proposta para revisar", func() bool {
		return e.pedir("GET", "/api/revisoes/"+strconv.FormatInt(tarefa, 10), nil, &r) == http.StatusOK
	})
	return r.Diff
}

func (e *ambiente) perguntaPronta(local, pergunta string) perguntaFeita {
	e.t.Helper()
	var p perguntaFeita
	if s := e.pedir("POST", "/api/perguntas", map[string]string{"local": local, "titulo": "Clube", "pergunta": pergunta}, &p); s != http.StatusCreated {
		e.t.Fatalf("POST /api/perguntas: %d", s)
	}
	e.esperar("resposta pronta", func() bool {
		var l []perguntaFeita
		e.pedir("GET", "/api/perguntas?local="+url.QueryEscape(local), nil, &l)
		for _, x := range l {
			if x.ID == p.ID {
				p = x
			}
		}
		return p.Estado == "pronto"
	})
	return p
}

// A resposta de uma pergunta entra no Tema pela proposta revisada; cada divergência segue a escolha
// do Lucas.
func TestIntegrarRespostaNoTema(t *testing.T) {
	e := novoAmbiente(t)
	pasta := filepath.Join(e.app.conteudo.raiz, "temas", "clube")
	os.MkdirAll(pasta, 0o755)
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("# Clube\n\n## Fundação\n\nFundado em 1930.\n"), 0o644)
	if err := e.app.conteudo.commit(context.Background(), "Tema de teste", "temas/clube"); err != nil {
		t.Fatal(err)
	}
	p := e.perguntaPronta("temas/clube", "Quando foi fundado?")

	for _, ruim := range []map[string]any{
		{"resposta": false, "escolhas": []map[string]any{{"divergencia": 0, "acao": "ignorar"}}},
		{"resposta": true, "escolhas": []map[string]any{{"divergencia": 5, "acao": "substituir"}}},
		{"resposta": true, "escolhas": []map[string]any{{"divergencia": 0, "acao": "apagar"}}},
	} {
		if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", ruim, nil); s != http.StatusBadRequest {
			t.Errorf("%v: %d", ruim, s)
		}
	}

	var r struct{ Tarefa int64 }
	pedido := map[string]any{"resposta": false, "escolhas": []map[string]any{{"divergencia": 0, "acao": "acrescentar"}}}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", pedido, &r); s != http.StatusAccepted {
		t.Fatalf("integrar: %d", s)
	}
	diff := e.revisaoPronta(r.Tarefa)
	if !strings.Contains(diff, "+Integrado ao texto (escolhas: Acrescentar; modo: ).") || !strings.Contains(diff, "temas/clube/indice.md") {
		t.Fatalf("diff:\n%s", diff)
	}
	var revisoes []tarefaComTitulo
	e.pedir("GET", "/api/revisoes", nil, &revisoes)
	if len(revisoes) != 1 || revisoes[0].Link != "#/ler/temas/clube/indice.md" || revisoes[0].Referencia != "Clube" {
		t.Fatalf("revisões: %+v", revisoes)
	}
	var fim map[string]string
	if s := e.pedir("POST", "/api/revisoes/"+strconv.FormatInt(r.Tarefa, 10)+"/aprovar", nil, &fim); s != http.StatusOK || fim["tema"] != "temas/clube" {
		t.Fatalf("aprovar: %d %v", s, fim)
	}
	b, _ := os.ReadFile(filepath.Join(pasta, "indice.md"))
	if !strings.Contains(string(b), "Integrado ao texto") {
		t.Fatalf("o Tema não mudou:\n%s", b)
	}

	// "Mais informações" no Tema: rejeitar deixa o texto como estava.
	const id = "9a1b2c3d-aaaa-4bbb-8ccc-000000000002"
	m := map[string]string{"local": "temas/clube/indice.md", "titulo": "Clube", "trecho": "1930", "antes": "Fundado em ", "depois": "."}
	e.pedir("PUT", "/api/detalhes/"+id, m, nil)
	e.esperar("explicação pronta", func() bool {
		var l []marcacao
		e.pedir("GET", "/api/detalhes?local="+url.QueryEscape("temas/clube/indice.md"), nil, &l)
		return len(l) == 1 && l[0].Estado == "pronto"
	})
	if s := e.pedir("POST", "/api/detalhes/"+id+"/integrar", map[string]string{"modo": "metade"}, nil); s != http.StatusBadRequest {
		t.Errorf("modo inválido: %d", s)
	}
	if s := e.pedir("POST", "/api/detalhes/"+id+"/integrar", map[string]string{"modo": "completo"}, &r); s != http.StatusAccepted {
		t.Fatalf("integrar detalhe: %d", s)
	}
	if diff := e.revisaoPronta(r.Tarefa); !strings.Contains(diff, "modo: O texto completo") {
		t.Fatalf("diff do detalhe:\n%s", diff)
	}
	antes, _ := os.ReadFile(filepath.Join(pasta, "indice.md"))
	if s := e.pedir("POST", "/api/revisoes/"+strconv.FormatInt(r.Tarefa, 10)+"/rejeitar", nil, nil); s != http.StatusOK {
		t.Fatalf("rejeitar: %d", s)
	}
	depois, _ := os.ReadFile(filepath.Join(pasta, "indice.md"))
	if string(antes) != string(depois) {
		t.Error("rejeitar mudou o Tema")
	}
}

// Numa Referência (fora do git), a proposta é uma cópia do texto; ao aprovar, o texto de antes
// fica como Versão anterior.
func TestIntegrarRespostaNaReferencia(t *testing.T) {
	e := novoAmbiente(t)
	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	antes := e.esperarEstado(id, "organizado").Textos["organizado"]
	p := e.perguntaPronta("referencias/"+id, "Quando?")

	var r struct{ Tarefa int64 }
	pedido := map[string]any{"resposta": true, "escolhas": []map[string]any{{"divergencia": 0, "acao": "substituir"}}}
	if s := e.pedir("POST", "/api/perguntas/"+p.ID+"/integrar", pedido, &r); s != http.StatusAccepted {
		t.Fatalf("integrar: %d", s)
	}
	diff := e.revisaoPronta(r.Tarefa)
	if !strings.Contains(diff, "a/referencias/"+id+"/organizado.md") || !strings.Contains(diff, "+Integrado ao texto (escolhas: Substituir; modo: ).") ||
		strings.Contains(diff, e.app.cfg.dados) {
		t.Fatalf("diff:\n%s", diff)
	}
	if e.referencia(id).Textos["organizado"] != antes {
		t.Fatal("o texto mudou antes da aprovação")
	}
	if s := e.pedir("POST", "/api/revisoes/"+strconv.FormatInt(r.Tarefa, 10)+"/aprovar", nil, nil); s != http.StatusOK {
		t.Fatalf("aprovar: %d", s)
	}
	d := e.referencia(id)
	if !strings.Contains(d.Textos["organizado"], "Integrado ao texto") || d.Textos["anterior"] != antes || d.Estado != "organizado" {
		t.Fatalf("depois de aprovar: estado %s\n%s", d.Estado, d.Textos["organizado"])
	}
	if _, err := os.Stat(e.app.pastaDaProposta(r.Tarefa)); !os.IsNotExist(err) {
		t.Error("a proposta ficou no disco")
	}
}
