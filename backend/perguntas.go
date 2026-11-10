package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Perguntar (D35): o Lucas faz uma pergunta sobre um Tema ou uma Referência. A IA vê quanto o texto
// responde e verifica a resposta na internet (regras/perguntar.md); o resultado aponta onde o texto
// está errado, desatualizado ou incompleto. Como em "Mais informações", fica guardado à parte.
type perguntaFeita struct {
	ID           string            `json:"id"`
	Local        string            `json:"local"` // temas/<pasta> ou referencias/<id>
	Titulo       string            `json:"titulo"`
	Pergunta     string            `json:"pergunta"`
	Estado       string            `json:"estado"` // processando, pronto ou falhou
	Erro         string            `json:"erro,omitempty"`
	Resultado    *respostaPergunta `json:"resultado"`
	TarefaID     int64             `json:"tarefa_id"`
	CriadoEm     string            `json:"criado_em"`
	AtualizadoEm string            `json:"atualizado_em"`
}

type respostaPergunta struct {
	Cobertura    string        `json:"cobertura"` // responde, em_parte, toca, nao_responde
	NoTexto      string        `json:"no_texto"`
	Resposta     string        `json:"resposta"`
	Selo         string        `json:"selo"` // confirmado, uma_fonte, divergem, nao_encontrado
	Divergencias []divergencia `json:"divergencias"`
	Fontes       []consultada  `json:"fontes"`
}

type divergencia struct {
	Tipo     string `json:"tipo"` // erro, desatualizado, omissao, melhoria
	Capitulo string `json:"capitulo"`
	NoTexto  string `json:"no_texto"`
	Correto  string `json:"correto"`
	Selo     string `json:"selo"`
	Fontes   []int  `json:"fontes"`
}

var localDePergunta = regexp.MustCompile(`^(temas/[a-z0-9/-]+|referencias/[A-Za-z0-9_-]+)$`)

const maxPergunta = 500

const esquemaPergunta = `{"type":"object","properties":{` +
	`"cobertura":{"type":"string","enum":["responde","em_parte","toca","nao_responde"]},` +
	`"no_texto":{"type":"string"},"resposta":{"type":"string"},` +
	`"selo":{"type":"string","enum":["confirmado","uma_fonte","divergem","nao_encontrado"]},` +
	`"divergencias":{"type":"array","items":{"type":"object","properties":{` +
	`"tipo":{"type":"string","enum":["erro","desatualizado","omissao","melhoria"]},` +
	`"capitulo":{"type":"string"},"no_texto":{"type":"string"},"correto":{"type":"string"},` +
	`"selo":{"type":"string","enum":["confirmado","uma_fonte","divergem","nao_encontrado"]},` +
	`"fontes":{"type":"array","items":{"type":"integer"}}},` +
	`"required":["tipo","capitulo","no_texto","correto","selo","fontes"]}},` +
	`"fontes":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"url":{"type":"string"}},"required":["titulo","url"]}}},` +
	`"required":["cobertura","no_texto","resposta","selo","divergencias","fontes"]}`

const consultaPergunta = `SELECT p.id, p.local, p.titulo, p.pergunta, p.resultado, p.tarefa_id, p.criado_em,
	p.atualizado_em, COALESCE(t.estado, ''), COALESCE(t.erro, '')
	FROM perguntas p LEFT JOIN tarefas t ON t.id = p.tarefa_id`

func lerPergunta(row linha) (perguntaFeita, error) {
	var p perguntaFeita
	var resultado, estadoTarefa, erroTarefa string
	err := row.Scan(&p.ID, &p.Local, &p.Titulo, &p.Pergunta, &resultado, &p.TarefaID, &p.CriadoEm,
		&p.AtualizadoEm, &estadoTarefa, &erroTarefa)
	if err != nil {
		return p, err
	}
	if resultado != "" {
		p.Resultado = &respostaPergunta{}
		json.Unmarshal([]byte(resultado), p.Resultado)
	}
	switch {
	case estadoTarefa == naFila || estadoTarefa == rodando:
		p.Estado = "processando"
	case p.Resultado != nil:
		p.Estado = "pronto"
	case estadoTarefa == cancelada:
		p.Estado, p.Erro = "falhou", "A pergunta foi cancelada na fila."
	default:
		p.Estado, p.Erro = "falhou", erroTarefa
		if p.Erro == "" {
			p.Erro = "A verificação terminou sem resposta."
		}
	}
	return p, nil
}

func (a *app) perguntaFeita(ctx context.Context, id string) (perguntaFeita, error) {
	return lerPergunta(a.db.QueryRowContext(ctx, consultaPergunta+` WHERE p.id = ?`, id))
}

// handlePerguntas lista as perguntas de um texto (?local=), a mais nova primeiro.
func (a *app) handlePerguntas(w http.ResponseWriter, r *http.Request) {
	local := r.URL.Query().Get("local")
	rows, err := a.db.QueryContext(r.Context(), consultaPergunta+` WHERE p.local = ? ORDER BY p.criado_em DESC, p.rowid DESC`, local)
	if err != nil {
		falha(w, err)
		return
	}
	defer rows.Close()
	lista := []perguntaFeita{}
	for rows.Next() {
		p, err := lerPergunta(rows)
		if err != nil {
			falha(w, err)
			return
		}
		lista = append(lista, p)
	}
	if err := rows.Err(); err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, lista)
}

// localValido confere que o texto existe: a pasta do Tema ou a Referência.
func (a *app) localValido(ctx context.Context, local string) bool {
	if !localDePergunta.MatchString(local) {
		return false
	}
	if id, ok := strings.CutPrefix(local, "referencias/"); ok {
		_, err := a.referencia(ctx, id)
		return err == nil
	}
	if _, err := caminhoDeTema(local); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(a.conteudo.raiz, filepath.FromSlash(local), "indice.md"))
	return err == nil
}

// handlePerguntar: POST /api/perguntas {local, titulo, pergunta}; a verificação vai para a fila.
func (a *app) handlePerguntar(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Local    string `json:"local"`
		Titulo   string `json:"titulo"`
		Pergunta string `json:"pergunta"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	texto := strings.TrimSpace(corpo.Pergunta)
	switch {
	case !a.localValido(r.Context(), corpo.Local):
		erroJSON(w, http.StatusBadRequest, "Texto inválido.")
		return
	case texto == "":
		erroJSON(w, http.StatusBadRequest, "Escreva a pergunta.")
		return
	case utf8.RuneCountInString(texto) > maxPergunta:
		erroJSON(w, http.StatusBadRequest, fmt.Sprintf("Pergunta longa demais: até %d caracteres.", maxPergunta))
		return
	}
	ctx := r.Context()
	id := novoID()
	_, err := a.db.ExecContext(ctx, `INSERT INTO perguntas (id, local, titulo, pergunta, criado_em, atualizado_em)
		VALUES (?, ?, ?, ?, ?, ?)`, id, corpo.Local, strings.TrimSpace(corpo.Titulo), texto, agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if err := a.verificarPergunta(ctx, id); err != nil {
		falha(w, err)
		return
	}
	a.responderPergunta(w, r, id, http.StatusCreated)
}

// handleRefazerPergunta verifica de novo; a resposta antiga sai.
func (a *app) handleRefazerPergunta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := a.perguntaFeita(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Pergunta não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	if p.Estado == "processando" {
		erroJSON(w, http.StatusConflict, "A pergunta ainda está na fila.")
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `UPDATE perguntas SET resultado = '', atualizado_em = ? WHERE id = ?`, agora(), id); err != nil {
		falha(w, err)
		return
	}
	if err := a.verificarPergunta(r.Context(), id); err != nil {
		falha(w, err)
		return
	}
	a.responderPergunta(w, r, id, http.StatusOK)
}

func (a *app) handleApagarPergunta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if p, err := a.perguntaFeita(r.Context(), id); err == nil && p.Estado == "processando" {
		a.fila.cancelarTarefa(r.Context(), p.TarefaID)
	}
	if _, err := a.db.ExecContext(r.Context(), `DELETE FROM perguntas WHERE id = ?`, id); err != nil {
		falha(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) responderPergunta(w http.ResponseWriter, r *http.Request, id string, status int) {
	p, err := a.perguntaFeita(r.Context(), id)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, status, p)
}

type paramsPergunta struct {
	Pergunta string `json:"pergunta"`
}

func (a *app) verificarPergunta(ctx context.Context, id string) error {
	tarefa, err := a.fila.enfileirar(ctx, "", "perguntar", paramsPergunta{id})
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE perguntas SET tarefa_id = ? WHERE id = ?`, tarefa, id)
	return err
}

// copiarTexto põe numa pasta só o texto da pergunta: os arquivos da página do Tema (sem os
// subtemas) ou o texto da Referência, em português quando houver tradução.
func (a *app) copiarTexto(ctx context.Context, local, destino string) error {
	if id, ok := strings.CutPrefix(local, "referencias/"); ok {
		ref, err := a.referencia(ctx, id)
		if err != nil {
			return errors.New("a Referência foi excluída")
		}
		texto, err := a.textoParaAcervo(ref)
		if err != nil {
			texto = filepath.Join(a.pastaReferencia(id), "organizado.md")
			if exigirArquivo(texto) != nil {
				return errors.New("a Referência ainda não tem texto organizado")
			}
		}
		return copiarArquivo(texto, filepath.Join(destino, "texto.md"))
	}
	origem := filepath.Join(a.conteudo.raiz, filepath.FromSlash(local))
	arquivos, err := filepath.Glob(filepath.Join(origem, "*.md"))
	if err != nil || len(arquivos) == 0 {
		return errors.New("o Tema não existe mais")
	}
	for _, arq := range arquivos {
		if err := copiarArquivo(arq, filepath.Join(destino, filepath.Base(arq))); err != nil {
			return err
		}
	}
	return nil
}

func copiarArquivo(de, para string) error {
	b, err := os.ReadFile(de)
	if err != nil {
		return err
	}
	return os.WriteFile(para, b, 0o644)
}

// perguntar é a tarefa da fila: lê só a cópia do texto e pesquisa na internet.
func (a *app) perguntar(ctx context.Context, t *tarefa) (resultado, error) {
	var params paramsPergunta
	if err := json.Unmarshal(t.Params, &params); err != nil {
		return pronto, err
	}
	p, err := a.perguntaFeita(ctx, params.Pergunta)
	if errors.Is(err, sql.ErrNoRows) {
		t.Erro = "A pergunta foi apagada."
		return pronto, nil
	}
	if err != nil {
		return pronto, err
	}
	pasta, err := os.MkdirTemp(a.cfg.dados, "pergunta-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(pasta)
	if err := a.copiarTexto(ctx, p.Local, pasta); err != nil {
		return pronto, err
	}

	titulo := p.Titulo
	if titulo == "" {
		titulo = p.Local
	}
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta: pasta,
		regra: "perguntar.md",
		instrucao: fmt.Sprintf("O Lucas está lendo o texto %q (os arquivos estão nesta pasta) e perguntou:\n\n%q\n\n"+
			"Veja quanto o texto responde, verifique a resposta na internet e devolva o resultado. Hoje é %s.",
			titulo, p.Pergunta, time.Now().Format("2006-01-02")),
		ferramentas: ferramentasLeitura + "," + ferramentasInternet,
		esquema:     esquemaPergunta,
	})
	if err != nil {
		return pronto, err
	}
	var saida respostaPergunta
	if err := dadosDaResposta(r, &saida); err != nil {
		return pronto, err
	}
	if strings.TrimSpace(saida.Resposta) == "" {
		return pronto, errors.New("o claude não devolveu resposta")
	}
	if saida.Divergencias == nil {
		saida.Divergencias = []divergencia{}
	}
	if saida.Fontes == nil {
		saida.Fontes = []consultada{}
	}
	resposta, _ := json.Marshal(saida)
	_, err = a.db.ExecContext(ctx, `UPDATE perguntas SET resultado = ?, atualizado_em = ? WHERE id = ? AND tarefa_id = ?`,
		string(resposta), agora(), p.ID, t.ID)
	return pronto, err
}
