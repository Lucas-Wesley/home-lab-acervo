package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Documentários do Streaming (D50). O Streaming manda cada filme ou episódio de documentário com o
// que se sabe dele (TMDB) e, para o texto, a legenda no idioma original (quando há uma confiável) ou
// o áudio original em mono. A legenda vira a transcrição na hora; o áudio vai ao Whisper, como um
// podcast, e fica guardado (comprimido) para as Dúvidas ouvirem de novo. Os episódios de uma série
// ficam ligados pela chave do TMDB: a lista os agrupa, cada episódio é organizado sabendo o que os
// anteriores já explicaram, e a sugestão de Tema procura o Tema onde os outros entraram.

type documentario struct {
	Chave           string `json:"chave"` // filme/123 ou serie/456 no TMDB: junta os episódios
	Nome            string `json:"nome"`
	Original        string `json:"original,omitempty"`
	Ano             string `json:"ano,omitempty"`
	Sinopse         string `json:"sinopse,omitempty"`
	Idioma          string `json:"idioma,omitempty"` // o idioma original, do TMDB
	Temporada       int    `json:"temporada,omitempty"`
	Episodio        int    `json:"episodio,omitempty"`
	Episodios       int    `json:"episodios,omitempty"` // quantos a temporada tem
	TituloEpisodio  string `json:"titulo_episodio,omitempty"`
	SinopseEpisodio string `json:"sinopse_episodio,omitempty"`
	Duracao         int    `json:"duracao,omitempty"` // segundos
	Capa            string `json:"capa,omitempty"`
	Link            string `json:"link,omitempty"` // a página do título no Streaming
	TMDB            string `json:"tmdb"`           // a página do filme ou do episódio no TMDB (é a URL da Referência)
	Origem          string `json:"origem,omitempty"`
	Texto           string `json:"texto"`             // "legenda" ou "audio"
	Legenda         string `json:"legenda,omitempty"` // qual legenda veio
	IdiomaDoTexto   string `json:"idioma_texto,omitempty"`
	Transcrito      string `json:"transcrito,omitempty"` // quem transcreveu o áudio (preenchido aqui)
}

var (
	reChaveDoDocumentario = regexp.MustCompile(`^(filme|serie)/\d+$`)
	reLinkDoTMDB          = regexp.MustCompile(`^https://www\.themoviedb\.org/(movie|tv)/\d+(/season/\d+/episode/\d+)?$`)
)

const maxEntradaDoDocumentario = 400 << 20

func (d documentario) ehEpisodio() bool { return d.Episodio > 0 }

// titulo: o do filme, ou "Série, episódio 2: Nome do episódio".
func (d documentario) titulo() string {
	if !d.ehEpisodio() {
		return d.Nome
	}
	if d.TituloEpisodio == "" {
		return fmt.Sprintf("%s, episódio %d", d.Nome, d.Episodio)
	}
	return fmt.Sprintf("%s, episódio %d: %s", d.Nome, d.Episodio, d.TituloEpisodio)
}

func (d documentario) rotuloDoEpisodio() string {
	if !d.ehEpisodio() {
		return ""
	}
	s := fmt.Sprintf("T%d E%d", d.Temporada, d.Episodio)
	if d.Episodios > 0 {
		s += fmt.Sprintf(" de %d", d.Episodios)
	}
	return s
}

func documentarioDe(ref *referencia) (documentario, error) {
	var d documentario
	if len(ref.Documentario) == 0 || string(ref.Documentario) == "null" {
		return d, errors.New("esta Referência não é um documentário")
	}
	err := json.Unmarshal(ref.Documentario, &d)
	return d, err
}

// handleCriarDocumentario: POST /api/documentarios, multipart com "dados" (JSON) e o arquivo
// "legenda" (SRT) ou "audio". O mesmo episódio de novo devolve 409 com o id, menos quando o
// anterior deu erro: aí ele é refeito com a entrada nova.
func (a *app) handleCriarDocumentario(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxEntradaDoDocumentario)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		erroJSON(w, http.StatusBadRequest, "Pedido inválido: "+err.Error())
		return
	}
	defer r.MultipartForm.RemoveAll()
	var d documentario
	if err := json.Unmarshal([]byte(r.FormValue("dados")), &d); err != nil {
		erroJSON(w, http.StatusBadRequest, "Os dados do documentário vieram ilegíveis.")
		return
	}
	d.Nome = strings.TrimSpace(d.Nome)
	if !reChaveDoDocumentario.MatchString(d.Chave) || d.Nome == "" || !reLinkDoTMDB.MatchString(d.TMDB) {
		erroJSON(w, http.StatusBadRequest, "Faltam a chave, o nome ou o link do TMDB do documentário.")
		return
	}
	campo := map[string]string{"legenda": "legenda", "audio": "audio"}[d.Texto]
	if campo == "" {
		erroJSON(w, http.StatusBadRequest, "O texto do documentário vem da legenda ou do áudio.")
		return
	}
	arquivo, cabecalho, err := r.FormFile(campo)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Falta o arquivo da "+map[string]string{"legenda": "legenda", "audio": "áudio"}[campo]+".")
		return
	}
	defer arquivo.Close()

	ctx := r.Context()
	var existente, estado string
	a.db.QueryRowContext(ctx, `SELECT id, estado FROM referencias WHERE tipo = 'documentario' AND url = ? AND estado != 'descartado'
		ORDER BY criado_em DESC LIMIT 1`, d.TMDB).Scan(&existente, &estado)
	if existente != "" && (estado != "erro" || a.ocupada(ctx, existente)) {
		writeJSON(w, http.StatusConflict, map[string]string{"erro": "Este documentário já está nas Referências.", "id": existente})
		return
	}
	id := existente
	if id == "" {
		id = novoID()
	}
	pasta := a.pastaReferencia(id)
	os.RemoveAll(pasta) // um erro de antes: começa do zero
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		falha(w, err)
		return
	}
	nome := "entrada.srt"
	if campo == "audio" {
		ext := strings.ToLower(filepath.Ext(cabecalho.Filename))
		if !regexp.MustCompile(`^\.[a-z0-9]{1,5}$`).MatchString(ext) {
			ext = ".ogg"
		}
		nome = "audio" + ext
	}
	f, err := os.Create(filepath.Join(pasta, nome))
	if err != nil {
		falha(w, err)
		return
	}
	_, err = io.Copy(f, arquivo)
	f.Close()
	if err != nil {
		falha(w, err)
		return
	}
	d.Transcrito = ""
	dados, _ := json.Marshal(d)
	duracao := ""
	if d.Duracao > 0 {
		duracao = marcaDeTempo(int64(d.Duracao) * 1000)
	}
	canal := strings.TrimSpace(strings.Join(naoVazios("Documentário", d.Ano, d.rotuloDoEpisodio()), " · "))
	if existente != "" {
		_, err = a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ?, canal = ?, duracao = ?, idioma = ?, estado = 'novo', erro = '',
			sugestao = '', documentario = ?, atualizado_em = ? WHERE id = ?`, cortar(d.titulo(), 200, false), canal, duracao, d.IdiomaDoTexto, string(dados), agora(), id)
	} else {
		_, err = a.db.ExecContext(ctx, `INSERT INTO referencias (id, url, tipo, estado, titulo, canal, duracao, idioma, documentario, criado_em, atualizado_em)
			VALUES (?, ?, 'documentario', 'novo', ?, ?, ?, ?, ?, ?, ?)`, id, d.TMDB, cortar(d.titulo(), 200, false), canal, duracao, d.IdiomaDoTexto, string(dados), agora(), agora())
	}
	if err != nil {
		falha(w, err)
		return
	}
	if _, err := a.fila.enfileirar(ctx, id, "extrair", nil); err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(ctx, id)
	a.completar(ctx, []*referencia{ref})
	writeJSON(w, http.StatusCreated, ref)
}

func naoVazios(s ...string) []string {
	var r []string
	for _, x := range s {
		if strings.TrimSpace(x) != "" {
			r = append(r, x)
		}
	}
	return r
}

// audioDoDocumentario: o áudio guardado na pasta da Referência, se houver.
func (a *app) audioDoDocumentario(id string) string {
	achados, _ := filepath.Glob(filepath.Join(a.pastaReferencia(id), "audio.*"))
	if len(achados) == 0 {
		return ""
	}
	return achados[0]
}

// extrairDocumentario é a etapa "extrair": a legenda vira a transcrição; o áudio vai ao Whisper.
func (a *app) extrairDocumentario(ctx context.Context, ref *referencia, pasta string) error {
	d, err := documentarioDe(ref)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "documentario-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if d.Capa != "" && !a.temCapa(ref.ID) {
		if err := a.capaDoPodcast(ctx, ref.ID, d.Capa, tmp); err != nil {
			log.Printf("capa do documentário %s: %v", ref.ID, err)
		}
	}
	var trechos []trechoFalado
	if d.Texto == "legenda" {
		b, err := os.ReadFile(filepath.Join(pasta, "entrada.srt"))
		if err != nil {
			return errors.New("A legenda do documentário não está guardada: mande de novo pelo Streaming.")
		}
		trechos = falasDaLegenda(b)
		if len(trechos) < 20 {
			return errors.New("A legenda do documentário veio quase vazia.")
		}
	} else {
		if a.cfg.whisper == "" && a.cfg.whisperReserva == "" {
			return errSemTranscricao
		}
		audio := a.audioDoDocumentario(ref.ID)
		if audio == "" {
			return errors.New("O áudio do documentário não está guardado: mande de novo pelo Streaming.")
		}
		ts, quem, _, err := a.transcreverAudio(ctx, audio, tmp, d.IdiomaDoTexto, promptDoDocumentario(d))
		if err != nil {
			return err
		}
		d.Transcrito = quem
		// Documentário tem trilha e silêncio: sobre a música, o Whisper inventa frases (como no Reel).
		for _, t := range ts {
			if !t.SemFala {
				trechos = append(trechos, t)
			}
		}
		if len(trechos) == 0 {
			return errSemFala
		}
		if d.IdiomaDoTexto == "" && trechos[0].Idioma != "" {
			d.IdiomaDoTexto = trechos[0].Idioma
		}
	}
	dados, _ := json.Marshal(d)
	a.db.ExecContext(ctx, `UPDATE referencias SET documentario = ?, idioma = ? WHERE id = ?`, string(dados), d.IdiomaDoTexto, ref.ID)
	ref.Idioma, ref.Documentario = d.IdiomaDoTexto, dados
	return os.WriteFile(filepath.Join(pasta, "cru.md"), []byte(documentarioParaCru(d, trechos)), 0o644)
}

// promptDoDocumentario ajuda o Whisper com os nomes: o documentário, o episódio e as sinopses.
func promptDoDocumentario(d documentario) string {
	return cortar(strings.Join(strings.Fields(strings.Join(naoVazios(d.Nome, d.TituloEpisodio, d.SinopseEpisodio, d.Sinopse), ". ")), " "), 600, false)
}

var (
	reTempoDaLegenda  = regexp.MustCompile(`(\d{1,2}):(\d{2}):(\d{2})[,.](\d{1,3})\s*-->`)
	reMarcasDaLegenda = regexp.MustCompile(`<[^>]*>|\{\\[^}]*\}`)
	// [música], (aplausos), ♪ ... ♪: o som descrito não é fala.
	reSomDescrito = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|♪[^♪]*♪?|#[^#]*#`)
)

// falasDaLegenda lê um SRT: cada fala com o tempo de início, sem marcas de formatação, sons
// descritos e as linhas que eram só isso.
func falasDaLegenda(srt []byte) []trechoFalado {
	texto := strings.ReplaceAll(strings.TrimPrefix(string(srt), "\ufeff"), "\r\n", "\n")
	var ts []trechoFalado
	for _, bloco := range regexp.MustCompile(`\n\s*\n`).Split(texto, -1) {
		linhas := strings.Split(strings.TrimSpace(bloco), "\n")
		for i, l := range linhas {
			m := reTempoDaLegenda.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			var falas []string
			for _, f := range linhas[i+1:] {
				f = html.UnescapeString(reMarcasDaLegenda.ReplaceAllString(f, ""))
				f = strings.TrimSpace(reSomDescrito.ReplaceAllString(f, ""))
				f = strings.TrimSpace(strings.TrimLeft(f, "-– "))
				if f != "" && strings.ContainsFunc(f, func(r rune) bool { return r > '9' }) {
					falas = append(falas, f)
				}
			}
			if len(falas) > 0 {
				ts = append(ts, trechoFalado{Inicio: segundosDaLegenda(m[1], m[2], m[3], m[4]), Texto: strings.Join(falas, " ")})
			}
			break
		}
	}
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].Inicio < ts[j].Inicio })
	return ts
}

func segundosDaLegenda(h, m, s, ms string) float64 {
	var hh, mm, ss, mil int
	fmt.Sscan(h, &hh)
	fmt.Sscan(m, &mm)
	fmt.Sscan(s, &ss)
	for len(ms) < 3 {
		ms += "0"
	}
	fmt.Sscan(ms, &mil)
	return float64(hh*3600+mm*60+ss) + float64(mil)/1000
}

// documentarioParaCru: o cabeçalho com o que se sabe do documentário e a fala com uma marca [mm:ss]
// a cada ~30 s, como a de um vídeo.
func documentarioParaCru(d documentario, trechos []trechoFalado) string {
	var s strings.Builder
	fmt.Fprintf(&s, "# %s\n\n", d.titulo())
	nome := d.Nome
	if d.Original != "" && d.Original != d.Nome {
		nome += " (" + d.Original + ")"
	}
	if d.Ano != "" {
		nome += ", " + d.Ano
	}
	fmt.Fprintf(&s, "- Documentário: %s\n", nome)
	if d.ehEpisodio() {
		fmt.Fprintf(&s, "- Episódio: %s", d.rotuloDoEpisodio())
		if d.TituloEpisodio != "" {
			fmt.Fprintf(&s, " — %s", d.TituloEpisodio)
		}
		s.WriteString("\n")
	}
	if d.Sinopse != "" {
		fmt.Fprintf(&s, "- Sinopse: %s\n", strings.Join(strings.Fields(d.Sinopse), " "))
	}
	if d.SinopseEpisodio != "" {
		fmt.Fprintf(&s, "- Sinopse do episódio: %s\n", strings.Join(strings.Fields(d.SinopseEpisodio), " "))
	}
	if d.Duracao > 0 {
		fmt.Fprintf(&s, "- Duração: %s\n", marcaDeTempo(int64(d.Duracao)*1000))
	}
	if d.Texto == "legenda" {
		fmt.Fprintf(&s, "- Texto: legenda do vídeo (%s), não transcrição do áudio\n", primeiro(d.Legenda, "idioma original"))
	} else {
		fmt.Fprintf(&s, "- Texto: transcrição do áudio (%s)\n", primeiro(d.Transcrito, "Whisper"))
	}
	fmt.Fprintf(&s, "- TMDB: %s\n", d.TMDB)
	if d.Link != "" {
		fmt.Fprintf(&s, "- Streaming: %s\n", d.Link)
	}
	s.WriteString("\n")
	s.WriteString(falasEmBlocos(trechos))
	return s.String()
}

// episodiosDoDocumentario: os outros episódios do mesmo documentário, em ordem (sem os descartados).
func (a *app) episodiosDoDocumentario(ctx context.Context, chave string) ([]*referencia, error) {
	rows, err := a.db.QueryContext(ctx, `SELECT `+colunasReferencia+` FROM referencias WHERE tipo = 'documentario'
		AND estado != 'descartado' AND json_extract(documentario, '$.chave') = ?`, chave)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var refs []*referencia
	for rows.Next() {
		r, err := lerReferencia(rows)
		if err != nil {
			return nil, err
		}
		refs = append(refs, r)
	}
	sort.SliceStable(refs, func(i, j int) bool {
		di, _ := documentarioDe(refs[i])
		dj, _ := documentarioDe(refs[j])
		if di.Temporada != dj.Temporada {
			return di.Temporada < dj.Temporada
		}
		return di.Episodio < dj.Episodio
	})
	return refs, rows.Err()
}

// episodioLigado é o que a página de um episódio mostra dos outros.
type episodioLigado struct {
	ID        string `json:"id"`
	Titulo    string `json:"titulo"`
	Temporada int    `json:"temporada"`
	Episodio  int    `json:"episodio"`
	Estado    string `json:"estado"`
}

func (a *app) episodiosLigados(ctx context.Context, ref *referencia) []episodioLigado {
	d, err := documentarioDe(ref)
	if err != nil || !d.ehEpisodio() {
		return nil
	}
	refs, _ := a.episodiosDoDocumentario(ctx, d.Chave)
	lista := []episodioLigado{}
	for _, r := range refs {
		e, _ := documentarioDe(r)
		lista = append(lista, episodioLigado{ID: r.ID, Titulo: primeiro(e.TituloEpisodio, fmt.Sprintf("Episódio %d", e.Episodio)), Temporada: e.Temporada, Episodio: e.Episodio, Estado: r.Estado})
	}
	return lista
}

// anterioresParaOrganizar escreve em outros-episodios.md o que os episódios anteriores já
// organizados explicaram (o resumo e as seções), para o texto deste não repetir e usar os mesmos
// nomes. Devolve o nome do arquivo, ou "" se não houver anteriores.
func (a *app) anterioresParaOrganizar(ctx context.Context, ref *referencia, pasta string) string {
	d, err := documentarioDe(ref)
	if err != nil || !d.ehEpisodio() {
		return ""
	}
	refs, _ := a.episodiosDoDocumentario(ctx, d.Chave)
	var s strings.Builder
	for _, r := range refs {
		e, _ := documentarioDe(r)
		if r.ID == ref.ID || e.Temporada > d.Temporada || (e.Temporada == d.Temporada && e.Episodio >= d.Episodio) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(a.pastaReferencia(r.ID), "organizado.md"))
		if err != nil {
			continue
		}
		fmt.Fprintf(&s, "## %s — %s\n\n", e.rotuloDoEpisodio(), primeiro(e.TituloEpisodio, "sem título"))
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "> ") || strings.HasPrefix(l, "## ") {
				if strings.HasPrefix(l, "## ") {
					l = "- " + strings.TrimPrefix(l, "## ")
				}
				s.WriteString(l + "\n")
			}
		}
		s.WriteString("\n")
	}
	if s.Len() == 0 {
		return ""
	}
	nome := "outros-episodios.md"
	conteudo := "# Episódios anteriores deste documentário\n\nO resumo e as seções de cada um, já organizados.\n\n" + s.String()
	if os.WriteFile(filepath.Join(pasta, nome), []byte(conteudo), 0o644) != nil {
		return ""
	}
	return nome
}

// pedidoDeTemaDoDocumentario entra na sugestão de Tema: os outros episódios e onde entraram.
func (a *app) pedidoDeTemaDoDocumentario(ctx context.Context, ref *referencia) string {
	d, err := documentarioDe(ref)
	if err != nil || !d.ehEpisodio() {
		return ""
	}
	refs, _ := a.episodiosDoDocumentario(ctx, d.Chave)
	var linhas []string
	for _, r := range refs {
		if r.ID == ref.ID {
			continue
		}
		e, _ := documentarioDe(r)
		var temas []string
		if rows, err := a.db.QueryContext(ctx, `SELECT tema FROM referencia_temas WHERE referencia_id = ?`, r.ID); err == nil {
			for rows.Next() {
				var t string
				if rows.Scan(&t) == nil {
					temas = append(temas, t)
				}
			}
			rows.Close()
		}
		onde := "ainda sem Tema"
		if len(temas) > 0 {
			onde = "integrado em " + strings.Join(temas, ", ")
		} else if len(r.Sugestao) > 0 && string(r.Sugestao) != "null" {
			var s sugestao
			if json.Unmarshal(r.Sugestao, &s) == nil && s.Titulo != "" {
				onde = fmt.Sprintf("sugestão: %s %s %s", s.Acao, s.Tema, s.Titulo)
			}
		}
		linhas = append(linhas, fmt.Sprintf("- %s %s (%s)", e.rotuloDoEpisodio(), primeiro(e.TituloEpisodio, ""), onde))
	}
	if len(linhas) == 0 {
		return fmt.Sprintf("\n\nEsta Referência é o %s do documentário %q, o primeiro mandado ao Acervo: sugira um Tema que possa receber os outros episódios também.", d.rotuloDoEpisodio(), d.Nome)
	}
	return fmt.Sprintf("\n\nEsta Referência é o %s do documentário %q. Os outros episódios no Acervo:\n%s\n"+
		"Os episódios de um documentário ficam juntos: se algum já entrou num Tema ou tem sugestão, sugira o mesmo Tema (integrar, ou um subtema dele para este episódio), a menos que este episódio trate de outro assunto.",
		d.rotuloDoEpisodio(), d.Nome, strings.Join(linhas, "\n"))
}

func notaDoDocumentario(ref *referencia) string {
	d, err := documentarioDe(ref)
	if err != nil {
		return ""
	}
	fonte := d.Nome
	if d.Ano != "" {
		fonte += " (" + d.Ano + ")"
	}
	if d.ehEpisodio() {
		fonte += ", " + d.rotuloDoEpisodio() + " " + d.TituloEpisodio
	}
	return fmt.Sprintf(" Esta Referência é um documentário: %s. As marcas [mm:ss] são do vídeo; a URL é a página dele no TMDB.", strings.TrimSpace(fonte))
}

// trechoDoDocumentario corta o áudio guardado entre inicio e fim (segundos), para as Dúvidas.
func (a *app) trechoDoDocumentario(ctx context.Context, ref *referencia, base string, inicio, fim float64) (string, error) {
	audio := a.audioDoDocumentario(ref.ID)
	if audio == "" {
		return "", errors.New("este documentário veio pela legenda: não há áudio guardado")
	}
	saida := base + ".flac"
	cmd := exec.CommandContext(ctx, a.cfg.ffmpeg, "-v", "error", "-ss", fmt.Sprintf("%.1f", inicio), "-t", fmt.Sprintf("%.1f", fim-inicio),
		"-i", audio, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "flac", saida)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("o ffmpeg não cortou o trecho: %v %s", err, cortar(string(out), 200, false))
	}
	return saida, nil
}
