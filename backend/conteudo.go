package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// conteudo é o repositório git do Acervo (D8). A main fica sempre limpa: edição manual é
// commit direto, e o que a IA propõe nasce numa worktree até ser aprovado (D16).
type conteudo struct {
	raiz     string
	revisoes string
	cfg      config
	mu       sync.Mutex // serializa o que mexe na main
}

const pastaTemas = "temas"

func abrirConteudo(ctx context.Context, raiz, revisoes string, cfg config) (*conteudo, error) {
	c := &conteudo{raiz: raiz, revisoes: revisoes, cfg: cfg}
	if err := os.MkdirAll(filepath.Join(raiz, pastaTemas), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(raiz, ".git")); errors.Is(err, fs.ErrNotExist) {
		if _, err := c.git(ctx, raiz, "init", "-q", "-b", "main"); err != nil {
			return nil, err
		}
		leia := "# Acervo\n\nConteúdo do Acervo, versionado só aqui no homelab. Não sobe para o GitHub.\n"
		if err := os.WriteFile(filepath.Join(raiz, "README.md"), []byte(leia), 0o644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(raiz, pastaTemas, ".gitkeep"), nil, 0o644); err != nil {
			return nil, err
		}
		if _, err := c.git(ctx, raiz, "add", "-A"); err != nil {
			return nil, err
		}
		if _, err := c.git(ctx, raiz, "commit", "-q", "-m", "Início do Acervo"); err != nil {
			return nil, err
		}
	}
	// Worktrees cujas pastas sumiram (ex.: dados restaurados) atrapalham o próximo "worktree add".
	c.git(ctx, raiz, "worktree", "prune")
	return c, nil
}

func (c *conteudo) git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+c.cfg.autorNome, "GIT_AUTHOR_EMAIL="+c.cfg.autorEmail,
		"GIT_COMMITTER_NAME="+c.cfg.autorNome, "GIT_COMMITTER_EMAIL="+c.cfg.autorEmail,
		"GIT_TERMINAL_PROMPT=0")
	var saida, erros bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erros
	if err := cmd.Run(); err != nil {
		return saida.String(), fmt.Errorf("git %s: %s", args[0], primeiraLinha(erros.String(), err.Error()))
	}
	return saida.String(), nil
}

var (
	errCaminhoInvalido = errors.New("caminho inválido")
	errNaoEncontrado   = errors.New("não encontrado")
	errJaExiste        = errors.New("já existe")
)

// caminhoDeArquivo valida um caminho relativo vindo da API: só arquivos .md dentro de temas/.
func caminhoDeArquivo(rel string) (string, error) {
	limpo := path.Clean("/" + rel)[1:]
	if limpo != rel || !strings.HasPrefix(limpo, pastaTemas+"/") || !strings.HasSuffix(limpo, ".md") {
		return "", errCaminhoInvalido
	}
	for _, parte := range strings.Split(limpo, "/") {
		if strings.HasPrefix(parte, ".") {
			return "", errCaminhoInvalido
		}
	}
	return limpo, nil
}

// caminhoDeTema valida a pasta de um Tema: temas/<slug>[/<slug>...].
func caminhoDeTema(rel string) (string, error) {
	limpo := path.Clean("/" + rel)[1:]
	if limpo != rel || !strings.HasPrefix(limpo, pastaTemas+"/") {
		return "", errCaminhoInvalido
	}
	for _, parte := range strings.Split(limpo, "/")[1:] {
		if !slugValido.MatchString(parte) {
			return "", errCaminhoInvalido
		}
	}
	return limpo, nil
}

var slugValido = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// slug transforma "Segunda Guerra Mundial" em "segunda-guerra-mundial".
func slug(titulo string) string {
	var b strings.Builder
	hifen := false
	for _, r := range norm.NFD.String(strings.ToLower(titulo)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			if hifen && b.Len() > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r)
			hifen = false
		default:
			hifen = true
		}
	}
	return b.String()
}

func hashDe(conteudo []byte) string {
	h := sha256.Sum256(conteudo)
	return hex.EncodeToString(h[:])
}

// ---------- Temas ----------

type tema struct {
	Caminho   string `json:"caminho"` // pasta: temas/origens-do-gaucho
	Titulo    string `json:"titulo"`
	Resumo    string `json:"resumo"`
	Pai       string `json:"pai"` // pasta do Tema que contém este, se houver
	Capitulos int    `json:"capitulos"`
}

type frente struct {
	Titulo      string
	Criado      string
	Referencias []fonte
}

type fonte struct {
	ID     string `json:"id"`
	Titulo string `json:"titulo"`
	URL    string `json:"url"`
}

// lerFrente lê o cabeçalho YAML do indice.md. Só o formato que as regras definem, sem biblioteca.
func lerFrente(texto string) (frente, string) {
	var f frente
	if !strings.HasPrefix(texto, "---\n") {
		return f, texto
	}
	fim := strings.Index(texto[4:], "\n---")
	if fim < 0 {
		return f, texto
	}
	cabeca, corpo := texto[4:4+fim], strings.TrimLeft(texto[4+fim+4:], "\n")
	var atual *fonte
	for _, l := range strings.Split(cabeca, "\n") {
		chave, valor, ok := strings.Cut(strings.TrimSpace(l), ":")
		if !ok {
			continue
		}
		valor = strings.Trim(strings.TrimSpace(valor), `"'`)
		if strings.HasPrefix(chave, "- ") {
			f.Referencias = append(f.Referencias, fonte{})
			atual = &f.Referencias[len(f.Referencias)-1]
			chave = strings.TrimSpace(chave[2:])
		}
		indentado := strings.HasPrefix(l, " ") && atual != nil
		switch {
		case indentado && chave == "id":
			atual.ID = valor
		case indentado && chave == "titulo":
			atual.Titulo = valor
		case indentado && chave == "url":
			atual.URL = valor
		case chave == "titulo":
			f.Titulo = valor
		case chave == "criado":
			f.Criado = valor
		}
	}
	return f, corpo
}

// tituloDe pega o título do cabeçalho ou o primeiro "# ".
func tituloDe(texto, alternativa string) string {
	f, corpo := lerFrente(texto)
	if f.Titulo != "" {
		return f.Titulo
	}
	for _, l := range strings.Split(corpo, "\n") {
		if strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(l[2:])
		}
	}
	return alternativa
}

// resumoDe pega o primeiro parágrafo de texto depois do título.
func resumoDe(texto string) string {
	_, corpo := lerFrente(texto)
	for _, p := range strings.Split(corpo, "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "#") || strings.HasPrefix(p, "<") {
			continue
		}
		p = strings.TrimPrefix(p, "> ")
		if r := []rune(p); len(r) > 220 {
			return strings.TrimSpace(string(r[:220])) + "…"
		}
		return p
	}
	return ""
}

func (c *conteudo) listarTemas() ([]tema, error) {
	return listarTemasEm(c.raiz)
}

func listarTemasEm(raiz string) ([]tema, error) {
	temas := []tema{}
	porPasta := map[string]bool{}
	err := filepath.WalkDir(filepath.Join(raiz, pastaTemas), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "indice.md" {
			return nil
		}
		rel, _ := filepath.Rel(raiz, filepath.Dir(p))
		rel = filepath.ToSlash(rel)
		texto, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		temas = append(temas, tema{Caminho: rel, Titulo: tituloDe(string(texto), path.Base(rel)), Resumo: resumoDe(string(texto)), Capitulos: contarCapitulos(string(texto))})
		porPasta[rel] = true
		return nil
	})
	for i := range temas {
		for pai := path.Dir(temas[i].Caminho); pai != pastaTemas && pai != "."; pai = path.Dir(pai) {
			if porPasta[pai] {
				temas[i].Pai = pai
				break
			}
		}
	}
	sort.Slice(temas, func(i, j int) bool { return temas[i].Caminho < temas[j].Caminho })
	return temas, err
}

// criarTemaVazio cria um Tema escrito à mão, sem Referência.
func (c *conteudo) criarTemaVazio(ctx context.Context, pasta, titulo string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	dir := filepath.Join(c.raiz, filepath.FromSlash(pasta))
	if _, err := os.Stat(dir); err == nil {
		return errJaExiste
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	texto := fmt.Sprintf("---\ntitulo: %s\ncriado: %s\n---\n\n# %s\n\n", titulo, time.Now().Format("2006-01-02"), titulo)
	if err := os.WriteFile(filepath.Join(dir, "indice.md"), []byte(texto), 0o644); err != nil {
		return err
	}
	return c.commit(ctx, "Novo Tema: "+titulo, pasta)
}

func (c *conteudo) commit(ctx context.Context, mensagem string, caminhos ...string) error {
	if _, err := c.git(ctx, c.raiz, append([]string{"add", "--"}, caminhos...)...); err != nil {
		return err
	}
	_, err := c.git(ctx, c.raiz, "commit", "-q", "-m", mensagem)
	return err
}

// ---------- Arquivos ----------

type arquivo struct {
	Caminho  string   `json:"caminho"`
	Conteudo string   `json:"conteudo"`
	Hash     string   `json:"hash"`
	Tema     string   `json:"tema"`     // pasta do Tema ao qual o arquivo pertence
	Migalhas []tema   `json:"migalhas"` // Temas que contêm este, do mais externo ao mais interno
	Fontes   []fonte  `json:"fontes"`   // referências do indice.md do Tema
	CitadoEm []citado `json:"citado_em"`
}

type citado struct {
	Caminho string `json:"caminho"`
	Titulo  string `json:"titulo"`
}

func (c *conteudo) ler(rel string) (*arquivo, error) {
	texto, err := os.ReadFile(filepath.Join(c.raiz, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, errNaoEncontrado
	} else if err != nil {
		return nil, err
	}
	a := &arquivo{Caminho: rel, Conteudo: string(texto), Hash: hashDe(texto), Migalhas: []tema{}, Fontes: []fonte{}}

	temas, err := c.listarTemas()
	if err != nil {
		return nil, err
	}
	porPasta := map[string]tema{}
	for _, t := range temas {
		porPasta[t.Caminho] = t
	}
	for dir := path.Dir(rel); dir != pastaTemas && dir != "."; dir = path.Dir(dir) {
		if t, ok := porPasta[dir]; ok {
			if a.Tema == "" {
				a.Tema = dir
			}
			a.Migalhas = append([]tema{t}, a.Migalhas...)
		}
	}
	if a.Tema != "" {
		if indice, err := os.ReadFile(filepath.Join(c.raiz, filepath.FromSlash(a.Tema), "indice.md")); err == nil {
			f, _ := lerFrente(string(indice))
			if f.Referencias != nil {
				a.Fontes = f.Referencias
			}
		}
	}
	a.CitadoEm, err = c.citacoes(rel)
	return a, err
}

var linkMarkdown = regexp.MustCompile(`\]\(([^)\s#]+\.md)(?:#[^)]*)?\)`)

// citacoes acha os arquivos que linkam para rel, fora da própria pasta (o "voltar ao índice"
// de cada tópico não conta como citação).
func (c *conteudo) citacoes(rel string) ([]citado, error) {
	lista := []citado{}
	alvoDir := path.Dir(rel)
	err := filepath.WalkDir(filepath.Join(c.raiz, pastaTemas), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return err
		}
		origem, _ := filepath.Rel(c.raiz, p)
		origem = filepath.ToSlash(origem)
		if path.Dir(origem) == alvoDir {
			return nil
		}
		texto, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range linkMarkdown.FindAllStringSubmatch(string(texto), -1) {
			if path.Join(path.Dir(origem), m[1]) == rel {
				lista = append(lista, citado{Caminho: origem, Titulo: tituloDe(string(texto), path.Base(origem))})
				break
			}
		}
		return nil
	})
	return lista, err
}

type conflito struct {
	Atual *arquivo
}

func (e *conflito) Error() string { return "o arquivo mudou no servidor" }

// salvar grava uma edição feita à mão. base é o hash que o aparelho tinha; se o arquivo mudou
// desde então, devolve conflito com a versão atual em vez de sobrescrever (D21).
func (c *conteudo) salvar(ctx context.Context, rel, texto, base string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cheio := filepath.Join(c.raiz, filepath.FromSlash(rel))
	atual, err := os.ReadFile(cheio)
	existe := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if (existe && hashDe(atual) != base) || (!existe && base != "") {
		a, _ := c.ler(rel)
		return "", &conflito{Atual: a}
	}
	if existe && string(atual) == texto {
		return base, nil
	}
	if !existe {
		if _, err := os.Stat(filepath.Join(filepath.Dir(cheio), "indice.md")); err != nil && path.Base(rel) != "indice.md" {
			return "", errCaminhoInvalido
		}
	}
	if err := os.WriteFile(cheio, []byte(texto), 0o644); err != nil {
		return "", err
	}
	verbo := "Edição"
	if !existe {
		verbo = "Novo arquivo"
	}
	if err := c.commit(ctx, verbo+": "+rel, rel); err != nil {
		return "", err
	}
	return hashDe([]byte(texto)), nil
}

// pacote devolve todos os arquivos de um Tema (com subtemas) para ler offline.
func (c *conteudo) pacote(pasta string) (map[string]arquivoOffline, error) {
	arquivos := map[string]arquivoOffline{}
	raiz := filepath.Join(c.raiz, filepath.FromSlash(pasta))
	if _, err := os.Stat(filepath.Join(raiz, "indice.md")); err != nil {
		return nil, errNaoEncontrado
	}
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return err
		}
		rel, _ := filepath.Rel(c.raiz, p)
		texto, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		arquivos[filepath.ToSlash(rel)] = arquivoOffline{Conteudo: string(texto), Hash: hashDe(texto)}
		return nil
	})
	return arquivos, err
}

type arquivoOffline struct {
	Conteudo string `json:"conteudo"`
	Hash     string `json:"hash"`
}

// ---------- Revisões (D16) ----------

func (c *conteudo) pastaRevisao(id int64) string {
	return filepath.Join(c.revisoes, fmt.Sprint(id))
}

func ramoRevisao(id int64) string { return fmt.Sprintf("revisao/%d", id) }

// abrirRevisao cria a worktree onde o Claude vai trabalhar, a partir da main atual.
func (c *conteudo) abrirRevisao(ctx context.Context, id int64) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.descartarRevisao(ctx, id)
	dir := c.pastaRevisao(id)
	_, err := c.git(ctx, c.raiz, "worktree", "add", "-q", "-b", ramoRevisao(id), dir, "main")
	return dir, err
}

func (c *conteudo) descartarRevisao(ctx context.Context, id int64) {
	c.git(ctx, c.raiz, "worktree", "remove", "--force", c.pastaRevisao(id))
	os.RemoveAll(c.pastaRevisao(id))
	c.git(ctx, c.raiz, "worktree", "prune")
	c.git(ctx, c.raiz, "branch", "-D", ramoRevisao(id))
}

// As comparações são com o HEAD da worktree, que é o ponto de onde a proposta partiu. Comparar
// com a main atual faria uma proposta antiga parecer que apaga o que foi aprovado depois dela.

// mudancas lista os arquivos alterados na worktree (inclusive os novos).
func (c *conteudo) mudancas(ctx context.Context, id int64) ([]string, error) {
	dir := c.pastaRevisao(id)
	if _, err := c.git(ctx, dir, "add", "-A"); err != nil {
		return nil, err
	}
	saida, err := c.git(ctx, dir, "diff", "--cached", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	return strings.Fields(saida), nil
}

func (c *conteudo) diff(ctx context.Context, id int64) (string, error) {
	dir := c.pastaRevisao(id)
	if _, err := c.git(ctx, dir, "add", "-A"); err != nil {
		return "", err
	}
	return c.git(ctx, dir, "diff", "--cached", "--no-color", "--find-renames", "HEAD")
}

// aprovarRevisao vira commit e entra na main.
func (c *conteudo) aprovarRevisao(ctx context.Context, id int64, mensagem string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	dir := c.pastaRevisao(id)
	if _, err := c.git(ctx, dir, "add", "-A"); err != nil {
		return err
	}
	if _, err := c.git(ctx, dir, "commit", "-q", "-m", mensagem); err != nil {
		return err
	}
	if _, err := c.git(ctx, c.raiz, "merge", "-q", "--no-edit", ramoRevisao(id)); err != nil {
		c.git(ctx, c.raiz, "merge", "--abort")
		return fmt.Errorf("a proposta conflita com mudanças feitas depois dela; rejeite e peça de novo (%v)", err)
	}
	c.descartarRevisao(ctx, id)
	return nil
}
