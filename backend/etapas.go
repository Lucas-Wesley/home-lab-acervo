package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func (a *app) executores() map[string]executor {
	return map[string]executor{
		"extrair":    a.naReferencia(a.extrair, true),
		"organizar":  a.naReferencia(a.organizar, true),
		"traduzir":   a.naReferencia(a.traduzir, true),
		"sugerir":    a.naReferencia(a.sugerir, false),
		"criar_tema": a.naReferencia(a.criarTema, false),
		"integrar":   a.naReferencia(a.integrar, false),
	}
}

// naReferencia deixa o erro visível na Referência (D17). Numa etapa do pipeline, a Referência
// passa ao estado "erro"; nas ações do Lucas, só a mensagem aparece.
func (a *app) naReferencia(fn func(context.Context, *tarefa, *referencia) (resultado, error), etapa bool) executor {
	return func(ctx context.Context, t *tarefa) (resultado, error) {
		ref, err := a.referencia(ctx, t.ReferenciaID)
		if err != nil {
			return pronto, err
		}
		res, err := fn(ctx, t, ref)
		if ctx.Err() != nil {
			return res, err
		}
		if err != nil {
			if etapa {
				a.db.Exec(`UPDATE referencias SET estado = 'erro', erro = ?, atualizado_em = ? WHERE id = ?`, err.Error(), agora(), ref.ID)
			} else {
				a.db.Exec(`UPDATE referencias SET erro = ?, atualizado_em = ? WHERE id = ?`, err.Error(), agora(), ref.ID)
			}
			return res, err
		}
		a.db.Exec(`UPDATE referencias SET erro = '', atualizado_em = ? WHERE id = ?`, agora(), ref.ID)
		return res, nil
	}
}

func (a *app) mudarEstado(ctx context.Context, id, estado string) error {
	_, err := a.db.ExecContext(ctx, `UPDATE referencias SET estado = ?, atualizado_em = ? WHERE id = ?`, estado, agora(), id)
	return err
}

func (a *app) extrair(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	pasta := a.pastaReferencia(ref.ID)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return pronto, err
	}
	info, lingua, err := a.baixarLegenda(ctx, ref.URL, pasta)
	if info.Titulo != "" {
		idioma := info.Idioma
		if idioma == "" {
			idioma = strings.TrimSuffix(lingua, "-orig")
		}
		a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ?, canal = ?, duracao = ?, idioma = ? WHERE id = ?`,
			info.Titulo, info.Canal, info.Duracao, idioma, ref.ID)
	}
	if err != nil {
		return pronto, err
	}
	if err := a.mudarEstado(ctx, ref.ID, "cru"); err != nil {
		return pronto, err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "organizar", nil)
	return pronto, err
}

func (a *app) organizar(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	pasta := a.pastaReferencia(ref.ID)
	os.Remove(filepath.Join(pasta, "organizado.md"))
	os.Remove(filepath.Join(pasta, "traducao.md"))
	_, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		regra:       "organizar.md",
		instrucao:   "Leia cru.md nesta pasta e escreva organizado.md nesta pasta. Responda só 'ok' ao terminar.",
		ferramentas: ferramentasEscrita,
	})
	if err != nil {
		return pronto, err
	}
	if err := exigirArquivo(filepath.Join(pasta, "organizado.md")); err != nil {
		return pronto, err
	}
	if err := a.mudarEstado(ctx, ref.ID, "organizado"); err != nil {
		return pronto, err
	}
	if precisaTraducao(ref.Idioma) {
		_, err = a.fila.enfileirar(ctx, ref.ID, "traduzir", nil)
	}
	return pronto, err
}

func precisaTraducao(idioma string) bool {
	return idioma != "" && idiomaBase(idioma) != "pt"
}

func (a *app) traduzir(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	pasta := a.pastaReferencia(ref.ID)
	_, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		regra:       "traduzir.md",
		instrucao:   "Leia organizado.md nesta pasta e escreva traducao.md nesta pasta. Responda só 'ok' ao terminar.",
		ferramentas: ferramentasEscrita,
	})
	if err != nil {
		return pronto, err
	}
	if err := exigirArquivo(filepath.Join(pasta, "traducao.md")); err != nil {
		return pronto, err
	}
	return pronto, a.mudarEstado(ctx, ref.ID, "organizado")
}

func exigirArquivo(caminho string) error {
	info, err := os.Stat(caminho)
	if err != nil || info.Size() < 50 {
		return fmt.Errorf("o claude terminou sem escrever %s", filepath.Base(caminho))
	}
	return nil
}

// textoParaAcervo: o Acervo é todo em português (D18).
func (a *app) textoParaAcervo(ref *referencia) (string, error) {
	pasta := a.pastaReferencia(ref.ID)
	for _, nome := range []string{"traducao.md", "organizado.md"} {
		if precisaTraducao(ref.Idioma) && nome == "organizado.md" {
			return "", errors.New("a tradução ainda não está pronta")
		}
		if exigirArquivo(filepath.Join(pasta, nome)) == nil {
			return filepath.Join(pasta, nome), nil
		}
	}
	return "", errors.New("a Referência ainda não foi organizada")
}

type sugestao struct {
	Acao   string `json:"acao"` // novo | subtema | integrar
	Tema   string `json:"tema"` // subtema: o pai; integrar: o Tema alvo
	Titulo string `json:"titulo"`
	Motivo string `json:"motivo"`
	Outros []struct {
		Tema   string `json:"tema"`
		Motivo string `json:"motivo"`
	} `json:"outros"`
}

const esquemaSugestao = `{"type":"object","properties":{` +
	`"acao":{"type":"string","enum":["novo","subtema","integrar"]},` +
	`"tema":{"type":"string"},"titulo":{"type":"string"},"motivo":{"type":"string"},` +
	`"outros":{"type":"array","items":{"type":"object","properties":{"tema":{"type":"string"},"motivo":{"type":"string"}},"required":["tema","motivo"]}}},` +
	`"required":["acao","tema","titulo","motivo","outros"]}`

func (a *app) sugerir(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	texto, err := a.textoParaAcervo(ref)
	if err != nil {
		return pronto, err
	}
	temas, err := a.conteudo.listarTemas()
	if err != nil {
		return pronto, err
	}
	var lista strings.Builder
	for _, tm := range temas {
		fmt.Fprintf(&lista, "- %s — %s: %s\n", tm.Caminho, tm.Titulo, tm.Resumo)
	}
	if len(temas) == 0 {
		lista.WriteString("(o Acervo ainda não tem Temas)\n")
	}
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:  a.conteudo.raiz,
		extras: []string{filepath.Dir(texto)},
		regra:  "sugerir.md",
		instrucao: fmt.Sprintf("A Referência está em %s. Os Temas do Acervo (pasta — título: resumo) são:\n\n%s\n"+
			"Os arquivos dos Temas estão em temas/ nesta pasta.", texto, lista.String()),
		ferramentas: ferramentasLeitura,
		esquema:     esquemaSugestao,
	})
	if err != nil {
		return pronto, err
	}
	var s sugestao
	if err := dadosDaResposta(r, &s); err != nil {
		return pronto, err
	}
	dados, _ := json.Marshal(s)
	_, err = a.db.ExecContext(ctx, `UPDATE referencias SET sugestao = ? WHERE id = ?`, string(dados), ref.ID)
	return pronto, err
}

type paramsTema struct {
	Pasta  string `json:"pasta"`
	Titulo string `json:"titulo"`
}

func (a *app) criarTema(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	var p paramsTema
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	if _, err := os.Stat(filepath.Join(a.conteudo.raiz, filepath.FromSlash(p.Pasta))); err == nil {
		return pronto, fmt.Errorf("a pasta %s já existe no Acervo", p.Pasta)
	}
	texto, err := a.textoParaAcervo(ref)
	if err != nil {
		return pronto, err
	}
	dir, err := a.conteudo.abrirRevisao(ctx, t.ID)
	if err != nil {
		return pronto, err
	}
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta:  dir,
		extras: []string{filepath.Dir(texto)},
		regra:  "criar-tema.md",
		instrucao: fmt.Sprintf("A Referência tem id %s, título %q e URL %s. O texto organizado está em %s. "+
			"O Lucas decidiu criar o Tema %q na pasta %s/ (caminho relativo a esta pasta). Hoje é %s. "+
			"Escreva só dentro dessa pasta. Responda só 'ok' ao terminar.",
			ref.ID, ref.Titulo, ref.URL, texto, p.Titulo, p.Pasta, time.Now().Format("2006-01-02")),
		ferramentas: ferramentasEscrita,
	})
	if err == nil {
		err = a.conferirProposta(ctx, t.ID, p.Pasta, true)
	}
	if err != nil {
		a.conteudo.descartarRevisao(context.Background(), t.ID)
		return pronto, err
	}
	return paraRevisao, nil
}

type paramsIntegrar struct {
	Tema string `json:"tema"`
}

func (a *app) integrar(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	var p paramsIntegrar
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	texto, err := a.textoParaAcervo(ref)
	if err != nil {
		return pronto, err
	}
	dir, err := a.conteudo.abrirRevisao(ctx, t.ID)
	if err != nil {
		return pronto, err
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(p.Tema), "indice.md")); err != nil {
		a.conteudo.descartarRevisao(context.Background(), t.ID)
		return pronto, fmt.Errorf("o Tema %s não existe", p.Tema)
	}
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta:  dir,
		extras: []string{filepath.Dir(texto)},
		regra:  "integrar.md",
		instrucao: fmt.Sprintf("A Referência tem id %s, título %q e URL %s. O texto organizado está em %s. "+
			"O Lucas decidiu integrá-la ao Tema da pasta %s/ (caminho relativo a esta pasta). Hoje é %s. "+
			"Escreva só dentro dessa pasta. Ao terminar, responda só 'ok', ou 'nada novo' se a Referência não "+
			"acrescentar nada ao Tema.",
			ref.ID, ref.Titulo, ref.URL, texto, p.Tema, time.Now().Format("2006-01-02")),
		ferramentas: ferramentasEscrita,
	})
	if err == nil {
		err = a.conferirProposta(ctx, t.ID, p.Tema, false)
	}
	if errors.Is(err, errSemMudancas) {
		a.conteudo.descartarRevisao(context.Background(), t.ID)
		t.Erro = "A Referência não acrescentou nada ao Tema."
		return pronto, nil
	}
	if err != nil {
		a.conteudo.descartarRevisao(context.Background(), t.ID)
		return pronto, err
	}
	return paraRevisao, nil
}

var errSemMudancas = errors.New("a IA não propôs nenhuma mudança")

// conferirProposta garante que a IA só mexeu dentro da pasta do Tema.
func (a *app) conferirProposta(ctx context.Context, id int64, pasta string, novo bool) error {
	arquivos, err := a.conteudo.mudancas(ctx, id)
	if err != nil {
		return err
	}
	if len(arquivos) == 0 {
		return errSemMudancas
	}
	for _, arq := range arquivos {
		if !strings.HasPrefix(arq, pasta+"/") {
			return fmt.Errorf("a IA mexeu fora do Tema (%s); a proposta foi descartada", arq)
		}
		if path.Ext(arq) != ".md" {
			return fmt.Errorf("a IA criou um arquivo que não é Markdown (%s); a proposta foi descartada", arq)
		}
	}
	if novo {
		if _, err := os.Stat(filepath.Join(a.conteudo.pastaRevisao(id), filepath.FromSlash(pasta), "indice.md")); err != nil {
			return errors.New("a IA não criou o indice.md do Tema")
		}
	}
	return nil
}
