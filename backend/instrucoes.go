package main

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// As instruções que o Claude segue em cada etapa. O padrão vem de regras/, no repositório; o
// Lucas pode editar qualquer uma pelo app (Ajustes), e a versão editada fica em DATA_DIR/regras e
// passa a valer no lugar do padrão. Apagar a edição volta ao padrão.
type instrucao struct {
	Arquivo   string `json:"arquivo"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Editada   bool   `json:"editada"`
	EditadaEm string `json:"editada_em,omitempty"`
	Conteudo  string `json:"conteudo,omitempty"`
}

// Na ordem em que as etapas acontecem; depois, o Pesquisador (D27), o resumo de capítulos (D30) e a
// pesquisa de um trecho (D26).
var instrucoes = []instrucao{
	{Arquivo: "organizar.md", Nome: "Organizar a transcrição", Descricao: "Transforma a transcrição bruta num texto organizado: tira propaganda e conversa, junta cada assunto num lugar e mantém tudo o que é conteúdo."},
	{Arquivo: "organizar-artigo.md", Nome: "Organizar um artigo", Descricao: "Faz o mesmo com o texto de uma página da web (artigo, blog): tira o que é do site e não do artigo e mantém tudo o que é conteúdo."},
	{Arquivo: "traduzir.md", Nome: "Traduzir", Descricao: "Passa o texto organizado para o português, quando o vídeo ou o artigo é em outra língua."},
	{Arquivo: "sugerir.md", Nome: "Sugerir o Tema", Descricao: "Propõe em qual Tema a Referência entra, ou se ela vira um Tema novo."},
	{Arquivo: "criar-tema.md", Nome: "Criar um Tema", Descricao: "Escreve a página de um Tema novo a partir da Referência."},
	{Arquivo: "integrar.md", Nome: "Integrar num Tema", Descricao: "Acrescenta a um Tema que já existe só o que a Referência traz de novo."},
	{Arquivo: "pesquisar-esboco.md", Nome: "Pesquisa: esboço", Descricao: "Divide o tema pedido em perguntas e escolhe as palavras de busca de cada fonte (livros, Wikipédia, YouTube)."},
	{Arquivo: "pesquisar.md", Nome: "Pesquisa: escrever", Descricao: "Escreve o texto da pesquisa a partir do material coletado (e dos sites, se escolhidos), com a fonte de cada parte."},
	{Arquivo: "resumir-livro.md", Nome: "Resumir capítulos de um livro", Descricao: "Sintetiza um ou mais capítulos de um livro da Biblioteca: fica o que o livro ensina, saem repetições e enchimento, e cada parte leva as páginas."},
	{Arquivo: "conferir.md", Nome: "Pesquisa: conferir", Descricao: "Na pesquisa detalhada, confere cada afirmação do texto contra o material guardado e corrige o que não se sustenta."},
	{Arquivo: "perguntar.md", Nome: "Perguntar e verificar", Descricao: "Vê quanto o texto responde à sua pergunta, verifica a resposta na internet em mais de uma fonte e aponta o que está errado, desatualizado ou faltando."},
	{Arquivo: "detalhar.md", Nome: "Mais informações", Descricao: "Pesquisa na internet sobre um trecho que você selecionou no texto e explica, com as fontes."},
	{Arquivo: "integrar-texto.md", Nome: "Integrar ao texto", Descricao: "Leva a resposta de uma pergunta ou um \"Mais informações\" para dentro do texto, no ponto certo, seguindo o que você escolheu em cada divergência."},
}

func instrucaoConhecida(arquivo string) (instrucao, bool) {
	for _, i := range instrucoes {
		if i.Arquivo == arquivo {
			return i, true
		}
	}
	return instrucao{}, false
}

func (a *app) pastaRegrasEditadas() string {
	return filepath.Join(a.cfg.dados, "regras")
}

// lerRegra devolve a instrução em vigor: a editada pelo Lucas, se houver, ou a padrão.
func (a *app) lerRegra(arquivo string) (texto string, editada bool, err error) {
	b, err := os.ReadFile(filepath.Join(a.pastaRegrasEditadas(), arquivo))
	if err == nil {
		return string(b), true, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", false, err
	}
	b, err = os.ReadFile(filepath.Join(a.cfg.regras, arquivo))
	return string(b), false, err
}

func (a *app) comEstado(i instrucao) instrucao {
	if info, err := os.Stat(filepath.Join(a.pastaRegrasEditadas(), i.Arquivo)); err == nil {
		i.Editada = true
		i.EditadaEm = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
	}
	return i
}

func (a *app) handleInstrucoes(w http.ResponseWriter, r *http.Request) {
	lista := make([]instrucao, 0, len(instrucoes))
	for _, i := range instrucoes {
		lista = append(lista, a.comEstado(i))
	}
	writeJSON(w, http.StatusOK, lista)
}

func (a *app) instrucaoDoPedido(w http.ResponseWriter, r *http.Request) (instrucao, bool) {
	i, ok := instrucaoConhecida(r.PathValue("arquivo"))
	if !ok {
		erroJSON(w, http.StatusNotFound, "Instrução não encontrada.")
	}
	return i, ok
}

func (a *app) handleInstrucao(w http.ResponseWriter, r *http.Request) {
	i, ok := a.instrucaoDoPedido(w, r)
	if !ok {
		return
	}
	texto, _, err := a.lerRegra(i.Arquivo)
	if err != nil {
		falha(w, err)
		return
	}
	i = a.comEstado(i)
	i.Conteudo = texto
	writeJSON(w, http.StatusOK, i)
}

func (a *app) handleSalvarInstrucao(w http.ResponseWriter, r *http.Request) {
	i, ok := a.instrucaoDoPedido(w, r)
	if !ok {
		return
	}
	var corpo struct {
		Conteudo string `json:"conteudo"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	if strings.TrimSpace(corpo.Conteudo) == "" {
		erroJSON(w, http.StatusBadRequest, "A instrução não pode ficar vazia. Para voltar ao texto original, use Voltar ao padrão.")
		return
	}
	if err := os.MkdirAll(a.pastaRegrasEditadas(), 0o755); err != nil {
		falha(w, err)
		return
	}
	// Grava ao lado e renomeia: uma tarefa da fila nunca lê a instrução pela metade.
	destino := filepath.Join(a.pastaRegrasEditadas(), i.Arquivo)
	if err := os.WriteFile(destino+".tmp", []byte(corpo.Conteudo), 0o644); err != nil {
		falha(w, err)
		return
	}
	if err := os.Rename(destino+".tmp", destino); err != nil {
		falha(w, err)
		return
	}
	a.handleInstrucao(w, r)
}

func (a *app) handleRestaurarInstrucao(w http.ResponseWriter, r *http.Request) {
	i, ok := a.instrucaoDoPedido(w, r)
	if !ok {
		return
	}
	if err := os.Remove(filepath.Join(a.pastaRegrasEditadas(), i.Arquivo)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		falha(w, err)
		return
	}
	a.handleInstrucao(w, r)
}
