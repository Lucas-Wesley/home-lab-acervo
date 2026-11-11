package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Reels do Instagram como Referência (D40). O yt-dlp baixa o Reel (um público vem sem login; quando
// o Instagram pede login, com os cookies de uma conta secundária), o Whisper transcreve a fala e o
// ffmpeg tira quadros a cada mudança de cena (e, sem mudança, um a cada 4 s). O Claude olha os
// quadros e escreve o que aparece na tela: texto sobreposto, slides, código, o que é mostrado. O
// cru.md junta a fala, a tela e a legenda do post. O vídeo não fica guardado.

const maxQuadros = 40

// Um quadro a cada mudança de cena (no máximo um por segundo) e, sem mudança, um a cada 4 s: um
// texto que entra devagar na tela, sem corte, também é pego.
const filtroDeQuadros = `select='isnan(prev_selected_t)+gte(t-prev_selected_t\,4)+gt(scene\,0.3)*gte(t-prev_selected_t\,1)',scale=-2:720,showinfo`

var (
	idDoReel           = regexp.MustCompile(`^[A-Za-z0-9_-]{5,40}$`)
	tempoDoQuadro      = regexp.MustCompile(`pts_time:([0-9.]+)`)
	duracaoDoFfmpeg    = regexp.MustCompile(`Duration: (\d+):(\d+):([0-9.]+)`)
	errLinkDeReel      = errors.New("Cole o link de um Reel do Instagram.")
	errInstagramLogin  = errors.New("O Instagram pediu login para baixar este Reel. Ponha os cookies da conta secundária (passo a passo em docs/INSTAGRAM.md).")
	errCookiesVencidos = errors.New("O Instagram recusou os cookies da conta secundária: eles venceram ou a conta foi deslogada. Exporte de novo (docs/INSTAGRAM.md).")
)

// reelDoLink aceita o link do "Compartilhar" do Instagram (com texto junto) e devolve o link limpo.
func reelDoLink(bruto string) (string, error) {
	for _, parte := range strings.Fields(bruto) {
		u, err := url.Parse(strings.TrimSpace(parte))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
			continue
		}
		host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
		if strings.TrimPrefix(host, "m.") != "instagram.com" {
			continue
		}
		// instagram.com/reel/<id>, /reels/<id> e /<conta>/reel/<id>
		partes := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i := 0; i+1 < len(partes); i++ {
			if (partes[i] == "reel" || partes[i] == "reels") && idDoReel.MatchString(partes[i+1]) {
				return "https://www.instagram.com/reel/" + partes[i+1] + "/", nil
			}
		}
	}
	return "", errLinkDeReel
}

func ehDoInstagram(link string) bool {
	return strings.Contains(link, "instagram.com/")
}

// cookiesDoInstagram: o arquivo de cookies da conta secundária, se foi posto no servidor.
func (a *app) cookiesDoInstagram() string {
	if a.cfg.instagramCookies == "" {
		return ""
	}
	if st, err := os.Stat(a.cfg.instagramCookies); err != nil || st.Size() == 0 {
		return ""
	}
	return a.cfg.instagramCookies
}

// erroDoInstagram troca o erro do yt-dlp por um que diga o que fazer, quando é falta de login.
func (a *app) erroDoInstagram(err error) error {
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "login") && !strings.Contains(msg, "rate-limit") && !strings.Contains(msg, "401") {
		return err
	}
	if a.cookiesDoInstagram() != "" {
		return errCookiesVencidos
	}
	return errInstagramLogin
}

type infoReel struct {
	ID        string  `json:"id"`
	Conta     string  `json:"channel"`  // o @ da conta
	Autor     string  `json:"uploader"` // o nome que aparece no perfil
	Legenda   string  `json:"description"`
	Duracao   float64 `json:"duration"`
	Publicado string  `json:"upload_date"`
	URL       string  `json:"webpage_url"`
}

// tituloDoReel: o Instagram não dá título ao Reel ("Video by nasa"); até o texto ser organizado (e
// o título dele valer), vale a primeira linha da legenda, cortada no fim de uma palavra.
func tituloDoReel(info infoReel) string {
	for _, linha := range strings.FieldsFunc(info.Legenda, func(r rune) bool { return r == '\n' || r == '\r' }) {
		t := strings.TrimSpace(linha)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) <= 100 {
			return t
		}
		t = cortar(t, 100, false)
		if i := strings.LastIndex(t, " "); i > 40 {
			t = t[:i]
		}
		return strings.TrimRight(t, " ,.;:") + "…"
	}
	return "Reel de @" + info.Conta
}

// quem: "@conta (Nome)", sem repetir quando o nome é o próprio @.
func (i infoReel) quem() string {
	if i.Autor == "" || strings.EqualFold(i.Autor, i.Conta) {
		return i.canal()
	}
	return strings.TrimSpace(i.canal() + " (" + i.Autor + ")")
}

func (i infoReel) canal() string {
	if i.Conta == "" {
		return i.Autor
	}
	return "@" + i.Conta
}

type quadro struct {
	Segundos float64
	Arquivo  string
}

// tirarQuadros grava os quadros como mm-ss.jpg (o nome é o momento do vídeo) e devolve também a
// duração e se o vídeo tem áudio.
func (a *app) tirarQuadros(ctx context.Context, video, pasta string) ([]quadro, float64, bool, error) {
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return nil, 0, false, err
	}
	cmd := exec.CommandContext(ctx, a.cfg.ffmpeg, "-hide_banner", "-nostats", "-i", video,
		"-vf", filtroDeQuadros, "-fps_mode", "vfr", "-q:v", "5", filepath.Join(pasta, "q%04d.jpg"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, 0, false, fmt.Errorf("o ffmpeg não tirou os quadros do Reel: %v %s", err, cortar(string(out), 300, false))
	}
	saida := string(out)
	var duracao float64
	if m := duracaoDoFfmpeg.FindStringSubmatch(saida); m != nil {
		h, _ := strconv.Atoi(m[1])
		mi, _ := strconv.Atoi(m[2])
		s, _ := strconv.ParseFloat(m[3], 64)
		duracao = float64(h*3600+mi*60) + s
	}
	temAudio := strings.Contains(saida, "Audio: ")

	var quadros []quadro
	for i, m := range tempoDoQuadro.FindAllStringSubmatch(saida, -1) {
		s, _ := strconv.ParseFloat(m[1], 64)
		de := filepath.Join(pasta, fmt.Sprintf("q%04d.jpg", i+1))
		para := filepath.Join(pasta, strings.ReplaceAll(marcaDeTempo(int64(s*1000)), ":", "-")+".jpg")
		if err := os.Rename(de, para); err != nil {
			return nil, 0, false, err
		}
		quadros = append(quadros, quadro{s, para})
	}
	if len(quadros) == 0 {
		return nil, 0, false, errors.New("o ffmpeg não tirou nenhum quadro do Reel")
	}
	// Um Reel longo: ficam maxQuadros, espalhados pelo vídeo.
	if len(quadros) > maxQuadros {
		var ficam []quadro
		for i, q := range quadros {
			if i*maxQuadros/len(quadros) != (i+1)*maxQuadros/len(quadros) {
				ficam = append(ficam, q)
			} else {
				os.Remove(q.Arquivo)
			}
		}
		quadros = ficam
	}
	return quadros, duracao, temAudio, nil
}

// extrairReel é a etapa "extrair" de um Reel: baixa, transcreve a fala, lê a tela e grava cru.md.
func (a *app) extrairReel(ctx context.Context, ref *referencia, pasta string) error {
	if a.cfg.whisper == "" && a.cfg.whisperReserva == "" {
		return errSemTranscricao
	}
	tmp, err := os.MkdirTemp("", "reel-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if _, err := a.ytdlp(ctx, "-q", "--no-playlist", "--write-info-json",
		"-f", "bv*[height<=720]+ba/b[height<=720]/bv*+ba/b", "--merge-output-format", "mp4",
		"-o", filepath.Join(tmp, "reel.%(ext)s"), ref.URL); err != nil {
		return a.erroDoInstagram(err)
	}
	var info infoReel
	b, err := os.ReadFile(filepath.Join(tmp, "reel.info.json"))
	if err != nil || json.Unmarshal(b, &info) != nil {
		return errors.New("o yt-dlp não trouxe os dados do Reel")
	}
	video := filepath.Join(tmp, "reel.mp4")
	if _, err := os.Stat(video); err != nil {
		return errors.New("o yt-dlp não baixou o vídeo do Reel")
	}
	a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ?, canal = ? WHERE id = ?`, tituloDoReel(info), info.canal(), ref.ID)

	ver := filepath.Join(tmp, "ver")
	quadros, duracao, temAudio, err := a.tirarQuadros(ctx, video, filepath.Join(ver, "quadros"))
	if err != nil {
		return err
	}
	if info.Duracao == 0 {
		info.Duracao = duracao
	}
	dur := marcaDeTempo(int64(info.Duracao * 1000))

	// A fala. Reel só com música não tem fala: o Whisper volta vazio, e o conteúdo é o da tela.
	var trechos []trechoFalado
	quem, idioma := "sem áudio", ""
	if temAudio {
		prompt := cortar(strings.Join(strings.Fields(info.canal()+". "+info.Legenda), " "), 600, false)
		trechos, quem, _, err = a.transcreverAudio(ctx, video, tmp, "", prompt)
		if errors.Is(err, errSemFala) {
			quem, err = "sem fala", nil
		}
		if err != nil {
			return err
		}
		trechos = soFala(trechos)
		if len(trechos) > 0 {
			idioma = trechos[0].Idioma
		} else {
			quem = "sem fala"
		}
	}
	a.db.ExecContext(ctx, `UPDATE referencias SET duracao = ?, idioma = ? WHERE id = ?`, dur, idioma, ref.ID)

	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n- Reel: %s\n- URL: %s\n- Duração: %s\n", tituloDoReel(info), info.quem(), ref.URL, dur)
	if p := info.Publicado; len(p) == 8 {
		fmt.Fprintf(&s, "- Publicado: %s-%s-%s\n", p[:4], p[4:6], p[6:])
	}
	fmt.Fprintf(&s, "- Transcrição: %s\n- Quadros: %d, lidos pelo Claude\n\n## Fala\n\n", quem, len(quadros))
	if len(trechos) > 0 {
		s.WriteString(falasEmBlocos(trechos))
	} else {
		s.WriteString("(o Reel não tem fala)\n\n")
	}
	contexto := s.String()
	if info.Legenda != "" {
		contexto += "## Legenda do post\n\n" + strings.TrimSpace(info.Legenda) + "\n"
	}
	if err := os.WriteFile(filepath.Join(ver, "contexto.md"), []byte(contexto), 0o644); err != nil {
		return err
	}
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta: ver,
		regra: "ver-quadros.md",
		instrucao: "Os quadros do Reel estão em quadros/ (o nome de cada um é o momento do vídeo: 00-04.jpg é [00:04]) e contexto.md tem a fala e a legenda do post. " +
			"Olhe todos os quadros e escreva tela.md nesta pasta. Responda só 'ok' ao terminar.",
		ferramentas: ferramentasEscrita,
	})
	if err != nil {
		return err
	}
	tela, err := os.ReadFile(filepath.Join(ver, "tela.md"))
	if err != nil || strings.TrimSpace(string(tela)) == "" {
		return errors.New("o Claude não escreveu o que aparece na tela do Reel")
	}

	s.WriteString("## Na tela\n\n" + strings.TrimSpace(string(tela)) + "\n\n")
	if info.Legenda != "" {
		s.WriteString("## Legenda do post\n\n" + strings.TrimSpace(info.Legenda) + "\n")
	}
	return os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(s.String()), 0o644)
}

// soFala tira o que o Whisper inventou sobre música ou silêncio, e trecho sem palavra ("!", "♪").
func soFala(trechos []trechoFalado) []trechoFalado {
	var fala []trechoFalado
	for _, t := range trechos {
		if !t.SemFala && strings.IndexFunc(t.Texto, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
			fala = append(fala, t)
		}
	}
	return fala
}

func notaDoReel(ref *referencia) string {
	return fmt.Sprintf(" Esta Referência é um Reel do Instagram (%s): as marcas [mm:ss] são do vídeo, e parte do conteúdo foi lida na tela.",
		strings.TrimSpace(ref.Canal+" · "+ref.URL))
}
