package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// D51: o foco dos flashcards, escrito ou falado, vai junto com o texto para o Flashcards.
func TestFocoDosFlashcards(t *testing.T) {
	e := temaPronto(t)
	f := &flashcardsFalso{}
	f.ligar(e)
	corpo := map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "baralho_id": "b1", "tipo": "basico", "instrucao": "  só a parte das datas, tipo, né  "}
	if s := e.pedir("POST", "/api/flashcards", corpo, nil); s != http.StatusCreated {
		t.Fatalf("enviar com foco: %d", s)
	}
	if got := f.textos[0].Instrucao; got != "só a parte das datas, tipo, né" {
		t.Fatalf("foco no Flashcards: %q", got)
	}
	corpo["instrucao"] = strings.Repeat("a", maxInstrucaoFlashcards+1)
	if s := e.pedir("POST", "/api/flashcards", corpo, nil); s != http.StatusBadRequest || len(f.textos) != 1 {
		t.Fatalf("foco longo demais: %d", s)
	}
}

// D51: a fala vai ao Whisper com o título, os capítulos e os nomes do texto; os trechos de silêncio
// que o Whisper inventa ficam de fora.
func TestFalarOFocoDosFlashcards(t *testing.T) {
	e := temaPronto(t)
	if s, _ := e.falar([]byte("OPUS"), nil); s != http.StatusServiceUnavailable {
		t.Fatalf("sem Whisper: %d", s)
	}
	var mu sync.Mutex
	var prompts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/saude" {
			w.Write([]byte(`{"ok":true}`))
			return
		}
		arq, _, err := r.FormFile("file")
		if err != nil || r.FormValue("language") != "pt" {
			http.Error(w, "pedido", http.StatusBadRequest)
			return
		}
		dados, _ := io.ReadAll(arq)
		if string(dados) != "OPUS" {
			http.Error(w, "áudio", http.StatusBadRequest)
			return
		}
		mu.Lock()
		prompts = append(prompts, r.FormValue("prompt"))
		mu.Unlock()
		w.Write([]byte(`{"language":"pt","segments":[{"start":0,"text":" Quero só a parte  das datas,","no_speech_prob":0.1,"avg_logprob":-0.3},` +
			`{"start":3,"text":" tipo, né.","no_speech_prob":0.2,"avg_logprob":-0.4},` +
			`{"start":9,"text":" Obrigado por assistir.","no_speech_prob":0.9,"avg_logprob":-1.5}]}`))
	}))
	t.Cleanup(srv.Close)
	e.app.cfg.whisper = srv.URL

	s, texto := e.falar([]byte("OPUS"), map[string]string{"arquivo": "temas/tema-de-teste/indice.md"})
	if s != http.StatusOK || texto != "Quero só a parte das datas, tipo, né." {
		t.Fatalf("transcrever: %d %q", s, texto)
	}
	if len(prompts) != 1 || !strings.HasPrefix(prompts[0], contextoDaFala+" Texto: Tema de Teste.") {
		t.Fatalf("prompt com o texto: %q", prompts)
	}
	// Sem origem, só o contexto da fala; com uma origem que não existe, nada vai ao Whisper.
	if s, _ := e.falar([]byte("OPUS"), nil); s != http.StatusOK || prompts[1] != contextoDaFala {
		t.Fatalf("sem origem: %d %q", s, prompts)
	}
	if s, _ := e.falar([]byte("OPUS"), map[string]string{"referencia": "nao-existe"}); s != http.StatusNotFound || len(prompts) != 2 {
		t.Fatalf("origem que não existe: %d", s)
	}
}

func TestNomesPropriosParaOWhisper(t *testing.T) {
	texto := "# O Japão\n\nA Restauração Meiji começou em 1868. O xogunato Tokugawa caiu, e o imperador Meiji voltou ao poder.\n" +
		"Depois, o Japão se industrializou. Em Tóquio, a Restauração Meiji mudou tudo; Tokugawa Yoshinobu renunciou."
	nomes := nomesProprios(texto, 10)
	quer := []string{"Japão", "Restauração Meiji", "Tokugawa", "Meiji", "Tóquio", "Tokugawa Yoshinobu"}
	if strings.Join(nomes, "|") != strings.Join(quer, "|") {
		t.Fatalf("nomes: %q", nomes)
	}
	p := promptDoTexto(textoParaCartoes{Titulo: "História › Japão", Texto: texto, Secoes: []secaoComLink{{Titulo: "Era Meiji"}}})
	if p != contextoDaFala+" Texto: História › Japão. Capítulos: Era Meiji. Nomes: "+strings.Join(quer, ", ")+"." {
		t.Fatalf("prompt: %q", p)
	}
	if longo := promptDoTexto(textoParaCartoes{Titulo: strings.Repeat("Nome ", 400)}); len([]rune(longo)) > maximoDoPrompt {
		t.Fatalf("prompt longo demais: %d", len([]rune(longo)))
	}
}

func (e *ambiente) falar(audio []byte, campos map[string]string) (int, string) {
	e.t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	for k, v := range campos {
		mw.WriteField(k, v)
	}
	parte, _ := mw.CreateFormFile("audio", "fala.webm")
	parte.Write(audio)
	mw.Close()
	res, err := http.Post(e.srv.URL+"/api/transcrever", mw.FormDataContentType(), &corpo)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	var r struct {
		Texto string `json:"texto"`
	}
	json.NewDecoder(res.Body).Decode(&r)
	return res.StatusCode, r.Texto
}
