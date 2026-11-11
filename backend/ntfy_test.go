package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ntfyFalso guarda o que os apps publicam.
func ntfyFalso(t *testing.T) (endereco string, recebidos chan map[string]any) {
	t.Helper()
	recebidos = make(chan map[string]any, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var m map[string]any
		json.NewDecoder(r.Body).Decode(&m)
		recebidos <- m
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/acervo", recebidos
}

func receberAviso(t *testing.T, recebidos chan map[string]any) map[string]any {
	t.Helper()
	select {
	case m := <-recebidos:
		return m
	case <-time.After(10 * time.Second):
		t.Fatal("nenhum aviso chegou")
		return nil
	}
}

// D42: um aviso quando o trabalho da Referência para (não a cada etapa) e um urgente quando falha.
func TestAvisos(t *testing.T) {
	e := novoAmbiente(t)
	endereco, recebidos := ntfyFalso(t)
	e.app.avisos = novoAvisador(endereco)

	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	m := receberAviso(t, recebidos)
	if m["topic"] != "acervo" || m["title"] != "Vídeo AAAAAAAAAAA" || m["message"] != "Pronto: Organizar." ||
		!strings.HasSuffix(m["click"].(string), "/#/referencia/"+id) || m["priority"] != nil {
		t.Errorf("aviso do fim: %v", m)
	}

	falha := e.adicionar("https://www.youtube.com/watch?v=FFFFFFFFFFF")
	e.esperarEstado(falha, "erro")
	m = receberAviso(t, recebidos)
	if !strings.HasPrefix(m["message"].(string), "Falhou: Extrair o texto. ") || m["priority"] != float64(4) {
		t.Errorf("aviso da falha: %v", m)
	}

	select {
	case m := <-recebidos:
		t.Errorf("aviso a mais (uma etapa no meio?): %v", m)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestAvisadorSemTopico(t *testing.T) {
	for _, endereco := range []string{"", "http://ntfy", "http://ntfy/", "http://ntfy/a/b"} {
		if novoAvisador(endereco) != nil {
			t.Errorf("%q devia desligar os avisos", endereco)
		}
	}
	if v := novoAvisador("http://ntfy/acervo/"); v == nil || v.servidor != "http://ntfy/" || v.topico != "acervo" {
		t.Errorf("com tópico: %+v", v)
	}
}
