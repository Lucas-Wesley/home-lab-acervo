package main

import (
	"encoding/json"
	"fmt"
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

func TestEpisodioDoLink(t *testing.T) {
	casos := map[string]string{
		"https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz?si=abc":            "https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz",
		"Ouça isto https://open.spotify.com/intl-pt/episode/5uWIuQMlrVbKvDucC8IRvz": "https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz",
		"https://podcasts.apple.com/br/podcast/um-podcast/id1234567?i=1000654321":   "https://podcasts.apple.com/podcast/id1234567?i=1000654321",
		"https://open.spotify.com/show/5uWIuQMlrVbKvDucC8IRvz":                      "",
		"https://podcasts.apple.com/br/podcast/um-podcast/id1234567":                "",
		"https://www.youtube.com/watch?v=AAAAAAAAAAA":                               "",
	}
	for entrada, quer := range casos {
		l, err := episodioDoLink(entrada)
		if quer == "" {
			if err == nil {
				t.Errorf("%s: devia recusar, deu %+v", entrada, l)
			}
			continue
		}
		if err != nil || l.URL != quer {
			t.Errorf("%s: %+v %v", entrada, l, err)
		}
	}
}

func TestDescricaoEmMarkdown(t *testing.T) {
	d := descricaoEmMarkdown(`<p>Da luz de <b>Hercule Florence</b> a Salgado.</p><p>LIVROS: <a href="https://loja.exemplo/livro">Um Livro</a><br>Site: <a href="https://exemplo.org">https://exemplo.org</a></p><ul><li>Item um</li><li>Item dois</li></ul><img src="https://img.exemplo/foto.jpg" alt="Foto">`)
	for _, deve := range []string{"Da luz de **Hercule Florence** a Salgado.", "LIVROS: [Um Livro](https://loja.exemplo/livro)\nSite: <https://exemplo.org>", "- Item um\n- Item dois", "![Foto](https://img.exemplo/foto.jpg)"} {
		if !strings.Contains(d, deve) {
			t.Errorf("faltou %q em:\n%s", deve, d)
		}
	}
	if d := descricaoEmMarkdown("Linha um\n\n\n\nLinha dois &amp; três"); d != "Linha um\n\nLinha dois & três" {
		t.Errorf("texto puro: %q", d)
	}
	if duracaoDoFeed("2265") != "37:45" || duracaoDoFeed("01:02:03") != "1:02:03" || duracaoDoFeed("x") != "" {
		t.Error("duração do feed")
	}
}

// Um episódio do Spotify vira Referência: o oEmbed do Spotify dá o título, a busca de episódios do
// Apple Podcasts dá o podcast e o feed, o feed dá o áudio e a descrição, e o Whisper transcreve os pedaços.
// Com o Whisper do Mac fora do ar, a transcrição vai para a reserva (o do servidor).
func TestReferenciaDePodcast(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("sem ffmpeg")
	}
	audio := filepath.Join(t.TempDir(), "ep.mp3")
	if out, err := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=50", audio).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v %s", err, out)
	}
	antes := segundosPorPedaco
	segundosPorPedaco = 30
	t.Cleanup(func() { segundosPorPedaco = antes })

	var pedidos atomic.Int32
	var prompt atomic.Value
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oembed":
			if r.URL.Query().Get("url") != "https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz" {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `{"title":"#165 a fotografia foi inventada no Brasil","provider_name":"Spotify"}`)
		case "/search":
			q := r.URL.Query()
			if q.Get("term") != "#165 a fotografia foi inventada no Brasil" || q.Get("entity") != "podcastEpisode" {
				http.Error(w, "termo", 400)
				return
			}
			fmt.Fprintf(w, `{"results":[{"trackName":"#16 outro","collectionName":"Outro","feedUrl":"%s/outro.xml"},{"trackName":"#165 A fotografia foi inventada no Brasil","collectionName":"Podcast de Teste","feedUrl":"%s/feed.xml"}]}`, srv.URL, srv.URL)
		case "/feed.xml":
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel>
<title>Podcast de Teste</title><language>pt-br</language><itunes:image href="https://img.exemplo/podcast.jpg"/>
<item><title>#164 outro episódio</title><enclosure url="%s/outro.mp3" type="audio/mpeg"/></item>
<item><title>#165 A Fotografia foi inventada no Brasil</title><pubDate>Tue, 10 Jun 2025 11:11:47 GMT</pubDate>
<itunes:duration>00:00:50</itunes:duration><itunes:image href="https://img.exemplo/ep165.jpg"/>
<description><![CDATA[<p>Da luz de Hercule Florence.</p><p>LIVROS INDICADOS: <a href="https://loja.exemplo/livro">Um Livro</a></p>]]></description>
<enclosure url="%s/ep.mp3" type="audio/mpeg"/></item>
</channel></rss>`, srv.URL, srv.URL)
		case "/ep.mp3":
			http.ServeFile(w, r, audio)
		case "/v1/audio/transcriptions":
			n := pedidos.Add(1)
			f, _, err := r.FormFile("file")
			if err != nil || r.FormValue("language") != "pt" || r.FormValue("model") != modeloDaReserva {
				http.Error(w, "pedido", 400)
				return
			}
			io.Copy(io.Discard, f)
			prompt.Store(r.FormValue("prompt"))
			json.NewEncoder(w).Encode(map[string]any{"segments": []map[string]any{
				{"start": 0.0, "text": fmt.Sprintf(" Fala do pedaço %d.", n)},
				{"start": 12.5, "text": " Continua."},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	e := novoAmbiente(t)
	e.app.cfg.spotify = srv.URL
	e.app.cfg.itunes = srv.URL
	e.app.cfg.whisper = "http://127.0.0.1:1" // o Mac fora do ar
	e.app.cfg.whisperReserva = srv.URL
	e.app.cfg.ffmpeg = "ffmpeg"

	id := e.adicionar("Ouça este episódio https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz?si=nr7g")
	d := e.esperarEstado(id, "organizado")
	if d.Tipo != "podcast" || d.URL != "https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz" || d.Titulo != "#165 A Fotografia foi inventada no Brasil" ||
		d.Canal != "Podcast de Teste" || d.Duracao != "00:50" || d.Idioma != "pt-br" {
		t.Errorf("metadados: %+v", d.referencia)
	}
	cru := d.Textos["cru"]
	for _, deve := range []string{"- Podcast: Podcast de Teste", "- Publicado: 2025-06-10", "- Capa: https://img.exemplo/ep165.jpg",
		"- Transcrição: Whisper large-v3-turbo (servidor)", "[00:00] Fala do pedaço", "[00:30] Fala do pedaço",
		"## Descrição do episódio", "LIVROS INDICADOS: [Um Livro](https://loja.exemplo/livro)"} {
		if !strings.Contains(cru, deve) {
			t.Errorf("faltou %q no cru:\n%s", deve, cru)
		}
	}
	if n := pedidos.Load(); n != 2 {
		t.Errorf("pedaços transcritos: %d", n)
	}
	if p, _ := prompt.Load().(string); !strings.Contains(p, "Hercule Florence") || strings.Contains(p, "https://") {
		t.Errorf("prompt do Whisper: %q", p)
	}
	// O áudio não fica guardado: só o endereço, no info.json.
	if _, err := os.Stat(filepath.Join(e.app.pastaReferencia(id), "info.json")); err != nil {
		t.Error("sem info.json")
	}
	if m, _ := filepath.Glob(filepath.Join(e.app.pastaReferencia(id), "*.mp3")); len(m) > 0 {
		t.Errorf("áudio guardado: %v", m)
	}

	// O mesmo episódio de novo é recusado.
	var erro map[string]string
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": "https://open.spotify.com/episode/5uWIuQMlrVbKvDucC8IRvz"}, &erro); s != http.StatusConflict || erro["id"] != id {
		t.Errorf("duplicado: %d %v", s, erro)
	}
}

// Um episódio exclusivo do Spotify não está no feed: a Referência para com o aviso.
func TestPodcastSemFeed(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oembed":
			fmt.Fprint(w, `{"title":"Exclusivo"}`)
		case "/search":
			fmt.Fprintf(w, `{"results":[{"trackName":"Exclusivo","collectionName":"Podcast X","feedUrl":"%s/feed.xml"}]}`, srv.URL)
		case "/feed.xml":
			fmt.Fprint(w, `<rss><channel><title>Podcast X</title><item><title>Outro</title><enclosure url="x"/></item></channel></rss>`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	e := novoAmbiente(t)
	e.app.cfg.spotify, e.app.cfg.itunes, e.app.cfg.whisper = srv.URL, srv.URL, srv.URL
	id := e.adicionar("https://open.spotify.com/episode/AAAAAAAAAAAAAAAAAAAAAA")
	d := e.esperarEstado(id, "erro")
	if !strings.Contains(d.Erro, "exclusivo do Spotify") || d.Titulo != "Exclusivo" {
		t.Errorf("erro: %q %+v", d.Erro, d.referencia)
	}
}
