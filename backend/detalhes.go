package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

// "Mais informações" (D26): o Lucas seleciona um trecho de um Tema ou de uma Referência e a IA
// pesquisa na internet sobre ele. A marcação fica guardada à parte, nunca dentro do texto: o app
// acha o trecho de novo ao abrir a página (pelo texto e pelo que vem antes e depois) e sublinha.
type marcacao struct {
	ID           string       `json:"id"`
	Local        string       `json:"local"` // temas/…/arquivo.md ou referencias/<id>/<aba>
	Titulo       string       `json:"titulo"`
	Trecho       string       `json:"trecho"`
	Antes        string       `json:"antes"`
	Depois       string       `json:"depois"`
	Estado       string       `json:"estado"` // processando, pronto ou falhou
	Erro         string       `json:"erro,omitempty"`
	Explicacao   string       `json:"explicacao"`
	Fontes       []consultada `json:"fontes"`
	TarefaID     int64        `json:"tarefa_id"`
	CriadoEm     string       `json:"criado_em"`
	AtualizadoEm string       `json:"atualizado_em"`
	Integracao   *pendencia   `json:"integracao"` // a proposta de Integrar ao texto, enquanto não decidida (D44)
}

type consultada struct {
	Titulo string `json:"titulo"`
	URL    string `json:"url"`
}

var (
	idDeDetalhe    = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)
	localDeDetalhe = regexp.MustCompile(`^(temas/[^?#]+\.md|referencias/[A-Za-z0-9_-]+/(organizado|traducao|cru|duvidas))$`)
)

const (
	maxTrecho   = 1000
	maxContexto = 400
)

const esquemaDetalhe = `{"type":"object","properties":{` +
	`"explicacao":{"type":"string"},` +
	`"fontes":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"url":{"type":"string"}},"required":["titulo","url"]}}},` +
	`"required":["explicacao","fontes"]}`

// O estado vem da tarefa: com a explicação pronta, vale ela; senão, como a tarefa terminou.
const consultaDetalhe = `SELECT d.id, d.local, d.titulo, d.trecho, d.antes, d.depois, d.explicacao, d.fontes,
	d.tarefa_id, d.criado_em, d.atualizado_em, COALESCE(t.estado, ''), COALESCE(t.erro, '')
	FROM detalhes d LEFT JOIN tarefas t ON t.id = d.tarefa_id`

func lerDetalhe(row linha) (marcacao, error) {
	var d marcacao
	var fontes, estadoTarefa, erroTarefa string
	err := row.Scan(&d.ID, &d.Local, &d.Titulo, &d.Trecho, &d.Antes, &d.Depois, &d.Explicacao, &fontes,
		&d.TarefaID, &d.CriadoEm, &d.AtualizadoEm, &estadoTarefa, &erroTarefa)
	if err != nil {
		return d, err
	}
	d.Fontes = []consultada{}
	json.Unmarshal([]byte(fontes), &d.Fontes)
	switch {
	case estadoTarefa == naFila || estadoTarefa == rodando:
		d.Estado = "processando"
	case d.Explicacao != "":
		d.Estado = "pronto"
	case estadoTarefa == cancelada:
		d.Estado, d.Erro = "falhou", "A pesquisa foi cancelada na fila."
	default:
		d.Estado, d.Erro = "falhou", erroTarefa
		if d.Erro == "" {
			d.Erro = "A pesquisa terminou sem explicação."
		}
	}
	return d, nil
}

func (a *app) detalhe(ctx context.Context, id string) (marcacao, error) {
	return lerDetalhe(a.db.QueryRowContext(ctx, consultaDetalhe+` WHERE d.id = ?`, id))
}

// handleDetalhes lista as marcações de um texto (?local=) ou todas.
func (a *app) handleDetalhes(w http.ResponseWriter, r *http.Request) {
	onde, args := "", []any{}
	if local := r.URL.Query().Get("local"); local != "" {
		onde, args = ` WHERE d.local = ?`, []any{local}
	}
	rows, err := a.db.QueryContext(r.Context(), consultaDetalhe+onde+` ORDER BY d.criado_em`, args...)
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	lista := []marcacao{}
	for rows.Next() {
		d, err := lerDetalhe(rows)
		if err != nil {
			falha(w, err)
			return
		}
		lista = append(lista, d)
	}
	if err := rows.Err(); err != nil {
		falha(w, err)
		return
	}
	pendentes, err := a.integracoesPendentes(r.Context(), "detalhe")
	if err != nil {
		falha(w, err)
		return
	}
	for i := range lista {
		lista[i].Integracao = pendentes[lista[i].ID]
	}
	writeJSON(w, http.StatusOK, lista)
}

func cortar(s string, max int, doFim bool) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	if doFim {
		return string(r[len(r)-max:])
	}
	return string(r[:max])
}

// handlePedirDetalhe cria a marcação e põe a pesquisa na fila. O id vem do aparelho: o pedido feito
// fora de casa sobe depois, e reenviar não pesquisa duas vezes.
func (a *app) handlePedirDetalhe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !idDeDetalhe.MatchString(id) {
		erroJSON(w, http.StatusBadRequest, "Id inválido.")
		return
	}
	if d, err := a.detalhe(r.Context(), id); err == nil {
		writeJSON(w, http.StatusOK, d)
		return
	}
	var corpo struct {
		Local  string `json:"local"`
		Titulo string `json:"titulo"`
		Trecho string `json:"trecho"`
		Antes  string `json:"antes"`
		Depois string `json:"depois"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	trecho := strings.TrimSpace(corpo.Trecho)
	switch {
	case !localDeDetalhe.MatchString(corpo.Local) || strings.Contains(corpo.Local, ".."):
		erroJSON(w, http.StatusBadRequest, "Texto inválido.")
		return
	case trecho == "":
		erroJSON(w, http.StatusBadRequest, "Selecione um trecho.")
		return
	case utf8.RuneCountInString(trecho) > maxTrecho:
		erroJSON(w, http.StatusBadRequest, fmt.Sprintf("Trecho longo demais: selecione até %d caracteres.", maxTrecho))
		return
	}
	ctx := r.Context()
	_, err := a.db.ExecContext(ctx, `INSERT INTO detalhes (id, local, titulo, trecho, antes, depois, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, id, corpo.Local, strings.TrimSpace(corpo.Titulo), trecho,
		cortar(corpo.Antes, maxContexto, true), cortar(corpo.Depois, maxContexto, false), agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if err := a.pesquisarDetalhe(ctx, id); err != nil {
		falha(w, err)
		return
	}
	a.responderDetalhe(w, r, id, http.StatusCreated)
}

// handleRefazerDetalhe pede a pesquisa de novo; a explicação antiga sai.
func (a *app) handleRefazerDetalhe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := a.detalhe(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Marcação não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	if d.Estado == "processando" {
		erroJSON(w, http.StatusConflict, "A pesquisa ainda está na fila.")
		return
	}
	if a.recusarSePendente(w, r.Context(), "detalhe", id) {
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE detalhes SET explicacao = '', fontes = '[]', atualizado_em = ? WHERE id = ?`, agora(), id); err != nil {
		falha(w, err)
		return
	}
	if err := a.pesquisarDetalhe(r.Context(), id); err != nil {
		falha(w, err)
		return
	}
	a.responderDetalhe(w, r, id, http.StatusOK)
}

func (a *app) handleApagarDetalhe(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if d, err := a.detalhe(r.Context(), id); err == nil && d.Estado == "processando" {
		a.fila.cancelarTarefa(r.Context(), d.TarefaID)
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM detalhes WHERE id = ?`, id); err != nil {
		falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) responderDetalhe(w http.ResponseWriter, r *http.Request, id string, status int) {
	d, err := a.detalhe(r.Context(), id)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, status, d)
}

type paramsDetalhe struct {
	Detalhe string `json:"detalhe"`
}

func (a *app) pesquisarDetalhe(ctx context.Context, id string) error {
	tarefa, err := a.fila.enfileirar(ctx, "", "detalhar", paramsDetalhe{id})
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE detalhes SET tarefa_id = ? WHERE id = ?`, tarefa, id)
	return err
}

// detalhar é a tarefa da fila: a única que pode pesquisar na internet.
func (a *app) detalhar(ctx context.Context, t *tarefa) (resultado, error) {
	var p paramsDetalhe
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	d, err := a.detalhe(ctx, p.Detalhe)
	if errors.Is(err, sql.ErrNoRows) {
		t.Erro = "A marcação foi apagada."
		return pronto, nil
	}
	if err != nil {
		return pronto, err
	}
	// Uma pasta vazia: a tarefa não lê nem escreve nada do Acervo.
	pasta, err := os.MkdirTemp(a.cfg.dados, "detalhe-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(pasta)

	titulo := d.Titulo
	if titulo == "" {
		titulo = d.Local
	}
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta: pasta,
		regra: "detalhar.md",
		instrucao: fmt.Sprintf("O Lucas está lendo o texto %q e selecionou um trecho para saber mais.\n\n"+
			"Trecho selecionado: %q\n\nO texto em volta (o trecho está entre ⟦ ⟧):\n\n…%s⟦%s⟧%s…\n\n"+
			"Pesquise sobre o trecho, no sentido em que ele aparece no texto, e devolva a explicação "+
			"em Markdown e as fontes que você consultou.",
			titulo, d.Trecho, d.Antes, d.Trecho, d.Depois),
		ferramentas: ferramentasInternet,
		esquema:     esquemaDetalhe,
	})
	if err != nil {
		return pronto, err
	}
	var saida struct {
		Explicacao string       `json:"explicacao"`
		Fontes     []consultada `json:"fontes"`
	}
	if err := dadosDaResposta(r, &saida); err != nil {
		return pronto, err
	}
	if strings.TrimSpace(saida.Explicacao) == "" {
		return pronto, errors.New("o claude não devolveu explicação")
	}
	if saida.Fontes == nil {
		saida.Fontes = []consultada{}
	}
	fontes, _ := json.Marshal(saida.Fontes)
	_, err = a.db.ExecContext(ctx, `UPDATE detalhes SET explicacao = ?, fontes = ?, atualizado_em = ? WHERE id = ? AND tarefa_id = ?`,
		strings.TrimSpace(saida.Explicacao), string(fontes), agora(), d.ID, t.ID)
	return pronto, err
}
