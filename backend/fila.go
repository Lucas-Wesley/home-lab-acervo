package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"sync"
)

// Estados de uma tarefa. "em_revisao" é uma proposta da IA aguardando o Lucas (D16).
const (
	naFila    = "na_fila"
	rodando   = "rodando"
	emRevisao = "em_revisao"
	concluida = "concluida"
	falhou    = "falhou"
	cancelada = "cancelada"
	rejeitada = "rejeitada"
)

type tarefa struct {
	ID           int64           `json:"id"`
	ReferenciaID string          `json:"referencia_id"`
	Tipo         string          `json:"tipo"`
	Params       json.RawMessage `json:"params"`
	Estado       string          `json:"estado"`
	Erro         string          `json:"erro"`
	CriadoEm     string          `json:"criado_em"`
	IniciadoEm   string          `json:"iniciado_em"`
	TerminadoEm  string          `json:"terminado_em"`
}

// resultado diz como a tarefa terminou quando não houve erro.
type resultado int

const (
	pronto      resultado = iota // concluída
	paraRevisao                  // proposta na worktree, espera aprovação
)

type executor func(ctx context.Context, t *tarefa) (resultado, error)

// fila roda uma tarefa por vez (D14): as etapas chamam o Claude e levam minutos.
type fila struct {
	db         *sql.DB
	executores map[string]executor
	acorda     chan struct{}

	mu       sync.Mutex
	atual    int64
	cancelar context.CancelFunc
}

func novaFila(db *sql.DB, executores map[string]executor) *fila {
	return &fila{db: db, executores: executores, acorda: make(chan struct{}, 1)}
}

// recuperar devolve à fila o que estava rodando quando o app parou.
func (f *fila) recuperar(ctx context.Context) error {
	_, err := f.db.ExecContext(ctx, `UPDATE tarefas SET estado = ?, iniciado_em = '' WHERE estado = ?`, naFila, rodando)
	return err
}

func (f *fila) enfileirar(ctx context.Context, referenciaID, tipo string, params any) (int64, error) {
	if params == nil {
		params = map[string]any{}
	}
	p, err := json.Marshal(params)
	if err != nil {
		return 0, err
	}
	res, err := f.db.ExecContext(ctx,
		`INSERT INTO tarefas (referencia_id, tipo, params, estado, criado_em) VALUES (?, ?, ?, ?, ?)`,
		referenciaID, tipo, string(p), naFila, agora())
	if err != nil {
		return 0, err
	}
	select {
	case f.acorda <- struct{}{}:
	default:
	}
	return res.LastInsertId()
}

func (f *fila) rodar(ctx context.Context) {
	for {
		t, err := f.proxima(ctx)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("fila: %v", err)
		}
		if t == nil {
			select {
			case <-ctx.Done():
				return
			case <-f.acorda:
			}
			continue
		}
		f.executar(ctx, t)
	}
}

func (f *fila) proxima(ctx context.Context) (*tarefa, error) {
	row := f.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` FROM tarefas WHERE estado = ? ORDER BY id LIMIT 1`, naFila)
	t, err := lerTarefa(row)
	if err != nil {
		return nil, err
	}
	t.Estado, t.IniciadoEm = rodando, agora()
	_, err = f.db.ExecContext(ctx, `UPDATE tarefas SET estado = ?, iniciado_em = ? WHERE id = ?`, t.Estado, t.IniciadoEm, t.ID)
	return t, err
}

func (f *fila) executar(ctx context.Context, t *tarefa) {
	exec, ok := f.executores[t.Tipo]
	if !ok {
		f.terminar(t, falhou, "tipo de tarefa desconhecido: "+t.Tipo)
		return
	}
	tctx, cancel := context.WithCancel(ctx)
	f.mu.Lock()
	f.atual, f.cancelar = t.ID, cancel
	f.mu.Unlock()

	res, err := exec(tctx, t)
	cancelou := tctx.Err() != nil && ctx.Err() == nil
	cancel()

	f.mu.Lock()
	f.atual, f.cancelar = 0, nil
	f.mu.Unlock()

	switch {
	case ctx.Err() != nil:
		// O app está parando: a tarefa volta para a fila no próximo início.
	case cancelou:
		f.terminar(t, cancelada, "")
	case err != nil:
		log.Printf("tarefa %d (%s): %v", t.ID, t.Tipo, err)
		f.terminar(t, falhou, err.Error())
	case res == paraRevisao:
		f.terminar(t, emRevisao, "")
	default:
		// Numa tarefa concluída, "erro" guarda só uma nota (ex.: nada a integrar).
		f.terminar(t, concluida, t.Erro)
	}
}

func (f *fila) terminar(t *tarefa, estado, erro string) {
	t.Estado, t.Erro = estado, erro
	_, err := f.db.Exec(`UPDATE tarefas SET estado = ?, erro = ?, terminado_em = ? WHERE id = ?`, estado, erro, agora(), t.ID)
	if err != nil {
		log.Printf("fila: %v", err)
	}
}

// cancelarTarefa tira da fila ou interrompe a que está rodando.
func (f *fila) cancelarTarefa(ctx context.Context, id int64) error {
	f.mu.Lock()
	if f.atual == id && f.cancelar != nil {
		f.cancelar()
		f.mu.Unlock()
		return nil
	}
	f.mu.Unlock()
	res, err := f.db.ExecContext(ctx, `UPDATE tarefas SET estado = ?, terminado_em = ? WHERE id = ? AND estado = ?`, cancelada, agora(), id, naFila)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNaoCancelavel
	}
	return nil
}

var errNaoCancelavel = errors.New("só dá para cancelar tarefa na fila ou rodando")

const colunasTarefa = `id, referencia_id, tipo, params, estado, erro, criado_em, iniciado_em, terminado_em`

type linha interface{ Scan(dest ...any) error }

func lerTarefa(row linha) (*tarefa, error) {
	var t tarefa
	var params string
	if err := row.Scan(&t.ID, &t.ReferenciaID, &t.Tipo, &params, &t.Estado, &t.Erro, &t.CriadoEm, &t.IniciadoEm, &t.TerminadoEm); err != nil {
		return nil, err
	}
	t.Params = json.RawMessage(params)
	return &t, nil
}

func (f *fila) tarefa(ctx context.Context, id int64) (*tarefa, error) {
	return lerTarefa(f.db.QueryRowContext(ctx, `SELECT `+colunasTarefa+` FROM tarefas WHERE id = ?`, id))
}

func (f *fila) listar(ctx context.Context, onde string, args ...any) ([]tarefa, error) {
	rows, err := f.db.QueryContext(ctx, `SELECT `+colunasTarefa+` FROM tarefas `+onde, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []tarefa{}
	for rows.Next() {
		t, err := lerTarefa(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, *t)
	}
	return lista, rows.Err()
}
