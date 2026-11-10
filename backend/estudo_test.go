package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Estudar (D41): resumo, conceitos anteriores e o banco do simulado saem do Tema inteiro (com os
// subtemas), cada um numa tarefa; o simulado respondido é corrigido; os conceitos e os erros viram
// flashcards. Se o Tema muda, a parte pronta fica marcada como desatualizada.
func TestEstudarUmTema(t *testing.T) {
	e := novoAmbiente(t)
	f := &flashcardsFalso{}
	f.ligar(e)
	raiz := e.app.conteudo.raiz
	escrever := func(rel, texto string) {
		os.MkdirAll(filepath.Dir(filepath.Join(raiz, rel)), 0o755)
		os.WriteFile(filepath.Join(raiz, rel), []byte(texto), 0o644)
	}
	escrever("temas/clube/indice.md", "---\ntitulo: Clube\n---\n\n# Clube\n\n## Fundação\n\nFundado em 1930.\n")
	escrever("temas/clube/estadio/indice.md", "# Estádio\n\nInaugurado em 1940.\n")
	escrever("temas/outro/indice.md", "# Outro\n")

	type resposta struct {
		Partes    map[string]estudo `json:"partes"`
		Simulados []simuladoFeito   `json:"simulados"`
	}
	var r resposta
	ler := func() resposta {
		var x resposta
		e.pedir("GET", "/api/estudos?tema=temas/clube", nil, &x)
		return x
	}
	if s := e.pedir("GET", "/api/estudos?tema=temas/clube", nil, &r); s != http.StatusOK || r.Partes["resumo"].Estado != "vazio" {
		t.Fatalf("antes: %d %+v", s, r)
	}
	for _, parte := range []string{"resumo", "anteriores", "simulado"} {
		if s := e.pedir("POST", "/api/estudos", map[string]string{"tema": "temas/clube", "parte": parte}, nil); s != http.StatusAccepted {
			t.Fatalf("preparar %s: %d", parte, s)
		}
	}
	if s := e.pedir("POST", "/api/estudos", map[string]string{"tema": "temas/clube", "parte": "outra"}, nil); s != http.StatusBadRequest {
		t.Errorf("parte inválida: %d", s)
	}
	if s := e.pedir("POST", "/api/estudos", map[string]string{"tema": "temas/nao-existe", "parte": "resumo"}, nil); s != http.StatusNotFound {
		t.Errorf("tema inexistente: %d", s)
	}
	e.esperar("partes prontas", func() bool {
		r = ler()
		return r.Partes["resumo"].Estado == "pronto" && r.Partes["anteriores"].Estado == "pronto" && r.Partes["simulado"].Estado == "pronto"
	})

	var resumo resumoDoTema
	json.Unmarshal(r.Partes["resumo"].Resultado, &resumo)
	if !strings.Contains(resumo.Texto, "./indice.md") || !strings.Contains(resumo.Texto, "./estadio/indice.md") {
		t.Errorf("o resumo não leu o Tema com os subtemas: %s", resumo.Texto)
	}
	var anteriores struct{ Conceitos []conceitoAnterior }
	json.Unmarshal(r.Partes["anteriores"].Resultado, &anteriores)
	if len(anteriores.Conceitos) != 2 || anteriores.Conceitos[0].Tema != "temas/outro" || anteriores.Conceitos[1].Tema != "" ||
		!strings.Contains(anteriores.Conceitos[0].Explicacao, "WebSearch") {
		t.Errorf("anteriores: %+v", anteriores.Conceitos)
	}
	var banco struct{ Perguntas []perguntaDoSimulado }
	json.Unmarshal(r.Partes["simulado"].Resultado, &banco)
	if len(banco.Perguntas) != 6 || r.Partes["simulado"].Desatualizado {
		t.Fatalf("simulado: %+v", r.Partes["simulado"])
	}

	// O simulado: três perguntas, uma falada e uma em branco.
	respostas := []respostaDoSimulado{
		{Pergunta: banco.Perguntas[0].Pergunta, Esperada: banco.Perguntas[0].Esperada, Capitulo: "Fundação", Resposta: "Em 1930."},
		{Pergunta: banco.Perguntas[1].Pergunta, Esperada: banco.Perguntas[1].Esperada, Resposta: "acho que foi em mil novecentos", PorVoz: true},
		{Pergunta: banco.Perguntas[2].Pergunta, Esperada: banco.Perguntas[2].Esperada},
	}
	if s := e.pedir("POST", "/api/simulados", map[string]any{"tema": "temas/clube", "respostas": []respostaDoSimulado{{Pergunta: "P?"}}}, nil); s != http.StatusBadRequest {
		t.Errorf("simulado sem nenhuma resposta: %d", s)
	}
	var sim simuladoFeito
	if s := e.pedir("POST", "/api/simulados", map[string]any{"tema": "temas/clube", "respostas": respostas}, &sim); s != http.StatusCreated {
		t.Fatalf("entregar: %d", s)
	}
	e.esperar("simulado corrigido", func() bool {
		e.pedir("GET", "/api/simulados/"+sim.ID, nil, &sim)
		return sim.Estado == "pronto"
	})
	c := sim.Correcao
	if c.Nota != 33 || len(c.Itens) != 3 || c.Itens[0].Avaliacao != "certa" || !strings.Contains(c.Comentario, "Teve resposta falada.") ||
		strings.Contains(c.Comentario, "Sem o Tema") {
		t.Errorf("correção: %+v", c)
	}
	if r = ler(); len(r.Simulados) != 1 || r.Simulados[0].ID != sim.ID {
		t.Errorf("histórico: %+v", r.Simulados)
	}

	// Flashcards: dos conceitos anteriores e do que se errou no simulado.
	for _, parte := range []string{"anteriores", "simulado:" + sim.ID} {
		if s := e.pedir("POST", "/api/flashcards", map[string]string{"estudo": "temas/clube", "parte": parte, "baralho": "Clube"}, nil); s != http.StatusCreated {
			t.Fatalf("flashcards de %s: %d", parte, s)
		}
	}
	if len(f.textos) != 2 || !strings.Contains(f.textos[0].Texto, "## Conceito base") ||
		!strings.Contains(f.textos[1].Texto, "Faltou 2.") || strings.Contains(f.textos[1].Texto, banco.Perguntas[0].Pergunta) ||
		!strings.HasSuffix(f.textos[1].Link, "/#/ler/temas/clube/indice.md?estudar") {
		t.Errorf("textos para o Flashcards: %+v", f.textos)
	}

	// O Tema mudou: as partes prontas avisam.
	escrever("temas/clube/estadio/indice.md", "# Estádio\n\nInaugurado em 1941.\n")
	if r = ler(); !r.Partes["resumo"].Desatualizado || r.Partes["resumo"].Estado != "pronto" {
		t.Errorf("depois de mudar o Tema: %+v", r.Partes["resumo"])
	}

	// Na fila, o título do Tema e o link da aba.
	var tarefas []tarefaComTitulo
	e.pedir("GET", "/api/tarefas", nil, &tarefas)
	achou := false
	for _, x := range tarefas {
		if x.Tipo == "estudar" && x.Referencia == "Clube · resumo" && x.Link == "#/ler/temas/clube/indice.md?estudar" {
			achou = true
		}
	}
	if !achou {
		t.Errorf("fila: %+v", tarefas)
	}
}
