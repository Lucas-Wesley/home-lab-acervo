package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Pesquisador (D27): o Lucas pede um tema, escolhe as fontes e a profundidade, e a IA produz um
// texto com fontes, que vira uma Referência do tipo "pesquisa" e segue o caminho de sempre
// (sugerir → criar ou integrar → revisão pelo diff). As etapas são tarefas da fila:
//
//	planejar (esboço: as perguntas e as palavras de busca de cada fonte)
//	→ [o Lucas aprova o esboço, se pediu] → coletar (livros, Wikipédia, YouTube; sem IA)
//	→ redigir (a IA escreve; com "Sites", ela mesma pesquisa na internet)
//	→ conferir (só na detalhada: cada afirmação contra o material guardado) → sugerir.
//
// Sem nenhuma fonte, é "só a resposta da IA": vai direto para redigir, sem internet.
//
// Aprofundar (D29) começa outra rodada na mesma Referência, com as fontes e a profundidade que o
// Lucas escolher: o texto atual vira anterior.md, o material das rodadas anteriores fica em
// fontes/, e o esboço novo procura o que falta. Em toda rodada, o que o Acervo já tem sobre o
// assunto (os Temas relacionados) é copiado para ja-escrito/, para o texto dizer o que há de novo.

const (
	fonteLivros    = "livros"
	fonteWikipedia = "wikipedia"
	fonteYoutube   = "youtube"
	fonteSites     = "sites"

	profRapida    = "rapida"
	profDetalhada = "detalhada"

	pastaFontes     = "fontes"
	pastaJaEscrito  = "ja-escrito"
	maxTema         = 500
	maxRelacionados = 3
)

var fontesConhecidas = []string{fonteLivros, fonteWikipedia, fonteYoutube, fonteSites}

// alcance: quanto cada profundidade pesquisa e quanto tempo a IA tem para escrever.
type alcance struct {
	minPerguntas, maxPerguntas int
	trechos, artigos, videos   int // por pergunta
	maxVideos                  int // na pesquisa toda
	tempo                      time.Duration
}

var alcances = map[string]alcance{
	profRapida:    {minPerguntas: 1, maxPerguntas: 3, trechos: 5, artigos: 1, videos: 1, maxVideos: 3, tempo: 20 * time.Minute},
	profDetalhada: {minPerguntas: 4, maxPerguntas: 8, trechos: 8, artigos: 2, videos: 2, maxVideos: 8, tempo: 50 * time.Minute},
}

type pedidoPesquisa struct {
	Tema          string           `json:"tema"`
	Fontes        []string         `json:"fontes"`
	Profundidade  string           `json:"profundidade"`
	RevisarEsboco bool             `json:"revisar_esboco"`
	Flashcards    bool             `json:"flashcards"` // pronta a pesquisa, o texto vai para o Flashcards (D28)
	Baralho       escolhaDeBaralho `json:"baralho"`    // para qual baralho (vazio: um novo, com o título da pesquisa)
	Esboco        []pergunta       `json:"esboco"`
	Coleta        []coletado       `json:"coleta"`
	Rodada        int              `json:"rodada"`       // 1 na primeira; cada "aprofundar" soma uma (0 = pesquisa antiga, a 1ª)
	Foco          string           `json:"foco"`         // o que o Lucas quer a mais, ao aprofundar
	Anteriores    []pergunta       `json:"anteriores"`   // as perguntas das rodadas anteriores
	Relacionados  []string         `json:"relacionados"` // Temas do Acervo sobre o assunto
}

// pergunta do esboço, com as palavras de busca de cada fonte (vazias quando a fonte não foi escolhida).
type pergunta struct {
	Pergunta  string `json:"pergunta"`
	Livros    string `json:"livros"`
	Wikipedia string `json:"wikipedia"`
	Youtube   string `json:"youtube"`
}

// coletado é uma fonte guardada em fontes/, ou o motivo de não ter vindo nada dela.
type coletado struct {
	Fonte   string `json:"fonte"`
	Titulo  string `json:"titulo"`
	URL     string `json:"url,omitempty"`
	Arquivo string `json:"arquivo,omitempty"`
	Erro    string `json:"erro,omitempty"`
	Rodada  int    `json:"rodada,omitempty"`
}

func (c coletado) rodada() int { return max(c.Rodada, 1) }

func (p pedidoPesquisa) rodada() int { return max(p.Rodada, 1) }

func (p pedidoPesquisa) usa(fonte string) bool { return slices.Contains(p.Fontes, fonte) }

func (p pedidoPesquisa) alcance() alcance { return alcances[p.Profundidade] }

func (a *app) pedidoDe(ref *referencia) (pedidoPesquisa, error) {
	var p pedidoPesquisa
	if ref.Tipo != "pesquisa" {
		return p, errors.New("esta Referência não é uma pesquisa")
	}
	err := json.Unmarshal(ref.Pesquisa, &p)
	return p, err
}

func (a *app) guardarPedido(ctx context.Context, id string, p pedidoPesquisa) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = a.db.ExecContext(ctx, `UPDATE referencias SET pesquisa = ?, atualizado_em = ? WHERE id = ?`, string(b), agora(), id)
	return err
}

// handleCriarPesquisa: POST /api/pesquisas {tema, fontes, profundidade, revisar_esboco}.
func (a *app) handleCriarPesquisa(w http.ResponseWriter, r *http.Request) {
	var p pedidoPesquisa
	if !lerCorpo(w, r, &p) {
		return
	}
	p.Tema = strings.TrimSpace(p.Tema)
	if utf8.RuneCountInString(p.Tema) < 3 || utf8.RuneCountInString(p.Tema) > maxTema {
		erroJSON(w, http.StatusBadRequest, fmt.Sprintf("Escreva o tema com 3 a %d caracteres.", maxTema))
		return
	}
	if msg := a.validarEscolhas(&p); msg != "" {
		erroJSON(w, http.StatusBadRequest, msg)
		return
	}
	if p.Flashcards && a.cfg.flashcards == "" {
		erroJSON(w, http.StatusBadRequest, "O Flashcards não está configurado neste servidor (FLASHCARDS_URL).")
		return
	}
	p.Esboco, p.Coleta, p.Anteriores, p.Relacionados = []pergunta{}, []coletado{}, []pergunta{}, []string{}
	p.Rodada, p.Foco = 1, ""

	dados, _ := json.Marshal(p)
	id := novoID()
	_, err := a.db.ExecContext(r.Context(),
		`INSERT INTO referencias (id, url, titulo, idioma, estado, tipo, pesquisa, criado_em, atualizado_em)
		 VALUES (?, '', ?, 'pt', 'novo', 'pesquisa', ?, ?, ?)`,
		id, cortar(p.Tema, 120, false), string(dados), agora(), agora())
	if err != nil {
		falha(w, err)
		return
	}
	if _, err := a.fila.enfileirar(r.Context(), id, a.primeiraEtapa(p), nil); err != nil {
		falha(w, err)
		return
	}
	ref, _ := a.referencia(r.Context(), id)
	a.completar(r.Context(), []*referencia{ref})
	writeJSON(w, http.StatusCreated, ref)
}

// validarEscolhas confere as fontes (na ordem conhecida) e a profundidade; devolve a mensagem do erro.
func (a *app) validarEscolhas(p *pedidoPesquisa) string {
	if _, ok := alcances[p.Profundidade]; !ok {
		return "Escolha a profundidade: rápida ou detalhada."
	}
	var fontes []string
	for _, f := range fontesConhecidas {
		if slices.Contains(p.Fontes, f) {
			fontes = append(fontes, f)
		}
	}
	if len(fontes) != len(p.Fontes) {
		return "Fonte desconhecida."
	}
	p.Fontes = fontes
	if p.usa(fonteLivros) && a.cfg.biblioteca == "" {
		return "A busca nos livros não está configurada neste servidor (BIBLIOTECA_URL)."
	}
	// Sem fonte, não há o que buscar: o esboço não teria uso.
	p.RevisarEsboco = p.RevisarEsboco && len(p.Fontes) > 0
	return ""
}

// aprofundar começa outra rodada da pesquisa (D29). Devolve a mensagem quando não dá.
func (a *app) aprofundar(ctx context.Context, ref *referencia, escolhas pedidoPesquisa) (string, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return err.Error(), nil
	}
	pasta := a.pastaReferencia(ref.ID)
	if exigirArquivo(filepath.Join(pasta, "organizado.md")) != nil {
		return "A pesquisa ainda não tem texto para aprofundar: use Tentar de novo.", nil
	}
	p.Fontes, p.Profundidade, p.RevisarEsboco = escolhas.Fontes, escolhas.Profundidade, escolhas.RevisarEsboco
	if msg := a.validarEscolhas(&p); msg != "" {
		return msg, nil
	}
	p.Foco = cortar(strings.TrimSpace(escolhas.Foco), maxTema, false)
	p.Anteriores = append(p.Anteriores, p.Esboco...)
	p.Esboco = []pergunta{}
	p.Rodada = p.rodada() + 1
	if err := os.Rename(filepath.Join(pasta, "organizado.md"), filepath.Join(pasta, "anterior.md")); err != nil {
		return "", err
	}
	os.Remove(filepath.Join(pasta, "cru.md"))
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return "", err
	}
	if _, err := a.db.ExecContext(ctx, `UPDATE referencias SET estado = 'novo', sugestao = '', atualizado_em = ? WHERE id = ?`, agora(), ref.ID); err != nil {
		return "", err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, a.primeiraEtapa(p), nil)
	return "", err
}

func (a *app) primeiraEtapa(p pedidoPesquisa) string {
	if len(p.Fontes) == 0 {
		return "redigir"
	}
	return "planejar"
}

// proximaEtapaPesquisa decide o que refazer depois de um erro, pelo que já existe.
func (a *app) proximaEtapaPesquisa(ref *referencia) string {
	p, err := a.pedidoDe(ref)
	pasta := a.pastaReferencia(ref.ID)
	switch {
	case err != nil || len(p.Fontes) == 0:
		return "redigir"
	case len(p.Esboco) == 0:
		return "planejar"
	case !existe(filepath.Join(pasta, "cru.md")):
		return "coletar"
	case !existe(filepath.Join(pasta, "organizado.md")):
		return "redigir"
	case p.Profundidade == profDetalhada:
		return "conferir"
	default:
		return "redigir"
	}
}

// ---------- planejar ----------

const esquemaEsboco = `{"type":"object","properties":{` +
	`"titulo":{"type":"string"},` +
	`"relacionados":{"type":"array","items":{"type":"string"}},` +
	`"perguntas":{"type":"array","items":{"type":"object","properties":{` +
	`"pergunta":{"type":"string"},"livros":{"type":"string"},"wikipedia":{"type":"string"},"youtube":{"type":"string"}},` +
	`"required":["pergunta","livros","wikipedia","youtube"]}}},` +
	`"required":["titulo","perguntas","relacionados"]}`

var nomesDasFontes = map[string]string{
	fonteLivros:    "os livros da biblioteca do Lucas (busca por palavras no texto dos livros)",
	fonteWikipedia: "a Wikipédia em português (busca de artigos)",
	fonteYoutube:   "vídeos do YouTube (busca de vídeos; a transcrição é lida)",
	fonteSites:     "sites da internet (a IA pesquisa na hora de escrever)",
}

func (p pedidoPesquisa) descreverFontes() string {
	var linhas []string
	for _, f := range p.Fontes {
		linhas = append(linhas, "- "+nomesDasFontes[f])
	}
	return strings.Join(linhas, "\n")
}

func (a *app) planejar(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return pronto, err
	}
	pasta := a.pastaReferencia(ref.ID)
	if err := os.MkdirAll(pasta, 0o755); err != nil {
		return pronto, err
	}
	al := p.alcance()
	instrucao := fmt.Sprintf("O Lucas quer uma pesquisa sobre: %q.\n\nProfundidade: %s, com %d a %d perguntas.\n"+
		"Fontes escolhidas:\n%s\n\n%s"+
		"Monte o esboço da pesquisa. Deixe vazias as palavras de busca das fontes que não foram escolhidas.",
		p.Tema, map[string]string{profRapida: "rápida", profDetalhada: "detalhada"}[p.Profundidade], al.minPerguntas, al.maxPerguntas,
		p.descreverFontes(), a.jaEscritoParaEsboco(ctx, ref, p))
	r, err := a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		extras:      []string{a.conteudo.raiz},
		regra:       "pesquisar-esboco.md",
		instrucao:   instrucao,
		ferramentas: ferramentasLeitura,
		esquema:     esquemaEsboco,
	})
	if err != nil {
		return pronto, err
	}
	var esboco struct {
		Titulo       string     `json:"titulo"`
		Perguntas    []pergunta `json:"perguntas"`
		Relacionados []string   `json:"relacionados"`
	}
	if err := dadosDaResposta(r, &esboco); err != nil {
		return pronto, err
	}
	p.Esboco = limparEsboco(esboco.Perguntas, p, al.maxPerguntas)
	if len(p.Esboco) == 0 {
		return pronto, errors.New("a IA não montou nenhuma pergunta para a pesquisa")
	}
	p.Relacionados = a.temasExistentes(append(a.temasDaReferencia(ctx, ref.ID), esboco.Relacionados...))
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return pronto, err
	}
	if titulo := strings.TrimSpace(esboco.Titulo); titulo != "" && p.rodada() == 1 {
		a.db.ExecContext(ctx, `UPDATE referencias SET titulo = ? WHERE id = ?`, cortar(titulo, 120, false), ref.ID)
	}
	if p.RevisarEsboco {
		return pronto, a.mudarEstado(ctx, ref.ID, "esboco")
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "coletar", nil)
	return pronto, err
}

// limparEsboco tira perguntas vazias e as buscas de fontes que não foram escolhidas.
func limparEsboco(lista []pergunta, p pedidoPesquisa, max int) []pergunta {
	limpo := []pergunta{}
	for _, q := range lista {
		q.Pergunta = strings.TrimSpace(q.Pergunta)
		if q.Pergunta == "" || len(limpo) == max {
			continue
		}
		busca := func(fonte, termos string) string {
			if !p.usa(fonte) {
				return ""
			}
			if termos = strings.TrimSpace(termos); termos != "" {
				return cortar(termos, 200, false)
			}
			return cortar(q.Pergunta, 200, false)
		}
		q.Livros = busca(fonteLivros, q.Livros)
		q.Wikipedia = busca(fonteWikipedia, q.Wikipedia)
		q.Youtube = busca(fonteYoutube, q.Youtube)
		limpo = append(limpo, q)
	}
	return limpo
}

// jaEscritoParaEsboco conta à IA o que já existe: a lista de Temas do Acervo e, ao aprofundar, o
// texto atual e as perguntas já respondidas.
func (a *app) jaEscritoParaEsboco(ctx context.Context, ref *referencia, p pedidoPesquisa) string {
	var s strings.Builder
	temas, _ := a.conteudo.listarTemas()
	s.WriteString("O que o Acervo já tem: os Temas (pasta — título: resumo) são:\n")
	for _, tm := range temas {
		fmt.Fprintf(&s, "- %s — %s: %s\n", tm.Caminho, tm.Titulo, tm.Resumo)
	}
	if len(temas) == 0 {
		s.WriteString("(o Acervo ainda não tem Temas)\n")
	}
	fmt.Fprintf(&s, "Os arquivos dos Temas estão em %s/temas/ e podem ser lidos.\n", a.conteudo.raiz)
	if ligados := a.temasDaReferencia(ctx, ref.ID); len(ligados) > 0 {
		fmt.Fprintf(&s, "Esta pesquisa já entrou nos Temas: %s.\n", strings.Join(ligados, ", "))
	}
	if p.rodada() > 1 {
		fmt.Fprintf(&s, "\nEsta é a rodada %d: o Lucas pediu para aprofundar a pesquisa. anterior.md, nesta pasta, é o texto atual. "+
			"Perguntas já respondidas nas rodadas anteriores:\n", p.rodada())
		for _, q := range p.Anteriores {
			fmt.Fprintf(&s, "- %s\n", q.Pergunta)
		}
		if p.Foco != "" {
			fmt.Fprintf(&s, "O que ele quer a mais: %q.\n", p.Foco)
		}
	}
	s.WriteString("\n")
	return s.String()
}

// temasDaReferencia: os Temas em que a Referência já entrou.
func (a *app) temasDaReferencia(ctx context.Context, id string) []string {
	var lista []string
	rows, err := a.db.QueryContext(ctx, `SELECT tema FROM referencia_temas WHERE referencia_id = ? ORDER BY integrado_em`, id)
	if err != nil {
		return lista
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if rows.Scan(&t) == nil {
			lista = append(lista, t)
		}
	}
	return lista
}

// temasExistentes fica com as pastas de Tema que existem, sem repetir, até maxRelacionados.
func (a *app) temasExistentes(pastas []string) []string {
	lista := []string{}
	for _, t := range pastas {
		t = strings.Trim(strings.TrimSpace(t), "/")
		if len(lista) == maxRelacionados || slices.Contains(lista, t) {
			continue
		}
		if _, err := caminhoDeTema(t); err == nil && existe(filepath.Join(a.conteudo.raiz, filepath.FromSlash(t), "indice.md")) {
			lista = append(lista, t)
		}
	}
	return lista
}

// copiarJaEscrito põe em ja-escrito/ os arquivos dos Temas relacionados (sem os subtemas), para a
// IA comparar. Não é fonte: serve para dizer o que há de novo.
func (a *app) copiarJaEscrito(pasta string, temas []string) ([]string, error) {
	destino := filepath.Join(pasta, pastaJaEscrito)
	os.RemoveAll(destino)
	var copiados []string
	for _, tema := range temas {
		origem := filepath.Join(a.conteudo.raiz, filepath.FromSlash(tema))
		entradas, err := os.ReadDir(origem)
		if err != nil {
			continue
		}
		for _, e := range entradas {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			b, err := os.ReadFile(filepath.Join(origem, e.Name()))
			if err != nil {
				continue
			}
			if err := os.MkdirAll(destino, 0o755); err != nil {
				return nil, err
			}
			nome := slug(tema) + "--" + e.Name()
			if err := os.WriteFile(filepath.Join(destino, nome), b, 0o644); err != nil {
				return nil, err
			}
			copiados = append(copiados, tema)
		}
	}
	return slices.Compact(copiados), nil
}

// aprovarEsboco: o Lucas revisou (e talvez editou) as perguntas; a coleta começa.
func (a *app) aprovarEsboco(ctx context.Context, ref *referencia, perguntas []pergunta) (string, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return err.Error(), nil
	}
	if ref.Estado != "esboco" {
		return "O esboço desta pesquisa já foi aprovado.", nil
	}
	if perguntas != nil {
		p.Esboco = limparEsboco(perguntas, p, alcances[profDetalhada].maxPerguntas)
	}
	if len(p.Esboco) == 0 {
		return "Deixe ao menos uma pergunta no esboço.", nil
	}
	if err := a.guardarPedido(ctx, ref.ID, p); err != nil {
		return "", err
	}
	if err := a.mudarEstado(ctx, ref.ID, "novo"); err != nil {
		return "", err
	}
	_, err = a.fila.enfileirar(ctx, ref.ID, "coletar", nil)
	return "", err
}

// ---------- redigir e conferir ----------

func (a *app) redigir(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return pronto, err
	}
	pasta := a.pastaReferencia(ref.ID)
	if err := os.MkdirAll(filepath.Join(pasta, pastaFontes), 0o755); err != nil {
		return pronto, err
	}
	os.Remove(filepath.Join(pasta, "organizado.md"))
	relacionados, err := a.copiarJaEscrito(pasta, a.temasExistentes(append(p.Relacionados, a.temasDaReferencia(ctx, ref.ID)...)))
	if err != nil {
		return pronto, err
	}
	ferramentas := ferramentasEscrita
	if p.usa(fonteSites) {
		ferramentas += "," + ferramentasInternet
	}
	var instrucao strings.Builder
	fmt.Fprintf(&instrucao, "Pesquisa pedida pelo Lucas: %q. Hoje é %s.\n\n", p.Tema, time.Now().Format("2006-01-02"))
	if len(p.Fontes) == 0 {
		instrucao.WriteString("Ele não escolheu nenhuma fonte: escreva só com o que você já sabe, sem pesquisar, como manda a seção \"Sem fontes\" das regras.\n\n")
	} else {
		instrucao.WriteString("Perguntas do esboço:\n")
		for i, q := range p.Esboco {
			fmt.Fprintf(&instrucao, "%d. %s\n", i+1, q.Pergunta)
		}
		if len(p.Anteriores) > 0 {
			instrucao.WriteString("Perguntas das rodadas anteriores (continuam valendo):\n")
			for _, q := range p.Anteriores {
				fmt.Fprintf(&instrucao, "- %s\n", q.Pergunta)
			}
		}
		instrucao.WriteString("\nO material coletado está em fontes/ nesta pasta; cru.md lista cada arquivo e de onde veio.\n")
		if p.usa(fonteSites) {
			instrucao.WriteString("Ele também escolheu sites da internet: pesquise com WebSearch e WebFetch e guarde em fontes/ o que usar, como mandam as regras.\n")
		} else {
			instrucao.WriteString("Use só o material de fontes/: não há internet nesta pesquisa.\n")
		}
		instrucao.WriteString("\n")
	}
	if p.rodada() > 1 {
		fmt.Fprintf(&instrucao, "Esta é a rodada %d: o Lucas pediu para aprofundar. anterior.md é o texto da rodada anterior.", p.rodada())
		if p.Foco != "" {
			fmt.Fprintf(&instrucao, " O que ele quer a mais: %q.", p.Foco)
		}
		instrucao.WriteString("\n")
	}
	if len(relacionados) > 0 {
		fmt.Fprintf(&instrucao, "ja-escrito/ tem o que o Acervo já tem sobre o assunto (Temas: %s).\n", strings.Join(relacionados, ", "))
	}
	if p.rodada() > 1 || len(relacionados) > 0 {
		instrucao.WriteString("Siga a seção \"Quando já existe texto\" das regras.\n\n")
	}
	instrucao.WriteString("Escreva organizado.md nesta pasta. Responda só 'ok' ao terminar.")

	ctx, cancelar := context.WithTimeout(ctx, p.alcance().tempo)
	defer cancelar()
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta:       pasta,
		regra:       "pesquisar.md",
		instrucao:   instrucao.String(),
		ferramentas: ferramentas,
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return pronto, fmt.Errorf("a pesquisa passou do limite de %d minutos", int(p.alcance().tempo.Minutes()))
	}
	if err != nil {
		return pronto, err
	}
	if err := exigirArquivo(filepath.Join(pasta, "organizado.md")); err != nil {
		return pronto, err
	}
	if p.Profundidade == profDetalhada && len(p.Fontes) > 0 {
		_, err = a.fila.enfileirar(context.WithoutCancel(ctx), ref.ID, "conferir", nil)
		return pronto, err
	}
	return pronto, a.pesquisaPronta(context.WithoutCancel(ctx), ref.ID)
}

func (a *app) conferir(ctx context.Context, t *tarefa, ref *referencia) (resultado, error) {
	p, err := a.pedidoDe(ref)
	if err != nil {
		return pronto, err
	}
	pasta := a.pastaReferencia(ref.ID)
	texto := filepath.Join(pasta, "organizado.md")
	if err := exigirArquivo(texto); err != nil {
		return pronto, err
	}
	antes, _ := os.ReadFile(texto)
	ctx, cancelar := context.WithTimeout(ctx, p.alcance().tempo)
	defer cancelar()
	_, err = a.chamarClaude(ctx, pedidoClaude{
		pasta: pasta,
		regra: "conferir.md",
		instrucao: fmt.Sprintf("Pesquisa sobre %q. Confira organizado.md contra o material guardado em fontes/ nesta pasta "+
			"e corrija organizado.md no lugar. Responda só 'ok' ao terminar.", p.Tema),
		ferramentas: ferramentasEscrita,
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = fmt.Errorf("a conferência passou do limite de %d minutos", int(p.alcance().tempo.Minutes()))
	}
	if err == nil {
		err = exigirArquivo(texto)
	}
	if err != nil {
		// O texto de antes da conferência continua valendo para tentar de novo.
		os.WriteFile(texto, antes, 0o644)
		return pronto, err
	}
	return pronto, a.pesquisaPronta(context.WithoutCancel(ctx), ref.ID)
}

// pesquisaPronta: o texto está pronto para o Acervo; a sugestão de Tema já vem pedida.
func (a *app) pesquisaPronta(ctx context.Context, id string) error {
	if err := a.mudarEstado(ctx, id, "organizado"); err != nil {
		return err
	}
	a.cartoesDaPesquisa(ctx, id)
	_, err := a.fila.enfileirar(ctx, id, "sugerir", nil)
	return err
}

// notaDoTipo entra no pedido de criar ou integrar um Tema: a pesquisa não tem uma URL só, e o
// resumo de um livro cita páginas.
func notaDoTipo(ref *referencia) string {
	if ref.Tipo == "livro" {
		return notaDoLivro(ref)
	}
	if ref.Tipo == "artigo" {
		return notaDoArtigo(ref)
	}
	if ref.Tipo == "podcast" {
		return notaDoPodcast(ref)
	}
	if ref.Tipo != "pesquisa" {
		return ""
	}
	return " Esta Referência é uma pesquisa, com várias fontes e sem URL própria (no cabeçalho, use url: pesquisa). " +
		"O texto marca as fontes de cada seção com [n] e as lista em \"## Fontes\": no Tema, em vez de " +
		"<sub>Fonte: id [mm:ss]</sub>, escreva <sub>Fontes: ...</sub> com as fontes daquela parte por extenso e curtas " +
		"(livro e página, artigo, vídeo com o minuto, site), cada uma com o seu link, e ponha a lista inteira em \"Citados\"."
}
