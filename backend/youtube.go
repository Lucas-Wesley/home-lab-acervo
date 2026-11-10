package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var idDeVideo = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// videoDoYouTube aceita os formatos de link mais comuns e devolve o id do vídeo.
// O texto pode vir do "Compartilhar" do celular, com o título junto: vale o primeiro link de vídeo.
func videoDoYouTube(bruto string) (string, error) {
	for _, parte := range strings.Fields(bruto) {
		if id, err := videoDoLink(parte); err == nil {
			return id, nil
		}
	}
	return "", errLinkInvalido
}

func videoDoLink(bruto string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(bruto))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", errLinkInvalido
	}
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	host = strings.TrimPrefix(host, "m.")
	var id string
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
	case "youtube.com", "music.youtube.com":
		if u.Path == "/watch" {
			id = u.Query().Get("v")
		} else if partes := strings.Split(strings.Trim(u.Path, "/"), "/"); len(partes) == 2 &&
			(partes[0] == "shorts" || partes[0] == "live" || partes[0] == "embed") {
			id = partes[1]
		}
	}
	if !idDeVideo.MatchString(id) {
		return "", errLinkInvalido
	}
	return id, nil
}

var (
	errLinkInvalido = errors.New("link do YouTube inválido")
	errSemLegenda   = errors.New("Não foi possível extrair a legenda deste vídeo. O processo não continuou.")
)

type infoVideo struct {
	ID          string                     `json:"id"`
	Titulo      string                     `json:"title"`
	Canal       string                     `json:"channel"`
	Duracao     string                     `json:"duration_string"`
	Idioma      string                     `json:"language"`
	URL         string                     `json:"webpage_url"`
	Publicado   string                     `json:"upload_date"`
	Legendas    map[string]json.RawMessage `json:"subtitles"`
	Automaticas map[string]json.RawMessage `json:"automatic_captions"`
}

// escolhaDeLegenda: manual no idioma do vídeo → automática original (-orig) →
// manual em português → qualquer manual → automática no idioma do vídeo.
func escolhaDeLegenda(info infoVideo) (lingua string, automatica bool, ok bool) {
	base := func(l string) string { return strings.SplitN(l, "-", 2)[0] }
	manuais := chaves(info.Legendas)
	autos := chaves(info.Automaticas)
	video := info.Idioma

	if video != "" {
		for _, l := range manuais {
			if l == video || base(l) == base(video) {
				return l, false, true
			}
		}
	}
	for _, l := range autos {
		if strings.HasSuffix(l, "-orig") {
			return l, true, true
		}
	}
	for _, l := range manuais {
		if base(l) == "pt" {
			return l, false, true
		}
	}
	if len(manuais) > 0 {
		return manuais[0], false, true
	}
	if video != "" {
		for _, l := range autos {
			if l == video {
				return l, true, true
			}
		}
	}
	return "", false, false
}

func chaves(m map[string]json.RawMessage) []string {
	var lista []string
	for k := range m {
		if k != "live_chat" {
			lista = append(lista, k)
		}
	}
	sort.Strings(lista)
	return lista
}

// idiomaBase devolve "pt", "en"... a partir de "pt-BR", "en-orig".
func idiomaBase(l string) string {
	return strings.ToLower(strings.SplitN(l, "-", 2)[0])
}

func (a *app) ytdlp(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.cfg.ytdlp, append(append([]string{}, a.cfg.ytdlpExtra...), args...)...)
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(erros.String())
		if i := strings.LastIndex(msg, "ERROR:"); i >= 0 {
			msg = msg[i:]
		}
		return nil, fmt.Errorf("yt-dlp falhou: %s", primeiraLinha(msg, err.Error()))
	}
	return saida.Bytes(), nil
}

func primeiraLinha(s, alternativa string) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if s == "" {
		return alternativa
	}
	return s
}

// baixarLegenda grava info.json, legenda.<lingua>.json3 e cru.md na pasta da Referência.
func (a *app) baixarLegenda(ctx context.Context, link, pasta string) (infoVideo, string, error) {
	var info infoVideo
	bruto, err := a.ytdlp(ctx, "-J", "--skip-download", "--no-playlist", link)
	if err != nil {
		return info, "", err
	}
	if err := json.Unmarshal(bruto, &info); err != nil {
		return info, "", fmt.Errorf("resposta do yt-dlp ilegível: %w", err)
	}
	if err := os.WriteFile(filepath.Join(pasta, "info.json"), bruto, 0o644); err != nil {
		return info, "", err
	}

	lingua, automatica, ok := escolhaDeLegenda(info)
	if !ok {
		return info, "", errSemLegenda
	}
	tipo := "--write-subs"
	if automatica {
		tipo = "--write-auto-subs"
	}
	if _, err := a.ytdlp(ctx, "--skip-download", "--no-playlist", tipo, "--sub-langs", lingua,
		"--sub-format", "json3", "-o", filepath.Join(pasta, "legenda"), link); err != nil {
		return info, "", err
	}
	legenda, err := os.ReadFile(filepath.Join(pasta, "legenda."+lingua+".json3"))
	if err != nil {
		return info, "", errSemLegenda
	}
	cru, err := legendaParaCru(legenda, info)
	if err != nil {
		return info, "", err
	}
	return info, lingua, os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(cru), 0o644)
}

const blocoMs = 30_000 // uma marca de tempo a cada ~30 s

var espacos = regexp.MustCompile(`\s+`)

// legendaParaCru transforma a legenda json3 em texto corrido com marcas [mm:ss].
func legendaParaCru(legenda []byte, info infoVideo) (string, error) {
	var dados struct {
		Events []struct {
			Inicio  int64 `json:"tStartMs"`
			Anexado int   `json:"aAppend"`
			Segs    []struct {
				Texto string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(legenda, &dados); err != nil {
		return "", fmt.Errorf("legenda ilegível: %w", err)
	}

	type bloco struct {
		inicio int64
		texto  []string
	}
	var blocos []bloco
	for _, ev := range dados.Events {
		if len(ev.Segs) == 0 || ev.Anexado != 0 {
			continue
		}
		var b strings.Builder
		for _, s := range ev.Segs {
			b.WriteString(s.Texto)
		}
		texto := strings.TrimSpace(strings.ReplaceAll(b.String(), "\n", " "))
		if texto == "" {
			continue
		}
		if len(blocos) == 0 || ev.Inicio-blocos[len(blocos)-1].inicio >= blocoMs {
			blocos = append(blocos, bloco{inicio: ev.Inicio})
		}
		atual := &blocos[len(blocos)-1]
		atual.texto = append(atual.texto, texto)
	}
	if len(blocos) == 0 {
		return "", errSemLegenda
	}

	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Canal: %s\n- URL: %s\n- Duração: %s\n- Publicado: %s\n\n",
		info.Titulo, info.Canal, info.URL, info.Duracao, info.Publicado)
	for _, b := range blocos {
		fmt.Fprintf(&s, "[%s] %s\n\n", marcaDeTempo(b.inicio), espacos.ReplaceAllString(strings.Join(b.texto, " "), " "))
	}
	return s.String(), nil
}

func marcaDeTempo(ms int64) string {
	s := ms / 1000
	h, m, seg := s/3600, s%3600/60, s%60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, seg)
	}
	return fmt.Sprintf("%02d:%02d", m, seg)
}

// transcreverVideo: vídeo sem legenda (D37) é transcrito inteiro pelo Whisper (o do Mac mini; com o
// Mac fora, o do servidor). O áudio é baixado pelo yt-dlp e não fica guardado.
func (a *app) transcreverVideo(ctx context.Context, info infoVideo, link, pasta string) error {
	tmp, err := os.MkdirTemp("", "video-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if _, err := a.ytdlp(ctx, "-q", "--no-playlist", "-f", "bestaudio/best", "-o", filepath.Join(tmp, "audio.%(ext)s"), link); err != nil {
		return err
	}
	achados, _ := filepath.Glob(filepath.Join(tmp, "audio.*"))
	if len(achados) == 0 {
		return errors.New("o yt-dlp não baixou o áudio do vídeo")
	}
	trechos, quem, _, err := a.transcreverAudio(ctx, achados[0], tmp, info.Idioma, cortar(info.Titulo+". "+info.Canal, 300, false))
	if err != nil {
		return err
	}
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Canal: %s\n- URL: %s\n- Duração: %s\n- Publicado: %s\n- Transcrição: %s (o vídeo não tem legenda)\n\n",
		info.Titulo, info.Canal, info.URL, info.Duracao, info.Publicado, quem)
	s.WriteString(falasEmBlocos(trechos))
	return os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(s.String()), 0o644)
}
