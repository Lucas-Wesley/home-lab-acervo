package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	nethtml "golang.org/x/net/html"
)

// Podcasts como Referência (D39). O Spotify protege o áudio, mas quase todo podcast publica os
// episódios também num feed RSS aberto (o mesmo que o Apple Podcasts lê). O servidor descobre pelo
// link qual é o episódio, acha o feed pela busca do Apple Podcasts, baixa o MP3 e manda transcrever
// no Whisper do Mac mini (whisper-mac/ no homelab; com o Mac fora, no Whisper do servidor). O cru.md
// tem as marcas [mm:ss], como o de um vídeo, e termina com a descrição do episódio (livros, links,
// imagens), que vira "Referências do episódio" no fim do texto. Episódios exclusivos do Spotify não
// têm feed aberto: esses não dá.

const (
	maxFeed         = 30 << 20
	maxAudio        = 1 << 30
	modeloDoMac     = "whisper-large-v3"
	modeloDaReserva = "deepdml/faster-whisper-large-v3-turbo-ct2"
)

// O áudio vai ao Whisper em pedaços de 10 min (variável só para o teste usar pedaços curtos).
var segundosPorPedaco = 600

var (
	clienteDoPodcast     = &http.Client{Timeout: 30 * time.Second}
	clienteDoAudio       = &http.Client{Timeout: 30 * time.Minute}
	clienteDoWhisper     = &http.Client{Timeout: 40 * time.Minute} // a reserva leva ~1,2× o tempo do áudio
	idDoSpotify          = regexp.MustCompile(`^[A-Za-z0-9]{22}$`)
	idDaApple            = regexp.MustCompile(`/id(\d+)`)
	soDigitos            = regexp.MustCompile(`^\d+$`)
	errSemFeed           = errors.New("Não achei este episódio no feed aberto do podcast. Ele pode ser exclusivo do Spotify, e aí o áudio não pode ser baixado.")
	errSemTranscricao    = errors.New("A transcrição não está configurada (WHISPER_URL).")
	errLinkDePodcast     = errors.New("Cole o link de um episódio do Spotify ou do Apple Podcasts.")
	espacosRepetidos     = regexp.MustCompile(`[ \t]+`)
	linhasEmBrancoDemais = regexp.MustCompile(`\n{3,}`)
)

// linkDePodcast é o episódio que o link aponta.
type linkDePodcast struct {
	Fonte     string // spotify ou apple
	ID        string // do episódio
	PodcastID string // só na Apple
	URL       string // o link limpo, guardado na Referência
}

// episodioDoLink aceita o link do "Compartilhar" do Spotify ou do Apple Podcasts (com texto junto).
func episodioDoLink(bruto string) (linkDePodcast, error) {
	for _, parte := range strings.Fields(bruto) {
		u, err := url.Parse(strings.TrimRight(parte, ".,;)"))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			continue
		}
		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		switch host {
		case "open.spotify.com":
			partes := strings.Split(strings.Trim(u.Path, "/"), "/")
			// /episode/<id> ou /intl-pt/episode/<id>
			for i := 0; i+1 < len(partes); i++ {
				if partes[i] == "episode" && idDoSpotify.MatchString(partes[i+1]) {
					return linkDePodcast{Fonte: "spotify", ID: partes[i+1], URL: "https://open.spotify.com/episode/" + partes[i+1]}, nil
				}
			}
		case "podcasts.apple.com":
			m := idDaApple.FindStringSubmatch(u.Path)
			ep := u.Query().Get("i")
			if m != nil && soDigitos.MatchString(ep) {
				return linkDePodcast{Fonte: "apple", ID: ep, PodcastID: m[1], URL: "https://podcasts.apple.com/podcast/id" + m[1] + "?i=" + ep}, nil
			}
		}
	}
	return linkDePodcast{}, errLinkDePodcast
}

// episodio junta o que se sabe do episódio: do link, do feed e da transcrição.
type episodio struct {
	Titulo     string `json:"titulo"`
	Podcast    string `json:"podcast"`
	URL        string `json:"url"`
	Feed       string `json:"feed"`
	Audio      string `json:"audio"`
	Publicado  string `json:"publicado"`
	Duracao    string `json:"duracao"`
	Idioma     string `json:"idioma"`
	Capa       string `json:"capa"`
	Descricao  string `json:"descricao"`  // em Markdown
	Transcrito string `json:"transcrito"` // qual Whisper transcreveu
}

func (a *app) pegar(ctx context.Context, cliente *http.Client, endereco string, limite int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Acervo-homelab")
	res, err := cliente.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s respondeu %d", req.URL.Host, res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, limite))
}

// doSpotify pega o título do episódio pelo oEmbed, a API pública do Spotify para prévias de link
// (a página do episódio é montada no navegador e não traz os dados).
func (a *app) doSpotify(ctx context.Context, id string) (string, error) {
	q := url.Values{"url": {"https://open.spotify.com/episode/" + id}}
	b, err := a.pegar(ctx, clienteDoPodcast, a.cfg.spotify+"/oembed?"+q.Encode(), 1<<20)
	if err != nil {
		return "", fmt.Errorf("O Spotify não respondeu sobre o episódio: %v", err)
	}
	var r struct {
		Titulo string `json:"title"`
	}
	if json.Unmarshal(b, &r) != nil || strings.TrimSpace(r.Titulo) == "" {
		return "", errors.New("O Spotify não disse o título do episódio.")
	}
	return strings.TrimSpace(r.Titulo), nil
}

type resultadoDaApple struct {
	Tipo      string `json:"wrapperType"`
	Kind      string `json:"kind"`
	Podcast   string `json:"collectionName"`
	Feed      string `json:"feedUrl"`
	TrackID   int64  `json:"trackId"`
	Titulo    string `json:"trackName"`
	Audio     string `json:"episodeUrl"`
	Publicado string `json:"releaseDate"`
}

func (a *app) apple(ctx context.Context, caminho string, q url.Values) ([]resultadoDaApple, error) {
	b, err := a.pegar(ctx, clienteDoPodcast, a.cfg.itunes+caminho+"?"+q.Encode(), 10<<20)
	if err != nil {
		return nil, fmt.Errorf("A busca do Apple Podcasts falhou: %v", err)
	}
	var r struct {
		Resultados []resultadoDaApple `json:"results"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("Resposta do Apple Podcasts ilegível: %v", err)
	}
	return r.Resultados, nil
}

// feedDoEpisodio acha o podcast e o feed pelo título do episódio, na busca de episódios do Apple
// Podcasts: primeiro o título igual (sem acento nem maiúscula), depois um contendo o outro.
func (a *app) feedDoEpisodio(ctx context.Context, titulo string) (feed, podcast string, err error) {
	rs, err := a.apple(ctx, "/search", url.Values{"term": {titulo}, "media": {"podcast"}, "entity": {"podcastEpisode"}, "country": {"BR"}, "limit": {"25"}})
	if err != nil {
		return "", "", err
	}
	alvo := slug(titulo)
	for _, igual := range []bool{true, false} {
		for _, r := range rs {
			t := slug(r.Titulo)
			if r.Feed != "" && t != "" && (t == alvo || !igual && (strings.Contains(t, alvo) || strings.Contains(alvo, t))) {
				return r.Feed, r.Podcast, nil
			}
		}
	}
	return "", "", errSemFeed
}

type itemDoFeed struct {
	Titulo    string `xml:"title"`
	Guid      string `xml:"guid"`
	Publicado string `xml:"pubDate"`
	Descricao string `xml:"description"`
	Conteudo  string `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Resumo    string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	Duracao   string `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
	Imagem    struct {
		Href string `xml:"href,attr"`
	} `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Anexo struct {
		URL  string `xml:"url,attr"`
		Tipo string `xml:"type,attr"`
	} `xml:"enclosure"`
}

type feedDoPodcast struct {
	Canal struct {
		Titulo string `xml:"title"`
		Idioma string `xml:"language"`
		Imagem struct {
			Href string `xml:"href,attr"`
		} `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
		Itens []itemDoFeed `xml:"item"`
	} `xml:"channel"`
}

func (a *app) lerFeed(ctx context.Context, endereco string) (*feedDoPodcast, error) {
	b, err := a.pegar(ctx, clienteDoPodcast, endereco, maxFeed)
	if err != nil {
		return nil, fmt.Errorf("O feed do podcast não abriu: %v", err)
	}
	var f feedDoPodcast
	d := xml.NewDecoder(bytes.NewReader(b))
	d.Strict = false
	d.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	if err := d.Decode(&f); err != nil {
		return nil, fmt.Errorf("O feed do podcast está ilegível: %v", err)
	}
	return &f, nil
}

// itemDoEpisodio acha o episódio no feed pelo título: igual, ou um contendo o outro.
func itemDoEpisodio(f *feedDoPodcast, titulo string) (*itemDoFeed, bool) {
	alvo := slug(titulo)
	if alvo == "" {
		return nil, false
	}
	for i := range f.Canal.Itens {
		if slug(f.Canal.Itens[i].Titulo) == alvo {
			return &f.Canal.Itens[i], true
		}
	}
	for i := range f.Canal.Itens {
		t := slug(f.Canal.Itens[i].Titulo)
		if t != "" && (strings.Contains(t, alvo) || strings.Contains(alvo, t)) {
			return &f.Canal.Itens[i], true
		}
	}
	return nil, false
}

// acharEpisodio vai do link ao episódio no feed aberto.
func (a *app) acharEpisodio(ctx context.Context, l linkDePodcast) (episodio, error) {
	ep := episodio{URL: l.URL}
	var titulo, feed string
	switch l.Fonte {
	case "spotify":
		t, err := a.doSpotify(ctx, l.ID)
		if err != nil {
			return ep, err
		}
		titulo, ep.Titulo = t, t
		if feed, ep.Podcast, err = a.feedDoEpisodio(ctx, t); err != nil {
			return ep, err
		}
	case "apple":
		rs, err := a.apple(ctx, "/lookup", url.Values{"id": {l.PodcastID}, "entity": {"podcastEpisode"}, "limit": {"300"}, "country": {"BR"}})
		if err != nil {
			return ep, err
		}
		for _, r := range rs {
			if r.Kind == "podcast" || (r.Tipo == "track" && r.Feed != "") {
				feed, ep.Podcast = r.Feed, r.Podcast
			}
			if strconv.FormatInt(r.TrackID, 10) == l.ID {
				titulo, ep.Titulo = r.Titulo, r.Titulo
			}
		}
		if feed == "" || titulo == "" {
			return ep, errSemFeed
		}
	}
	ep.Feed = feed
	f, err := a.lerFeed(ctx, feed)
	if err != nil {
		return ep, err
	}
	item, ok := itemDoEpisodio(f, titulo)
	if !ok || item.Anexo.URL == "" {
		return ep, errSemFeed
	}
	ep.Titulo = strings.TrimSpace(item.Titulo)
	if ep.Podcast == "" {
		ep.Podcast = strings.TrimSpace(f.Canal.Titulo)
	}
	ep.Audio = item.Anexo.URL
	ep.Idioma = strings.TrimSpace(f.Canal.Idioma)
	ep.Capa = primeiro(item.Imagem.Href, f.Canal.Imagem.Href)
	if t, err := time.Parse(time.RFC1123Z, strings.TrimSpace(item.Publicado)); err == nil {
		ep.Publicado = t.Format("2006-01-02")
	} else if t, err := time.Parse(time.RFC1123, strings.TrimSpace(item.Publicado)); err == nil {
		ep.Publicado = t.Format("2006-01-02")
	}
	ep.Duracao = duracaoDoFeed(item.Duracao)
	ep.Descricao = descricaoEmMarkdown(primeiro(item.Conteudo, item.Descricao, item.Resumo))
	return ep, nil
}

// duracaoDoFeed: "2265", "37:45" ou "00:37:45" → "37:45" (como o yt-dlp mostra a de um vídeo).
func duracaoDoFeed(s string) string {
	s = strings.TrimSpace(s)
	total := 0
	for _, p := range strings.Split(s, ":") {
		n, err := strconv.Atoi(p)
		if err != nil {
			return ""
		}
		total = total*60 + n
	}
	if total == 0 {
		return ""
	}
	return marcaDeTempo(int64(total) * 1000)
}

// descricaoEmMarkdown: a descrição do feed vem em HTML (ou texto puro). Links, listas, negrito e
// imagens passam para Markdown; o resto vira parágrafos.
func descricaoEmMarkdown(bruto string) string {
	bruto = strings.TrimSpace(bruto)
	if bruto == "" {
		return ""
	}
	if !strings.Contains(bruto, "<") {
		return limparLinhas(html.UnescapeString(bruto))
	}
	doc, err := nethtml.Parse(strings.NewReader("<body>" + bruto + "</body>"))
	if err != nil {
		return limparLinhas(bruto)
	}
	var s strings.Builder
	var andar func(*nethtml.Node)
	andar = func(n *nethtml.Node) {
		switch {
		case n.Type == nethtml.TextNode:
			s.WriteString(espacosRepetidos.ReplaceAllString(strings.ReplaceAll(n.Data, "\n", " "), " "))
			return
		case n.Type != nethtml.ElementNode && n.Type != nethtml.DocumentNode:
			return
		}
		switch n.Data {
		case "br":
			s.WriteString("\n")
			return
		case "img":
			if src := atributo(n, "src"); src != "" {
				fmt.Fprintf(&s, "\n\n![%s](%s)\n\n", atributo(n, "alt"), src)
			}
			return
		case "a":
			href := atributo(n, "href")
			texto := strings.TrimSpace(textoDoNo(n))
			switch {
			case href == "" || strings.HasPrefix(href, "javascript:"):
				s.WriteString(texto)
			case texto == "" || texto == href:
				fmt.Fprintf(&s, "<%s>", href)
			default:
				fmt.Fprintf(&s, "[%s](%s)", texto, href)
			}
			return
		case "p", "div", "ul", "ol", "h1", "h2", "h3", "h4", "blockquote":
			s.WriteString("\n\n")
			defer s.WriteString("\n\n")
		case "li":
			s.WriteString("\n- ")
		case "strong", "b":
			s.WriteString("**")
			defer s.WriteString("**")
		case "em", "i":
			s.WriteString("_")
			defer s.WriteString("_")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			andar(c)
		}
	}
	andar(doc)
	return limparLinhas(s.String())
}

func limparLinhas(s string) string {
	linhas := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	for i, l := range linhas {
		linhas[i] = strings.TrimSpace(l)
	}
	return strings.TrimSpace(linhasEmBrancoDemais.ReplaceAllString(strings.Join(linhas, "\n"), "\n\n"))
}

// trechoFalado é um pedaço da transcrição, com o tempo desde o começo do episódio.
type trechoFalado struct {
	Inicio float64
	Texto  string
}

// extrairPodcast é a etapa "extrair" de um podcast: acha, baixa, transcreve e grava cru.md.
func (a *app) extrairPodcast(ctx context.Context, ref *referencia, pasta string) error {
	if a.cfg.whisper == "" && a.cfg.whisperReserva == "" {
		return errSemTranscricao
	}
	l, err := episodioDoLink(ref.URL)
	if err != nil {
		return err
	}
	ep, err := a.acharEpisodio(ctx, l)
	if ep.Titulo != "" {
		a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ?, canal = ?, duracao = ?, idioma = ? WHERE id = ?`,
			cortar(ep.Titulo, 200, false), ep.Podcast, ep.Duracao, ep.Idioma, ref.ID)
		ref.Idioma = ep.Idioma
	}
	if err != nil {
		return err
	}

	// O áudio fica numa pasta temporária: depois da transcrição, só o endereço dele é guardado.
	tmp, err := os.MkdirTemp("", "podcast-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	audio := filepath.Join(tmp, "audio")
	if err := a.baixarAudio(ctx, ep.Audio, audio); err != nil {
		return err
	}
	pedacos, err := a.cortarAudio(ctx, audio, tmp)
	if err != nil {
		return err
	}
	prompt := promptDoEpisodio(ep)
	var trechos []trechoFalado
	for i, p := range pedacos {
		ts, quem, err := a.transcrever(ctx, p, ep.Idioma, prompt)
		if err != nil {
			return fmt.Errorf("A transcrição do pedaço %d de %d falhou: %v", i+1, len(pedacos), err)
		}
		ep.Transcrito = quem
		for _, t := range ts {
			t.Inicio += float64(i * segundosPorPedaco)
			trechos = append(trechos, t)
		}
	}
	if len(trechos) == 0 {
		return errors.New("A transcrição voltou vazia: o áudio pode estar sem fala.")
	}
	if ep.Duracao == "" {
		ep.Duracao = marcaDeTempo(int64(len(pedacos)*segundosPorPedaco) * 1000)
	}
	info, _ := json.MarshalIndent(ep, "", "  ")
	if err := os.WriteFile(filepath.Join(pasta, "info.json"), info, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(podcastParaCru(ep, trechos)), 0o644)
}

func (a *app) baixarAudio(ctx context.Context, endereco, destino string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Acervo-homelab")
	res, err := clienteDoAudio.Do(req)
	if err != nil {
		return fmt.Errorf("O áudio do episódio não baixou: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("O áudio do episódio respondeu %d.", res.StatusCode)
	}
	f, err := os.Create(destino)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(res.Body, maxAudio)); err != nil {
		return fmt.Errorf("O áudio do episódio não baixou: %v", err)
	}
	return f.Close()
}

// cortarAudio faz pedaços de 10 min, mono a 16 kHz em FLAC (o que o Whisper usa; ~8 MB cada).
func (a *app) cortarAudio(ctx context.Context, audio, pasta string) ([]string, error) {
	cmd := exec.CommandContext(ctx, a.cfg.ffmpeg, "-v", "error", "-i", audio, "-vn", "-ac", "1", "-ar", "16000",
		"-f", "segment", "-segment_time", strconv.Itoa(segundosPorPedaco), "-c:a", "flac",
		filepath.Join(pasta, "pedaco-%03d.flac"))
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("Não consegui ler o áudio do episódio: %v %s", err, cortar(string(out), 300, false))
	}
	pedacos, _ := filepath.Glob(filepath.Join(pasta, "pedaco-*.flac"))
	sort.Strings(pedacos)
	if len(pedacos) == 0 {
		return nil, errors.New("O áudio do episódio está vazio.")
	}
	return pedacos, nil
}

// promptDoEpisodio ajuda o Whisper com os nomes: o podcast, o título e o começo da descrição.
func promptDoEpisodio(ep episodio) string {
	desc := regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)|<[^>]+>|https?://\S+`).ReplaceAllString(ep.Descricao, "$1")
	return cortar(strings.Join(strings.Fields(ep.Podcast+". "+ep.Titulo+". "+desc), " "), 600, false)
}

// transcrever manda um pedaço ao Whisper do Mac; se o Mac não responde, ao Whisper do servidor.
func (a *app) transcrever(ctx context.Context, arquivo, idioma, prompt string) ([]trechoFalado, string, error) {
	if a.cfg.whisper != "" && (a.cfg.whisperReserva == "" || a.noAr(ctx, a.cfg.whisper)) {
		ts, err := a.pedirTranscricao(ctx, a.cfg.whisper, modeloDoMac, arquivo, idioma, prompt)
		if err == nil {
			return ts, "Whisper large-v3 (Mac mini)", nil
		}
		var erroDeRede net.Error
		if a.cfg.whisperReserva == "" || ctx.Err() != nil || !(errors.As(err, &erroDeRede) || errors.Is(err, errServicoFora)) {
			return nil, "", err
		}
	}
	ts, err := a.pedirTranscricao(ctx, a.cfg.whisperReserva, modeloDaReserva, arquivo, idioma, prompt)
	return ts, "Whisper large-v3-turbo (servidor)", err
}

var errServicoFora = errors.New("serviço de transcrição fora do ar")

// noAr: o Mac desligado não responde, e a conexão poderia ficar pendurada; pergunta antes, rápido.
func (a *app) noAr(ctx context.Context, base string) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/saude", nil)
	if err != nil {
		return false
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	res.Body.Close()
	return res.StatusCode == http.StatusOK
}

func (a *app) pedirTranscricao(ctx context.Context, base, modelo, arquivo, idioma, prompt string) ([]trechoFalado, error) {
	f, err := os.Open(arquivo)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	parte, _ := mw.CreateFormFile("file", filepath.Base(arquivo))
	if _, err := io.Copy(parte, f); err != nil {
		return nil, err
	}
	mw.WriteField("model", modelo)
	mw.WriteField("response_format", "verbose_json")
	if idioma = strings.ToLower(strings.SplitN(idioma, "-", 2)[0]); idioma != "" {
		mw.WriteField("language", idioma)
	}
	if prompt != "" {
		mw.WriteField("prompt", prompt)
	}
	mw.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSuffix(base, "/")+"/v1/audio/transcriptions", &corpo)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := clienteDoWhisper.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if res.StatusCode == http.StatusBadGateway || res.StatusCode == http.StatusServiceUnavailable {
		return nil, errServicoFora
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("o Whisper respondeu %d: %s", res.StatusCode, cortar(string(b), 200, false))
	}
	var r struct {
		Segmentos []struct {
			Inicio float64 `json:"start"`
			Texto  string  `json:"text"`
		} `json:"segments"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("resposta do Whisper ilegível: %v", err)
	}
	var ts []trechoFalado
	for _, s := range r.Segmentos {
		if t := strings.TrimSpace(s.Texto); t != "" {
			ts = append(ts, trechoFalado{Inicio: s.Inicio, Texto: t})
		}
	}
	return ts, nil
}

// podcastParaCru: o cabeçalho, a fala com uma marca [mm:ss] a cada ~30 s (como a de um vídeo) e a
// descrição do episódio no fim.
func podcastParaCru(ep episodio, trechos []trechoFalado) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Podcast: %s\n- URL: %s\n- Duração: %s\n", ep.Titulo, ep.Podcast, ep.URL, ep.Duracao)
	if ep.Publicado != "" {
		fmt.Fprintf(&s, "- Publicado: %s\n", ep.Publicado)
	}
	if ep.Capa != "" {
		fmt.Fprintf(&s, "- Capa: %s\n", ep.Capa)
	}
	fmt.Fprintf(&s, "- Transcrição: %s\n\n", ep.Transcrito)
	inicio := -1.0
	var bloco []string
	fechar := func() {
		if len(bloco) > 0 {
			fmt.Fprintf(&s, "[%s] %s\n\n", marcaDeTempo(int64(inicio*1000)), strings.Join(bloco, " "))
		}
		bloco = nil
	}
	for _, t := range trechos {
		if inicio < 0 || t.Inicio-inicio >= blocoMs/1000 {
			fechar()
			inicio = t.Inicio
		}
		bloco = append(bloco, t.Texto)
	}
	fechar()
	if ep.Descricao != "" {
		fmt.Fprintf(&s, "## Descrição do episódio\n\n%s\n", ep.Descricao)
	}
	return s.String()
}

func notaDoPodcast(ref *referencia) string {
	return fmt.Sprintf(" Esta Referência é um episódio de podcast (%s), transcrito do áudio: as marcas [mm:ss] são do episódio.",
		strings.TrimSpace(ref.Canal+" · "+ref.URL))
}
