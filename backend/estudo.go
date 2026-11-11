package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// Estudar (D41): a aba Estudar de um Tema junta o que ajuda a aprender o que está nele. A IA lê o
// Tema inteiro (com os subtemas) e prepara, cada parte numa tarefa da fila e com a sua regra:
//   - resumo: o resumo inteligente (regras/estudar-resumo.md);
//   - anteriores: os conceitos anteriores, o que ajuda a entender o Tema e não está nele
//     (regras/estudar-anteriores.md, com a internet para as fontes);
//   - simulado: um banco de perguntas abertas (regras/estudar-simulado.md). O app sorteia algumas,
//     o Lucas responde digitando ou falando, e a IA corrige (regras/corrigir-simulado.md).
// Os flashcards saem do Tema (D28), dos conceitos anteriores ou do que se errou num simulado.
// Cada parte guarda a versão do Tema de onde saiu: se o texto mudou, a aba avisa.

var partesDoEstudo = map[string]string{
	"resumo":     "estudar-resumo.md",
	"anteriores": "estudar-anteriores.md",
	"simulado":   "estudar-simulado.md",
}

const (
	perguntasPorSimulado = 5
	maxResposta          = 3000
)

type estudo struct {
	Parte         string          `json:"parte"`
	Estado        string          `json:"estado"` // vazio, processando, pronto ou falhou
	Erro          string          `json:"erro,omitempty"`
	Resultado     json.RawMessage `json:"resultado"`
	Desatualizado bool            `json:"desatualizado"`
	AtualizadoEm  string          `json:"atualizado_em,omitempty"`
}

type resumoDoTema struct {
	Texto string `json:"texto"`
}

type conceitoAnterior struct {
	Nome       string       `json:"nome"`
	Tipo       string       `json:"tipo"` // base ou vizinho
	PorQue     string       `json:"por_que"`
	Explicacao string       `json:"explicacao"`
	Tema       string       `json:"tema"` // a pasta de um Tema do Acervo que trata dele, se houver
	Fontes     []consultada `json:"fontes"`
}

type perguntaDoSimulado struct {
	Pergunta string `json:"pergunta"`
	Esperada string `json:"esperada"`
	Capitulo string `json:"capitulo"`
	Nivel    string `json:"nivel"` // lembrar, entender, aplicar
}

const (
	esquemaResumo = `{"type":"object","properties":{"texto":{"type":"string"}},"required":["texto"]}`

	esquemaAnteriores = `{"type":"object","properties":{"conceitos":{"type":"array","items":{"type":"object","properties":{` +
		`"nome":{"type":"string"},"tipo":{"type":"string","enum":["base","vizinho"]},"por_que":{"type":"string"},` +
		`"explicacao":{"type":"string"},"tema":{"type":"string"},` +
		`"fontes":{"type":"array","items":{"type":"object","properties":{"titulo":{"type":"string"},"url":{"type":"string"}},"required":["titulo","url"]}}},` +
		`"required":["nome","tipo","por_que","explicacao","tema","fontes"]}}},"required":["conceitos"]}`

	esquemaSimulado = `{"type":"object","properties":{"perguntas":{"type":"array","items":{"type":"object","properties":{` +
		`"pergunta":{"type":"string"},"esperada":{"type":"string"},"capitulo":{"type":"string"},` +
		`"nivel":{"type":"string","enum":["lembrar","entender","aplicar"]}},` +
		`"required":["pergunta","esperada","capitulo","nivel"]}}},"required":["perguntas"]}`

	esquemaCorrecao = `{"type":"object","properties":{"nota":{"type":"integer","minimum":0,"maximum":100},"comentario":{"type":"string"},` +
		`"itens":{"type":"array","items":{"type":"object","properties":{"n":{"type":"integer"},` +
		`"avaliacao":{"type":"string","enum":["certa","parcial","errada","em_branco"]},"comentario":{"type":"string"},"faltou":{"type":"string"}},` +
		`"required":["n","avaliacao","comentario","faltou"]}},"revisar":{"type":"array","items":{"type":"string"}}},` +
		`"required":["nota","comentario","itens","revisar"]}`
)

// ---------- O texto do Tema ----------

// O estudo é de um Tema (temas/<pasta>, com os subtemas) ou de uma Referência (referencias/<id>:
// o texto organizado, em português quando houver tradução). O campo e a coluna continuam "tema".

func referenciaDoEstudo(local string) (string, bool) {
	return strings.CutPrefix(local, "referencias/")
}

// textoDaReferenciaParaEstudo: o arquivo com o texto que a Referência estuda.
func (a *app) textoDaReferenciaParaEstudo(ctx context.Context, id string) (*referencia, string, error) {
	ref, err := a.referencia(ctx, id)
	if err != nil {
		return nil, "", errNaoEncontrado
	}
	texto, err := a.textoParaAcervo(ref)
	if err != nil {
		texto = filepath.Join(a.pastaReferencia(id), "organizado.md")
		if exigirArquivo(texto) != nil {
			return ref, "", errors.New("a Referência ainda não tem o texto organizado")
		}
	}
	return ref, texto, nil
}

// temaDoEstudo confere o Tema (a pasta com indice.md) ou a Referência e devolve o título.
func (a *app) temaDoEstudo(ctx context.Context, tema string) (string, error) {
	if !localDePergunta.MatchString(tema) {
		return "", errCaminhoInvalido
	}
	if id, ok := referenciaDoEstudo(tema); ok {
		ref, _, err := a.textoDaReferenciaParaEstudo(ctx, id)
		if err != nil {
			return "", errNaoEncontrado
		}
		if ref.Titulo == "" {
			return ref.URL, nil
		}
		return ref.Titulo, nil
	}
	if _, err := caminhoDeTema(tema); err != nil {
		return "", errCaminhoInvalido
	}
	b, err := os.ReadFile(filepath.Join(a.conteudo.raiz, filepath.FromSlash(tema), "indice.md"))
	if err != nil {
		return "", errNaoEncontrado
	}
	return tituloDe(string(b), tema), nil
}

// arquivosDoTema: os .md do Tema e dos subtemas, em ordem; numa Referência, o texto dela, que na
// pasta da tarefa vira indice.md. Devolve a pasta de onde os caminhos partem.
func (a *app) arquivosDoTema(ctx context.Context, tema string) (map[string]string, []string, error) {
	if id, ok := referenciaDoEstudo(tema); ok {
		_, texto, err := a.textoDaReferenciaParaEstudo(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		return map[string]string{"indice.md": texto}, []string{"indice.md"}, nil
	}
	raiz := filepath.Join(a.conteudo.raiz, filepath.FromSlash(tema))
	origem := map[string]string{}
	var lista []string
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".md") {
			rel, _ := filepath.Rel(raiz, p)
			lista = append(lista, rel)
			origem[rel] = p
		}
		return nil
	})
	sort.Strings(lista)
	return origem, lista, err
}

// versaoDoTema muda quando qualquer arquivo do Tema (ou o texto da Referência) muda.
func (a *app) versaoDoTema(ctx context.Context, tema string) string {
	origem, arquivos, err := a.arquivosDoTema(ctx, tema)
	if err != nil {
		return ""
	}
	h := sha256.New()
	for _, rel := range arquivos {
		b, _ := os.ReadFile(origem[rel])
		fmt.Fprintf(h, "%s\x00%d\x00", rel, len(b))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// copiarTema põe na pasta da tarefa os arquivos do Tema, com os subtemas nas suas pastas.
func (a *app) copiarTema(ctx context.Context, tema, destino string) error {
	origem, arquivos, err := a.arquivosDoTema(ctx, tema)
	if err != nil || len(arquivos) == 0 {
		if _, ok := referenciaDoEstudo(tema); ok {
			return errors.New("a Referência não tem mais o texto")
		}
		return errors.New("o Tema não existe mais")
	}
	for _, rel := range arquivos {
		if err := os.MkdirAll(filepath.Join(destino, filepath.Dir(rel)), 0o755); err != nil {
			return err
		}
		if err := copiarArquivo(origem[rel], filepath.Join(destino, rel)); err != nil {
			return err
		}
	}
	return nil
}

func naoEncontrado(tema string) string {
	if _, ok := referenciaDoEstudo(tema); ok {
		return "Referência sem texto para estudar."
	}
	return "Tema não encontrado."
}

// linkDoEstudo abre a aba Estudar do Tema ou da Referência.
func linkDoEstudo(tema string) string {
	if id, ok := referenciaDoEstudo(tema); ok {
		return "#/referencia/" + id + "?estudar"
	}
	return "#/ler/" + tema + "/indice.md?estudar"
}

// ---------- As partes ----------

func estadoPelaTarefa(temResultado bool, estadoTarefa, erroTarefa string) (string, string) {
	switch {
	case estadoTarefa == naFila || estadoTarefa == rodando:
		return "processando", ""
	case temResultado:
		return "pronto", ""
	case estadoTarefa == "":
		return "vazio", ""
	case estadoTarefa == cancelada:
		return "falhou", "Cancelado na fila."
	case erroTarefa == "":
		return "falhou", "A tarefa terminou sem resultado."
	}
	return "falhou", erroTarefa
}

func (a *app) estudosDoTema(ctx context.Context, tema string) (map[string]estudo, error) {
	versao := a.versaoDoTema(ctx, tema)
	saida := map[string]estudo{}
	for parte := range partesDoEstudo {
		saida[parte] = estudo{Parte: parte, Estado: "vazio", Resultado: json.RawMessage("null")}
	}
	rows, err := a.db.QueryContext(ctx, `SELECT e.parte, e.resultado, e.versao, e.atualizado_em, COALESCE(t.estado, ''), COALESCE(t.erro, '')
		FROM estudos e LEFT JOIN tarefas t ON t.id = e.tarefa_id WHERE e.tema = ?`, tema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e estudo
		var resultado, versaoDele, estadoTarefa, erroTarefa string
		if err := rows.Scan(&e.Parte, &resultado, &versaoDele, &e.AtualizadoEm, &estadoTarefa, &erroTarefa); err != nil {
			return nil, err
		}
		e.Estado, e.Erro = estadoPelaTarefa(resultado != "", estadoTarefa, erroTarefa)
		e.Resultado = json.RawMessage("null")
		if resultado != "" {
			e.Resultado = json.RawMessage(resultado)
			e.Desatualizado = versaoDele != versao
		}
		saida[e.Parte] = e
	}
	return saida, rows.Err()
}

// handleEstudos: GET /api/estudos?tema=temas/<pasta> — as partes e os últimos simulados.
func (a *app) handleEstudos(w http.ResponseWriter, r *http.Request) {
	tema := strings.TrimSuffix(r.URL.Query().Get("tema"), "/")
	if _, err := a.temaDoEstudo(r.Context(), tema); err != nil {
		erroJSON(w, http.StatusNotFound, naoEncontrado(tema))
		return
	}
	partes, err := a.estudosDoTema(r.Context(), tema)
	if err != nil {
		falha(w, err)
		return
	}
	simulados, err := a.simuladosDoTema(r.Context(), tema, 10)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"partes": partes, "simulados": simulados})
}

type paramsEstudo struct {
	Tema  string `json:"tema"`
	Parte string `json:"parte"`
}

// handlePrepararEstudo: POST /api/estudos {tema, parte} — prepara (ou refaz) uma parte na fila.
func (a *app) handlePrepararEstudo(w http.ResponseWriter, r *http.Request) {
	var p paramsEstudo
	if !lerCorpo(w, r, &p) {
		return
	}
	p.Tema = strings.TrimSuffix(p.Tema, "/")
	if _, ok := partesDoEstudo[p.Parte]; !ok {
		erroJSON(w, http.StatusBadRequest, "Parte inválida.")
		return
	}
	if _, err := a.temaDoEstudo(r.Context(), p.Tema); err != nil {
		erroJSON(w, http.StatusNotFound, naoEncontrado(p.Tema))
		return
	}
	ctx := r.Context()
	if partes, err := a.estudosDoTema(ctx, p.Tema); err == nil && partes[p.Parte].Estado == "processando" {
		erroJSON(w, http.StatusConflict, "Esta parte já está na fila.")
		return
	}
	tarefa, err := a.fila.enfileirar(ctx, "", "estudar", p)
	if err != nil {
		falha(w, err)
		return
	}
	// O resultado anterior fica até o novo chegar.
	if _, err := a.db.ExecContext(ctx, `INSERT INTO estudos (tema, parte, tarefa_id, atualizado_em) VALUES (?, ?, ?, ?)
		ON CONFLICT (tema, parte) DO UPDATE SET tarefa_id = excluded.tarefa_id`, p.Tema, p.Parte, tarefa, agora()); err != nil {
		falha(w, err)
		return
	}
	partes, err := a.estudosDoTema(ctx, p.Tema)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, partes[p.Parte])
}

// estudar é a tarefa da fila: prepara uma parte a partir da cópia do Tema.
func (a *app) estudar(ctx context.Context, t *tarefa) (resultado, error) {
	var p paramsEstudo
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	titulo, err := a.temaDoEstudo(ctx, p.Tema)
	if err != nil {
		return pronto, errors.New("o Tema não existe mais")
	}
	pasta, err := os.MkdirTemp(a.cfg.dados, "estudo-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(pasta)
	versao := a.versaoDoTema(ctx, p.Tema)
	if err := a.copiarTema(ctx, p.Tema, pasta); err != nil {
		return pronto, err
	}

	pedido := pedidoClaude{pasta: pasta, regra: partesDoEstudo[p.Parte], ferramentas: ferramentasLeitura}
	inicio := fmt.Sprintf("O Lucas está estudando o Tema %q. Os arquivos estão nesta pasta: indice.md é a página do Tema; "+
		"as pastas são os subtemas, e os outros .md, capítulos com página própria.", titulo)
	if _, ok := referenciaDoEstudo(p.Tema); ok {
		inicio = fmt.Sprintf("O Lucas está estudando a Referência %q (um conteúdo que ainda não entrou, ou não só, num Tema do Acervo). "+
			"O texto dela está em indice.md, nesta pasta: nas regras, onde se lê Tema, entenda este texto; os capítulos são as suas seções (## ).", titulo)
	}
	switch p.Parte {
	case "resumo":
		pedido.instrucao = inicio + " Escreva o resumo inteligente do Tema inteiro."
		pedido.esquema = esquemaResumo
	case "anteriores":
		temas, _ := a.conteudo.listarTemas()
		var lista strings.Builder
		for _, tm := range temas {
			if tm.Caminho != p.Tema {
				fmt.Fprintf(&lista, "- %s — %s\n", tm.Caminho, tm.Titulo)
			}
		}
		if lista.Len() == 0 {
			lista.WriteString("(nenhum outro)\n")
		}
		pedido.instrucao = inicio + " Diga os conceitos anteriores: o que ajuda a entender este Tema e não está explicado nele. " +
			"Os outros Temas do Acervo (pasta — título):\n\n" + lista.String()
		pedido.esquema = esquemaAnteriores
		pedido.ferramentas = ferramentasLeitura + "," + ferramentasInternet
	case "simulado":
		pedido.instrucao = inicio + " Monte o banco de perguntas do simulado sobre o Tema inteiro."
		pedido.esquema = esquemaSimulado
	default:
		return pronto, fmt.Errorf("parte desconhecida: %s", p.Parte)
	}
	r, err := a.chamarClaude(ctx, pedido)
	if err != nil {
		return pronto, err
	}
	var saida any
	switch p.Parte {
	case "resumo":
		var s resumoDoTema
		if err := dadosDaResposta(r, &s); err != nil {
			return pronto, err
		}
		if strings.TrimSpace(s.Texto) == "" {
			return pronto, errors.New("o claude não devolveu o resumo")
		}
		saida = s
	case "anteriores":
		var s struct {
			Conceitos []conceitoAnterior `json:"conceitos"`
		}
		if err := dadosDaResposta(r, &s); err != nil {
			return pronto, err
		}
		for i := range s.Conceitos {
			if s.Conceitos[i].Fontes == nil {
				s.Conceitos[i].Fontes = []consultada{}
			}
			// Só vale a pasta de um Tema que existe.
			if tm := s.Conceitos[i].Tema; tm != "" {
				if _, err := a.temaDoEstudo(ctx, tm); err != nil || tm == p.Tema || !strings.HasPrefix(tm, "temas/") {
					s.Conceitos[i].Tema = ""
				}
			}
		}
		if s.Conceitos == nil {
			s.Conceitos = []conceitoAnterior{}
		}
		saida = s
	case "simulado":
		var s struct {
			Perguntas []perguntaDoSimulado `json:"perguntas"`
		}
		if err := dadosDaResposta(r, &s); err != nil {
			return pronto, err
		}
		if len(s.Perguntas) == 0 {
			return pronto, errors.New("o claude não devolveu perguntas")
		}
		saida = s
	}
	b, _ := json.Marshal(saida)
	_, err = a.db.ExecContext(ctx, `UPDATE estudos SET resultado = ?, versao = ?, atualizado_em = ? WHERE tema = ? AND parte = ? AND tarefa_id = ?`,
		string(b), versao, agora(), p.Tema, p.Parte, t.ID)
	return pronto, err
}

// ---------- O simulado ----------

type respostaDoSimulado struct {
	Pergunta string `json:"pergunta"`
	Esperada string `json:"esperada"`
	Capitulo string `json:"capitulo"`
	Resposta string `json:"resposta"`
	PorVoz   bool   `json:"por_voz"`
}

type correcaoDoSimulado struct {
	Nota       int    `json:"nota"`
	Comentario string `json:"comentario"`
	Itens      []struct {
		N          int    `json:"n"`
		Avaliacao  string `json:"avaliacao"`
		Comentario string `json:"comentario"`
		Faltou     string `json:"faltou"`
	} `json:"itens"`
	Revisar []string `json:"revisar"`
}

type simuladoFeito struct {
	ID        string               `json:"id"`
	Tema      string               `json:"tema"`
	Respostas []respostaDoSimulado `json:"respostas"`
	Estado    string               `json:"estado"`
	Erro      string               `json:"erro,omitempty"`
	Correcao  *correcaoDoSimulado  `json:"correcao"`
	TarefaID  int64                `json:"tarefa_id"`
	CriadoEm  string               `json:"criado_em"`
}

const consultaSimulado = `SELECT s.id, s.tema, s.respostas, s.correcao, s.tarefa_id, s.criado_em, COALESCE(t.estado, ''), COALESCE(t.erro, '')
	FROM simulados s LEFT JOIN tarefas t ON t.id = s.tarefa_id`

func lerSimulado(row linha) (simuladoFeito, error) {
	var s simuladoFeito
	var respostas, correcao, estadoTarefa, erroTarefa string
	if err := row.Scan(&s.ID, &s.Tema, &respostas, &correcao, &s.TarefaID, &s.CriadoEm, &estadoTarefa, &erroTarefa); err != nil {
		return s, err
	}
	json.Unmarshal([]byte(respostas), &s.Respostas)
	if correcao != "" {
		s.Correcao = &correcaoDoSimulado{}
		json.Unmarshal([]byte(correcao), s.Correcao)
	}
	s.Estado, s.Erro = estadoPelaTarefa(s.Correcao != nil, estadoTarefa, erroTarefa)
	return s, nil
}

func (a *app) simulado(ctx context.Context, id string) (simuladoFeito, error) {
	return lerSimulado(a.db.QueryRowContext(ctx, consultaSimulado+` WHERE s.id = ?`, id))
}

func (a *app) simuladosDoTema(ctx context.Context, tema string, max int) ([]simuladoFeito, error) {
	rows, err := a.db.QueryContext(ctx, consultaSimulado+` WHERE s.tema = ? ORDER BY s.criado_em DESC, s.rowid DESC LIMIT ?`, tema, max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []simuladoFeito{}
	for rows.Next() {
		s, err := lerSimulado(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

// handleEntregarSimulado: POST /api/simulados {tema, respostas} — a correção vai para a fila.
func (a *app) handleEntregarSimulado(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Tema      string               `json:"tema"`
		Respostas []respostaDoSimulado `json:"respostas"`
	}
	if !lerCorpo(w, r, &corpo) {
		return
	}
	corpo.Tema = strings.TrimSuffix(corpo.Tema, "/")
	if _, err := a.temaDoEstudo(r.Context(), corpo.Tema); err != nil {
		erroJSON(w, http.StatusNotFound, naoEncontrado(corpo.Tema))
		return
	}
	if len(corpo.Respostas) == 0 || len(corpo.Respostas) > 20 {
		erroJSON(w, http.StatusBadRequest, "Simulado sem perguntas.")
		return
	}
	respondidas := 0
	for i, x := range corpo.Respostas {
		if strings.TrimSpace(x.Pergunta) == "" {
			erroJSON(w, http.StatusBadRequest, "Pergunta vazia.")
			return
		}
		if utf8.RuneCountInString(x.Resposta) > maxResposta {
			erroJSON(w, http.StatusBadRequest, fmt.Sprintf("A resposta %d passou de %d caracteres.", i+1, maxResposta))
			return
		}
		if strings.TrimSpace(x.Resposta) != "" {
			respondidas++
		}
	}
	if respondidas == 0 {
		erroJSON(w, http.StatusBadRequest, "Responda pelo menos uma pergunta.")
		return
	}
	ctx := r.Context()
	id := novoID()
	respostas, _ := json.Marshal(corpo.Respostas)
	if _, err := a.db.ExecContext(ctx, `INSERT INTO simulados (id, tema, respostas, criado_em) VALUES (?, ?, ?, ?)`,
		id, corpo.Tema, string(respostas), agora()); err != nil {
		falha(w, err)
		return
	}
	tarefa, err := a.fila.enfileirar(ctx, "", "corrigir_simulado", map[string]string{"simulado": id})
	if err != nil {
		falha(w, err)
		return
	}
	a.db.ExecContext(ctx, `UPDATE simulados SET tarefa_id = ? WHERE id = ?`, tarefa, id)
	s, err := a.simulado(ctx, id)
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// handleSimulado: GET /api/simulados/{id}.
func (a *app) handleSimulado(w http.ResponseWriter, r *http.Request) {
	s, err := a.simulado(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		erroJSON(w, http.StatusNotFound, "Simulado não encontrado.")
		return
	}
	if err != nil {
		falha(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// corrigirSimulado é a tarefa da fila: a IA compara cada resposta com o Tema e a resposta esperada.
func (a *app) corrigirSimulado(ctx context.Context, t *tarefa) (resultado, error) {
	var p struct {
		Simulado string `json:"simulado"`
	}
	if err := json.Unmarshal(t.Params, &p); err != nil {
		return pronto, err
	}
	s, err := a.simulado(ctx, p.Simulado)
	if errors.Is(err, sql.ErrNoRows) {
		t.Erro = "O simulado foi apagado."
		return pronto, nil
	}
	if err != nil {
		return pronto, err
	}
	titulo, err := a.temaDoEstudo(ctx, s.Tema)
	if err != nil {
		return pronto, errors.New("o Tema não existe mais")
	}
	pasta, err := os.MkdirTemp(a.cfg.dados, "simulado-")
	if err != nil {
		return pronto, err
	}
	defer os.RemoveAll(pasta)
	if err := a.copiarTema(ctx, s.Tema, filepath.Join(pasta, "tema")); err != nil {
		return pronto, err
	}
	var md strings.Builder
	fmt.Fprintf(&md, "# Simulado: %s\n\n", titulo)
	for i, x := range s.Respostas {
		fmt.Fprintf(&md, "## Pergunta %d\n\n%s\n\n", i+1, x.Pergunta)
		if x.Capitulo != "" {
			fmt.Fprintf(&md, "Capítulo: %s\n\n", x.Capitulo)
		}
		fmt.Fprintf(&md, "### Resposta esperada\n\n%s\n\n### Resposta do Lucas", x.Esperada)
		if x.PorVoz {
			md.WriteString(" (falada; transcrita pelo navegador)")
		}
		resposta := strings.TrimSpace(x.Resposta)
		if resposta == "" {
			resposta = "(em branco)"
		}
		fmt.Fprintf(&md, "\n\n%s\n\n", resposta)
	}
	if err := os.WriteFile(filepath.Join(pasta, "simulado.md"), []byte(md.String()), 0o644); err != nil {
		return pronto, err
	}
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta: pasta,
		regra: "corrigir-simulado.md",
		instrucao: fmt.Sprintf("Corrija o simulado em simulado.md, nesta pasta, sobre o Tema %q (o texto do Tema está na pasta tema/). "+
			"Há %d perguntas.", titulo, len(s.Respostas)),
		ferramentas: ferramentasLeitura,
		esquema:     esquemaCorrecao,
	})
	if err != nil {
		return pronto, err
	}
	var c correcaoDoSimulado
	if err := dadosDaResposta(r, &c); err != nil {
		return pronto, err
	}
	if c.Revisar == nil {
		c.Revisar = []string{}
	}
	b, _ := json.Marshal(c)
	_, err = a.db.ExecContext(ctx, `UPDATE simulados SET correcao = ? WHERE id = ? AND tarefa_id = ?`, string(b), s.ID, t.ID)
	return pronto, err
}

// ---------- Flashcards do estudo ----------

// textoDoEstudo: os conceitos anteriores, ou o que se errou num simulado, como texto para o Flashcards.
func (a *app) textoDoEstudo(ctx context.Context, tema, parte string) (string, textoParaCartoes, error) {
	var t textoParaCartoes
	titulo, err := a.temaDoEstudo(ctx, tema)
	if err != nil {
		return "", t, errNaoEncontrado
	}
	t.Baralho = titulo
	t.Link = a.cfg.acervoLink + "/" + linkDoEstudo(tema)
	// Numa Referência, o baralho é o mesmo dos cartões do texto dela (o do Tema onde entrou, se entrou).
	if id, ok := referenciaDoEstudo(tema); ok {
		if ref, err := a.referencia(ctx, id); err == nil {
			if doTexto, err := a.textoDaReferencia(ctx, ref); err == nil {
				t.Baralho = doTexto.Baralho
			}
		}
	}
	var md strings.Builder
	if id, ok := strings.CutPrefix(parte, "simulado:"); ok {
		s, err := a.simulado(ctx, id)
		if err != nil || s.Tema != tema {
			return "", t, errNaoEncontrado
		}
		if s.Correcao == nil {
			return "", t, errors.New("O simulado ainda não foi corrigido.")
		}
		for _, it := range s.Correcao.Itens {
			if it.Avaliacao == "certa" || it.N < 1 || it.N > len(s.Respostas) {
				continue
			}
			x := s.Respostas[it.N-1]
			fmt.Fprintf(&md, "## %s\n\n%s\n\n", x.Pergunta, x.Esperada)
			if it.Faltou != "" {
				fmt.Fprintf(&md, "O que faltou na resposta: %s\n\n", it.Faltou)
			}
		}
		if md.Len() == 0 {
			return "", t, errors.New("Nenhuma resposta errada neste simulado: não há o que virar cartão.")
		}
		t.Titulo = titulo + " › O que errei no simulado"
		t.Contexto = fmt.Sprintf("São as perguntas que o Lucas errou ou acertou em parte num simulado do Tema %q, com a resposta esperada. Faça cartões que fixem exatamente o que faltou.", titulo)
		t.Texto = md.String()
		return "estudo:" + tema + "#" + parte, t, nil
	}
	// "anteriores" são todos os conceitos; "anteriores:<nome>", um só (o botão de cada conceito).
	so, umSo := strings.CutPrefix(parte, "anteriores:")
	if parte != "anteriores" && !umSo {
		return "", t, errors.New("Parte inválida.")
	}
	partes, err := a.estudosDoTema(ctx, tema)
	if err != nil {
		return "", t, err
	}
	var s struct {
		Conceitos []conceitoAnterior `json:"conceitos"`
	}
	if partes["anteriores"].Estado != "pronto" || json.Unmarshal(partes["anteriores"].Resultado, &s) != nil || len(s.Conceitos) == 0 {
		return "", t, errors.New("Os conceitos anteriores ainda não foram preparados.")
	}
	if umSo {
		for _, c := range s.Conceitos {
			if c.Nome == so {
				fmt.Fprintf(&md, "# %s\n\n%s\n\nPor que ajuda a entender %s: %s\n", c.Nome, c.Explicacao, titulo, c.PorQue)
			}
		}
		if md.Len() == 0 {
			return "", t, errNaoEncontrado
		}
		t.Titulo = titulo + " › " + so
		t.Contexto = fmt.Sprintf("É um conceito que ajuda a entender o Tema %q e não está explicado nele.", titulo)
		t.Texto = md.String()
		return "estudo:" + tema + "#" + parte, t, nil
	}
	fmt.Fprintf(&md, "# Conceitos anteriores: %s\n\n", titulo)
	for _, c := range s.Conceitos {
		fmt.Fprintf(&md, "## %s\n\n%s\n\nPor que ajuda: %s\n\n", c.Nome, c.Explicacao, c.PorQue)
	}
	t.Titulo = titulo + " › Conceitos anteriores"
	t.Contexto = fmt.Sprintf("São os conceitos que ajudam a entender o Tema %q e não estão explicados nele.", titulo)
	t.Texto = md.String()
	return "estudo:" + tema + "#anteriores", t, nil
}

// tarefaDeEstudo: na fila, o título do Tema e a parte; o link abre a aba Estudar.
func (a *app) tarefaDeEstudo(r *http.Request, t tarefa) tarefaComTitulo {
	var p struct {
		Tema     string `json:"tema"`
		Parte    string `json:"parte"`
		Simulado string `json:"simulado"`
	}
	json.Unmarshal(t.Params, &p)
	if p.Simulado != "" {
		if s, err := a.simulado(r.Context(), p.Simulado); err == nil {
			p.Tema = s.Tema
		}
	}
	titulo, err := a.temaDoEstudo(r.Context(), p.Tema)
	if err != nil {
		if _, ok := referenciaDoEstudo(p.Tema); ok {
			return tarefaComTitulo{t, "(Referência apagada)", ""}
		}
		return tarefaComTitulo{t, "(Tema apagado)", ""}
	}
	nomes := map[string]string{"resumo": "resumo", "anteriores": "conceitos anteriores", "simulado": "perguntas do simulado"}
	if n := nomes[p.Parte]; n != "" {
		titulo += " · " + n
	}
	return tarefaComTitulo{t, titulo, linkDoEstudo(p.Tema)}
}
