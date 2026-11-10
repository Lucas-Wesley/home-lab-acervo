package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bibliotecaParaMarcar: "grande" tem três capítulos que dão dois lotes; "escaneado" não tem texto;
// "sem-sumario" tem texto (45 páginas) mas nenhum capítulo.
func bibliotecaParaMarcar(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/books":
		writeJSON(w, http.StatusOK, []map[string]any{{"id": "grande", "title": "História do Clube", "author": "Autor"}, {"id": "escaneado", "title": "Escaneado"}, {"id": "sem-sumario", "title": "Sem sumário"}})
	case "/api/books/grande/chapters":
		writeJSON(w, http.StatusOK, map[string]any{"indexed": true, "chapters": []map[string]any{
			{"n": 1, "title": "Fundação", "first_page": 1, "last_page": 20, "chars": 40000},
			{"n": 2, "title": "Segunda parte", "first_page": 21, "last_page": 21, "chars": 50},
			{"n": 3, "title": "Os títulos", "first_page": 21, "last_page": 40, "chars": 30000},
		}})
	case "/api/books/escaneado/chapters":
		writeJSON(w, http.StatusOK, map[string]any{"indexed": false, "chapters": []any{}})
	case "/api/books/sem-sumario/chapters":
		writeJSON(w, http.StatusOK, map[string]any{"indexed": true, "pages": 45, "chapters": []any{}})
	case "/api/books/sem-sumario/text":
		q := r.URL.Query()
		if q.Get("chapters") != "" {
			http.Error(w, "sem capítulos", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"parts": []map[string]any{{"title": "Páginas " + q.Get("from") + " a " + q.Get("to"),
			"passages": []map[string]any{{"page": 1, "text": "Texto das páginas " + q.Get("from") + "."}}}}})
	case "/api/books/grande/text":
		var parts []map[string]any
		for _, n := range strings.Split(r.URL.Query().Get("chapters"), ",") {
			parts = append(parts, map[string]any{"title": "Capítulo " + n, "first_page": 1, "last_page": 2,
				"passages": []map[string]any{{"page": 1, "text": "Texto do capítulo " + n + "."}}})
		}
		writeJSON(w, http.StatusOK, map[string]any{"parts": parts})
	default:
		http.NotFound(w, r)
	}
}

func TestLivrosMarcadosPorAssunto(t *testing.T) {
	biblioteca := httptest.NewServer(http.HandlerFunc(bibliotecaParaMarcar))
	defer biblioteca.Close()
	e := novoAmbiente(t)
	e.app.cfg.biblioteca = biblioteca.URL
	pasta := filepath.Join(e.app.conteudo.raiz, "temas", "clube")
	os.MkdirAll(pasta, 0o755)
	os.WriteFile(filepath.Join(pasta, "indice.md"), []byte("---\ntitulo: Clube\n---\n\n# Clube\n\n## Fundação\n\nFundado em 1930.\n"), 0o644)
	ctx := context.Background()

	marcados := func() map[string]map[string]any {
		var l []map[string]any
		e.pedir("GET", "/api/livros-marcados", nil, &l)
		m := map[string]map[string]any{}
		for _, x := range l {
			m[x["livro"].(string)] = x
		}
		return m
	}
	// Com a fila parada, o primeiro livro entra; um lote por tarefa, até o fim.
	if err := e.app.proximoLivro(ctx); err != nil {
		t.Fatal(err)
	}
	e.esperar("livro marcado", func() bool { return marcados()["grande"]["estado"] == "pronto" })
	if m := marcados()["grande"]; m["marcados"].(float64) != 3 || m["faltam"].(float64) != 0 {
		var ts []tarefa
		e.pedir("GET", "/api/tarefas", nil, &ts)
		t.Fatalf("marcas: %v\n%+v", m, ts)
	}
	var tarefas []tarefaComTitulo
	e.pedir("GET", "/api/tarefas", nil, &tarefas)
	lotes := 0
	for _, x := range tarefas {
		if x.Tipo == "marcar_livro" {
			lotes++
			if x.Referencia != "História do Clube" {
				t.Errorf("título na fila: %+v", x)
			}
		}
	}
	if lotes != 2 {
		t.Errorf("tarefas de marcar: %d (dois lotes)", lotes)
	}

	// O próximo não indexado falha e fica para depois; o marcado não entra de novo.
	e.app.proximoLivro(ctx)
	e.esperar("escaneado", func() bool { return marcados()["escaneado"]["estado"] == "falhou" })

	// Sem sumário: lido em blocos de 20 páginas, cada um marcado como um capítulo.
	e.app.proximoLivro(ctx)
	e.esperar("sem sumário", func() bool { return marcados()["sem-sumario"]["estado"] == "pronto" })
	if m := marcados()["sem-sumario"]; m["marcados"].(float64) != 3 {
		t.Fatalf("blocos: %v", m)
	}
	var feitos string
	e.app.db.QueryRow(`SELECT capitulos FROM livros_marcados WHERE livro = 'sem-sumario'`).Scan(&feitos)
	if !strings.Contains(feitos, `"titulo":"Páginas 41 a 45","de":41,"ate":45`) {
		t.Errorf("blocos marcados: %s", feitos)
	}

	e.app.proximoLivro(ctx)
	e.pedir("GET", "/api/tarefas", nil, &tarefas)
	if len(tarefas) != 3+2 {
		t.Errorf("entrou de novo: %d tarefas", len(tarefas))
	}

	// No Tema: os capítulos marcados com ele, com o modelo menor.
	var livros []capituloSobre
	if s := e.pedir("GET", "/api/temas-livros?tema=temas/clube", nil, &livros); s != http.StatusOK || len(livros) != 6 ||
		!strings.Contains(livros[0].Resumo, "modelo: haiku") || livros[0].Titulo != "História do Clube" || !strings.Contains(livros[0].Link, "/#/livro/grande") {
		t.Fatalf("livros do Tema: %d %+v", s, livros)
	}
	if s := e.pedir("GET", "/api/temas-livros?tema=../x", nil, nil); s != http.StatusBadRequest {
		t.Errorf("tema inválido: %d", s)
	}
	// E entram como pista no Perguntar.
	if p := e.perguntaPronta("temas/clube", "Quem fundou?"); !strings.Contains(p.Resultado.Resposta, "Com livros da biblioteca.") {
		t.Errorf("perguntar sem os livros: %s", p.Resultado.Resposta)
	}
}
