package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// flashcardsFalso faz o papel do app Flashcards: guarda o que recebeu e devolve uma proposta.
type flashcardsFalso struct {
	mu        sync.Mutex
	textos    []textoParaCartoes
	acoes     []string
	foraDoAr  bool
	aprovados []any
}

func (f *flashcardsFalso) ligar(e *ambiente) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.foraDoAr {
			http.Error(w, "fora", http.StatusServiceUnavailable)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/baralhos":
			writeJSON(w, http.StatusOK, []map[string]any{{"id": "b1", "nome": "Tema de Teste", "cartoes": 3}})
		case r.Method == "POST" && r.URL.Path == "/api/textos":
			var t textoParaCartoes
			json.NewDecoder(r.Body).Decode(&t)
			f.textos = append(f.textos, t)
			writeJSON(w, http.StatusCreated, map[string]any{"id": len(f.textos), "estado": "na_fila"})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/geracoes/"):
			writeJSON(w, http.StatusOK, map[string]any{"id": 1, "estado": "em_revisao", "proposta": map[string]any{
				"nome_baralho": "Tema de Teste", "cartoes": []map[string]string{{"frente": "P?", "verso": "R"}}}})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/geracoes/"):
			f.acoes = append(f.acoes, r.URL.Path)
			if strings.HasSuffix(r.URL.Path, "/aprovar") {
				var corpo any
				json.NewDecoder(r.Body).Decode(&corpo)
				f.aprovados = append(f.aprovados, corpo)
				writeJSON(w, http.StatusOK, map[string]any{"baralho_id": "b1", "criadas": 1})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id": 1})
		default:
			http.NotFound(w, r)
		}
	}))
	e.t.Cleanup(srv.Close)
	e.app.cfg.flashcards = srv.URL
	e.app.cfg.flashcardsLink = "https://flashcards.lan"
	e.app.cfg.acervoLink = "https://acervo.lan"
}

func TestFlashcardsDeUmCapituloEDoTemaInteiro(t *testing.T) {
	e := temaPronto(t)
	f := &flashcardsFalso{}
	f.ligar(e)
	antes := e.indice()
	capitulos := secoes(antes.Conteudo)
	if len(capitulos) < 2 {
		t.Fatalf("o Tema de teste precisa de dois capítulos:\n%s", antes.Conteudo)
	}
	primeiro, segundo := capitulos[0].Titulo, capitulos[1].Titulo

	// Para escolher o baralho: os que existem no Flashcards e o nome sugerido, o do Tema.
	var escolha struct {
		Baralhos []struct{ ID, Nome string } `json:"baralhos"`
		Sugestao string                       `json:"sugestao"`
	}
	if s := e.pedir("GET", "/api/flashcards/baralhos?arquivo=temas/tema-de-teste/indice.md&capitulo="+url.QueryEscape(primeiro), nil, &escolha); s != http.StatusOK ||
		len(escolha.Baralhos) != 1 || escolha.Baralhos[0].ID != "b1" || escolha.Sugestao != "Tema de Teste" {
		t.Fatalf("baralhos: %d %+v", s, escolha)
	}

	// Um capítulo: só o texto dele, com o caminho no Acervo, no baralho escolhido; o link abre no capítulo.
	var envio envioFlashcards
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "capitulo": primeiro, "baralho_id": "b1"}, &envio); s != http.StatusCreated {
		t.Fatalf("enviar capítulo: %d", s)
	}
	got := f.textos[0]
	linkDoPrimeiro := "https://acervo.lan/#/ler/temas/tema-de-teste/indice.md?capitulo=" + url.PathEscape(primeiro)
	if envio.GeracaoID != 1 || got.BaralhoID != "b1" || got.Baralho != "" || got.Titulo != "Tema de Teste › "+primeiro ||
		got.Link != linkDoPrimeiro || len(got.Secoes) != 1 {
		t.Fatalf("envio %+v, texto %+v", envio, got)
	}
	if !strings.HasPrefix(got.Texto, "## "+primeiro) || strings.Contains(got.Texto, "## "+segundo) || !strings.Contains(got.Contexto, "Onde fica no Acervo") {
		t.Errorf("texto do capítulo:\n%s\ncontexto: %s", got.Texto, got.Contexto)
	}

	// Separado em página própria, o capítulo entra no texto do Tema inteiro no lugar do link.
	e.pedir("POST", "/api/capitulos/separar", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "titulo": primeiro, "base": antes.Hash}, nil)
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "baralho": " Revisão geral "}, nil); s != http.StatusCreated {
		t.Fatalf("enviar Tema: %d", s)
	}
	inteiro := f.textos[1].Texto
	if strings.Contains(inteiro, "Ler o capítulo →") || !strings.Contains(inteiro, "## "+segundo) || strings.HasPrefix(inteiro, "---") {
		t.Errorf("Tema inteiro:\n%s", inteiro)
	}
	// Baralho novo, com o nome dado; cada capítulo com o link dele (o separado, a página própria).
	links := map[string]string{}
	for _, s := range f.textos[1].Secoes {
		links[s.Titulo] = s.Link
	}
	if f.textos[1].Baralho != "Revisão geral" || f.textos[1].BaralhoID != "" ||
		!strings.HasPrefix(links[primeiro], "https://acervo.lan/#/ler/temas/tema-de-teste/") || strings.Contains(links[primeiro], "?capitulo=") ||
		links[segundo] != "https://acervo.lan/#/ler/temas/tema-de-teste/indice.md?capitulo="+url.PathEscape(segundo) {
		t.Errorf("baralho %q, seções %v", f.textos[1].Baralho, links)
	}

	var lista struct {
		Disponivel bool              `json:"disponivel"`
		Link       string            `json:"link"`
		Envios     []envioFlashcards `json:"envios"`
	}
	e.pedir("GET", "/api/flashcards?alvo="+alvoDeArquivo("temas/tema-de-teste/indice.md", primeiro), nil, &lista)
	if !lista.Disponivel || lista.Link != "https://flashcards.lan" || len(lista.Envios) != 1 {
		t.Errorf("lista: %+v", lista)
	}

	// Capítulo que não existe e arquivo fora dos Temas.
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "capitulo": "Não existe"}, nil); s != http.StatusNotFound {
		t.Errorf("capítulo inexistente: %d", s)
	}
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"arquivo": "../acervo.db"}, nil); s != http.StatusBadRequest {
		t.Errorf("arquivo inválido: %d", s)
	}

	// Flashcards fora do ar: o erro volta e o envio fica anotado, sem pedido lá.
	f.foraDoAr = true
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"arquivo": "temas/tema-de-teste/indice.md", "capitulo": segundo}, nil); s != http.StatusBadGateway {
		t.Errorf("fora do ar: %d", s)
	}
}

func TestPesquisaComFlashcardsAprovadosNoAcervo(t *testing.T) {
	e := novoAmbiente(t)
	f := &flashcardsFalso{}
	f.ligar(e)
	id := e.pesquisar(map[string]any{"tema": "origem da laranja", "fontes": []string{}, "profundidade": "rapida", "flashcards": true, "baralho": map[string]string{"baralho_id": "b1"}})
	var lista struct {
		Envios []envioFlashcards `json:"envios"`
	}
	e.esperar("envio ao Flashcards", func() bool {
		e.pedir("GET", "/api/flashcards?alvo="+alvoDeReferencia(id), nil, &lista)
		return len(lista.Envios) == 1
	})
	if f.textos[0].Link != "https://acervo.lan/#/referencia/"+id || !strings.Contains(f.textos[0].Contexto, "pesquisa") || f.textos[0].BaralhoID != "b1" {
		t.Errorf("texto enviado: %+v", f.textos[0])
	}
	geracao := strconv.FormatInt(lista.Envios[0].GeracaoID, 10)

	var g map[string]any
	if s := e.pedir("GET", "/api/flashcards/"+geracao, nil, &g); s != http.StatusOK || g["estado"] != "em_revisao" {
		t.Fatalf("proposta: %d %v", s, g)
	}
	if s := e.pedir("POST", "/api/flashcards/"+geracao+"/aprovar", map[string]any{"nome_baralho": "Laranja", "cartoes": []map[string]string{{"frente": "P?", "verso": "R"}}}, nil); s != http.StatusOK {
		t.Fatalf("aprovar: %d", s)
	}
	// Só passa adiante o que foi enviado daqui.
	if s := e.pedir("GET", "/api/flashcards/99", nil, nil); s != http.StatusNotFound {
		t.Errorf("proposta alheia: %d", s)
	}
	if s := e.pedir("POST", "/api/flashcards/"+geracao+"/apagar", nil, nil); s != http.StatusNotFound {
		t.Errorf("ação desconhecida: %d", s)
	}

	// Excluir a pesquisa descarta a proposta de cartões dela.
	if s := e.pedir("DELETE", "/api/referencias/"+id, nil, nil); s != http.StatusNoContent {
		t.Fatalf("excluir: %d", s)
	}
	if f.acoes[len(f.acoes)-1] != "/api/geracoes/"+geracao+"/descartar" {
		t.Errorf("ações: %v", f.acoes)
	}
}

func TestReferenciaOrganizadaViraFlashcards(t *testing.T) {
	e := novoAmbiente(t)
	f := &flashcardsFalso{}
	f.ligar(e)
	id := e.adicionar("https://www.youtube.com/watch?v=AAAAAAAAAAA")
	e.esperarEstado(id, "organizado")
	// Organizada já vai, mesmo sem validar; sem escolha, para um baralho novo com o título dela.
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"referencia": id}, nil); s != http.StatusCreated {
		t.Errorf("organizada: %d", s)
	}
	if f.textos[0].Baralho == "" || f.textos[0].BaralhoID != "" || f.textos[0].Tipo != "" {
		t.Errorf("baralho: %+v", f.textos[0])
	}
	// De leitura (D47): o tipo vai junto; outro tipo é recusado.
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"referencia": id, "tipo": "invertido"}, nil); s != http.StatusBadRequest {
		t.Errorf("tipo inválido: %d", s)
	}
	f.textos = nil
	e.app.db.Exec(`DELETE FROM flashcards`)
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"referencia": id, "tipo": "leitura"}, nil); s != http.StatusCreated || f.textos[0].Tipo != "leitura" {
		t.Errorf("leitura: %d %+v", s, f.textos)
	}
	e.pedir("DELETE", "/api/referencias/"+id, nil, nil)
	if s := e.pedir("POST", "/api/flashcards", map[string]string{"referencia": id}, nil); s != http.StatusNotFound {
		t.Errorf("excluída: %d", s)
	}
}
