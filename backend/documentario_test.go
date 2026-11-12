package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFalasDaLegenda(t *testing.T) {
	srt := "\ufeff1\r\n00:00:01,000 --> 00:00:03,500\r\n<i>NARRADOR: O Egito antigo</i>\r\n- durou três mil anos.\r\n\r\n" +
		"2\n00:00:04,000 --> 00:00:05,000\n[MÚSICA DRAMÁTICA]\n\n" +
		"3\n00:00:06,2 --> 00:00:08,000\n♪ lá lá lá ♪\n\n" +
		"4\n00:01:02,000 --> 00:01:04,000\n{\\an8}Khufu &amp; Khafre (aplausos)\n\n" +
		"5\n00:00:05,500 --> 00:00:06,000\n1989\n\n" +
		"6\n00:08:09,781 --> 00:08:12,125\n- DR. CRANE: Pearl Harbor\ninfuriated the American people.\n"
	ts := falasDaLegenda([]byte(srt))
	if len(ts) != 3 {
		t.Fatalf("falas: %+v", ts)
	}
	// "08" e "09" não são octal.
	if ts[2].Inicio != 489.781 || ts[2].Texto != "DR. CRANE: Pearl Harbor infuriated the American people." {
		t.Errorf("fala do minuto 8: %+v", ts[2])
	}
	if ts[0].Inicio != 1 || ts[0].Texto != "NARRADOR: O Egito antigo durou três mil anos." {
		t.Errorf("primeira fala: %+v", ts[0])
	}
	if ts[1].Inicio != 62 || ts[1].Texto != "Khufu & Khafre" {
		t.Errorf("segunda fala: %+v", ts[1])
	}
}

// mandarDocumentario faz o pedido do Streaming: os dados e a legenda ou o áudio.
func (e *ambiente) mandarDocumentario(d documentario, campo, arquivo string, conteudo []byte) (int, map[string]any) {
	e.t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	dados, _ := json.Marshal(d)
	mw.WriteField("dados", string(dados))
	if campo != "" {
		parte, _ := mw.CreateFormFile(campo, arquivo)
		parte.Write(conteudo)
	}
	mw.Close()
	resp, err := http.Post(e.srv.URL+"/api/documentarios", mw.FormDataContentType(), &corpo)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	var r map[string]any
	json.NewDecoder(resp.Body).Decode(&r)
	return resp.StatusCode, r
}

type detalheDoDocumentario struct {
	detalhe
	Episodios []episodioLigado
}

func TestDocumentarioDoStreaming(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("sem ffmpeg")
	}
	antes := segundosPorPedaco
	segundosPorPedaco = 30
	t.Cleanup(func() { segundosPorPedaco = antes })
	audio := filepath.Join(t.TempDir(), "ep2.ogg")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=45", "-ac", "1", "-c:a", "libopus", "-b:a", "24k", audio).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v %s", err, out)
	}
	capa := filepath.Join(t.TempDir(), "capa.jpg")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=yellow:size=500x750", "-frames:v", "1", capa).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v %s", err, out)
	}
	var prompt atomic.Value
	var idioma atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/capa.jpg":
			http.ServeFile(w, r, capa)
		case "/v1/audio/transcriptions":
			f, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "pedido", 400)
				return
			}
			io.Copy(io.Discard, f)
			prompt.Store(r.FormValue("prompt"))
			idioma.Store(r.FormValue("language"))
			json.NewEncoder(w).Encode(map[string]any{"language": "en", "segments": []map[string]any{
				{"start": 0.0, "text": " The pyramid was built for Khufu."},
				{"start": 9.0, "text": " Thank you for watching!", "no_speech_prob": 0.9}, // a trilha: o Whisper inventa
				{"start": 15.0, "text": " Workers lived nearby."},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	e := novoAmbiente(t)
	e.app.cfg.whisperReserva = srv.URL
	e.app.cfg.ffmpeg = "ffmpeg"

	serie := documentario{
		Chave: "serie/900", Nome: "O Segredo das Pirâmides", Original: "Secrets of the Pyramids", Ano: "2019", Idioma: "en",
		Sinopse: "Uma série sobre as pirâmides do Egito.", Temporada: 1, Episodios: 3, Capa: srv.URL + "/capa.jpg",
		Link: "https://streaming.internal/#/titulo/serie/900", Origem: "assinatura",
	}

	// Episódio 1 pela legenda (o OpenSubtitles tinha uma no idioma original).
	ep1 := serie
	ep1.Episodio, ep1.TituloEpisodio, ep1.SinopseEpisodio, ep1.Duracao = 1, "A Grande Pirâmide", "Como Khufu ergueu Gizé.", 2640
	ep1.TMDB, ep1.Texto, ep1.Legenda, ep1.IdiomaDoTexto = "https://www.themoviedb.org/tv/900/season/1/episode/1", "legenda", "OpenSubtitles, en: Secrets.S01E01.WEB", "en"
	var srt strings.Builder
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&srt, "%d\n00:%02d:%02d,000 --> 00:%02d:%02d,500\nFala número %d sobre Gizé.\n\n", i+1, i/6, (i%6)*10, i/6, (i%6)*10+2, i)
	}
	s, r := e.mandarDocumentario(ep1, "legenda", "ep1.srt", []byte(srt.String()))
	if s != http.StatusCreated {
		t.Fatalf("ep1: %d %v", s, r)
	}
	id1 := r["id"].(string)
	d1 := e.esperarEstado(id1, "organizado")
	if d1.Tipo != "documentario" || d1.Titulo != "O Segredo das Pirâmides, episódio 1: A Grande Pirâmide" || d1.Canal != "Documentário · 2019 · T1 E1 de 3" ||
		d1.Duracao != "44:00" || d1.URL != ep1.TMDB {
		t.Errorf("metadados: %+v", d1.referencia)
	}
	for _, deve := range []string{"- Documentário: O Segredo das Pirâmides (Secrets of the Pyramids), 2019", "- Episódio: T1 E1 de 3 — A Grande Pirâmide",
		"- Sinopse do episódio: Como Khufu ergueu Gizé.", "- Texto: legenda do vídeo (OpenSubtitles, en: Secrets.S01E01.WEB)", "[00:00] Fala número 0 sobre Gizé.",
		"[00:30] Fala número 3", "- Streaming: https://streaming.internal/#/titulo/serie/900"} {
		if !strings.Contains(d1.Textos["cru"], deve) {
			t.Errorf("faltou %q no cru:\n%s", deve, d1.Textos["cru"])
		}
	}
	if !d1.Capa {
		t.Error("o documentário ficou sem capa")
	}
	if strings.Contains(d1.Textos["organizado"], "anterior") {
		t.Errorf("o primeiro episódio não tem anteriores: %s", d1.Textos["organizado"])
	}
	// Inglês: a tradução vem depois, como num vídeo.
	e.esperar("tradução do ep1", func() bool { return e.referencia(id1).Textos["traducao"] != "" })

	// Episódio 2 pelo áudio: o Whisper transcreve; o áudio fica para as Dúvidas.
	ep2 := serie
	ep2.Episodio, ep2.TituloEpisodio = 2, "Os construtores"
	ep2.TMDB, ep2.Texto, ep2.IdiomaDoTexto = "https://www.themoviedb.org/tv/900/season/1/episode/2", "audio", ""
	conteudo, _ := os.ReadFile(audio)
	s, r = e.mandarDocumentario(ep2, "audio", "ep2.ogg", conteudo)
	if s != http.StatusCreated {
		t.Fatalf("ep2: %d %v", s, r)
	}
	id2 := r["id"].(string)
	e.esperarEstado(id2, "organizado")
	var d2 detalheDoDocumentario
	e.pedir("GET", "/api/referencias/"+id2, nil, &d2)
	cru := d2.Textos["cru"]
	if !strings.Contains(cru, "- Texto: transcrição do áudio (Whisper large-v3-turbo (servidor))") || !strings.Contains(cru, "Khufu") ||
		!strings.Contains(cru, "Workers lived nearby.") || strings.Contains(cru, "Thank you for watching") {
		t.Errorf("cru do ep2:\n%s", cru)
	}
	if p, _ := prompt.Load().(string); !strings.Contains(p, "O Segredo das Pirâmides") || !strings.Contains(p, "Os construtores") {
		t.Errorf("prompt do Whisper: %q", p)
	}
	if l, _ := idioma.Load().(string); l != "" {
		t.Errorf("sem idioma do texto, o Whisper reconhece: %q", l)
	}
	if d2.Idioma != "en" {
		t.Errorf("idioma reconhecido: %q", d2.Idioma)
	}
	if !strings.Contains(d2.Textos["organizado"], "Leu 1 episódio(s) anterior(es).") {
		t.Errorf("o ep2 não recebeu o ep1: %s", d2.Textos["organizado"])
	}
	if _, err := os.Stat(filepath.Join(e.app.pastaReferencia(id2), "outros-episodios.md")); err == nil {
		t.Error("outros-episodios.md ficou na pasta")
	}
	if e.app.audioDoDocumentario(id2) == "" {
		t.Error("o áudio do documentário não ficou guardado")
	}
	if len(d2.Episodios) != 2 || d2.Episodios[0].ID != id1 || d2.Episodios[1].ID != id2 || d2.Episodios[0].Titulo != "A Grande Pirâmide" {
		t.Errorf("episódios ligados: %+v", d2.Episodios)
	}
	// As Dúvidas ouvem de novo um trecho do áudio guardado.
	ref2, _ := e.app.referencia(t.Context(), id2)
	trecho, err := e.app.trechoDeAudio(t.Context(), ref2, t.TempDir(), 1, 5, 20)
	if err != nil {
		t.Fatalf("trecho: %v", err)
	}
	if st, err := os.Stat(trecho); err != nil || st.Size() < 1000 {
		t.Errorf("trecho cortado: %v", err)
	}
	ref1, _ := e.app.referencia(t.Context(), id1)
	if _, err := e.app.trechoDeAudio(t.Context(), ref1, t.TempDir(), 1, 5, 20); err == nil {
		t.Error("o ep1 veio pela legenda: não há áudio para cortar")
	}

	// A sugestão de Tema sabe dos outros episódios.
	if s := e.pedir("POST", "/api/referencias/"+id2+"/sugerir", nil, nil); s >= 300 {
		t.Fatalf("sugerir: %d", s)
	}
	e.esperar("sugestão do ep2", func() bool { return strings.Contains(string(e.referencia(id2).Sugestao), "Junto dos outros episódios.") })

	// O mesmo episódio de novo é recusado, com o id.
	s, r = e.mandarDocumentario(ep2, "audio", "ep2.ogg", conteudo)
	if s != http.StatusConflict || r["id"] != id2 {
		t.Errorf("duplicado: %d %v", s, r)
	}
	// Pedidos errados.
	if s, _ := e.mandarDocumentario(ep2, "", "", nil); s != http.StatusBadRequest {
		t.Errorf("sem arquivo: %d", s)
	}
	ruim := ep2
	ruim.TMDB = "https://exemplo.com/x"
	if s, _ := e.mandarDocumentario(ruim, "audio", "a.ogg", conteudo); s != http.StatusBadRequest {
		t.Errorf("link errado: %d", s)
	}

	// Um episódio que deu erro (legenda vazia) é refeito quando chega de novo.
	ep3 := serie
	ep3.Episodio, ep3.TMDB, ep3.Texto = 3, "https://www.themoviedb.org/tv/900/season/1/episode/3", "legenda"
	s, r = e.mandarDocumentario(ep3, "legenda", "ep3.srt", []byte("1\n00:00:01,000 --> 00:00:02,000\nSó isto.\n"))
	if s != http.StatusCreated {
		t.Fatalf("ep3: %d %v", s, r)
	}
	id3 := r["id"].(string)
	if d := e.esperarEstado(id3, "erro"); !strings.Contains(d.Erro, "quase vazia") {
		t.Errorf("erro do ep3: %q", d.Erro)
	}
	s, r = e.mandarDocumentario(ep3, "legenda", "ep3.srt", []byte(srt.String()))
	if s != http.StatusCreated || r["id"] != id3 {
		t.Fatalf("ep3 de novo: %d %v", s, r)
	}
	e.esperarEstado(id3, "organizado")
}
