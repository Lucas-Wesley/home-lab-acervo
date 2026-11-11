package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Conversar (D48): no lugar do Perguntar (D35), uma conversa com a IA sobre o texto aberto (Tema ou
// Referência), com as mensagens anteriores como memória. A IA entende a intenção: responde, procura
// no Acervo, pesquisa na internet quando precisa (para conferir ou acrescentar fatos) e, quando o
// Lucas pede para mudar ou acrescentar, devolve as alterações como "antes → depois". Nada muda no
// texto sem o Lucas tocar em Aplicar; no Tema a mudança vira commit, na Referência o texto de antes
// fica como Versão anterior. A mensagem pode ser sobre um trecho selecionado.

const (
	modoConversa      = "conversa"
	maxMensagem       = 2000
	maxTrechoConversa = 4000
	// memoriaDaConversa: quantas mensagens anteriores do mesmo texto a IA recebe.
	memoriaDaConversa = 8
)

var arquivoDeTema = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*\.md$`)

type respostaConversa struct {
	Resposta   string       `json:"resposta"`
	Pesquisou  bool         `json:"pesquisou"`
	Fontes     []consultada `json:"fontes"`
	NoAcervo   []noAcervo   `json:"no_acervo"`
	Alteracoes []alteracao  `json:"alteracoes"`
}

type noAcervo struct {
	Titulo  string `json:"titulo"`
	Caminho string `json:"caminho"` // temas/<pasta>[/arquivo.md] ou referencias/<id>
}

// alteracao: um trecho do texto como está (exato, aparece uma vez só no arquivo) e como fica.
type alteracao struct {
	Arquivo string `json:"arquivo"` // no Tema, o arquivo na pasta do Tema; na Referência, texto.md
	Antes   string `json:"antes"`
	Depois  string `json:"depois"`
	Resumo  string `json:"resumo"`
	Estado  string `json:"estado"` // proposta, aplicada ou descartada
}

const esquemaConversa = `{"type":"object","properties":{` +
	`"resposta":{"type":"string"},"pesquisou":{"type":"boolean"},` +
	`"fontes":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"url":{"type":"string"}},"required":["titulo","url"]}},` +
	`"no_acervo":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"caminho":{"type":"string"}},"required":["titulo","caminho"]}},` +
	`"alteracoes":{"type":"array","items":{"type":"object","properties":{` +
	`"arquivo":{"type":"string"},"antes":{"type":"string"},"depois":{"type":"string"},"resumo":{"type":"string"}},` +
	`"required":["arquivo","antes","depois","resumo"]}}},` +
	`"required":["resposta","pesquisou","fontes","no_acervo","alteracoes"]}`

// conversar é a tarefa da fila para uma mensagem do modo conversa.
func (a *app) conversar(ctx context.Context, t *tarefa, p perguntaFeita) (resultado, error) {
	pasta, err := os.MkdirTemp(a.cfg.dados, "conversa-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(pasta)
	texto := filepath.Join(pasta, "texto")
	if err := os.MkdirAll(texto, 0o755); err != nil {
		return pronto, err
	}
	if err := a.copiarTexto(ctx, p.Local, texto); err != nil {
		return pronto, err
	}
	if err := os.WriteFile(filepath.Join(pasta, "acervo.md"), []byte(a.indiceDoAcervo(ctx)), 0o644); err != nil {
		return pronto, err
	}

	titulo := primeiro(p.Titulo, p.Local)
	var s strings.Builder
	ehTema := strings.HasPrefix(p.Local, "temas/")
	if ehTema {
		fmt.Fprintf(&s, "O Lucas está lendo o Tema %q do Acervo dele. Os arquivos do Tema estão em texto/ (o indice.md e os capítulos separados).", titulo)
	} else {
		fmt.Fprintf(&s, "O Lucas está lendo a Referência %q do Acervo dele. O texto está em texto/texto.md.", titulo)
	}
	s.WriteString(" O mapa do Acervo inteiro (Temas e Referências) está em acervo.md; os Temas podem ser lidos em " +
		filepath.Join(a.conteudo.raiz, pastaTemas) + " e as Referências em " + filepath.Join(a.cfg.dados, "referencias") + ".")
	if anteriores := a.memoriaDaConversa(ctx, p); anteriores != "" {
		s.WriteString("\n\nA conversa até aqui sobre este texto (da mais antiga para a mais nova):\n\n" + anteriores)
	}
	if p.Trecho != "" {
		onde := ""
		if p.Arquivo != "" {
			onde = fmt.Sprintf(" (no arquivo texto/%s)", p.Arquivo)
		}
		fmt.Fprintf(&s, "\n\nA mensagem é sobre este trecho, que o Lucas selecionou%s:\n\n<trecho>\n%s\n</trecho>\n\n"+
			"O trecho vem como aparece na tela, sem a formatação Markdown: ache-o no arquivo.", onde, p.Trecho)
	}
	fmt.Fprintf(&s, "\n\nA mensagem do Lucas:\n\n<mensagem>\n%s\n</mensagem>\n\nHoje é %s.", p.Pergunta, time.Now().Format("2006-01-02"))
	if ehTema {
		if lista, _ := a.livrosSobre(ctx, p.Local, p.Titulo, 6); len(lista) > 0 {
			s.WriteString("\n\nLivros da biblioteca do Lucas que tratam deste Tema (pista de fonte; cite com o capítulo e as páginas se usar):\n")
			for _, l := range lista {
				fmt.Fprintf(&s, "- %s (%s), capítulo %q, p. %d–%d: %s\n", l.Titulo, l.Autor, l.Capitulo, l.De, l.Ate, l.Resumo)
			}
		}
	}

	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		extras:      []string{filepath.Join(a.conteudo.raiz, pastaTemas), filepath.Join(a.cfg.dados, "referencias")},
		regra:       "conversar.md",
		instrucao:   s.String(),
		ferramentas: ferramentasLeitura + "," + ferramentasInternet,
		esquema:     esquemaConversa,
	})
	if err != nil {
		return pronto, err
	}
	var saida respostaConversa
	if err := dadosDaResposta(r, &saida); err != nil {
		return pronto, err
	}
	saida.Resposta = strings.TrimSpace(saida.Resposta)
	a.limparConversa(ctx, &saida, texto, ehTema)
	if saida.Resposta == "" && len(saida.Alteracoes) == 0 {
		return pronto, errors.New("o claude não devolveu resposta")
	}
	resposta, _ := json.Marshal(saida)
	_, err = a.db.ExecContext(ctx, `UPDATE perguntas SET resultado = ?, atualizado_em = ? WHERE id = ? AND tarefa_id = ?`,
		string(resposta), agora(), p.ID, t.ID)
	return pronto, err
}

// limparConversa fica só com o que o app consegue usar: alterações cujo "antes" aparece exatamente
// uma vez no arquivo, e lugares do Acervo que existem. O que cair vira uma nota na resposta.
func (a *app) limparConversa(ctx context.Context, c *respostaConversa, texto string, ehTema bool) {
	if c.Fontes == nil {
		c.Fontes = []consultada{}
	}
	var validas []alteracao
	caidas := 0
	for _, alt := range c.Alteracoes {
		alt.Arquivo = path.Base(strings.TrimPrefix(strings.TrimSpace(alt.Arquivo), "texto/"))
		if !ehTema {
			alt.Arquivo = "texto.md"
		}
		alt.Resumo = strings.TrimSpace(alt.Resumo)
		conteudo, err := os.ReadFile(filepath.Join(texto, alt.Arquivo))
		if err != nil || !arquivoDeTema.MatchString(alt.Arquivo) || alt.Antes == "" || alt.Antes == alt.Depois ||
			strings.Count(string(conteudo), alt.Antes) != 1 {
			caidas++
			continue
		}
		alt.Estado = "proposta"
		validas = append(validas, alt)
	}
	if validas == nil {
		validas = []alteracao{}
	}
	c.Alteracoes = validas
	if caidas > 0 {
		c.Resposta += fmt.Sprintf("\n\n*(%d alteração proposta não pôde ser localizada no texto e ficou de fora: peça de novo.)*", caidas)
	}
	lugares := []noAcervo{}
	for _, l := range c.NoAcervo {
		l.Caminho = strings.Trim(strings.TrimSpace(l.Caminho), "/")
		if a.lugarExiste(ctx, l.Caminho) {
			lugares = append(lugares, l)
		}
	}
	c.NoAcervo = lugares
}

func (a *app) lugarExiste(ctx context.Context, caminho string) bool {
	if id, ok := strings.CutPrefix(caminho, "referencias/"); ok {
		_, err := a.referencia(ctx, id)
		return err == nil
	}
	if !strings.HasPrefix(caminho, pastaTemas+"/") || strings.Contains(caminho, "..") {
		return false
	}
	return existe(filepath.Join(a.conteudo.raiz, filepath.FromSlash(caminho)))
}

// indiceDoAcervo: o mapa dos Temas e das Referências, para a IA responder "isto já está no Acervo?".
func (a *app) indiceDoAcervo(ctx context.Context) string {
	var s strings.Builder
	s.WriteString("# O Acervo do Lucas\n\n## Temas (caminho: título — resumo)\n\n")
	if temas, err := a.conteudo.listarTemas(); err == nil {
		for _, t := range temas {
			fmt.Fprintf(&s, "- %s: %s — %s\n", t.Caminho, t.Titulo, cortar(t.Resumo, 200, false))
		}
	}
	s.WriteString("\n## Referências (caminho: título — tipo)\n\n")
	rows, err := a.db.QueryContext(ctx, `SELECT id, titulo, tipo FROM referencias WHERE estado != 'descartada' ORDER BY criado_em DESC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, titulo, tipo string
			if rows.Scan(&id, &titulo, &tipo) == nil {
				fmt.Fprintf(&s, "- referencias/%s: %s — %s\n", id, titulo, tipo)
			}
		}
	}
	return s.String()
}

// memoriaDaConversa: as últimas mensagens do mesmo texto, com o que a IA respondeu e o que o Lucas
// fez com as alterações, para "agora resume isso" funcionar.
func (a *app) memoriaDaConversa(ctx context.Context, atual perguntaFeita) string {
	rows, err := a.db.QueryContext(ctx, consultaPergunta+` WHERE p.local = ? AND p.id != ? AND p.criado_em <= ?
		ORDER BY p.criado_em DESC, p.rowid DESC LIMIT ?`, atual.Local, atual.ID, atual.CriadoEm, memoriaDaConversa)
	if err != nil {
		return ""
	}
	defer rows.Close()
	var itens []string
	for rows.Next() {
		p, err := lerPergunta(rows)
		if err != nil || p.Estado != "pronto" {
			continue
		}
		var s strings.Builder
		fmt.Fprintf(&s, "**Lucas:** %s\n", p.Pergunta)
		if p.Trecho != "" {
			fmt.Fprintf(&s, "(sobre o trecho: “%s”)\n", cortar(p.Trecho, 300, false))
		}
		switch {
		case p.Conversa != nil:
			fmt.Fprintf(&s, "**IA:** %s\n", cortar(p.Conversa.Resposta, 1500, false))
			for _, alt := range p.Conversa.Alteracoes {
				fmt.Fprintf(&s, "- Alteração proposta (%s): %s\n", alt.Estado, alt.Resumo)
			}
		case p.Resultado != nil:
			fmt.Fprintf(&s, "**IA:** %s\n", cortar(p.Resultado.Resposta, 1500, false))
		}
		itens = append([]string{s.String()}, itens...)
	}
	return strings.Join(itens, "\n")
}

// handleAlteracao: POST /api/perguntas/{id}/alteracoes/{n}/{decisao}, decisao = aplicar ou descartar.
func (a *app) handleAlteracao(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	p, err := a.perguntaFeita(ctx, r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Mensagem não encontrada.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || p.Conversa == nil || n < 0 || n >= len(p.Conversa.Alteracoes) {
		erroJSON(w, http.StatusNotFound, "Alteração não encontrada.")
		return
	}
	alt := &p.Conversa.Alteracoes[n]
	if alt.Estado != "proposta" {
		erroJSON(w, http.StatusConflict, "Esta alteração já foi decidida.")
		return
	}
	switch r.PathValue("decisao") {
	case "aplicar":
		if err := a.aplicarAlteracao(ctx, p, *alt); err != nil {
			erroJSON(w, http.StatusConflict, err.Error())
			return
		}
		alt.Estado = "aplicada"
	case "descartar":
		alt.Estado = "descartada"
	default:
		erroJSON(w, http.StatusNotFound, "Decisão desconhecida.")
		return
	}
	b, _ := json.Marshal(p.Conversa)
	if _, err := a.db.ExecContext(ctx, `UPDATE perguntas SET resultado = ?, atualizado_em = ? WHERE id = ?`, string(b), agora(), p.ID); err != nil {
		falha(w, err)
		return
	}
	a.responderPergunta(w, r, p.ID, http.StatusOK)
}

var errTrechoMudou = errors.New("O texto mudou nesse trecho desde a resposta: peça a alteração de novo.")

// aplicarAlteracao troca o trecho no texto de verdade. No Tema, grava e faz commit (o git guarda o de
// antes); na Referência, o texto de antes vira anterior.md, como numa proposta aprovada (D36).
func (a *app) aplicarAlteracao(ctx context.Context, p perguntaFeita, alt alteracao) error {
	if id, ok := strings.CutPrefix(p.Local, "referencias/"); ok {
		ref, err := a.referencia(ctx, id)
		if err != nil {
			return errors.New("a Referência foi excluída")
		}
		if a.ocupada(ctx, ref.ID) {
			return errors.New("A Referência tem uma tarefa em andamento: espere terminar.")
		}
		original, err := a.textoParaAcervo(ref)
		if err != nil {
			return err
		}
		atual, err := os.ReadFile(original)
		if err != nil {
			return err
		}
		if strings.Count(string(atual), alt.Antes) != 1 {
			return errTrechoMudou
		}
		if err := os.WriteFile(filepath.Join(a.pastaReferencia(ref.ID), "anterior.md"), atual, 0o644); err != nil {
			return err
		}
		return os.WriteFile(original, []byte(strings.Replace(string(atual), alt.Antes, alt.Depois, 1)), 0o644)
	}
	rel := path.Join(p.Local, alt.Arquivo)
	atual, err := os.ReadFile(filepath.Join(a.conteudo.raiz, filepath.FromSlash(rel)))
	if err != nil {
		return errors.New("o arquivo do Tema não existe mais")
	}
	if strings.Count(string(atual), alt.Antes) != 1 {
		return errTrechoMudou
	}
	novo := strings.Replace(string(atual), alt.Antes, alt.Depois, 1)
	if _, err := a.conteudo.salvarComMensagem(ctx, rel, novo, hashDe(atual), "Conversar: "+cortar(primeiro(alt.Resumo, p.Pergunta), 80, false)); err != nil {
		var c *conflito
		if errors.As(err, &c) {
			return errTrechoMudou
		}
		return err
	}
	return nil
}
