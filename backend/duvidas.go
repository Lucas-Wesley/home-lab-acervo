package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Dúvidas resolvidas pelo áudio e pela internet (D37). A maioria das dúvidas de um vídeo é erro da
// legenda automática: o servidor acha o momento de cada uma (a marca [mm:ss] da dúvida ou, nas
// Referências antigas, o trecho da transcrição que ela cita), corta o áudio ali (±20 s) e manda ao
// Whisper large-v3 do Mac mini, com os nomes como contexto. A IA compara a legenda, o áudio e a
// internet (regras/resolver-duvidas.md) e diz se a dúvida foi resolvida pelo áudio, confirmada na
// internet, continua incerta ou é falta no material. O Lucas decide: Corrigir no texto (uma
// proposta revisada, D36), Pesquisar isto (Perguntar ou Aprofundar) ou deixar.

type resolucao struct {
	Estado     string       `json:"estado"` // resolvida_audio, confirmada_internet, incerta ou lacuna
	NoAudio    string       `json:"no_audio"`
	Explicacao string       `json:"explicacao"`
	Trecho     string       `json:"trecho"`  // o que está errado no texto (vazio: nada a corrigir)
	Correto    string       `json:"correto"` // como fica
	Fontes     []consultada `json:"fontes"`
}

type duvidaResolvida struct {
	ID           string     `json:"id"`
	ReferenciaID string     `json:"referencia_id"`
	Texto        string     `json:"texto"`
	Momento      string     `json:"momento"` // [mm:ss] usado, se achado
	Audio        string     `json:"audio"`   // o que o Whisper ouviu no trecho
	Estado       string     `json:"estado"`  // processando, pronto ou falhou
	Erro         string     `json:"erro,omitempty"`
	Resultado    *resolucao `json:"resultado"`
	TarefaID     int64      `json:"tarefa_id"`
	AtualizadoEm string     `json:"atualizado_em"`
}

const (
	folgaAntes     = 20.0
	folgaDepois    = 25.0
	maxTrechoAudio = 120.0
	maxDuvidas     = 40
)

const esquemaDuvidas = `{"type":"object","properties":{"duvidas":{"type":"array","items":{"type":"object","properties":{` +
	`"indice":{"type":"integer"},` +
	`"estado":{"type":"string","enum":["resolvida_audio","confirmada_internet","incerta","lacuna"]},` +
	`"no_audio":{"type":"string"},"explicacao":{"type":"string"},"trecho":{"type":"string"},"correto":{"type":"string"},` +
	`"fontes":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"url":{"type":"string"}},"required":["titulo","url"]}}},` +
	`"required":["indice","estado","no_audio","explicacao","trecho","correto","fontes"]}}},"required":["duvidas"]}`

var (
	secaoDuvidas = regexp.MustCompile(`(?s)\n## (?:Dúvidas|Doubts|Uncertainties)\n(.*?)(?:\n## |\z)`)
	marcaNoTexto = regexp.MustCompile(`(\d{1,2}:\d{2}(?::\d{2})?)`)
	linhaDoCru   = regexp.MustCompile(`(?m)^\[(\d{1,2}:\d{2}(?::\d{2})?)\] (.*)$`)
	termoCitado  = regexp.MustCompile(`["“*_]([^"”*_\n]{3,80})["”*_]`)
)

// listaDeDuvidas: os itens da seção "## Dúvidas" do texto (um item pode ocupar várias linhas).
func listaDeDuvidas(texto string) []string {
	m := secaoDuvidas.FindStringSubmatch("\n" + texto)
	if m == nil {
		return nil
	}
	var itens []string
	for _, l := range strings.Split(m[1], "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "- ") || strings.HasPrefix(l, "* "):
			itens = append(itens, strings.TrimSpace(l[2:]))
		case l != "" && len(itens) > 0:
			itens[len(itens)-1] += " " + l
		}
	}
	return itens
}

func segundosDaMarca(m string) float64 {
	total := 0.0
	for _, p := range strings.Split(m, ":") {
		n, _ := strconv.Atoi(p)
		total = total*60 + float64(n)
	}
	return total
}

type blocoDoCru struct {
	inicio float64
	marca  string
	texto  string
}

func blocosDoCru(cru string) []blocoDoCru {
	var bs []blocoDoCru
	for _, m := range linhaDoCru.FindAllStringSubmatch(cru, -1) {
		bs = append(bs, blocoDoCru{segundosDaMarca(m[1]), m[1], m[2]})
	}
	return bs
}

// momentoDaDuvida: a marca escrita na dúvida (e a de fim, num intervalo) ou, sem marca, o bloco da
// transcrição que mais tem o que a dúvida cita. ok = false se não há onde ouvir.
func momentoDaDuvida(duvida string, blocos []blocoDoCru) (inicio, fim float64, ok bool) {
	if ms := marcaNoTexto.FindAllString(duvida, 2); len(ms) > 0 {
		inicio = segundosDaMarca(ms[0])
		fim = inicio
		if len(ms) == 2 {
			if f := segundosDaMarca(ms[1]); f > inicio && f-inicio <= 180 {
				fim = f
			}
		}
		return inicio, fim, true
	}
	// Sem marca: o bloco que contém mais termos citados na dúvida ("…", *…*), sem acento nem
	// maiúscula. Uma dúvida que não cita nada da fala fica sem momento.
	var termos []string
	for _, t := range termoCitado.FindAllStringSubmatch(duvida, -1) {
		if s := slug(t[1]); len(s) >= 3 {
			termos = append(termos, s)
		}
	}
	melhor, pontos := -1, 0
	for i, b := range blocos {
		texto, n := slug(b.texto), 0
		for _, t := range termos {
			if strings.Contains(texto, t) {
				n++
			}
		}
		if n > pontos {
			melhor, pontos = i, n
		}
	}
	if melhor < 0 {
		return 0, 0, false
	}
	return blocos[melhor].inicio, blocos[melhor].inicio + 30, true
}

// trechoDaLegenda: os blocos da transcrição em volta do momento, para a IA comparar com o áudio.
func trechoDaLegenda(blocos []blocoDoCru, inicio, fim float64) string {
	var s strings.Builder
	for _, b := range blocos {
		if b.inicio >= inicio-60 && b.inicio <= fim+45 {
			fmt.Fprintf(&s, "[%s] %s\n", b.marca, b.texto)
		}
	}
	return s.String()
}

const consultaDuvida = `SELECT d.id, d.referencia_id, d.texto, d.momento, d.audio, d.resultado, d.tarefa_id, d.atualizado_em,
	COALESCE(t.estado, ''), COALESCE(t.erro, '') FROM duvidas d LEFT JOIN tarefas t ON t.id = d.tarefa_id`

func lerDuvida(row linha) (duvidaResolvida, error) {
	var d duvidaResolvida
	var res, estado, erro string
	if err := row.Scan(&d.ID, &d.ReferenciaID, &d.Texto, &d.Momento, &d.Audio, &res, &d.TarefaID, &d.AtualizadoEm, &estado, &erro); err != nil {
		return d, err
	}
	if res != "" {
		d.Resultado = &resolucao{}
		json.Unmarshal([]byte(res), d.Resultado)
	}
	switch {
	case estado == naFila || estado == rodando:
		d.Estado = "processando"
	case d.Resultado != nil:
		d.Estado = "pronto"
	case estado == cancelada:
		d.Estado, d.Erro = "falhou", "Cancelada na fila."
	default:
		d.Estado, d.Erro = "falhou", primeiro(erro, "Terminou sem resposta.")
	}
	return d, nil
}

func (a *app) duvidasDe(ctx context.Context, refID string) ([]duvidaResolvida, error) {
	rows, err := a.db.QueryContext(ctx, consultaDuvida+` WHERE d.referencia_id = ? ORDER BY d.criado_em`, refID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []duvidaResolvida{}
	for rows.Next() {
		d, err := lerDuvida(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, d)
	}
	return lista, rows.Err()
}

// handleDuvidas: GET /api/referencias/{id}/duvidas, as dúvidas já mandadas resolver.
func (a *app) handleDuvidas(w http.ResponseWriter, r *http.Request) {
	ref, ok := a.referenciaDoPedido(w, r)
	if !ok {
		return
	}
	lista, err := a.duvidasDe(r.Context(), ref.ID)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lista)
}

// handleResolverDuvidas: POST /api/referencias/{id}/duvidas {texto}. Com texto, só aquela dúvida;
// sem, todas as do texto que ainda não foram resolvidas (serve às Referências antigas também).
func (a *app) handleResolverDuvidas(w http.ResponseWriter, r *http.Request) {
	ref, ok := a.referenciaDoPedido(w, r)
	if !ok {
		return
	}
	var corpo struct {
		Texto string `json:"texto"`
	}
	if r.ContentLength != 0 && !lerCorpo(w, r, &corpo) {
		return
	}
	ctx := r.Context()
	caminho, err := a.textoParaAcervo(ref)
	if err != nil {
		erroJSON(w, http.StatusConflict, err.Error())
		return
	}
	b, _ := os.ReadFile(caminho)
	todas := listaDeDuvidas(string(b))
	ja, err := a.duvidasDe(ctx, ref.ID)
	if err != nil {
		falha(w, err)
		return
	}
	feitas := map[string]duvidaResolvida{}
	for _, d := range ja {
		feitas[d.Texto] = d
	}
	var escolhidas []string
	for _, t := range todas {
		d, existe := feitas[t]
		if corpo.Texto != "" {
			if t == strings.TrimSpace(corpo.Texto) {
				escolhidas = append(escolhidas, t)
			}
		} else if !existe || d.Estado == "falhou" {
			escolhidas = append(escolhidas, t)
		}
	}
	if len(escolhidas) == 0 {
		erroJSON(w, http.StatusBadRequest, "Nenhuma dúvida para resolver.")
		return
	}
	if len(escolhidas) > maxDuvidas {
		escolhidas = escolhidas[:maxDuvidas]
	}
	if a.ocupada(ctx, ref.ID) {
		erroJSON(w, http.StatusConflict, "A Referência já tem uma tarefa em andamento.")
		return
	}
	ids := make([]string, len(escolhidas))
	for i, t := range escolhidas {
		if d, existe := feitas[t]; existe {
			ids[i] = d.ID
		} else {
			ids[i] = novoID()
			if _, err := a.db.ExecContext(ctx, `INSERT INTO duvidas (id, referencia_id, texto, criado_em, atualizado_em) VALUES (?, ?, ?, ?, ?)`,
				ids[i], ref.ID, t, agora(), agora()); err != nil {
				falha(w, err)
				return
			}
		}
	}
	tid, err := a.fila.enfileirar(ctx, ref.ID, "resolver_duvidas", map[string]any{"duvidas": ids})
	if err != nil {
		falha(w, err)
		return
	}
	marcas := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := []any{tid, agora()}
	for _, id := range ids {
		args = append(args, id)
	}
	a.db.ExecContext(ctx, `UPDATE duvidas SET tarefa_id = ?, resultado = '', atualizado_em = ? WHERE id IN (`+marcas+`)`, args...)
	lista, _ := a.duvidasDe(ctx, ref.ID)
	writeJSON(w, http.StatusAccepted, lista)
}

// trechoDeAudio corta o áudio da Referência entre inicio e fim (segundos) e devolve o arquivo.
func (a *app) trechoDeAudio(ctx context.Context, ref *referencia, pasta string, n int, inicio, fim float64) (string, error) {
	base := filepath.Join(pasta, fmt.Sprintf("trecho-%02d", n))
	switch ref.Tipo {
	case "video", "reel":
		secao := fmt.Sprintf("*%.0f-%.0f", inicio, fim)
		if _, err := a.ytdlp(ctx, "-q", "--no-playlist", "-f", "bestaudio/best", "--download-sections", secao, "-o", base+".%(ext)s", ref.URL); err != nil {
			return "", err
		}
		achados, _ := filepath.Glob(base + ".*")
		if len(achados) == 0 {
			return "", errors.New("o yt-dlp não baixou o trecho")
		}
		return achados[0], nil
	case "podcast":
		var ep episodio
		b, err := os.ReadFile(filepath.Join(a.pastaReferencia(ref.ID), "info.json"))
		if err != nil || json.Unmarshal(b, &ep) != nil || ep.Audio == "" {
			return "", errors.New("o endereço do áudio do episódio não está guardado")
		}
		saida := base + ".flac"
		cmd := exec.CommandContext(ctx, a.cfg.ffmpeg, "-v", "error", "-ss", fmt.Sprintf("%.1f", inicio), "-t", fmt.Sprintf("%.1f", fim-inicio),
			"-i", ep.Audio, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "flac", saida)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("o ffmpeg não cortou o trecho: %v %s", err, cortar(string(out), 200, false))
		}
		return saida, nil
	}
	return "", errors.New("esta Referência não tem áudio")
}

func (a *app) resolverDuvidas(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	var p struct {
		Duvidas []string `json:"duvidas"`
	}
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	var lista []duvidaResolvida
	for _, id := range p.Duvidas {
		d, err := lerDuvida(a.db.QueryRowContext(ctx, consultaDuvida+` WHERE d.id = ?`, id))
		if err == nil {
			lista = append(lista, d)
		}
	}
	if len(lista) == 0 {
		return pronto, errors.New("as dúvidas não existem mais")
	}
	pastaRef := a.pastaReferencia(ref.ID)
	cru, _ := os.ReadFile(filepath.Join(pastaRef, "cru.md"))
	blocos := blocosDoCru(string(cru))
	temAudio := (ref.Tipo == "video" || ref.Tipo == "podcast" || ref.Tipo == "reel") && (a.cfg.whisper != "" || a.cfg.whisperReserva != "")

	tmp, err := os.MkdirTemp("", "duvidas-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(tmp)
	var material strings.Builder
	fmt.Fprintf(&material, "# Dúvidas da Referência %q (%s, %s)\n\n", ref.Titulo, ref.Tipo, ref.URL)
	for i, d := range lista {
		fmt.Fprintf(&material, "## Dúvida %d\n\n%s\n\n", i, d.Texto)
		inicio, fim, achou := momentoDaDuvida(d.Texto, blocos)
		momento, ouvido := "", ""
		if achou {
			momento = marcaDeTempo(int64(inicio * 1000))
			fmt.Fprintf(&material, "Momento: [%s]\n\n### O que a transcrição tem ali\n\n%s\n", momento, trechoDaLegenda(blocos, inicio, fim))
			if temAudio {
				de := max(0, inicio-folgaAntes)
				ate := min(fim+folgaDepois, de+maxTrechoAudio)
				arquivo, err := a.trechoDeAudio(ctx, ref, tmp, i, de, ate)
				if err == nil {
					var ts []trechoFalado
					ts, _, err = a.transcrever(ctx, arquivo, ref.Idioma, cortar(ref.Titulo+". "+d.Texto, 600, false))
					var s strings.Builder
					for _, x := range ts {
						fmt.Fprintf(&s, "[%s] %s\n", marcaDeTempo(int64((de+x.Inicio)*1000)), x.Texto)
					}
					ouvido = s.String()
				}
				if err != nil {
					fmt.Fprintf(&material, "\n(O áudio deste trecho não pôde ser ouvido: %v)\n", err)
				} else {
					fmt.Fprintf(&material, "\n### O que o Whisper large-v3 ouviu no áudio (%s a %s)\n\n%s\n", marcaDeTempo(int64(de*1000)), marcaDeTempo(int64(ate*1000)), ouvido)
				}
			}
		} else {
			material.WriteString("Momento: não achado na transcrição.\n")
		}
		material.WriteString("\n")
		a.db.ExecContext(ctx, `UPDATE duvidas SET momento = ?, audio = ? WHERE id = ?`, momento, ouvido, d.ID)
	}
	if err := os.WriteFile(filepath.Join(tmp, "duvidas.md"), []byte(material.String()), 0o644); err != nil {
		return pronto, err
	}
	texto, err := a.textoParaAcervo(ref)
	if err != nil {
		return pronto, err
	}
	b, _ := os.ReadFile(texto)
	os.WriteFile(filepath.Join(tmp, "texto.md"), b, 0o644)

	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:       tmp,
		regra:       "resolver-duvidas.md",
		instrucao:   fmt.Sprintf("Resolva as dúvidas de duvidas.md (o texto organizado está em texto.md, nesta pasta). Hoje é %s.", time.Now().Format("2006-01-02")),
		ferramentas: ferramentasLeitura + "," + ferramentasInternet,
		esquema:     esquemaDuvidas,
	})
	if err != nil {
		return pronto, err
	}
	var saida struct {
		Duvidas []struct {
			Indice int `json:"indice"`
			resolucao
		} `json:"duvidas"`
	}
	if err := dadosDaResposta(r, &saida); err != nil {
		return pronto, err
	}
	respondidas := 0
	for _, x := range saida.Duvidas {
		if x.Indice < 0 || x.Indice >= len(lista) {
			continue
		}
		res := x.resolucao
		if res.Fontes == nil {
			res.Fontes = []consultada{}
		}
		j, _ := json.Marshal(res)
		a.db.ExecContext(ctx, `UPDATE duvidas SET resultado = ?, atualizado_em = ? WHERE id = ?`, string(j), agora(), lista[x.Indice].ID)
		respondidas++
	}
	if respondidas == 0 {
		return pronto, errors.New("a IA não respondeu nenhuma dúvida")
	}
	return pronto, nil
}

// handleIntegrarDuvida: POST /api/duvidas/{id}/integrar — a correção vira proposta revisada (D36).
func (a *app) handleIntegrarDuvida(w http.ResponseWriter, r *http.Request) {
	d, err := lerDuvida(a.db.QueryRowContext(r.Context(), consultaDuvida+` WHERE d.id = ?`, r.PathValue("id")))
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Dúvida não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	if d.Resultado == nil || d.Estado != "pronto" {
		erroJSON(w, http.StatusConflict, "A dúvida ainda não foi resolvida.")
		return
	}
	ref, err := a.referencia(r.Context(), d.ReferenciaID)
	if err != nil {
		erroJSON(w, http.StatusNotFound, "Referência não encontrada.")
		return
	}
	a.enfileirarIntegracao(w, r, "referencias/"+d.ReferenciaID, paramsIntegrarTexto{Origem: "duvida", ID: d.ID, Titulo: ref.Titulo})
}
