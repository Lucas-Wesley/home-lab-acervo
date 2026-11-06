package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVideoDoYouTube(t *testing.T) {
	validos := map[string]string{
		"https://youtu.be/XQkGVoznDiY?si=bPAJenUlEbO9o9M6":         "XQkGVoznDiY",
		"https://www.youtube.com/watch?v=yAgLfb-e8sY":              "yAgLfb-e8sY",
		"https://m.youtube.com/watch?v=3iGPclknn_8&t=30s":          "3iGPclknn_8",
		"https://youtube.com/shorts/3iGPclknn_8":                   "3iGPclknn_8",
		"  https://www.youtube.com/live/3iGPclknn_8  ":             "3iGPclknn_8",
		"https://music.youtube.com/watch?v=3iGPclknn_8&list=abcde": "3iGPclknn_8",
		// Texto do "Compartilhar" do celular: título e link juntos.
		"Como o café chegou ao Brasil https://youtu.be/XQkGVoznDiY?si=x": "XQkGVoznDiY",
	}
	for link, esperado := range validos {
		if id, err := videoDoYouTube(link); err != nil || id != esperado {
			t.Errorf("%q: %q, %v", link, id, err)
		}
	}
	for _, link := range []string{"", "youtube.com/watch?v=3iGPclknn_8", "https://vimeo.com/123", "https://youtu.be/curto", "javascript:alert(1)"} {
		if _, err := videoDoYouTube(link); err == nil {
			t.Errorf("%q deveria ser recusado", link)
		}
	}
}

func TestEscolhaDeLegenda(t *testing.T) {
	casos := []struct {
		nome, json string
		lingua     string
		automatica bool
		semLegenda bool
	}{
		{"manual no idioma do vídeo", `{"language":"pt-BR","subtitles":{"pt":[],"en":[]},"automatic_captions":{"pt-orig":[]}}`, "pt", false, false},
		{"automática original", `{"language":"pt-BR","subtitles":{},"automatic_captions":{"pt":[],"pt-orig":[],"pt-PT":[]}}`, "pt-orig", true, false},
		{"sem idioma, manual em português", `{"subtitles":{"pt-BR":[]},"automatic_captions":{"pt-BR":[]}}`, "pt-BR", false, false},
		{"ignora o chat", `{"language":"pt","subtitles":{"live_chat":[]},"automatic_captions":{}}`, "", false, true},
		{"sem nada", `{"language":"en"}`, "", false, true},
	}
	for _, c := range casos {
		var info infoVideo
		json.Unmarshal([]byte(c.json), &info)
		lingua, auto, ok := escolhaDeLegenda(info)
		if ok == c.semLegenda || lingua != c.lingua || auto != c.automatica {
			t.Errorf("%s: %q %v %v", c.nome, lingua, auto, ok)
		}
	}
}

func TestLegendaParaCru(t *testing.T) {
	legenda := `{"events":[
		{"tStartMs":0,"segs":[{"utf8":"Olá,"},{"utf8":" pessoal"}]},
		{"tStartMs":1000,"aAppend":1,"segs":[{"utf8":"\n"}]},
		{"tStartMs":12000,"segs":[{"utf8":"tudo\nbem?"}]},
		{"tStartMs":40000,"segs":[{"utf8":"Segundo bloco."}]},
		{"tStartMs":3725000,"segs":[{"utf8":"Fim."}]}]}`
	cru, err := legendaParaCru([]byte(legenda), infoVideo{Titulo: "Título", URL: "https://x"})
	if err != nil {
		t.Fatal(err)
	}
	for _, trecho := range []string{"# Título", "[00:00] Olá, pessoal tudo bem?", "[00:40] Segundo bloco.", "[1:02:05] Fim."} {
		if !strings.Contains(cru, trecho) {
			t.Errorf("faltou %q em:\n%s", trecho, cru)
		}
	}
	if _, err := legendaParaCru([]byte(`{"events":[]}`), infoVideo{}); err != errSemLegenda {
		t.Errorf("legenda vazia: %v", err)
	}
}

func TestSlug(t *testing.T) {
	casos := map[string]string{
		"Segunda Guerra Mundial":      "segunda-guerra-mundial",
		"Café: o pó preto que vicia!": "cafe-o-po-preto-que-vicia",
		"  Alemanha × Rússia (1941) ": "alemanha-russia-1941",
		"Leonardo da Vinci":           "leonardo-da-vinci",
		"???":                         "",
	}
	for titulo, esperado := range casos {
		if s := slug(titulo); s != esperado {
			t.Errorf("%q: %q", titulo, s)
		}
	}
}

func TestCaminhos(t *testing.T) {
	for _, ok := range []string{"temas/cafe/indice.md", "temas/a/b/c.md"} {
		if _, err := caminhoDeArquivo(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, ruim := range []string{"temas/../README.md", "README.md", "temas/a/b.txt", "/temas/a.md", "temas/.git/x.md", "temas//a.md"} {
		if _, err := caminhoDeArquivo(ruim); err == nil {
			t.Errorf("%q deveria ser recusado", ruim)
		}
	}
	if _, err := caminhoDeTema("temas/segunda-guerra/alemanha-russia"); err != nil {
		t.Error(err)
	}
	for _, ruim := range []string{"temas/../x", "temas/Com Espaco", "outra/x", "temas/a/"} {
		if _, err := caminhoDeTema(ruim); err == nil {
			t.Errorf("%q deveria ser recusado", ruim)
		}
	}
}

func TestLerFrente(t *testing.T) {
	texto := "---\ntitulo: As origens do gaúcho\ncriado: 2026-11-06\nreferencias:\n  - id: yAgLfb-e8sY\n    titulo: AS ORIGENS: EDUARDO BUENO\n    url: https://www.youtube.com/watch?v=yAgLfb-e8sY\n  - id: outra\n    titulo: Outra\n    url: https://x\n---\n\n# Título do corpo\n\nPrimeiro parágrafo.\n"
	f, corpo := lerFrente(texto)
	if f.Titulo != "As origens do gaúcho" || f.Criado != "2026-11-06" || len(f.Referencias) != 2 {
		t.Fatalf("%+v", f)
	}
	if r := f.Referencias[0]; r.ID != "yAgLfb-e8sY" || r.Titulo != "AS ORIGENS: EDUARDO BUENO" || r.URL != "https://www.youtube.com/watch?v=yAgLfb-e8sY" {
		t.Errorf("%+v", r)
	}
	if !strings.HasPrefix(corpo, "# Título do corpo") {
		t.Errorf("corpo: %q", corpo)
	}
	if resumoDe(texto) != "Primeiro parágrafo." {
		t.Errorf("resumo: %q", resumoDe(texto))
	}
	if tituloDe("# Só corpo\n", "x") != "Só corpo" {
		t.Error("título do corpo")
	}
}
