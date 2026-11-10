package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Integrar ao texto (D36): o que uma pergunta (D35) ou um "Mais informações" (D26) achou entra no
// texto, no capítulo e no ponto certos, pela regra integrar-texto.md. Em cada divergência da
// pergunta, o Lucas escolhe Substituir, Manter as duas versões (ou Acrescentar, no que falta) ou
// Ignorar; no "Mais informações", o texto completo ou só o que serve. Tudo vira proposta para
// revisar pelo diff, como uma integração (D16): num Tema, na worktree do git; numa Referência
// (fora do git), numa cópia do texto, e o texto de antes fica como Versão anterior ao aprovar.

type escolhaDeDivergencia struct {
	Divergencia int    `json:"divergencia"` // índice em resultado.divergencias
	Acao        string `json:"acao"`        // substituir, manter, acrescentar ou ignorar
}

type paramsIntegrarTexto struct {
	Origem   string                 `json:"origem"` // pergunta ou detalhe
	ID       string                 `json:"id"`
	Tema     string                 `json:"tema,omitempty"` // a pasta do Tema; vazio numa Referência
	Arquivo  string                 `json:"arquivo,omitempty"`
	Titulo   string                 `json:"titulo"`
	Resposta bool                   `json:"resposta,omitempty"`
	Escolhas []escolhaDeDivergencia `json:"escolhas,omitempty"`
	Modo     string                 `json:"modo,omitempty"` // completo ou adaptado
}

var acoesDeDivergencia = map[string]string{
	"substituir":  "Substituir",
	"manter":      "Manter as duas versões",
	"acrescentar": "Acrescentar",
	"ignorar":     "Ignorar",
}

// handleIntegrarPergunta: POST /api/perguntas/{id}/integrar {resposta, escolhas}.
func (a *app) handleIntegrarPergunta(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, err := a.perguntaFeita(ctx, r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusNotFound, "Pergunta não encontrada.")
		return
	}
	if p.Resultado == nil || p.Estado != "pronto" {
		erroJSON(w, http.StatusConflict, "A pergunta ainda não tem resposta.")
		return
	}
	var corpo struct {
		Resposta bool                   `json:"resposta"`
		Escolhas []escolhaDeDivergencia `json:"escolhas"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	algo := corpo.Resposta
	vistas := map[int]bool{}
	for _, e := range corpo.Escolhas {
		if e.Divergencia < 0 || e.Divergencia >= len(p.Resultado.Divergencias) || vistas[e.Divergencia] || acoesDeDivergencia[e.Acao] == "" {
			erroJSON(w, http.StatusBadRequest, "Escolha inválida.")
			return
		}
		vistas[e.Divergencia] = true
		algo = algo || e.Acao != "ignorar"
	}
	if !algo {
		erroJSON(w, http.StatusBadRequest, "Nada para integrar: tudo ficou em Ignorar.")
		return
	}
	params := paramsIntegrarTexto{Origem: "pergunta", ID: p.ID, Titulo: p.Titulo, Resposta: corpo.Resposta, Escolhas: corpo.Escolhas}
	a.enfileirarIntegracao(w, r, p.Local, params)
}

// handleIntegrarDetalhe: POST /api/detalhes/{id}/integrar {modo: completo | adaptado}.
func (a *app) handleIntegrarDetalhe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d, err := a.detalhe(ctx, r.PathValue("id"))
	if err != nil {
		erroJSON(w, http.StatusNotFound, "Marcação não encontrada.")
		return
	}
	if d.Estado != "pronto" {
		erroJSON(w, http.StatusConflict, "A pesquisa deste trecho ainda não terminou.")
		return
	}
	var corpo struct {
		Modo string `json:"modo"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	if corpo.Modo != "completo" && corpo.Modo != "adaptado" {
		erroJSON(w, http.StatusBadRequest, "Escolha o texto completo ou só o que serve ao texto.")
		return
	}
	local := d.Local
	params := paramsIntegrarTexto{Origem: "detalhe", ID: d.ID, Titulo: d.Titulo, Modo: corpo.Modo}
	if strings.HasPrefix(local, "temas/") {
		params.Arquivo = local
		local = path.Dir(local)
	} else {
		local = path.Dir(local) // referencias/<id>/<aba> → referencias/<id>
	}
	a.enfileirarIntegracao(w, r, local, params)
}

// enfileirarIntegracao acha o texto (Tema ou Referência) e põe a tarefa na fila.
func (a *app) enfileirarIntegracao(w http.ResponseWriter, r *http.Request, local string, p paramsIntegrarTexto) {
	ctx := r.Context()
	refID := ""
	if id, ok := strings.CutPrefix(local, "referencias/"); ok {
		ref, err := a.referencia(ctx, id)
		if err != nil {
			erroJSON(w, http.StatusNotFound, "Referência não encontrada.")
			return
		}
		if _, err := a.textoParaAcervo(ref); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
		if a.ocupada(ctx, ref.ID) {
			erroJSON(w, http.StatusConflict, "A Referência já tem uma tarefa em andamento.")
			return
		}
		refID = ref.ID
	} else {
		// No Tema, a pasta que tem o indice.md (o arquivo pode ser um capítulo separado).
		tema := local
		for tema != "temas" && !existe(filepath.Join(a.conteudo.raiz, filepath.FromSlash(tema), "indice.md")) {
			tema = path.Dir(tema)
		}
		if tema == "temas" || !strings.HasPrefix(tema, "temas/") {
			erroJSON(w, http.StatusNotFound, "Tema não encontrado.")
			return
		}
		p.Tema = tema
	}
	id, err := a.fila.enfileirar(ctx, refID, "integrar_texto", p)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"tarefa": id})
}

// materialDaIntegracao escreve o que integrar, com as escolhas do Lucas, para a IA ler.
func (a *app) materialDaIntegracao(ctx context.Context, p paramsIntegrarTexto) (string, error) {
	var s strings.Builder
	fontes := []consultada{}
	switch p.Origem {
	case "pergunta":
		q, err := a.perguntaFeita(ctx, p.ID)
		if err != nil || q.Resultado == nil {
			return "", errors.New("a pergunta não existe mais")
		}
		res := q.Resultado
		fontes = res.Fontes
		fmt.Fprintf(&s, "# Integrar ao texto: resposta a uma pergunta\n\n## A pergunta do Lucas\n\n%s\n\n", q.Pergunta)
		if p.Resposta {
			fmt.Fprintf(&s, "## A resposta verificada (integrar: sim)\n\n%s\n\n", res.Resposta)
		} else {
			s.WriteString("## A resposta verificada (integrar: não; só as divergências escolhidas abaixo)\n\n")
		}
		escolhas := map[int]string{}
		for _, e := range p.Escolhas {
			escolhas[e.Divergencia] = e.Acao
		}
		if len(res.Divergencias) > 0 {
			s.WriteString("## Divergências e a escolha do Lucas\n\n")
		}
		for i, d := range res.Divergencias {
			acao := escolhas[i]
			if acao == "" {
				acao = "ignorar"
			}
			fmt.Fprintf(&s, "%d. **%s** (%s", i+1, acoesDeDivergencia[acao], d.Tipo)
			if d.Capitulo != "" {
				fmt.Fprintf(&s, ", capítulo %q", d.Capitulo)
			}
			s.WriteString(")\n")
			if d.NoTexto != "" {
				fmt.Fprintf(&s, "   - O texto diz: %q\n", d.NoTexto)
			}
			fmt.Fprintf(&s, "   - As fontes dizem: %s\n", d.Correto)
			if len(d.Fontes) > 0 {
				ns := make([]string, len(d.Fontes))
				for j, n := range d.Fontes {
					ns[j] = fmt.Sprintf("[%d]", n)
				}
				fmt.Fprintf(&s, "   - Fontes: %s\n", strings.Join(ns, " "))
			}
		}
	case "detalhe":
		d, err := a.detalhe(ctx, p.ID)
		if err != nil || d.Explicacao == "" {
			return "", errors.New("a marcação não existe mais")
		}
		fontes = d.Fontes
		modo := "**O texto completo**: a explicação entra inteira, como está, logo depois do parágrafo do trecho."
		if p.Modo == "adaptado" {
			modo = "**Só o que serve ao texto**: use apenas o que acrescenta, reescrito no estilo do texto."
		}
		fmt.Fprintf(&s, "# Integrar ao texto: \"Mais informações\" sobre um trecho\n\nModo escolhido pelo Lucas: %s\n\n", modo)
		if p.Arquivo != "" {
			fmt.Fprintf(&s, "Arquivo do trecho: `%s`\n\n", p.Arquivo)
		}
		fmt.Fprintf(&s, "## O trecho selecionado\n\n> %s\n\n(antes: “…%s”; depois: “%s…”)\n\n## A explicação\n\n%s\n\n",
			strings.ReplaceAll(d.Trecho, "\n", "\n> "), d.Antes, d.Depois, d.Explicacao)
	default:
		return "", errors.New("origem desconhecida")
	}
	if len(fontes) > 0 {
		s.WriteString("\n## Fontes\n\n")
		for i, f := range fontes {
			fmt.Fprintf(&s, "%d. [%s](%s)\n", i+1, primeiro(f.Titulo, f.URL), f.URL)
		}
	}
	return s.String(), nil
}

func (a *app) integrarTexto(ctx context.Context, t *tarefa) (resultado, error) {
	var p paramsIntegrarTexto
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	material, err := a.materialDaIntegracao(ctx, p)
	if err != nil {
		return pronto, err
	}
	tmp, err := os.MkdirTemp("", "integrar-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(tmp)
	if err := os.WriteFile(filepath.Join(tmp, "material.md"), []byte(material), 0o644); err != nil {
		return pronto, err
	}
	hoje := time.Now().Format("2006-01-02")

	if p.Tema != "" {
		dir, err := a.conteudo.abrirRevisao(ctx, t.ID)
		if err != nil {
			return pronto, err
		}
		_, err = a.chamarClaude(ctx, pedidoClaude{
			pasta:  dir,
			extras: []string{tmp},
			regra:  "integrar-texto.md",
			instrucao: fmt.Sprintf("O que integrar está em %s. O Lucas pediu para integrar ao texto do Tema da pasta %s/ (caminho relativo a esta pasta). "+
				"Hoje é %s. Escreva só dentro dessa pasta. Ao terminar, responda só 'ok', ou 'nada novo' se não houver nada a mudar.",
				filepath.Join(tmp, "material.md"), p.Tema, hoje),
			ferramentas: ferramentasEscrita,
		})
		if err == nil {
			err = a.conferirProposta(ctx, t.ID, p.Tema, false)
		}
		if errors.Is(err, errSemMudancas) {
			a.conteudo.descartarRevisao(context.Background(), t.ID)
			t.Erro = "Nada mudou no texto."
			return pronto, nil
		}
		if err != nil {
			a.conteudo.descartarRevisao(context.Background(), t.ID)
			return pronto, err
		}
		return paraRevisao, nil
	}

	// Referência: a IA edita uma cópia do texto, numa pasta de proposta.
	ref, err := a.referencia(ctx, t.ReferenciaID)
	if err != nil {
		return pronto, err
	}
	original, err := a.textoParaAcervo(ref)
	if err != nil {
		return pronto, err
	}
	antes, err := os.ReadFile(original)
	if err != nil {
		return pronto, err
	}
	prop := a.pastaDaProposta(t.ID)
	os.RemoveAll(prop)
	if err := os.MkdirAll(prop, 0o755); err != nil {
		return pronto, err
	}
	if err := os.WriteFile(filepath.Join(prop, "texto.md"), antes, 0o644); err != nil {
		return pronto, err
	}
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta:  prop,
		extras: []string{tmp},
		regra:  "integrar-texto.md",
		instrucao: fmt.Sprintf("O que integrar está em %s. O Lucas pediu para integrar ao texto desta Referência (%q): edite texto.md nesta pasta. "+
			"Hoje é %s. Ao terminar, responda só 'ok', ou 'nada novo' se não houver nada a mudar.",
			filepath.Join(tmp, "material.md"), ref.Titulo, hoje),
		ferramentas: ferramentasEscrita,
	})
	if err != nil {
		os.RemoveAll(prop)
		return pronto, err
	}
	depois, err := os.ReadFile(filepath.Join(prop, "texto.md"))
	if err != nil || strings.TrimSpace(string(depois)) == "" {
		os.RemoveAll(prop)
		return pronto, errors.New("a IA apagou o texto; a proposta foi descartada")
	}
	if string(depois) == string(antes) {
		os.RemoveAll(prop)
		t.Erro = "Nada mudou no texto."
		return pronto, nil
	}
	return paraRevisao, os.WriteFile(filepath.Join(prop, "arquivo"), []byte(filepath.Base(original)), 0o644)
}

// pastaDaProposta: onde fica a cópia editada do texto de uma Referência até o Lucas decidir.
func (a *app) pastaDaProposta(id int64) string {
	return filepath.Join(a.cfg.dados, "revisoes", fmt.Sprintf("texto-%d", id))
}

// diffDaProposta mostra a mudança no texto da Referência no mesmo formato do git das outras revisões.
func (a *app) diffDaProposta(ctx context.Context, t *tarefa) (string, error) {
	prop := a.pastaDaProposta(t.ID)
	nome, err := os.ReadFile(filepath.Join(prop, "arquivo"))
	if err != nil {
		return "", errors.New("a proposta não existe mais")
	}
	original := filepath.Join(a.pastaReferencia(t.ReferenciaID), string(nome))
	cmd := exec.CommandContext(ctx, "git", "diff", "--no-index", "--no-color", original, filepath.Join(prop, "texto.md"))
	out, err := cmd.Output()
	var saida *exec.ExitError
	if err != nil && !(errors.As(err, &saida) && saida.ExitCode() == 1) { // 1 = há diferença
		return "", err
	}
	// Os caminhos de disco viram o caminho do texto, como nas outras revisões.
	logico := "referencias/" + t.ReferenciaID + "/" + string(nome)
	diff := strings.NewReplacer(
		"a"+original, "a/"+logico,
		"b"+filepath.Join(prop, "texto.md"), "b/"+logico,
	).Replace(string(out))
	return diff, nil
}

// decidirProposta aprova (o texto de antes vira anterior.md) ou descarta a proposta numa Referência.
func (a *app) decidirProposta(t *tarefa, aprovar bool) error {
	prop := a.pastaDaProposta(t.ID)
	defer os.RemoveAll(prop)
	if !aprovar {
		return nil
	}
	nome, err := os.ReadFile(filepath.Join(prop, "arquivo"))
	if err != nil {
		return errors.New("a proposta não existe mais")
	}
	pasta := a.pastaReferencia(t.ReferenciaID)
	original := filepath.Join(pasta, string(nome))
	novo, err := os.ReadFile(filepath.Join(prop, "texto.md"))
	if err != nil {
		return err
	}
	if atual, err := os.ReadFile(original); err == nil {
		if err := os.WriteFile(filepath.Join(pasta, "anterior.md"), atual, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(original, novo, 0o644)
}

// decidirIntegracao: aprovar ou rejeitar a proposta de integrar ao texto.
func (a *app) decidirIntegracao(w http.ResponseWriter, r *http.Request, t *tarefa) {
	ctx := r.Context()
	var p paramsIntegrarTexto
	json.Unmarshal(t.Params, &p)
	aprovar := r.PathValue("decisao") == "aprovar"
	if !aprovar && r.PathValue("decisao") != "rejeitar" {
		erroJSON(w, http.StatusNotFound, "Decisão desconhecida.")
		return
	}
	if p.Tema == "" {
		if err := a.decidirProposta(t, aprovar); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
	} else if aprovar {
		if err := a.conteudo.aprovarRevisao(ctx, t.ID, "Integrado ao texto: "+cortar(p.Titulo, 80, false)); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
	} else {
		a.conteudo.descartarRevisao(ctx, t.ID)
	}
	estado := rejeitada
	if aprovar {
		estado = concluida
	}
	a.fila.terminar(t, estado, "")
	writeJSON(w, http.StatusOK, map[string]string{"tema": p.Tema})
}
