package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// fontesFalsas sobe uma Biblioteca, uma Wikipédia (em português e em inglês) e uma página da web
// de mentira e aponta o app para elas.
type fontesFalsas struct {
	mu          sync.Mutex
	buscasLivro []string
	semLivros   bool
	wikiFalha   bool // a Wikipédia responde 503 a tudo
	wikiPedidos int
}

// artigoLongo passa do mínimo para não buscar o artigo em inglês.
func artigoLongo(titulo string) string {
	return "Texto do artigo " + titulo + ". " + strings.Repeat("Mais texto do artigo. ", 200)
}

// wikipediaFalsa: "Inexistente…" não existe; "…curto" é curto e tem versão em inglês; o resto é longo.
// A busca devolve "Zé Ramalho" (que não combina com nada pedido) e o que combina com "Citricultura".
func (f *fontesFalsas) wikipediaFalsa(lingua string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.wikiPedidos++
		falha := f.wikiFalha
		f.mu.Unlock()
		if falha {
			http.Error(w, "indisponível", http.StatusServiceUnavailable)
			return
		}
		q := r.URL.Query()
		if q.Get("list") == "search" {
			titulos := []map[string]string{{"title": "Zé Ramalho"}}
			if strings.Contains(q.Get("srsearch"), "Citricultura") {
				titulos = append(titulos, map[string]string{"title": "Citricultura no Brasil e no mundo"})
			}
			writeJSON(w, http.StatusOK, map[string]any{"query": map[string]any{"search": titulos}})
			return
		}
		titulo := q.Get("titles")
		pagina := map[string]any{"title": titulo, "extract": artigoLongo(titulo)}
		switch {
		case strings.HasPrefix(titulo, "Inexistente") || titulo == "Citricultura no Brasil":
			pagina = map[string]any{"title": titulo, "missing": ""}
		case strings.HasSuffix(titulo, "curto"):
			pagina["extract"] = "Texto curto de " + titulo + "."
			pagina["langlinks"] = []map[string]string{{"lang": "en", "*": "Short article"}}
		}
		if _, falta := pagina["missing"]; lingua == "en" && !falta {
			pagina["extract"] = "English article " + titulo + ". " + strings.Repeat("More text. ", 300)
		}
		writeJSON(w, http.StatusOK, map[string]any{"query": map[string]any{"pages": map[string]any{"1": pagina}}})
	}
}

func (f *fontesFalsas) ligar(e *ambiente) {
	esperasDaColeta = []time.Duration{time.Millisecond, time.Millisecond}
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
		// O segundo exemplar do mesmo livro fica de fora.
		writeJSON(w, http.StatusOK, map[string]any{"results": []map[string]any{{
			"book_id": "livro1", "title": "Frutas do Brasil", "author": "Autora", "page": 42, "chapter": "Os citros",
			"text": "A laranja doce veio da Ásia.", "anchor": `{"pagina":42}`,
		}, {
			"book_id": "livro1-copia", "title": "Frutas do Brasil", "author": "Autora", "page": 42, "chapter": "Os citros",
			"text": "A laranja doce veio da Ásia (cópia).", "anchor": `{"pagina":42}`,
		}}})
	}))
	wikipedia := httptest.NewServer(f.wikipediaFalsa("pt"))
	pagina := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html lang=pt><head><title>Laranja no Brasil - Embrapa</title></head><body><article><h1>Laranja</h1><p>%s</p></article></body></html>",
			strings.Repeat("A laranja chegou ao Brasil com os portugueses no século XVI. ", 20))
	}))
	e.t.Cleanup(biblioteca.Close)
	e.t.Cleanup(wikipedia.Close)
	e.t.Cleanup(pagina.Close)
	e.app.cfg.biblioteca = biblioteca.URL
	e.app.cfg.bibliotecaLink = "https://biblioteca.lan"
	e.app.cfg.wikipedia = wikipedia.URL
	e.t.Setenv("PESQUISA_PAGINA_URL", pagina.URL+"/laranja")
	// Os capítulos do livro, marcados por assunto (D38).
	e.app.db.Exec(`INSERT INTO livros_marcados (livro, titulo, autor, estado, capitulos, atualizado_em) VALUES
		('livro1', 'Frutas do Brasil', 'Autora', 'pronto', '[{"n":2,"titulo":"O caju","de":5,"ate":9,"resumo":"O caju e a castanha.","temas":["Caju"]}]', 'x')`)
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
	// D49: o capítulo que existe no mapa dos livros marcados fica, uma vez só; o que não existe sai.
	// Sem "sites", as páginas ficam de fora.
	if len(p.Capitulos) != 1 || p.Capitulos[0].Capitulo != "O caju" || p.Capitulos[0].De != 5 || p.Capitulos[0].Pergunta != 2 || len(p.Paginas) != 0 {
		t.Fatalf("capítulos e páginas: %+v %+v", p.Capitulos, p.Paginas)
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
	if strings.Contains(string(livros), "(cópia)") {
		t.Errorf("o exemplar repetido do livro deveria sair:\n%s", livros)
	}
	artigo, _ := os.ReadFile(filepath.Join(pasta, "fontes", "wikipedia-laranja-fruta.md"))
	if !strings.Contains(string(artigo), "Texto do artigo Laranja (fruta).") || !strings.Contains(string(artigo), "/wiki/Laranja_%28fruta%29") {
		t.Errorf("artigo:\n%s", artigo)
	}
	// "Citricultura no Brasil" não existe: vale o resultado da busca que combina, não o "Zé Ramalho".
	if !existe(filepath.Join(pasta, "fontes", "wikipedia-citricultura-no-brasil-e-no-mundo.md")) || existe(filepath.Join(pasta, "fontes", "wikipedia-ze-ramalho.md")) {
		t.Errorf("artigo pela busca")
	}
	// O capítulo inteiro, com as páginas e a marcação do Lucas.
	capitulo, _ := os.ReadFile(filepath.Join(pasta, "fontes", "livro-frutas-do-brasil-p5.md"))
	if !strings.Contains(string(capitulo), "O caju é nativo do Nordeste") || !strings.Contains(string(capitulo), "Nota do Lucas: rota do caju") ||
		!strings.Contains(string(capitulo), "[p. 6](https://biblioteca.lan/#/livro/livro1?em=") {
		t.Errorf("capítulo:\n%s", capitulo)
	}
	// O vídeo curto e o sem legenda ficam de fora; o mesmo vídeo não entra duas vezes.
	video, _ := os.ReadFile(filepath.Join(pasta, "fontes", "video-AAAAAAAAAAA.md"))
	if !strings.Contains(string(video), "[00:31] Em 1494") {
		t.Errorf("transcrição:\n%s", video)
	}
	entradas, _ := os.ReadDir(filepath.Join(pasta, "fontes"))
	if len(entradas) != 6 {
		t.Errorf("fontes/ deveria ter 2 de livros, 1 capítulo, 2 artigos e 1 vídeo: %v", entradas)
	}

	p = pedidoDaResposta(t, d)
	if !strings.Contains(d.Textos["cru"], "[Laranja (fruta)](") || !strings.Contains(d.Textos["cru"], "`fontes/video-AAAAAAAAAAA.md` (pergunta 1)") ||
		!strings.Contains(d.Textos["cru"], "**Capítulo de livro:** [“O caju” (p. 5–9), de Frutas do Brasil]") {
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
	// D49: a página do esboço foi lida pelo servidor (o vídeo do YouTube não é página); sem livros, nada de capítulos.
	p := pedidoDaResposta(t, d)
	if len(p.Paginas) != 1 || len(p.Capitulos) != 0 {
		t.Fatalf("páginas: %+v capítulos: %+v", p.Paginas, p.Capitulos)
	}
	sites, _ := filepath.Glob(filepath.Join(e.app.pastaReferencia(id), "fontes", "site-*.md"))
	if len(sites) != 1 {
		t.Fatalf("site: %v", sites)
	}
	if b, _ := os.ReadFile(sites[0]); !strings.Contains(string(b), "chegou ao Brasil com os portugueses") || !strings.Contains(string(b), "- URL: http") {
		t.Errorf("página:\n%s", b)
	}
	// Sem livros nem vídeos escolhidos, as buscas deles ficam vazias.
	if p.Esboco[0].Livros != "" || p.Esboco[0].Youtube != "" || p.Esboco[0].Wikipedia != "Laranja (fruta)" {
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
	e.app.db.Exec(`DELETE FROM livros_marcados`) // sem capítulos marcados, só a busca, que não acha nada
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

// D49: o artigo pelo nome exato; a busca só quando não existe, e só com título que combine; o
// artigo em inglês quando o em português é curto; e a consulta que falha por um momento tenta de novo.
func TestArtigosDaWikipedia(t *testing.T) {
	e := novoAmbiente(t)
	f := &fontesFalsas{}
	f.ligar(e)
	en := httptest.NewServer(f.wikipediaFalsa("en"))
	t.Cleanup(en.Close)
	e.app.cfg.wikipediaEN = en.URL
	pasta := t.TempDir()
	os.MkdirAll(filepath.Join(pasta, "fontes"), 0o755)
	vistos := map[string]bool{}

	c := e.app.coletarWikipedia(t.Context(), pasta, "João Ramalho | Inexistente Ramalho | João Ramalho", 3, vistos)
	if len(c) != 2 || c[0].Titulo != "João Ramalho" || c[0].Erro != "" || c[1].Erro != "nenhum artigo com esse nome" {
		t.Fatalf("pelo nome: %+v", c)
	}
	c = e.app.coletarWikipedia(t.Context(), pasta, "Laranja curto", 1, vistos)
	if len(c) != 2 || c[1].Titulo != "Short article (em inglês)" || !existe(filepath.Join(pasta, "fontes", "wikipedia-en-short-article.md")) {
		t.Fatalf("em inglês: %+v", c)
	}
	// O mesmo artigo não volta.
	if c = e.app.coletarWikipedia(t.Context(), pasta, "João Ramalho", 1, vistos); len(c) != 0 {
		t.Errorf("repetido: %+v", c)
	}
	f.mu.Lock()
	f.wikiFalha, f.wikiPedidos = true, 0
	f.mu.Unlock()
	c = e.app.coletarWikipedia(t.Context(), pasta, "Outro", 1, vistos)
	if len(c) != 1 || !strings.Contains(c[0].Erro, "503") || f.wikiPedidos != 3 {
		t.Errorf("falha depois de tentar 3 vezes: %+v (%d pedidos)", c, f.wikiPedidos)
	}

	for _, caso := range []struct {
		nome, titulo string
		combina      bool
	}{
		{"João Ramalho", "Zé Ramalho", false},
		{"Origem das plantas cultivadas", "Plantae", false},
		{"Rio São Francisco integração nacional", "Ministério da Integração Nacional", false},
		{"Cultivo de citros", "Cultivo de citros no Brasil", true},
		{"Diogo Álvares Correia", "Diogo Álvares Correia", true},
	} {
		if tituloCombina(caso.nome, caso.titulo) != caso.combina {
			t.Errorf("%q × %q: deveria ser %v", caso.nome, caso.titulo, caso.combina)
		}
	}
}

// D49: "Buscar de novo o que faltou" refaz só as buscas que falharam e escreve o texto de novo.
func TestBuscarDeNovoOQueFaltou(t *testing.T) {
	e := novoAmbiente(t)
	f := &fontesFalsas{wikiFalha: true}
	f.ligar(e)
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{"livros", "wikipedia"}, "profundidade": "rapida"})
	var d detalhe
	e.esperar("pronta", func() bool {
		d = e.referencia(id)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	p := pedidoDaResposta(t, d)
	if p.buscasQueFalharam() != 2 || !strings.Contains(d.Textos["cru"], "## Sem resultado") {
		t.Fatalf("as duas buscas da Wikipédia falharam: %+v", p.Coleta)
	}
	antes := d.Textos["organizado"]

	f.mu.Lock()
	f.wikiFalha = false
	buscasAntes := len(f.buscasLivro)
	f.mu.Unlock()
	if s := e.pedir("POST", "/api/referencias/"+id+"/recoletar", nil, nil); s != http.StatusOK {
		t.Fatalf("recoletar: %d", s)
	}
	e.esperar("escrita de novo", func() bool {
		d = e.referencia(id)
		return len(d.Tarefas) == 7 && d.Estado == "organizado" && string(d.Sugestao) != "null" && d.Tarefas[0].Estado == "concluida"
	})
	p = pedidoDaResposta(t, d)
	if p.buscasQueFalharam() != 0 || !strings.Contains(d.Textos["organizado"], "wikipedia-laranja-fruta.md") {
		t.Errorf("depois: %+v\n%s", p.Coleta, d.Textos["organizado"])
	}
	f.mu.Lock()
	if len(f.buscasLivro) != buscasAntes {
		t.Errorf("os livros, que deram certo, não deveriam ser buscados de novo")
	}
	f.mu.Unlock()
	if b, _ := os.ReadFile(filepath.Join(e.app.pastaReferencia(id), "anterior.md")); string(b) != antes {
		t.Errorf("o texto de antes deveria ficar como Versão anterior")
	}
	if s := e.pedir("POST", "/api/referencias/"+id+"/recoletar", nil, nil); s != http.StatusConflict {
		t.Errorf("sem nada faltando: %d", s)
	}
}

func TestTextoDoPDF(t *testing.T) {
	got := limparTextoDoPDF("Primeira linha\ncontinua aqui.\n\nSegundo  parágrafo.\f\nTerceiro.")
	if got != "Primeira linha continua aqui.\n\nSegundo parágrafo.\n\nTerceiro." {
		t.Errorf("%q", got)
	}
}
