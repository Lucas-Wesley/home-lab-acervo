package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type ambiente struct {
	t   *testing.T
	srv *httptest.Server
	app *app
}

// novoAmbiente sobe o app com yt-dlp e claude falsos (testdata/) e a fila rodando.
func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	// Dados num caminho relativo, como no "make dev-api" (DATA_DIR=../data).
	t.Chdir(t.TempDir())
	os.Mkdir("backend", 0o755)
	t.Chdir("backend")
	dados := "../data"
	testdata, _ := filepath.Abs(filepath.Join(raizDoBackend, "testdata"))
	cfg := configDoAmbiente(dados)
	cfg.regras = filepath.Join(raizDoBackend, "..", "regras")
	cfg = config{
		dados:      cfg.dados,
		regras:     cfg.regras,
		ytdlp:      filepath.Join(testdata, "yt-dlp-falso"),
		claude:     filepath.Join(testdata, "claude-falso"),
		autorNome:  "Teste",
		autorEmail: "teste@exemplo.com",
	}
	os.MkdirAll(cfg.dados, 0o755)
	db, err := abrirBanco(filepath.Join(cfg.dados, "acervo.db"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	a, err := novoApp(ctx, db, cfg)
	if err != nil {
		t.Fatal(err)
	}
	go a.fila.rodar(ctx)
	srv := httptest.NewServer(newServer(db, fstest.MapFS{"index.html": {Data: []byte("app")}}, a, "teste"))
	t.Cleanup(func() {
		srv.Close()
		cancel()
		db.Close()
	})
	return &ambiente{t: t, srv: srv, app: a}
}

func (e *ambiente) pedir(metodo, caminho string, corpo any, destino any) int {
	e.t.Helper()
	var leitor *bytes.Reader
	if corpo != nil {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	} else {
		leitor = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(metodo, e.srv.URL+caminho, leitor)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	if destino != nil {
		json.NewDecoder(resp.Body).Decode(destino)
	}
	return resp.StatusCode
}

// esperar repete até a condição valer ou o tempo acabar.
func (e *ambiente) esperar(descricao string, cond func() bool) {
	e.t.Helper()
	limite := time.Now().Add(10 * time.Second)
	for time.Now().Before(limite) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	e.t.Fatalf("tempo esgotado esperando: %s", descricao)
}

// detalhe espelha detalheReferencia para ler a resposta (o JSON não preenche ponteiro embutido).
type detalhe struct {
	referencia
	Textos  map[string]string
	Tarefas []tarefa
}

func (e *ambiente) referencia(id string) detalhe {
	var d detalhe
	e.pedir("GET", "/api/referencias/"+id, nil, &d)
	return d
}

func (e *ambiente) esperarEstado(id, estado string) detalhe {
	e.t.Helper()
	var d detalhe
	e.esperar("Referência "+estado, func() bool {
		d = e.referencia(id)
		return d.Estado == estado && (d.Tarefa == nil || d.Tarefa.Estado == emRevisao)
	})
	return d
}

func (e *ambiente) adicionar(link string) string {
	e.t.Helper()
	var ref referencia
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": link}, &ref); s != http.StatusCreated {
		e.t.Fatalf("POST /api/referencias: %d", s)
	}
	return ref.ID
}

var raizDoBackend, _ = os.Getwd()

func TestFluxoCompleto(t *testing.T) {
	e := novoAmbiente(t)
	id := e.adicionar("https://youtu.be/AAAAAAAAAAA?si=x")

	// Extrair → organizar, sem tradução (vídeo em português).
	d := e.esperarEstado(id, "organizado")
	if d.Titulo != "Vídeo AAAAAAAAAAA" || d.Idioma != "pt-BR" {
		t.Errorf("metadados: %+v", d)
	}
	if !strings.Contains(d.Textos["cru"], "[00:31] Em 1494") || d.Textos["organizado"] == "" || d.Textos["traducao"] != "" {
		t.Errorf("textos: %v", d.Textos)
	}

	// O mesmo vídeo de novo é recusado.
	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": "https://www.youtube.com/watch?v=AAAAAAAAAAA"}, nil); s != http.StatusConflict {
		t.Errorf("duplicado: %d", s)
	}

	// Sugestão da IA.
	e.pedir("POST", "/api/referencias/"+id+"/sugerir", nil, nil)
	e.esperar("sugestão", func() bool { return string(e.referencia(id).Sugestao) != "null" })
	var sug sugestao
	json.Unmarshal(e.referencia(id).Sugestao, &sug)
	if sug.Acao != "novo" || sug.Titulo != "Tema de Teste" {
		t.Errorf("sugestão: %+v", sug)
	}

	// Criar o Tema: vira revisão, a main não muda até aprovar.
	if s := e.pedir("POST", "/api/referencias/"+id+"/criar-tema", map[string]string{"titulo": "Tema de Teste"}, nil); s != http.StatusOK {
		t.Fatalf("criar-tema: %d", s)
	}
	var revisoes []tarefaComTitulo
	e.esperar("revisão", func() bool {
		e.pedir("GET", "/api/revisoes", nil, &revisoes)
		return len(revisoes) == 1
	})
	if existe(filepath.Join(e.app.conteudo.raiz, "temas/tema-de-teste")) {
		t.Fatal("o Tema apareceu na main antes da aprovação")
	}
	var rev struct{ Diff string }
	e.pedir("GET", "/api/revisoes/"+itoa(revisoes[0].ID), nil, &rev)
	if !strings.Contains(rev.Diff, "+++ b/temas/tema-de-teste/indice.md") {
		t.Errorf("diff: %s", rev.Diff)
	}
	if s := e.pedir("POST", "/api/revisoes/"+itoa(revisoes[0].ID)+"/aprovar", nil, nil); s != http.StatusOK {
		t.Fatalf("aprovar: %d", s)
	}
	d = e.esperarEstado(id, "integrado")
	if len(d.Temas) != 1 || d.Temas[0] != "temas/tema-de-teste" {
		t.Errorf("temas da Referência: %v", d.Temas)
	}
	if log := git(t, e.app.conteudo.raiz, "log", "--oneline"); !strings.Contains(log, "Novo Tema: Tema de Teste") {
		t.Errorf("log: %s", log)
	}
	if existe(e.app.conteudo.pastaRevisao(revisoes[0].ID)) {
		t.Error("a worktree ficou para trás")
	}

	// Temas e leitura, com fontes do índice.
	var temas []tema
	e.pedir("GET", "/api/temas", nil, &temas)
	if len(temas) != 1 || temas[0].Titulo != "Tema de Teste" || temas[0].Capitulos != 2 || temas[0].Resumo != "Introdução do Tema." {
		t.Fatalf("temas: %+v", temas)
	}
	var arq arquivo
	e.pedir("GET", "/api/arquivos/temas/tema-de-teste/indice.md", nil, &arq)
	if arq.Tema != "temas/tema-de-teste" || len(arq.Fontes) != 1 || arq.Fontes[0].ID != "r1" || len(arq.CitadoEm) != 0 {
		t.Errorf("arquivo: %+v", arq)
	}

	// Edição manual com controle de versão por hash.
	novo := arq.Conteudo + "\nNota minha.\n"
	var salvo struct{ Hash string }
	if s := e.pedir("PUT", "/api/arquivos/temas/tema-de-teste/indice.md", map[string]string{"conteudo": novo, "base": arq.Hash}, &salvo); s != http.StatusOK || salvo.Hash == "" {
		t.Fatalf("salvar: %d", s)
	}
	var conflito struct {
		Atual arquivo `json:"atual"`
	}
	if s := e.pedir("PUT", "/api/arquivos/temas/tema-de-teste/indice.md", map[string]string{"conteudo": "outra", "base": arq.Hash}, &conflito); s != http.StatusConflict {
		t.Errorf("edição com base velha: %d", s)
	}
	if conflito.Atual.Conteudo != novo {
		t.Errorf("o conflito devia trazer a versão atual")
	}

	// Subtema escrito à mão, linkando o Tema: aparece em "citado em".
	if s := e.pedir("POST", "/api/temas", map[string]string{"titulo": "Detalhe", "pai": "temas/tema-de-teste"}, nil); s != http.StatusCreated {
		t.Fatalf("criar subtema: %d", s)
	}
	e.pedir("GET", "/api/arquivos/temas/tema-de-teste/detalhe/indice.md", nil, &arq)
	texto := arq.Conteudo + "Veja [o Tema](../indice.md).\n"
	e.pedir("PUT", "/api/arquivos/temas/tema-de-teste/detalhe/indice.md", map[string]string{"conteudo": texto, "base": arq.Hash}, nil)
	e.pedir("GET", "/api/temas", nil, &temas)
	if len(temas) != 2 || temas[0].Pai != "" || temas[1].Pai != "temas/tema-de-teste" {
		t.Errorf("hierarquia: %+v", temas)
	}
	e.pedir("GET", "/api/arquivos/temas/tema-de-teste/indice.md", nil, &arq)
	if len(arq.CitadoEm) != 1 || arq.CitadoEm[0].Caminho != "temas/tema-de-teste/detalhe/indice.md" {
		t.Errorf("citado em: %+v", arq.CitadoEm)
	}
	e.pedir("GET", "/api/arquivos/temas/tema-de-teste/detalhe/indice.md", nil, &arq)
	if len(arq.Migalhas) != 2 || arq.Migalhas[0].Titulo != "Tema de Teste" {
		t.Errorf("migalhas: %+v", arq.Migalhas)
	}

	// Pacote para offline traz o Tema com subtemas.
	var pacote struct{ Arquivos map[string]arquivoOffline }
	e.pedir("GET", "/api/pacotes/temas/tema-de-teste", nil, &pacote)
	if len(pacote.Arquivos) != 2 {
		t.Errorf("pacote: %v", len(pacote.Arquivos))
	}

	// Integrar: a proposta acrescenta e, rejeitada, não deixa rastro.
	id2 := e.adicionar("https://www.youtube.com/watch?v=EEEEEEEEEEE")
	d2 := e.esperarEstado(id2, "organizado")
	if d2.Textos["traducao"] == "" {
		t.Fatal("vídeo em inglês sem tradução")
	}
	e.pedir("POST", "/api/referencias/"+id2+"/integrar", map[string]string{"tema": "temas/tema-de-teste"}, nil)
	e.esperar("revisão da integração", func() bool {
		e.pedir("GET", "/api/revisoes", nil, &revisoes)
		return len(revisoes) == 1
	})
	e.pedir("GET", "/api/revisoes/"+itoa(revisoes[0].ID), nil, &rev)
	if !strings.Contains(rev.Diff, "+Parágrafo integrado.") {
		t.Errorf("diff da integração: %s", rev.Diff)
	}
	e.pedir("POST", "/api/revisoes/"+itoa(revisoes[0].ID)+"/rejeitar", nil, nil)
	if b, _ := os.ReadFile(filepath.Join(e.app.conteudo.raiz, "temas/tema-de-teste/indice.md")); strings.Contains(string(b), "Parágrafo integrado") {
		t.Error("a integração rejeitada entrou na main")
	}
	if ramos := git(t, e.app.conteudo.raiz, "branch"); strings.Contains(ramos, "revisao/") {
		t.Errorf("ramo da revisão ficou: %s", ramos)
	}
}

func TestErrosVisiveis(t *testing.T) {
	e := novoAmbiente(t)

	sem := e.adicionar("https://www.youtube.com/watch?v=SSSSSSSSSSS")
	d := e.esperarEstado(sem, "erro")
	if d.Erro != errSemLegenda.Error() {
		t.Errorf("sem legenda: %q", d.Erro)
	}
	var lista []referencia
	e.pedir("GET", "/api/referencias", nil, &lista)
	if len(lista) != 1 || lista[0].Erro == "" {
		t.Errorf("a lista devia mostrar o erro: %+v", lista)
	}

	indisponivel := e.adicionar("https://www.youtube.com/watch?v=FFFFFFFFFFF")
	if d := e.esperarEstado(indisponivel, "erro"); !strings.Contains(d.Erro, "Video unavailable") {
		t.Errorf("indisponível: %q", d.Erro)
	}

	if s := e.pedir("POST", "/api/referencias", map[string]string{"url": "https://exemplo.com"}, nil); s != http.StatusBadRequest {
		t.Errorf("link inválido: %d", s)
	}

	// Claude falhando: erro visível; reprocessar retoma da etapa que falhou.
	t.Setenv("CLAUDE_FALSO_FALHA", "1")
	id := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	if d := e.esperarEstado(id, "erro"); !strings.Contains(d.Erro, "limite de uso") {
		t.Errorf("falha do claude: %q", d.Erro)
	}
	os.Unsetenv("CLAUDE_FALSO_FALHA")
	e.pedir("POST", "/api/referencias/"+id+"/reprocessar", nil, nil)
	if d := e.referencia(id); d.Erro != "" {
		t.Errorf("o erro antigo devia sumir ao pedir de novo: %q", d.Erro)
	}
	d = e.esperarEstado(id, "organizado")
	if d.Erro != "" || d.Tarefas[0].Tipo != "organizar" {
		t.Errorf("reprocessar: erro %q, tarefa %s", d.Erro, d.Tarefas[0].Tipo)
	}

	// A IA mexendo fora do Tema: a proposta é descartada.
	t.Setenv("CLAUDE_FALSO_FORA", "1")
	e.pedir("POST", "/api/referencias/"+id+"/criar-tema", map[string]string{"titulo": "Fora"}, nil)
	e.esperar("erro de proposta", func() bool { return strings.Contains(e.referencia(id).Erro, "fora do Tema") })
	var revisoes []tarefaComTitulo
	e.pedir("GET", "/api/revisoes", nil, &revisoes)
	if len(revisoes) != 0 || existe(filepath.Join(e.app.conteudo.raiz, "temas/fora")) {
		t.Error("a proposta fora do Tema não foi descartada")
	}
}

func TestFila(t *testing.T) {
	e := novoAmbiente(t)
	// Uma tarefa de tipo desconhecido falha sem travar a fila.
	idRuim, _ := e.app.fila.enfileirar(context.Background(), "x", "inexistente", nil)
	id := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	ruim, _ := e.app.fila.tarefa(context.Background(), idRuim)
	if ruim.Estado != falhou {
		t.Errorf("tarefa desconhecida: %s", ruim.Estado)
	}

	// Tarefa na fila pode ser cancelada; concluída não.
	outra, _ := e.app.fila.enfileirar(context.Background(), id, "inexistente", nil)
	var lista []tarefaComTitulo
	e.pedir("GET", "/api/tarefas", nil, &lista)
	if len(lista) < 3 || lista[len(lista)-1].Referencia == "" && lista[len(lista)-1].ReferenciaID != "x" {
		t.Errorf("lista: %+v", lista)
	}
	if s := e.pedir("POST", "/api/tarefas/"+itoa(idRuim)+"/cancelar", nil, nil); s != http.StatusConflict {
		t.Errorf("cancelar tarefa terminada: %d", s)
	}
	_ = outra
}

func TestRecuperarFila(t *testing.T) {
	dados := t.TempDir()
	db, _ := abrirBanco(filepath.Join(dados, "acervo.db"))
	defer db.Close()
	f := novaFila(db, nil)
	id, _ := f.enfileirar(context.Background(), "r", "organizar", nil)
	db.Exec(`UPDATE tarefas SET estado = ? WHERE id = ?`, rodando, id)
	if err := f.recuperar(context.Background()); err != nil {
		t.Fatal(err)
	}
	tr, _ := f.tarefa(context.Background(), id)
	if tr.Estado != naFila {
		t.Errorf("depois de reiniciar: %s", tr.Estado)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	saida, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s", args, saida)
	}
	return string(saida)
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// Duas propostas abertas ao mesmo tempo: aprovar uma não pode fazer a outra parecer que apaga
// o que acabou de entrar.
func TestPropostasSimultaneas(t *testing.T) {
	e := novoAmbiente(t)
	a := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	b := e.adicionar("https://www.youtube.com/watch?v=EEEEEEEEEEE")
	e.esperarEstado(a, "organizado")
	e.esperarEstado(b, "organizado")
	e.pedir("POST", "/api/referencias/"+a+"/criar-tema", map[string]string{"titulo": "Primeiro"}, nil)
	e.pedir("POST", "/api/referencias/"+b+"/criar-tema", map[string]string{"titulo": "Segundo"}, nil)
	var revisoes []tarefaComTitulo
	e.esperar("duas revisões", func() bool {
		e.pedir("GET", "/api/revisoes", nil, &revisoes)
		return len(revisoes) == 2
	})
	e.pedir("POST", "/api/revisoes/"+itoa(revisoes[0].ID)+"/aprovar", nil, nil)

	var rev struct{ Diff string }
	e.pedir("GET", "/api/revisoes/"+itoa(revisoes[1].ID), nil, &rev)
	if strings.Contains(rev.Diff, "temas/primeiro") || !strings.Contains(rev.Diff, "temas/segundo/indice.md") {
		t.Fatalf("o diff da segunda proposta devia ter só o Tema dela:\n%s", rev.Diff)
	}
	if s := e.pedir("POST", "/api/revisoes/"+itoa(revisoes[1].ID)+"/aprovar", nil, nil); s != http.StatusOK {
		t.Fatalf("aprovar a segunda: %d", s)
	}
	var temas []tema
	e.pedir("GET", "/api/temas", nil, &temas)
	if len(temas) != 2 {
		t.Errorf("os dois Temas deviam estar no Acervo: %+v", temas)
	}
}

func TestExcluirReferencia(t *testing.T) {
	e := novoAmbiente(t)
	id := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	pasta := e.app.pastaReferencia(id)
	if _, err := os.Stat(pasta); err != nil {
		t.Fatalf("pasta antes: %v", err)
	}
	if s := e.pedir("DELETE", "/api/referencias/"+id, nil, nil); s != http.StatusNoContent {
		t.Fatalf("excluir: %d", s)
	}
	if s := e.pedir("GET", "/api/referencias/"+id, nil, nil); s != http.StatusNotFound {
		t.Errorf("depois: %d", s)
	}
	if _, err := os.Stat(pasta); !os.IsNotExist(err) {
		t.Errorf("pasta ficou: %v", err)
	}
	var n int
	e.app.db.QueryRow(`SELECT COUNT(*) FROM tarefas WHERE referencia_id = ?`, id).Scan(&n)
	if n != 0 {
		t.Errorf("tarefas ficaram: %d", n)
	}
	// O mesmo vídeo pode voltar depois.
	if novo := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA"); novo == "" {
		t.Error("não voltou")
	}
	if s := e.pedir("DELETE", "/api/referencias/"+id, nil, nil); s != http.StatusNotFound {
		t.Errorf("de novo: %d", s)
	}
}
