package main

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// bibliotecaDeLivros imita a API de livros da Biblioteca (R33 de lá): "livro1" tem dois capítulos;
// "escaneado" ainda não tem texto.
func bibliotecaDeLivros(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/books":
		writeJSON(w, http.StatusOK, []map[string]any{
			{"id": "livro1", "title": "Frutas do Brasil", "author": "Autora", "format": "epub", "has_cover": true, "percent": 0.4, "read_at": "2026-11-05T10:00:00Z"},
			{"id": "escaneado", "title": "Escaneado", "author": "", "format": "pdf", "has_cover": false},
		})
	case "/api/books/livro1/cover":
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("\x89PNG capa"))
	case "/api/books/livro1/chapters":
		writeJSON(w, http.StatusOK, map[string]any{"id": "livro1", "title": "Frutas do Brasil", "author": "Autora", "format": "epub",
			"indexed": true, "pages": 9, "chapters": []map[string]any{
				{"n": 1, "title": "Os citros", "level": 1, "first_page": 1, "last_page": 4, "chars": 6000},
				{"n": 2, "title": "O caju", "level": 1, "first_page": 5, "last_page": 9, "chars": 7000, "highlights": 1},
			}})
	case "/api/books/escaneado/chapters":
		writeJSON(w, http.StatusOK, map[string]any{"id": "escaneado", "title": "Escaneado", "format": "pdf",
			"indexed": false, "error": "o livro não tem texto (PDF escaneado?)", "chapters": []any{}})
	case "/api/books/livro1/text":
		if r.URL.Query().Get("chapters") != "2" {
			http.Error(w, "pedido inesperado: "+r.URL.RawQuery, http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"title": "Frutas do Brasil", "parts": []map[string]any{{
			"title": "O caju", "first_page": 5, "last_page": 9, "passages": []map[string]any{
				{"page": 5, "text": "O caju é nativo do Nordeste brasileiro. " + strings.Repeat("Texto do capítulo. ", 10), "anchor": `{"href":"cap2.xhtml"}`},
				{"page": 5, "text": "A castanha vem junto, presa ao pedúnculo.", "anchor": `{"href":"cap2.xhtml"}`},
				{"page": 6, "text": "Os portugueses levaram o caju para a Índia e a África.", "anchor": `{"href":"cap2.xhtml","p":2}`},
			}, "highlights": []map[string]any{
				{"quote": "Os portugueses levaram o caju\npara a Índia", "note": "rota do caju", "page": 6, "anchor": `{"href":"cap2.xhtml","p":2}`},
			}}}})
	default:
		http.NotFound(w, r)
	}
}

func TestResumoDeUmCapitulo(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)

	var livros []map[string]any
	if s := e.pedir("GET", "/api/livros", nil, &livros); s != http.StatusOK || len(livros) != 2 || livros[0]["has_cover"] != true || livros[0]["read_at"] == nil {
		t.Fatalf("livros: %d %v", s, livros)
	}
	if s := e.pedir("GET", "/api/livros/livro1/capa", nil, nil); s != http.StatusOK {
		t.Errorf("capa: %d", s)
	}
	if s := e.pedir("GET", "/api/livros/escaneado/capa", nil, nil); s != http.StatusNotFound {
		t.Errorf("sem capa: %d", s)
	}
	var sumario sumarioDoLivro
	if s := e.pedir("GET", "/api/livros/livro1", nil, &sumario); s != http.StatusOK || len(sumario.Chapters) != 2 || sumario.Chapters[1].Highlights != 1 {
		t.Fatalf("sumário: %d %+v", s, sumario)
	}

	var ref referencia
	if s := e.pedir("POST", "/api/livros", map[string]any{"livro": "livro1", "capitulos": []int{2}, "foco": "a castanha"}, &ref); s != http.StatusCreated {
		t.Fatalf("POST /api/livros: %d", s)
	}
	if ref.Tipo != "livro" || ref.Titulo != "O caju — Frutas do Brasil" || ref.URL != "https://biblioteca.lan/#/livro/livro1" {
		t.Fatalf("Referência criada: %+v", ref)
	}

	// Ler (sem IA) → resumir → pronta, com a sugestão de Tema já pedida.
	var d detalhe
	e.esperar("resumo pronto", func() bool {
		d = e.referencia(ref.ID)
		return d.Estado == "organizado" && string(d.Sugestao) != "null"
	})
	if got := tiposDasTarefas(d); !slices.Equal(got, []string{"ler_livro", "resumir", "sugerir"}) {
		t.Errorf("etapas: %v", got)
	}
	cru := d.Textos["cru"]
	link5 := "[p. 5](https://biblioteca.lan/#/livro/livro1?em=%7B%22href%22%3A%22cap2.xhtml%22%7D)"
	if !strings.Contains(cru, "## O caju (p. 5–9)") || strings.Count(cru, "[p. 5]") != 1 || !strings.Contains(cru, link5) ||
		!strings.Contains(cru, "[p. 6](") || !strings.Contains(cru, "> Autor: Autora") {
		t.Errorf("cru.md:\n%s", cru)
	}
	if org := d.Textos["organizado"]; !strings.Contains(org, "<sub>Páginas: "+link5) || !strings.Contains(org, "Foco recebido.") || !strings.Contains(org, "Marcações recebidas.") {
		t.Errorf("organizado.md:\n%s", org)
	}
	// As marcações do Lucas vão à parte, com a página, o link e a nota: são os pontos-chave do resumo.
	marcacoes, _ := os.ReadFile(filepath.Join(e.app.pastaReferencia(ref.ID), arquivoDasMarcacoes))
	if m := string(marcacoes); !strings.Contains(m, "## O caju") || !strings.Contains(m, "[p. 6](https://biblioteca.lan/#/livro/livro1?em=") ||
		!strings.Contains(m, "“Os portugueses levaram o caju para a Índia”") || !strings.Contains(m, "Nota do Lucas: rota do caju") || !strings.Contains(m, "1 trechos") {
		t.Errorf("marcacoes.md:\n%s", m)
	}
	nota := notaDoTipo(&d.referencia)
	if !strings.Contains(nota, `o capítulo "O caju" do livro "Frutas do Brasil", de Autora`) || !strings.Contains(nota, "Fonte: Frutas do Brasil, p. N") {
		t.Errorf("nota para criar o Tema: %s", nota)
	}
	if !podeVirarCartoes(&d.referencia) {
		t.Error("o resumo pronto pode virar flashcards")
	}

	// Sem o texto, tentar de novo começa pela leitura.
	os.Remove(filepath.Join(e.app.pastaReferencia(ref.ID), "cru.md"))
	if got := e.app.proximaEtapa(&d.referencia); got != "ler_livro" {
		t.Errorf("próxima etapa sem cru.md: %s", got)
	}
}

func TestPedidoDeResumoInvalido(t *testing.T) {
	e := novoAmbiente(t)
	(&fontesFalsas{}).ligar(e)
	casos := []struct {
		corpo  map[string]any
		status int
	}{
		{map[string]any{"livro": "livro1", "capitulos": []int{3}}, http.StatusBadRequest},
		{map[string]any{"livro": "livro1"}, http.StatusBadRequest},
		{map[string]any{"livro": "livro1", "de": 5, "ate": 20}, http.StatusBadRequest},
		{map[string]any{"livro": "nao-existe", "capitulos": []int{1}}, http.StatusBadRequest},
		{map[string]any{"livro": "escaneado", "de": 1, "ate": 2}, http.StatusConflict},
	}
	for _, c := range casos {
		if s := e.pedir("POST", "/api/livros", c.corpo, nil); s != c.status {
			t.Errorf("%v: %d, queria %d", c.corpo, s, c.status)
		}
	}
	// Páginas, quando o livro não tem os capítulos escolhidos.
	var ref referencia
	if s := e.pedir("POST", "/api/livros", map[string]any{"livro": "livro1", "de": 2, "ate": 3}, &ref); s != http.StatusCreated || ref.Titulo != "Frutas do Brasil, p. 2 a 3" {
		t.Errorf("por páginas: %d %+v", s, ref)
	}

	e.app.cfg.biblioteca = ""
	if s := e.pedir("GET", "/api/livros", nil, nil); s != http.StatusServiceUnavailable {
		t.Errorf("sem Biblioteca: %d", s)
	}
}
