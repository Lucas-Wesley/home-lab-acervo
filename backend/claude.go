package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Ferramentas que o Claude pode usar: nas etapas, só ler e escrever arquivos na pasta da tarefa.
// A internet fica para "Mais informações" (D26) e "Perguntar" (D35), que não mexem em arquivo nenhum.
const (
	ferramentasEscrita   = "Read,Write,Edit,Glob,Grep"
	ferramentasLeitura   = "Read,Glob,Grep"
	ferramentasInternet  = "WebSearch,WebFetch"
	ferramentasProibidas = "Bash,WebFetch,WebSearch,Task,NotebookEdit"
)

// proibidas tira da lista de proibidas o que a tarefa pode usar.
func proibidas(permitidas string) string {
	var lista []string
	for _, f := range strings.Split(ferramentasProibidas, ",") {
		if !strings.Contains(","+permitidas+",", ","+f+",") {
			lista = append(lista, f)
		}
	}
	return strings.Join(lista, ",")
}

type pedidoClaude struct {
	pasta       string   // diretório de trabalho
	extras      []string // outras pastas que ele pode ler
	regra       string   // arquivo em regras/, ex.: "organizar.md"; vale a versão editada no app, se houver
	instrucao   string   // o pedido específico desta tarefa
	ferramentas string
	esquema     string // JSON Schema da resposta, quando a tarefa devolve dados
	modelo      string // outro modelo só nesta chamada (ex.: "haiku" para marcar livros, D38)
}

type respostaClaude struct {
	Resultado   string          `json:"result"`
	Erro        bool            `json:"is_error"`
	Estruturado json.RawMessage `json:"structured_output"`
}

func (a *app) chamarClaude(ctx context.Context, p pedidoClaude) (respostaClaude, error) {
	var r respostaClaude
	regra, _, err := a.lerRegra(p.regra)
	if err != nil {
		return r, fmt.Errorf("regra %s não encontrada: %w", p.regra, err)
	}
	prompt := p.instrucao + "\n\nSiga as regras abaixo.\n\n" + regra

	args := []string{"-p", prompt, "--output-format", "json",
		"--allowedTools", p.ferramentas, "--disallowedTools", proibidas(p.ferramentas)}
	for _, d := range p.extras {
		args = append(args, "--add-dir", d)
	}
	if p.esquema != "" {
		args = append(args, "--json-schema", p.esquema)
	}
	if modelo := primeiro(p.modelo, a.cfg.modelo); modelo != "" {
		args = append(args, "--model", modelo)
	}

	cmd := exec.CommandContext(ctx, a.cfg.claude, args...)
	cmd.Env = ambienteDoClaude()
	cmd.Dir = p.pasta
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	errExec := cmd.Run()
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if jerr := json.Unmarshal(saida.Bytes(), &r); jerr != nil {
		msg := primeiraLinha(erros.String(), primeiraLinha(saida.String(), "sem resposta"))
		if errExec != nil && msg == "sem resposta" {
			msg = errExec.Error()
		}
		if errExec != nil {
			return r, fmt.Errorf("claude falhou: %s", msg)
		}
		return r, fmt.Errorf("resposta do claude ilegível: %s", msg)
	}
	if r.Erro || errExec != nil {
		msg := primeiraLinha(r.Resultado, primeiraLinha(erros.String(), "erro sem mensagem"))
		if strings.Contains(msg, "Not logged in") || strings.Contains(msg, "/login") {
			return r, errors.New("O Claude recusou o login: o token da conta ativa venceu ou falta. Cole um token novo ou troque de conta no Hub, na tela Claude.")
		}
		return r, fmt.Errorf("claude falhou: %s", msg)
	}
	return r, nil
}

// dadosDaResposta lê a resposta estruturada; sem ela, procura um JSON no texto.
func dadosDaResposta(r respostaClaude, destino any) error {
	if len(r.Estruturado) > 0 && string(r.Estruturado) != "null" {
		return json.Unmarshal(r.Estruturado, destino)
	}
	texto := r.Resultado
	ini, fim := strings.Index(texto, "{"), strings.LastIndex(texto, "}")
	if ini < 0 || fim < ini {
		return fmt.Errorf("o claude não devolveu dados: %s", primeiraLinha(texto, "resposta vazia"))
	}
	return json.Unmarshal([]byte(texto[ini:fim+1]), destino)
}
