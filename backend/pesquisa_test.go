package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// fontesFalsas sobe uma Biblioteca e uma Wikipédia de mentira e aponta o app para elas.
type fontesFalsas struct {
	mu          sync.Mutex
	buscasLivro []string
	semLivros   bool
}

func (f *fontesFalsas) ligar(e *ambiente) {
	biblioteca := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/books") {
			bibliotecaDeLivros(w, r)
			return
		}
		f.mu.Lock()
		f.buscasLivro = append(f.buscasLivro, r.URL.Query().Get("q"))
		vazio := f.semLivros
		f.mu.Unlock()
		if vazio {
			writeJSON(w, http.StatusOK, map[string]any{"results": []any{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": []map[string]any{{
			"book_id": "livro1", "title": "Frutas do Brasil", "author": "Autora", "page": 42, "chapter": "Os citros",
			"text": "A laranja doce veio da Ásia.", "anchor": `{"pagina":42}`,
		}}})
	}))
	wikipedia := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("list") == "search" {
			writeJSON(w, http.StatusOK, map[string]any{"query": map[string]any{"search": []map[string]string{
				{"title": "Laranja (fruta)"}, {"title": "Citricultura no Brasil"},
			}}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"query": map[string]any{"pages": map[string]any{
			"1": map[string]string{"title": q.Get("titles"), "extract": "Texto do artigo " + q.Get("titles") + "."},
		}}})
	}))
	e.t.Cleanup(biblioteca.Close)
	e.t.Cleanup(wikipedia.Close)
	e.app.cfg.biblioteca = biblioteca.URL
	e.app.cfg.bibliotecaLink = "https://biblioteca.lan"
	e.app.cfg.wikipedia = wikipedia.URL
}

func (e *ambiente) pesquisar(pedido map[string]any) string {
	e.t.Helper()
	var ref referencia
	if s := e.pedir("POST", "/api/pesquisas", pedido, &ref); s != http.StatusCreated {
		e.t.Fatalf("POST /api/pesquisas: %d", s)
	}
	if ref.Tipo != "pesquisa" || ref.Estado != "novo" {
		e.t.Fatalf("pesquisa criada: %+v", ref)
	}
	return ref.ID
}

func pedidoDaResposta(t *testing.T, d detalhe) pedidoPesquisa {
	t.Helper()
	var p pedidoPesquisa
	if err := json.Unmarshal(d.Pesquisa, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func tiposDasTarefas(d detalhe) []string {
	var tipos []string
	for _, t := range d.Tarefas {
		tipos = append([]string{t.Tipo}, tipos...)
	}
	return tipos
}

func TestPesquisaDetalhadaComEsbocoRevisado(t *testing.T) {
	e := novoAmbiente(t)
	f := &fontesFalsas{}
	f.ligar(e)
	id := e.pesquisar(map[string]any{
		"tema": "Qual é a origem da laranja?", "fontes": []string{"youtube", "livros", "wikipedia"},
		"profundidade": "detalhada", "revisar_esboco": true,
	})

	// O esboço para e espera o Lucas. A pergunta vazia sai; a busca vazia vira a própria pergunta.
	d := e.esperarEstado(id, "esboco")
	p := pedidoDaResposta(t, d)
	if d.Titulo != "A origem da laranja" || len(p.Esboco) != 2 || p.Esboco[1].Livros != "Como chegou ao Brasil?" {
		t.Fatalf("esboço: %q %+v", d.Titulo, p.Esboco)
	}
	if !slices.Equal(p.Fontes, []string{"livros", "wikipedia", "youtube"}) {
		t.Errorf("fontes na ordem conhecida: %v", p.Fontes)
	}
	if s := e.pedir("POST", "/api/referencias/"+id+"/reprocessar", nil, nil); s != http.StatusConflict {
		t.Errorf("reprocessar no esboço: %d", s)
	}

	// O Lucas edita uma pergunta e aprova.
	p.Esboco[0].Livros = "laranja doce"
	if s := e.pedir("POST", "/api/referencias/"+id+"/pesquisar", map[string]any{"esboco": p.Esboco}, nil); s != http.StatusOK {
		t.Fatalf("aprovar esboço: %d", s)
	}
	if s := e.pedir("POST", "/api/referencias/"+id+"/pesquisar", map[string]any{}, nil); s != http.StatusConflict {
		t.Errorf("aprovar de novo: %d", s)
	}

	// Coleta → escrita → conferência → pronta, com a sugestão de Tema já pedida.
	e.esperar("sugestão", func() bool {
		d = e.referencia(id)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	if got := tiposDasTarefas(d); !slices.Equal(got, []string{"planejar", "coletar", "redigir", "conferir", "sugerir"}) {
		t.Errorf("etapas: %v", got)
	}
	f.mu.Lock()
	if !slices.Equal(f.buscasLivro, []string{"laranja doce", "Como chegou ao Brasil?"}) {
		t.Errorf("buscas nos livros: %v", f.buscasLivro)
	}
	f.mu.Unlock()

	pasta := e.app.pastaReferencia(id)
	livros, _ := os.ReadFile(filepath.Join(pasta, "fontes", "livros-1.md"))
	if !strings.Contains(string(livros), "Frutas do Brasil, de Autora — p. 42 (Os citros)") ||
		!strings.Contains(string(livros), "https://biblioteca.lan/#/livro/livro1?em=%7B%22pagina%22%3A42%7D") {
		t.Errorf("trechos dos livros:\n%s", livros)
	}
	artigo, _ := os.ReadFile(filepath.Join(pasta, "fontes", "wikipedia-laranja-fruta.md"))
	if !strings.Contains(string(artigo), "Texto do artigo Laranja (fruta).") || !strings.Contains(string(artigo), "/wiki/Laranja_%28fruta%29") {
		t.Errorf("artigo:\n%s", artigo)
	}
	// O vídeo curto e o sem legenda ficam de fora; o mesmo vídeo não entra duas vezes.
	video, _ := os.ReadFile(filepath.Join(pasta, "fontes", "video-AAAAAAAAAAA.md"))
	if !strings.Contains(string(video), "[00:31] Em 1494") {
		t.Errorf("transcrição:\n%s", video)
	}
	entradas, _ := os.ReadDir(filepath.Join(pasta, "fontes"))
	if len(entradas) != 5 {
		t.Errorf("fontes/ deveria ter 2 de livros, 2 artigos e 1 vídeo: %v", entradas)
	}

	p = pedidoDaResposta(t, d)
	if !strings.Contains(d.Textos["cru"], "[Laranja (fruta)](") || !strings.Contains(d.Textos["cru"], "`fontes/video-AAAAAAAAAAA.md`") {
		t.Errorf("material coletado:\n%s", d.Textos["cru"])
	}
	texto := d.Textos["organizado"]
	if strings.Contains(texto, "WebSearch") || !strings.Contains(texto, "- video-AAAAAAAAAAA.md") || !strings.Contains(texto, "Conferência: 1 afirmação") {
		t.Errorf("texto:\n%s", texto)
	}
	if d.Idioma != "pt" {
		t.Errorf("a pesquisa entra no Acervo em português: %q", d.Idioma)
	}
}

func TestPesquisaRapidaComSitesSegueSemParar(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{"sites", "wikipedia"}, "profundidade": "rapida"})

	var d detalhe
	e.esperar("sugestão", func() bool {
		d = e.referencia(id)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	// Rápida: sem conferência. Com sites, a escrita tem internet e guarda o que usou.
	if got := tiposDasTarefas(d); !slices.Equal(got, []string{"planejar", "coletar", "redigir", "sugerir"}) {
		t.Errorf("etapas: %v", got)
	}
	if !strings.Contains(d.Textos["organizado"], "WebSearch,WebFetch") || !existe(filepath.Join(e.app.pastaReferencia(id), "fontes", "web-1.md")) {
		t.Errorf("texto:\n%s", d.Textos["organizado"])
	}
	// Sem livros nem vídeos escolhidos, as buscas deles ficam vazias.
	if p := pedidoDaResposta(t, d); p.Esboco[0].Livros != "" || p.Esboco[0].Youtube != "" || p.Esboco[0].Wikipedia != "Laranja (fruta)" {
		t.Errorf("esboço: %+v", p.Esboco)
	}
}

func TestSoARespostaDaIA(t *testing.T) {
	e := novoAmbiente(t)
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{}, "profundidade": "detalhada", "revisar_esboco": true})
	var d detalhe
	e.esperar("sugestão", func() bool {
		d = e.referencia(id)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	if got := tiposDasTarefas(d); !slices.Equal(got, []string{"redigir", "sugerir"}) {
		t.Errorf("etapas: %v", got)
	}
	if !strings.Contains(d.Textos["organizado"], "Sem fontes") || strings.Contains(d.Textos["organizado"], "WebSearch") {
		t.Errorf("texto:\n%s", d.Textos["organizado"])
	}
}

func TestPesquisaSemMaterialFalhaEDaParaTentarDeNovo(t *testing.T) {
	e := novoAmbiente(t)
	f := &fontesFalsas{semLivros: true}
	f.ligar(e)
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{"livros"}, "profundidade": "rapida"})
	d := e.esperarEstado(id, "erro")
	if !strings.Contains(d.Erro, "nenhuma fonte trouxe material") || !strings.Contains(d.Erro, "nenhum trecho nos livros") {
		t.Errorf("erro: %q", d.Erro)
	}

	// Com livros, "Tentar de novo" volta da coleta, sem refazer o esboço.
	f.mu.Lock()
	f.semLivros = false
	f.mu.Unlock()
	e.pedir("POST", "/api/referencias/"+id+"/reprocessar", nil, nil)
	e.esperar("sugestão", func() bool {
		d = e.referencia(id)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	if got := tiposDasTarefas(d); !slices.Equal(got, []string{"planejar", "coletar", "coletar", "redigir", "sugerir"}) {
		t.Errorf("etapas: %v", got)
	}
}

func TestConferenciaQueFalhaGuardaOTextoDeAntes(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)
	t.Setenv("CLAUDE_FALSO_CONFERIR_FALHA", "1")
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{"wikipedia"}, "profundidade": "detalhada"})
	d := e.esperarEstado(id, "erro")
	if !strings.Contains(d.Textos["organizado"], "Conteúdo da pesquisa") {
		t.Errorf("o texto escrito deveria continuar:\n%s", d.Textos["organizado"])
	}
	os.Unsetenv("CLAUDE_FALSO_CONFERIR_FALHA")
	e.pedir("POST", "/api/referencias/"+id+"/reprocessar", nil, nil)
	e.esperar("pronta", func() bool { return e.referencia(id).Estado == "organizado" })
	if got := tiposDasTarefas(e.referencia(id)); got[len(got)-2] != "conferir" {
		t.Errorf("tentar de novo deveria refazer só a conferência: %v", got)
	}
}

func TestPedidoDePesquisaInvalido(t *testing.T) {
	e := novoAmbiente(t)
	casos := []map[string]any{
		{"tema": "ab", "fontes": []string{}, "profundidade": "rapida"},
		{"tema": "laranja", "fontes": []string{"jornal"}, "profundidade": "rapida"},
		{"tema": "laranja", "fontes": []string{}, "profundidade": "funda"},
		// Sem BIBLIOTECA_URL, os livros não estão disponíveis.
		{"tema": "laranja", "fontes": []string{"livros"}, "profundidade": "rapida"},
	}
	for _, c := range casos {
		if s := e.pedir("POST", "/api/pesquisas", c, nil); s != http.StatusBadRequest {
			t.Errorf("%v: %d", c, s)
		}
	}
}

// Um banco de antes do Pesquisador: video_id obrigatório e sem tipo.
func TestMigracaoDasReferencias(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "acervo.db")
	velho, err := sql.Open("sqlite", caminho)
	if err != nil {
		t.Fatal(err)
	}
	_, err = velho.Exec(`CREATE TABLE referencias (id TEXT PRIMARY KEY, url TEXT NOT NULL, video_id TEXT NOT NULL UNIQUE,
		titulo TEXT NOT NULL DEFAULT '', canal TEXT NOT NULL DEFAULT '', duracao TEXT NOT NULL DEFAULT '', idioma TEXT NOT NULL DEFAULT '',
		estado TEXT NOT NULL, erro TEXT NOT NULL DEFAULT '', sugestao TEXT NOT NULL DEFAULT '', criado_em TEXT NOT NULL, atualizado_em TEXT NOT NULL);
		INSERT INTO referencias (id, url, video_id, titulo, estado, criado_em, atualizado_em) VALUES ('r1', 'u', 'AAAAAAAAAAA', 'Vídeo', 'integrado', 'x', 'x');`)
	velho.Close()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // a segunda abertura não refaz nada
		db, err := abrirBanco(caminho)
		if err != nil {
			t.Fatal(err)
		}
		r, err := lerReferencia(db.QueryRow(`SELECT ` + colunasReferencia + ` FROM referencias WHERE id = 'r1'`))
		if err != nil || r.Titulo != "Vídeo" || r.Tipo != "video" || r.Estado != "integrado" {
			t.Fatalf("depois da migração: %+v %v", r, err)
		}
		for _, id := range []string{"p1", "p2"} {
			db.Exec(`INSERT OR IGNORE INTO referencias (id, url, estado, tipo, criado_em, atualizado_em) VALUES (?, '', 'novo', 'pesquisa', 'x', 'x')`, id)
		}
		var n int
		db.QueryRow(`SELECT COUNT(*) FROM referencias WHERE tipo = 'pesquisa'`).Scan(&n)
		if n != 2 {
			t.Errorf("duas pesquisas sem video_id: %d", n)
		}
		db.Close()
	}
}
