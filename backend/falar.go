package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Falar o foco dos flashcards (D51): o aparelho grava a fala e o servidor manda ao Whisper large-v3 do
// Mac mini (com o do servidor de reserva), como num podcast, mas direto, sem a fila: é curto e o Lucas
// está esperando. Para acertar os nomes, o Whisper recebe o título, os capítulos e os nomes próprios
// do texto de onde saem os cartões. O texto volta ao campo, e lá o Lucas confere antes de mandar.

const (
	tamanhoMaximoDaFala = 25 << 20 // ~25 min de Opus; um foco falado tem um ou dois
	contextoDaFala      = "Instrução ditada para gerar flashcards de um texto: em que parte focar e como quer os cartões."
	maximoDoPrompt      = 700 // o Whisper do Mac corta em 800
)

func (a *app) handleTranscrever(w http.ResponseWriter, r *http.Request) {
	if a.cfg.whisper == "" && a.cfg.whisperReserva == "" {
		erroJSON(w, http.StatusServiceUnavailable, "A transcrição não está configurada (WHISPER_URL).")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, tamanhoMaximoDaFala+1<<20)
	arquivo, cab, err := r.FormFile("audio")
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "Envie a gravação (até 25 MB).")
		return
	}
	defer arquivo.Close()

	prompt := contextoDaFala
	o := origemDosCartoes{Arquivo: r.FormValue("arquivo"), Capitulo: r.FormValue("capitulo"), Referencia: r.FormValue("referencia"),
		Estudo: r.FormValue("estudo"), Parte: r.FormValue("parte")}
	if o.Arquivo != "" || o.Referencia != "" || o.Estudo != "" {
		_, t, ok := a.texto(w, r, o)
		if !ok {
			return
		}
		prompt = promptDoTexto(t)
	}

	pasta, err := os.MkdirTemp("", "fala-")
	if err != nil {
		falha(w, err)
		return
	}
	defer os.RemoveAll(pasta)
	nome := "fala.webm"
	if strings.Contains(cab.Header.Get("Content-Type"), "mp4") || strings.HasSuffix(cab.Filename, ".mp4") {
		nome = "fala.mp4"
	}
	caminho := filepath.Join(pasta, nome)
	f, err := os.Create(caminho)
	if err != nil {
		falha(w, err)
		return
	}
	n, err := io.Copy(f, arquivo)
	f.Close()
	if err != nil || n == 0 {
		erroJSON(w, http.StatusBadRequest, "A gravação veio vazia.")
		return
	}

	trechos, _, err := a.transcrever(r.Context(), caminho, "pt", prompt)
	if err != nil {
		erroJSON(w, http.StatusBadGateway, "Não deu para transcrever: "+err.Error())
		return
	}
	texto := juntarFala(trechos)
	if texto == "" {
		erroJSON(w, http.StatusUnprocessableEntity, "Não ouvi nenhuma fala na gravação.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"texto": texto})
}

// juntarFala: os trechos, sem os que o Whisper inventa no silêncio (a regra dele, a mesma do documentário).
func juntarFala(trechos []trechoFalado) string {
	var partes []string
	for _, t := range trechos {
		if !semFalaNoDocumentario(t) {
			partes = append(partes, t.Texto)
		}
	}
	return strings.Join(strings.Fields(strings.Join(partes, " ")), " ")
}

// promptDoTexto: o contexto da fala, o título, os capítulos e os nomes próprios mais citados do texto.
func promptDoTexto(t textoParaCartoes) string {
	var b strings.Builder
	b.WriteString(contextoDaFala)
	b.WriteString(" Texto: " + t.Titulo + ".")
	var capitulos []string
	for _, s := range t.Secoes {
		capitulos = append(capitulos, s.Titulo)
	}
	if len(capitulos) > 0 {
		b.WriteString(" Capítulos: " + strings.Join(capitulos, ", ") + ".")
	}
	if nomes := nomesProprios(t.Texto, 40); len(nomes) > 0 {
		b.WriteString(" Nomes: " + strings.Join(nomes, ", ") + ".")
	}
	return cortar(strings.Join(strings.Fields(b.String()), " "), maximoDoPrompt, false)
}

var palavraComMaiuscula = regexp.MustCompile(`\p{Lu}[\p{L}\p{M}'’-]+(?:\s+(?:(?:de|da|do|dos|das|e|del|von|van)\s+)?\p{Lu}[\p{L}\p{M}'’-]+)*`)

// nomesProprios: palavras com maiúscula fora do começo da frase (onde toda palavra tem), as mais
// citadas primeiro. Não precisa ser perfeito: é só uma dica para o Whisper.
func nomesProprios(texto string, max int) []string {
	conta := map[string]int{}
	primeira := map[string]int{}
	for _, m := range palavraComMaiuscula.FindAllStringIndex(texto, -1) {
		nome := texto[m[0]:m[1]]
		if comecoDeFrase(texto[:m[0]]) {
			// A primeira palavra tem maiúscula por começar a frase ("Em Tóquio"): vale o que vem depois.
			_, resto, _ := strings.Cut(nome, " ")
			resto = strings.TrimSpace(resto)
			for _, conector := range []string{"de ", "da ", "do ", "dos ", "das ", "e ", "del ", "von ", "van "} {
				resto = strings.TrimPrefix(resto, conector)
			}
			if r, _ := utf8.DecodeRuneInString(resto); !unicode.IsUpper(r) {
				continue
			}
			m[0] = m[1] - len(resto)
			nome = resto
		}
		if utf8.RuneCountInString(nome) < 3 {
			continue
		}
		if _, ok := primeira[nome]; !ok {
			primeira[nome] = m[0]
		}
		conta[nome]++
	}
	nomes := make([]string, 0, len(conta))
	for n := range conta {
		nomes = append(nomes, n)
	}
	sort.Slice(nomes, func(i, j int) bool {
		if conta[nomes[i]] != conta[nomes[j]] {
			return conta[nomes[i]] > conta[nomes[j]]
		}
		return primeira[nomes[i]] < primeira[nomes[j]]
	})
	if len(nomes) > max {
		nomes = nomes[:max]
	}
	return nomes
}

// comecoDeFrase: o que vem antes termina uma frase, um título, um item de lista ou o próprio texto.
func comecoDeFrase(antes string) bool {
	antes = strings.TrimRightFunc(antes, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '*' || r == '_' || r == '"' || r == '“' || r == '('
	})
	if antes == "" {
		return true
	}
	ultimo, _ := utf8.DecodeLastRuneInString(antes)
	return ultimo == '\n' || ultimo == '.' || ultimo == '!' || ultimo == '?' || ultimo == ':' || ultimo == '#' || ultimo == '-' || ultimo == '>' || ultimo == '|' || unicode.IsDigit(ultimo)
}
