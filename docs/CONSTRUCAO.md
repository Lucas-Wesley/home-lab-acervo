# Construção do Acervo

> Andamento e arquitetura da primeira versão. Começado em **06/11/2026**, com o Lucas ausente e
> autorização para construir. As decisões estão em [`DECISOES.md`](DECISOES.md); aqui fica o
> **como** e o que já foi feito. Sem commit, push ou deploy até o Lucas revisar.

---

## Arquitetura

Mesma base da Biblioteca Digital: um binário Go serve a API e o PWA Vue embutido; SQLite em
`DATA_DIR`.

```
DATA_DIR/
  acervo.db                 ← SQLite: referências, fila, ligações Referência ↔ Tema
  referencias/<id>/         ← info.json, legenda.json3, cru.md, organizado.md, traducao.md
  conteudo/                 ← repositório git do Acervo (NÃO vai pro GitHub)
    temas/<slug>/indice.md  ← Tema: uma página com capítulos (D23); subtema = pasta dentro
    temas/<slug>/<cap>.md   ← só os capítulos que o Lucas separou
  revisoes/<tarefa>/        ← git worktree de cada proposta da IA, até aprovar ou rejeitar
```

### Fila (D14)
- Tabela `tarefas`; **um worker**, uma tarefa por vez, a mais antiga primeiro.
- Ao iniciar o app, tarefas que estavam `rodando` voltam para `na_fila`.
- Tipos: `extrair` → `organizar` → (`traduzir` se o idioma não for português) e, sob comando do
  Lucas, `sugerir`, `criar_tema`, `integrar`.
- Cancelar mata o processo (`claude` ou `yt-dlp`) pelo contexto.

### Revisão por diff (D16) com git worktree
- `criar_tema` e `integrar` rodam numa **worktree própria** (`revisoes/<tarefa>`, branch
  `revisao/<tarefa>`). A `main` do conteúdo fica sempre limpa.
- O app mostra `git diff main...` da worktree. **Aprovar** = commit na branch + merge na `main`.
  **Rejeitar** = remover worktree e branch.
- Edição manual de um arquivo é commit direto na `main`.

### Edição e offline (D19, D21)
- Cada arquivo tem um **hash** (SHA-256 do conteúdo). Salvar envia o hash que o aparelho tinha;
  se o arquivo mudou no servidor, a API responde **409** com a versão atual e o app mostra as duas
  para escolher.
- Baixar um Tema guarda todos os arquivos dele no Cache do navegador; o service worker responde
  com a rede quando há rede e com o cache quando não há. Edições offline ficam numa fila no
  aparelho e sobem ao voltar.

### Chamadas externas
- `yt-dlp`: `-J` para metadados e escolha da legenda (manual no idioma do vídeo → automática
  `<idioma>-orig` → manual em português → qualquer manual); depois baixa só a legenda em `json3`.
  Sem legenda = erro visível e o processo para (D17).
- `claude -p` com as regras de [`regras/`](../regras/), ferramentas limitadas a
  `Read,Write,Edit,Glob,Grep` e diretório de trabalho da tarefa. Login por
  `CLAUDE_CODE_OAUTH_TOKEN` (D13).
- Binários configuráveis (`YTDLP_BIN`, `CLAUDE_BIN`) para os testes usarem versões falsas.

---

## Checklist

- [x] Validação manual com 3 vídeos; regras `organizar` e `criar-tema`.
- [x] Back-end: banco, fila, extração, organizar/traduzir, sugerir, criar Tema, integrar, revisões,
      leitura/edição de arquivos, citações (links que chegam), baixar Tema. Testes Go passando.
- [x] Regras `traduzir`, `sugerir` e `integrar` (escritas, ainda não testadas com o Claude real).
- [x] PWA: menu lateral, Acervo (Temas), leitor de Markdown com navegação e "voltar", editor,
      Referências (lista, detalhe com abas, ações), Fila, Revisões com diff, Baixados/offline.
- [x] Testes Go (com `yt-dlp` e `claude` falsos), Vue (13) e Playwright (5), todos passando.
- [x] Rodar os 3 vídeos de verdade pelo app, de ponta a ponta (ver "Teste real" abaixo).
- [x] Docker (`compose.yaml`, `compose.prod.yaml`, `deploy.sh`) e bloco no Caddy (porta 8444).
      Imagem testada no Mac: extração real dentro do container funciona.
- [x] Plano (só texto) da integração com a Biblioteca e do módulo Pesquisador:
      [`PESQUISADOR.md`](PESQUISADOR.md).

## Teste real (06/11/2026, no Mac, `yt-dlp` e Claude de verdade)

| Etapa | Resultado | Tempo |
|-------|-----------|-------|
| Extrair (3 vídeos) | ok; legenda manual em 2, automática `pt-orig` no do café | ~5 s cada |
| Organizar | ok | 1 a 2,5 min |
| Sugerir Tema | ok, resposta estruturada: "novo — O gaúcho: origem e história" | 7 s |
| Criar Tema (3) | ok, 12 a 15 tópicos cada, fontes com marca de tempo | 1 a 2,5 min |
| Aprovar | Gaúcho e Café aprovados (commits no git do conteúdo); **Leonardo da Vinci deixado em Revisões para o Lucas** | — |
| Integrar sem lacuna | Café → Tema do gaúcho: "A Referência não acrescentou nada ao Tema"; nada mudou | 11 s |
| Integrar com lacuna | Numa cópia dos dados (não no Acervo real): gaúcho → Tema "Rio Grande do Sul" mínimo. Manteve o texto existente, inseriu em ordem cronológica, com fonte em cada trecho, e criou tópicos novos | ~1 min |

Bug achado pelo Lucas e corrigido: com duas propostas abertas ao mesmo tempo, aprovar uma fazia a
outra mostrar os arquivos da primeira como "apagados" (o diff comparava com a main atual, e não com
o ponto de partida da proposta). Só a tela estava errada; o merge não apagaria nada. Agora o diff é
contra o `HEAD` da worktree, com teste.

Bug achado pelo teste real e corrigido: com `DATA_DIR` relativo, a worktree nascia no lugar errado
e o Claude falhava com "sem resposta". Os caminhos agora são absolutos, e há teste para isso.

## Reestruturação em capítulos (06/11/2026, D23)

- Servidor: `capitulos.go` (separar/juntar com hash e commit; juntar recusa se o arquivo do capítulo
  é citado em outro lugar). A lista de Temas conta capítulos.
- Leitor: sumário entre a introdução e o corpo; botão "Capítulos" no cabeçalho fixo, que lista os
  capítulos de qualquer ponto do texto e destaca o que está na tela; menu "⋯" em cada capítulo (editar só ele, separar,
  abrir, trazer de volta); "Novo capítulo" entra antes dos apêndices; subtemas listados sozinhos;
  anterior/próximo nas páginas de capítulo separado.
- Regras `criar-tema` e `integrar` reescritas para a página única.
- Conteúdo existente migrado (Café e Gaúcho: os tópicos viraram capítulos, sem perder texto; commit
  no git do conteúdo). A proposta do Leonardo foi rejeitada e refeita com a regra nova pelo Claude
  real: uma página, 20 capítulos.

## Identidade visual e compartilhar (06/11/2026)

- **Design system comum** com a Biblioteca ([`DESIGN-SYSTEM.md`](DESIGN-SYSTEM.md)): neutros
  quentes, azul e terracota só para contraste, vidro com blur no que flutua, botões de 32 px e
  listas em linhas finas, sem cartões. Os nomes antigos dos tokens (`--ok`, `--aviso`,
  `--perigo`, `--mais`, `--menos`) continuam, agora apontando para o azul ou a terracota.
- **Celular:** o menu virou **abas de vidro** no pé da tela e some no leitor. Não há mais botão
  flutuante de menu nem o recuo de 56 px nas barras.
- **Compartilhar:** `share_target` (GET em `/`) no manifest. O `App.vue` lê `title`, `text` e
  `url`, manda para `POST /api/referencias`, troca o endereço para `#/referencias` e mostra um
  aviso rápido. `videoDoYouTube` acha o primeiro link de vídeo num texto. No iPhone não existe
  *share target*, e a saída é um atalho do app Atalhos (README). Há teste de navegador.
- **Ícone próprio:** livros numa estante, no mesmo traço do livro aberto da Biblioteca.

## Ajustes: instruções da IA e sugestões (06/11/2026, D25)

- **Ajustes** é o último item do menu (no celular, a sexta aba).
- **Instruções da IA:** as cinco regras de [`regras/`](../regras/), com nome e descrição, para ler
  e editar pelo celular.
  - A versão editada vai para `DATA_DIR/regras/<arquivo>` e vale no lugar da do repositório
    (`lerRegra`, usada por `chamarClaude`). "Voltar ao padrão" apaga a edição.
  - A gravação é atômica (arquivo temporário + rename), e só os cinco arquivos conhecidos são
    aceitos.
  - API: `GET /api/instrucoes`, `GET|PUT|DELETE /api/instrucoes/{arquivo}`.
  - Os parágrafos das regras passaram a ocupar uma linha só, para editar no celular sem quebras
    no meio da frase.
- **Sugestões de melhoria:** o Lucas anota o que pode melhorar; o Claude Code lê, faz e responde.
  - Tabela `sugestoes`. API comum a todas as apps: `GET /api/sugestoes`,
    `PUT /api/sugestoes/{id}` (o id vem do aparelho, e reenviar não duplica),
    `PATCH /api/sugestoes/{id}` com `{estado, resposta}` e `DELETE /api/sugestoes/{id}`.
  - Estados: `nova`, `feita`, `descartada`. A sugestão guarda a tela de onde o Lucas foi para
    Ajustes.
- Testes: Go (instrução editada chega ao Claude; sugestões) e navegador (Ajustes).

## Mais informações sobre um trecho (06/11/2026, D26)

- **Seleção:** ao selecionar texto num Tema ou numa Referência aparece uma barra escura com dois
  ícones, **Mais informações** e **Copiar** (`CamadaDetalhes.vue`). No celular ela fica abaixo da
  seleção, porque em cima o sistema mostra o menu dele. Cabe mais ícone ali no futuro.
- **Marcação fora do texto:** tabela `detalhes` com o trecho, até 300 caracteres antes e depois, o
  `local` (`temas/…/arquivo.md` ou `referencias/<id>/<aba>`) e a tarefa. Ao abrir a página, o
  app junta os nós de texto do HTML renderizado, escolhe a ocorrência do trecho com mais contexto
  igual e envolve em `<mark class="detalhe">` (`detalhes.js`). Se o trecho sumiu do texto, a
  marcação aparece em "Marcações que não estão mais no texto", no fim da página.
- **Pesquisa:** tarefa `detalhar` na fila (uma por vez, D14), com a regra `regras/detalhar.md` e
  só as ferramentas `WebSearch` e `WebFetch`, numa pasta temporária vazia. A resposta é
  estruturada (`explicacao` em Markdown e `fontes`). O estado (`processando`, `pronto`, `falhou`)
  vem da tarefa; a Fila mostra o trecho e leva de volta ao texto.
- **Fora da rede:** as explicações já vistas ficam no aparelho; um pedido novo fica guardado e
  sobe quando a rede volta (o id vem do aparelho, e reenviar não pesquisa duas vezes).
- API: `GET /api/detalhes?local=`, `PUT /api/detalhes/{id}`, `POST /api/detalhes/{id}/refazer`,
  `DELETE /api/detalhes/{id}`.
- Testes: Go (pesquisa com internet só nesta tarefa, reenvio, refazer, validação, falha),
  unidade (achar e sublinhar o trecho, pedido fora da rede) e navegador (selecionar, copiar,
  pedir, abrir o modal, apagar).

## Pesquisador (06/11/2026, D27)

- **Pedido:** `POST /api/pesquisas` com `{tema, fontes, profundidade, revisar_esboco}` cria uma Referência com `tipo = 'pesquisa'`; o pedido, o esboço e a coleta ficam na coluna `pesquisa` (JSON). A tabela `referencias` foi refeita uma vez (`migrarReferencias`) para o `video_id` deixar de ser obrigatório.
- **Etapas:** `planejar` (resposta estruturada com as perguntas e as buscas), `coletar` (`coleta.go`, sem IA), `redigir` e `conferir` (`pesquisa.go`). Com "Revisar o esboço", a Referência para no estado `esboco` até `POST /api/referencias/{id}/pesquisar` com o esboço editado. "Tentar de novo" retoma da etapa que falta (`proximaEtapaPesquisa`).
- **Fontes:**
  - **Livros:** `GET {BIBLIOTECA_URL}/api/search` (R32 da Biblioteca); cada trecho vem com o link de volta ao ponto do livro (`BIBLIOTECA_LINK`).
  - **Wikipédia:** API do pt.wikipedia (busca e texto do artigo, até 60 mil caracteres).
  - **YouTube:** `yt-dlp --flat-playlist -J "ytsearchN:<busca>"`, vídeos de 2 a 90 minutos com legenda, pela mesma extração das Referências de vídeo.
  - **Sites:** só na escrita, com `WebSearch`/`WebFetch`; a IA guarda em `fontes/web-*.md` o que usou.
- **Telas:** formulário em Referências (as últimas escolhas ficam no aparelho), editor do esboço na Referência, aba "Material coletado" (`cru.md`) e ícone de pesquisa na lista.
- Testes: Go (detalhada com esboço revisado, rápida com sites, só a IA, sem material e tentar de novo, conferência que falha, pedido inválido, migração), unidade e navegador (pedido → esboço → pesquisa → sugestão).

## O que ficou de fora desta versão

- **Verificação de veracidade (item 7):** só a aba "Dúvidas" e o botão "Marcar como validada".
- **Excluir ou mover Tema** pelo app (mover exige reescrever os links que apontam para ele).
- **Tradução:** a regra existe e o fluxo está testado com o Claude falso, mas falta um vídeo em
  inglês de verdade.

## Notas da construção

- **Resposta estruturada:** `claude -p --json-schema` devolve `structured_output`; a sugestão de
  Tema usa isso (7 s no teste real).
- **`yt-dlp` no container:** o YouTube exige um motor de JavaScript; no Mac é o `deno` (dependência
  do Homebrew), no container é o próprio Node (`--js-runtimes node`), que já está lá pelo Claude.
- **Deploy sem Playwright no servidor:** diferente da Biblioteca, os testes de navegador criam
  Referências e Temas; rodar contra produção sujaria o Acervo. O `deploy.sh` confere só saúde,
  versão, PWA e API.
- **Sem token, erro claro:** a etapa do Claude falha com "O Claude não está autenticado neste
  servidor: falta o CLAUDE_CODE_OAUTH_TOKEN".
- **No Mac, o `claude` lê o `~/.claude` do Lucas** (instruções globais). No servidor, `HOME` é
  `/data/home`, limpo. Se as instruções pessoais atrapalharem as regras, rodar com
  `--setting-sources` restrito.
