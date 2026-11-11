package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestListaEMomentoDasDuvidas(t *testing.T) {
	texto := "# T\n\n## Seção\n\nTexto.\n\n## Dúvidas\n\n- \"Zeferin\" [17:16]: nome incerto.\n- **[56:22]–[56:53]**: nomes dos mortos\n  continuam incertos.\n- Sem marca, cita \"Em 1494\" da fala.\n"
	ds := listaDeDuvidas(texto)
	if len(ds) != 3 || ds[1] != "**[56:22]–[56:53]**: nomes dos mortos continuam incertos." {
		t.Fatalf("lista: %q", ds)
	}
	blocos := blocosDoCru("# V\n\n[00:00] Abertura.\n\n[00:31] Em 1494 o tratado.\n\n[01:02] Fim.\n")
	casos := []struct {
		duvida      string
		inicio, fim float64
		ok          bool
	}{
		{ds[0], 1036, 1036, true},
		{ds[1], 3382, 3413, true},
		{ds[2], 31, 61, true},
		{"Nada citado da fala.", 0, 0, false},
	}
	for _, c := range casos {
		i, f, ok := momentoDaDuvida(c.duvida, blocos)
		if i != c.inicio || f != c.fim || ok != c.ok {
			t.Errorf("%q: %v %v %v", c.duvida, i, f, ok)
		}
	}
}

// whisperFalso responde a transcrição de qualquer áudio; conta os pedidos.
func whisperFalso(t *testing.T, fala string) (*httptest.Server, *atomic.Int32) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/saude" {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "sem arquivo", 400)
			return
		}
		io.Copy(io.Discard, f)
		n.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"segments": []map[string]any{{"start": 2.0, "text": " " + fala}}})
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// As dúvidas de um vídeo: o trecho de cada uma é ouvido de novo pelo Whisper (com marca ou pelo
// que ela cita), a IA diz o que se apurou, e a correção vira proposta revisada.
func TestResolverDuvidasPeloAudio(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("sem ffmpeg")
	}
	whisper, ouvidos := whisperFalso(t, "Zéphyrin Ferrez, o avô.")
	e := novoAmbiente(t)
	e.app.cfg.whisper = whisper.URL
	id := e.adicionar("https://youtu.be/AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	organizado := "# Organizado\n\n## Seção\n\nTexto organizado.\n\n## Dúvidas\n\n- \"Zeferin\" [00:31]: nome incerto.\n- Cita \"Em 1494\" sem marca.\n- O vídeo não diz o ano.\n"
	os.WriteFile(filepath.Join(e.app.pastaReferencia(id), "organizado.md"), []byte(organizado), 0o644)

	var lista []duvidaResolvida
	if s := e.pedir("POST", "/api/referencias/"+id+"/duvidas", nil, &lista); s != http.StatusAccepted || len(lista) != 3 || lista[0].Estado != "processando" {
		t.Fatalf("resolver: %d %+v", s, lista)
	}
	e.esperar("dúvidas resolvidas", func() bool {
		e.pedir("GET", "/api/referencias/"+id+"/duvidas", nil, &lista)
		return len(lista) == 3 && lista[0].Estado == "pronto"
	})
	if n := ouvidos.Load(); n != 2 {
		t.Errorf("trechos ouvidos: %d (só as duas dúvidas com momento)", n)
	}
	d0, d1, d2 := lista[0], lista[1], lista[2]
	if d0.Momento != "00:31" || !strings.Contains(d0.Audio, "Zéphyrin Ferrez") || d0.Resultado.Estado != "resolvida_audio" ||
		!strings.Contains(d0.Resultado.NoAudio, "Zéphyrin") || d0.Resultado.Correto != "Texto corrigido pelo áudio." {
		t.Errorf("dúvida com marca: %+v %+v", d0, d0.Resultado)
	}
	if d1.Momento != "00:31" || d1.Resultado.Estado != "resolvida_audio" {
		t.Errorf("dúvida antiga, sem marca: %+v %+v", d1, d1.Resultado)
	}
	if d2.Momento != "" || d2.Audio != "" || d2.Resultado.Estado != "lacuna" {
		t.Errorf("dúvida sem momento: %+v %+v", d2, d2.Resultado)
	}

	// De novo, sem texto: nada novo para resolver. Com o texto de uma, só ela.
	if s := e.pedir("POST", "/api/referencias/"+id+"/duvidas", nil, nil); s != http.StatusBadRequest {
		t.Errorf("sem dúvidas novas: %d", s)
	}

	var r struct{ Tarefa int64 }
	if s := e.pedir("POST", "/api/duvidas/"+d0.ID+"/integrar", nil, &r); s != http.StatusAccepted {
		t.Fatalf("integrar: %d", s)
	}
	if diff := e.revisaoPronta(r.Tarefa); !strings.Contains(diff, "+Integrado ao texto") {
		t.Fatalf("diff:\n%s", diff)
	}

	// Corrigir de novo devolve a mesma proposta (D44); a aba mostra como o texto fica, e resolver
	// essa dúvida de novo espera a decisão.
	var r2 struct{ Tarefa int64 }
	if s := e.pedir("POST", "/api/duvidas/"+d0.ID+"/integrar", nil, &r2); s != http.StatusOK || r2.Tarefa != r.Tarefa {
		t.Fatalf("corrigir de novo: %d %d", s, r2.Tarefa)
	}
	e.pedir("GET", "/api/referencias/"+id+"/duvidas", nil, &lista)
	c := lista[0].Correcao
	if c == nil || c.TarefaID != r.Tarefa || c.Estado != emRevisao || len(c.Trechos) != 1 || c.Trechos[0] != (trechoMudado{"Texto organizado.", "Texto corrigido pelo áudio."}) {
		t.Fatalf("correção na aba: %+v", c)
	}
	if lista[1].Correcao != nil {
		t.Error("a outra dúvida aparece com correção")
	}
	if s := e.pedir("POST", "/api/referencias/"+id+"/duvidas", map[string]string{"texto": d0.Texto}, nil); s != http.StatusConflict {
		t.Errorf("resolver com correção esperando: %d", s)
	}
}

// Vídeo sem legenda: em vez de parar com erro, o Whisper transcreve o vídeo inteiro.
func TestVideoSemLegendaTranscritoPeloWhisper(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("sem ffmpeg")
	}
	whisper, _ := whisperFalso(t, "Fala do vídeo sem legenda.")
	e := novoAmbiente(t)
	e.app.cfg.whisper = whisper.URL
	e.app.cfg.ffmpeg = "ffmpeg"
	id := e.adicionar("https://youtu.be/SSSSSSSSSSS")
	d := e.esperarEstado(id, "organizado")
	if !strings.Contains(d.Textos["cru"], "[00:02] Fala do vídeo sem legenda.") || !strings.Contains(d.Textos["cru"], "o vídeo não tem legenda") {
		t.Fatalf("cru:\n%s", d.Textos["cru"])
	}
}
