package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Pedidos que viram proposta são idempotentes (D44). Enquanto a proposta de integrar ao texto de uma
// dúvida, pergunta ou marcação está na fila, rodando ou esperando decisão, pedir de novo devolve a
// mesma tarefa, e refazer o que a originou fica bloqueado até o Lucas aprovar ou descartar. Uma
// Referência com proposta de Tema esperando em Revisões não pede outra, e um texto com flashcards
// esperando decisão no Flashcards não manda outro pedido.

type pendencia struct {
	TarefaID int64          `json:"tarefa_id"`
	Estado   string         `json:"estado"`            // na_fila, rodando ou em_revisao
	Trechos  []trechoMudado `json:"trechos,omitempty"` // numa dúvida em revisão: como o texto fica
}

const consultaPendentes = `SELECT id, estado, COALESCE(json_extract(params, '$.id'), '') FROM tarefas
	WHERE tipo = 'integrar_texto' AND estado IN (?, ?, ?) AND json_extract(params, '$.origem') = ?`

// integracoesPendentes: as propostas em aberto de uma origem (duvida, pergunta ou detalhe), pelo id.
func (a *app) integracoesPendentes(ctx context.Context, origem string) (map[string]*pendencia, error) {
	rows, err := a.db.QueryContext(ctx, consultaPendentes, naFila, rodando, emRevisao, origem)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]*pendencia{}
	for rows.Next() {
		p, id := &pendencia{}, ""
		if err := rows.Scan(&p.TarefaID, &p.Estado, &id); err != nil {
			return nil, err
		}
		m[id] = p
	}
	return m, rows.Err()
}

func (a *app) integracaoPendente(ctx context.Context, origem, id string) *pendencia {
	p := &pendencia{}
	var achado string
	err := a.db.QueryRowContext(ctx, consultaPendentes+` AND json_extract(params, '$.id') = ? LIMIT 1`,
		naFila, rodando, emRevisao, origem, id).Scan(&p.TarefaID, &p.Estado, &achado)
	if err != nil {
		return nil
	}
	return p
}

// recusarSePendente responde 409 quando há proposta em aberto para o item; true = recusou.
func (a *app) recusarSePendente(w http.ResponseWriter, ctx context.Context, origem, id string) bool {
	if a.integracaoPendente(ctx, origem, id) == nil {
		return false
	}
	erroJSON(w, http.StatusConflict, "Há uma proposta deste item esperando: aprove ou descarte antes de pedir de novo.")
	return true
}

// propostaDeTemaPendente: a tarefa de criar ou integrar Tema desta Referência que espera em Revisões.
func (a *app) propostaDeTemaPendente(ctx context.Context, refID string) int64 {
	var id int64
	a.db.QueryRowContext(ctx, `SELECT id FROM tarefas WHERE referencia_id = ? AND tipo IN ('integrar', 'criar_tema') AND estado = ? LIMIT 1`,
		refID, emRevisao).Scan(&id)
	return id
}

// geracaoEmAberto: o último pedido de flashcards deste texto, se ainda não foi decidido lá.
func (a *app) geracaoEmAberto(ctx context.Context, alvo string) int64 {
	var id int64
	if a.db.QueryRowContext(ctx, `SELECT geracao_id FROM flashcards WHERE alvo = ? AND geracao_id > 0 ORDER BY id DESC LIMIT 1`, alvo).Scan(&id) != nil {
		return 0
	}
	var g struct {
		Estado string `json:"estado"`
	}
	if _, err := a.chamarFlashcards(ctx, "GET", fmt.Sprintf("/api/geracoes/%d", id), nil, &g); err != nil {
		return 0
	}
	if g.Estado == naFila || g.Estado == rodando || g.Estado == emRevisao {
		return id
	}
	return 0
}

// ---------- A correção pequena, vista na própria aba ----------

// trechoMudado: um pedaço do texto antes e depois da proposta (parágrafos inteiros).
type trechoMudado struct {
	Antes  string `json:"antes"`
	Depois string `json:"depois"`
}

// trechosMudados compara os parágrafos dos dois textos, sem a seção de Dúvidas (tirar a dúvida
// resolvida de lá é parte da correção, e não precisa ser mostrado).
func trechosMudados(antes, depois string) []trechoMudado {
	semDuvidas := func(s string) []string {
		s = secaoDuvidas.ReplaceAllStringFunc("\n"+s, func(m string) string {
			if strings.HasSuffix(m, "\n## ") {
				return "\n## "
			}
			return ""
		})
		var ps []string
		for _, p := range strings.Split(s, "\n\n") {
			if p = strings.TrimSpace(p); p != "" {
				ps = append(ps, p)
			}
		}
		return ps
	}
	a, b := semDuvidas(antes), semDuvidas(depois)
	// Maior subsequência comum de parágrafos; o que sobra entre os iguais é um trecho mudado.
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var trechos []trechoMudado
	var de, para []string
	fechar := func() {
		if len(de) > 0 || len(para) > 0 {
			trechos = append(trechos, trechoMudado{strings.Join(de, "\n\n"), strings.Join(para, "\n\n")})
		}
		de, para = nil, nil
	}
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			fechar()
			i, j = i+1, j+1
		case j < len(b) && (i == len(a) || lcs[i][j+1] >= lcs[i+1][j]):
			para = append(para, b[j])
			j++
		default:
			de = append(de, a[i])
			i++
		}
	}
	fechar()
	return trechos
}

// trechosDaProposta: o que a proposta de uma Referência muda no texto, para mostrar na hora.
func (a *app) trechosDaProposta(ctx context.Context, tarefaID int64) []trechoMudado {
	t, err := a.fila.tarefa(ctx, tarefaID)
	if err != nil {
		return nil
	}
	prop := a.pastaDaProposta(t.ID)
	nome, err := os.ReadFile(filepath.Join(prop, "arquivo"))
	if err != nil {
		return nil
	}
	antes, err1 := os.ReadFile(filepath.Join(a.pastaReferencia(t.ReferenciaID), string(nome)))
	depois, err2 := os.ReadFile(filepath.Join(prop, "texto.md"))
	if err1 != nil || err2 != nil {
		return nil
	}
	return trechosMudados(string(antes), string(depois))
}

// ---------- Várias propostas no mesmo texto ----------

var errPropostaCruzada = errors.New("O texto mudou depois desta proposta, no mesmo trecho: descarte e peça de novo.")

// juntarComAtual: a proposta foi feita sobre base.md; se o texto já mudou (outra proposta aprovada
// antes), as duas mudanças se juntam como num merge do git. Mudanças no mesmo trecho são recusadas.
func juntarComAtual(ctx context.Context, atual, base, proposta []byte) ([]byte, error) {
	if string(atual) == string(base) {
		return proposta, nil
	}
	tmp, err := os.MkdirTemp("", "juntar-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	arquivos := map[string][]byte{"atual": atual, "base": base, "proposta": proposta}
	for nome, b := range arquivos {
		if err := os.WriteFile(filepath.Join(tmp, nome), b, 0o644); err != nil {
			return nil, err
		}
	}
	cmd := exec.CommandContext(ctx, "git", "merge-file", "-p", filepath.Join(tmp, "atual"), filepath.Join(tmp, "base"), filepath.Join(tmp, "proposta"))
	out, err := cmd.Output()
	var saida *exec.ExitError
	if errors.As(err, &saida) && saida.ExitCode() > 0 {
		return nil, errPropostaCruzada
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

// respostaJaPedida: o pedido repetido devolve a proposta que já existe, sem criar outra.
func respostaJaPedida(w http.ResponseWriter, p *pendencia) {
	writeJSON(w, http.StatusOK, map[string]any{"tarefa": p.TarefaID, "estado": p.Estado, "ja_pedida": true})
}
