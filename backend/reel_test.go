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
	"testing"
)

func TestReelDoLink(t *testing.T) {
	casos := map[string]string{
		"https://www.instagram.com/reel/CDUMkliABpa/?igsh=MWQ1ZGUxMzBkMA==": "https://www.instagram.com/reel/CDUMkliABpa/",
		"Olha isso https://instagram.com/reels/Chunk8-jurw":                 "https://www.instagram.com/reel/Chunk8-jurw/",
		"https://www.instagram.com/nasa/reel/CoLINANJtMV/":                  "https://www.instagram.com/reel/CoLINANJtMV/",
		"https://m.instagram.com/reel/CoLINANJtMV":                          "https://www.instagram.com/reel/CoLINANJtMV/",
		"https://www.instagram.com/p/CoLINANJtMV/":                          "",
		"https://www.instagram.com/nasa/":                                   "",
		"https://www.youtube.com/watch?v=AAAAAAAAAAA":                       "",
		"https://exemplo.com/reel/CoLINANJtMV":                              "",
	}
	for entrada, quer := range casos {
		l, err := reelDoLink(entrada)
		if quer == "" {
			if err == nil {
				t.Errorf("%s: devia recusar, deu %s", entrada, l)
			}
			continue
		}
		if err != nil || l != quer {
			t.Errorf("%s: %q %v", entrada, l, err)
		}
	}
	if tituloDoReel(infoReel{Conta: "nasa", Legenda: "\r\t\r  Primeira linha\r\t\rsegunda"}) != "Primeira linha" || tituloDoReel(infoReel{Conta: "nasa"}) != "Reel de @nasa" {
		t.Error("título do Reel")
	}
}

// Um Reel vira Referência: o yt-dlp baixa o vídeo e o info.json, o Whisper transcreve a fala, o ffmpeg
// tira os quadros e o Claude escreve o que aparece na tela. O cru.md junta fala, tela e legenda.
func TestReferenciaDeReel(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("sem ffmpeg")
	}
	whisper := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, _, err := r.FormFile("file")
		if err != nil || r.FormValue("language") != "" || !strings.Contains(r.FormValue("prompt"), "Três dicas de estudo") {
			http.Error(w, "pedido", 400)
			return
		}
		io.Copy(io.Discard, f)
		json.NewEncoder(w).Encode(map[string]any{"language": "en", "segments": []map[string]any{
			{"start": 0.0, "text": " First tip.", "no_speech_prob": 0.02}, {"start": 5.0, "text": " Second tip.", "compression_ratio": 1.2},
			{"start": 8.0, "text": " Thanks for watching!", "no_speech_prob": 0.8}, {"start": 9.0, "text": " stress stress stress", "compression_ratio": 9.1}, {"start": 9.5, "text": " ! ♪"},
		}})
	}))
	defer whisper.Close()

	e := novoAmbiente(t)
	e.app.cfg.whisper = whisper.URL
	e.app.cfg.ffmpeg = "ffmpeg"
	args := filepath.Join(t.TempDir(), "args")
	t.Setenv("YTDLP_FALSO_ARGS", args)

	var r referencia
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": "https://www.instagram.com/p/AAAAAA/", "tipo": "reel"}, nil); s != http.StatusBadRequest {
		t.Errorf("post que não é Reel: %d", s)
	}
	e.pedir("POST", "/api/referencias", map[string]string{"url": "Veja https://www.instagram.com/reel/AbCdEf123/?igsh=xyz", "tipo": "reel"}, &r)
	d := e.esperarEstado(r.ID, "organizado")
	if d.Tipo != "reel" || d.URL != "https://www.instagram.com/reel/AbCdEf123/" || d.Titulo != "Três dicas de estudo" ||
		d.Canal != "@conta" || d.Duracao != "00:10" || d.Idioma != "en" {
		t.Errorf("metadados: %+v", d.referencia)
	}
	cru := d.Textos["cru"]
	for _, deve := range []string{"- Reel: @conta (Conta de Teste)", "- Publicado: 2026-03-01", "- Transcrição: Whisper large-v3 (Mac mini)",
		"## Fala\n\n[00:00] First tip. Second tip.", "## Na tela\n\n[00:00] Texto na tela: \"quadro 00-00\".", "[00:04]", "[00:08]",
		"Leu o contexto.", "## Legenda do post\n\nTrês dicas de estudo\n\nMais na legenda. #estudo"} {
		if !strings.Contains(cru, deve) {
			t.Errorf("faltou %q no cru:\n%s", deve, cru)
		}
	}
	// O vídeo e os quadros não ficam guardados.
	if m, _ := filepath.Glob(filepath.Join(e.app.pastaReferencia(r.ID), "*.*4")); len(m) > 0 {
		t.Errorf("vídeo guardado: %v", m)
	}
	// Sem cookies no servidor, o yt-dlp vai sem login.
	if b, _ := os.ReadFile(args); strings.Contains(string(b), "--cookies") {
		t.Errorf("cookies sem arquivo: %s", b)
	}
	// O mesmo Reel de novo é recusado.
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": "https://www.instagram.com/reel/AbCdEf123/"}, nil); s != http.StatusConflict {
		t.Errorf("duplicado: %d", s)
	}
}

// Quando o Instagram pede login: sem cookies, o aviso diz para pô-los; com cookies, que venceram (e
// o yt-dlp foi chamado com eles).
func TestReelPedeLogin(t *testing.T) {
	e := novoAmbiente(t)
	e.app.cfg.whisper = "http://127.0.0.1:1"
	e.app.cfg.instagramCookies = filepath.Join(t.TempDir(), "instagram-cookies.txt")
	args := filepath.Join(t.TempDir(), "args")
	t.Setenv("YTDLP_FALSO_ARGS", args)

	var r referencia
	e.pedir("POST", "/api/referencias", map[string]string{"url": "https://www.instagram.com/reel/LOGIN1/"}, &r)
	if d := e.esperarEstado(r.ID, "erro"); d.Erro != errInstagramLogin.Error() {
		t.Errorf("sem cookies: %q", d.Erro)
	}

	os.WriteFile(e.app.cfg.instagramCookies, []byte("# Netscape HTTP Cookie File\n"), 0o600)
	e.pedir("POST", "/api/referencias", map[string]string{"url": "https://www.instagram.com/reel/LOGIN2/"}, &r)
	if d := e.esperarEstado(r.ID, "erro"); d.Erro != errCookiesVencidos.Error() {
		t.Errorf("com cookies: %q", d.Erro)
	}
	if b, _ := os.ReadFile(args); !strings.Contains(string(b), "--cookies "+e.app.cfg.instagramCookies) {
		t.Errorf("o yt-dlp não recebeu os cookies: %s", b)
	}
}
