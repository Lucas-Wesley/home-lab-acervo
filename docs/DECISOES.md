# Acervo — Plano do Projeto

> Gerenciador de conhecimento pessoal. Registro das decisões até **06/11/2026**.
> Fonte da verdade do projeto enquanto o código não existe. Ao clonar/retomar em outra máquina,
> leia este arquivo inteiro antes de qualquer coisa.

---

## 1. O que é

Sistema pessoal para transformar conteúdos (hoje **vídeos do YouTube**; no futuro artigos e outras
fontes) em uma **base de conhecimento permanente** — organizada, verificada, cronológica e
versionada — construída incrementalmente ao longo do tempo.

**Não é um "resumidor de vídeo".** O vídeo é só matéria-prima. O produto é a base de conhecimento
que cresce e fica cada vez mais completa.

---

## 2. Vocabulário — as duas áreas

- **Acervo (base permanente):** o conhecimento consolidado, dividido por **Temas** (Segunda Guerra
  Mundial, Café, ...). Permanente, cumulativo, **nunca substituído — só enriquecido**.
- **Referências (efêmeras):** os conteúdos extraídos de fontes. Matéria-prima com ciclo de vida
  (estados). Eu gerencio: posso integrar, manter ou descartar.

### Estrutura de um Tema — uma página com capítulos (D23, revisto em 06/11/2026)

Cada Tema é uma **pasta** com um **`indice.md`** que tem **o texto inteiro**, dividido em
**capítulos** (`## `), mais "Linha do tempo" e "Citados" no fim. O app monta o **sumário** a partir
dos capítulos; cada item leva ao capítulo **na mesma página**.

- **Por padrão, tudo fica na página principal.** Tópicos em arquivos separados deixavam textos
  curtos demais.
- **Separar é manual:** quando um capítulo cresce, o Lucas escolhe "Separar em página própria" no
  menu do capítulo. O texto vai para `<pasta>/<capitulo>.md` e, na página principal, fica o título
  com o link "Ler o capítulo →". O caminho inverso ("Trazer de volta") também existe.
- Cada capítulo pode ser **editado sozinho** (o resto da página não muda).
- A IA cria Temas numa página só e integra no capítulo certo; se o capítulo estiver separado,
  escreve no arquivo dele.

> **Por que isso e não RAG:** a página com capítulos mantém o texto legível de ponta a ponta, e
> separar capítulos grandes mantém cada operação pequena. A estrutura resolve a escala.

### Temas, subtemas e links (D20, revisto em 06/11/2026)

Estrutura simples, decidida pelo Lucas:
- Cada Tema é uma **pasta com `indice.md`**. Um subtema é uma **pasta dentro** do Tema que o contém
  (ex.: `temas/segunda-guerra-mundial/alemanha-e-russia/`), também com o próprio `indice.md`.
- **Onde cada Tema mora é decidido no cadastro**, pelo Lucas. A IA só sugere.
- Para usar um Tema em outro contexto, **linka-se o `indice.md` dele** — é só link, o texto não é
  copiado. Ex.: "Alemanha e Rússia" nasce como Tema principal; depois "Segunda Guerra Mundial" é
  criado e o índice dele linka "Alemanha e Rússia".
- O link sempre aponta para um `indice.md`, que pode ter o texto direto ou levar a capítulos separados e
  subtemas.
- **Navegação no PWA:** entrar num link e **voltar ao texto de onde veio** tem que ser fluido. Cada
  Tema mostra também **onde é citado** (links que chegam nele).
- Links são caminhos relativos de Markdown comum; se um Tema mudar de pasta, o app reescreve os
  links que apontam para ele.

### Estados de uma Referência

```
cru ─> organizado ─> em verificação ─> validado ─> integrado
 │                                              └─> descartado
 └─> erro (ex.: vídeo sem legenda)
```
- **cru:** texto extraído bruto da fonte, no idioma original (preservado, para poder reverter).
- **organizado:** reestruturado e com complementos temporais consolidados (ver fluxo, passo 2).
  Se a fonte não for em português, ganha também a **tradução** do organizado (D18).
- **em verificação / validado:** passou pela checagem de veracidade com decisão humana.
- **integrado / descartado:** eu decido o destino. Uma Referência pode alimentar **vários Temas**,
  então "integrado" vale **por Tema**.
- **erro:** a etapa falhou e o processo parou. O motivo aparece na lista de Referências.

---

## 3. Fluxo completo (da URL ao Acervo)

1. **Extrair** a transcrição da fonte. YouTube → `yt-dlp`. → estado **`cru`**.
   - **Sem legenda:** o processo para e a Referência mostra na lista "não foi possível extrair a
     legenda deste vídeo". Nada mais é feito por enquanto; o Whisper fica para depois (D17).
2. **Organizar / reorganizar:** estruturar o texto e **consolidar complementos temporais** — num
   podcast, algo dito no minuto 5 é complementado no minuto 20; a reorganização junta isso num
   fluxo lógico/temático, **não** na ordem falada. (O `cru` fica preservado.) → **`organizado`**.
   - *Observação:* são duas ordens diferentes — a **ordem do discurso** (como foi falado) vira a
     **ordem lógica/cronológica do conteúdo** (como o assunto se organiza de fato).
   - **Fonte em outro idioma:** traduz o **organizado** para português. A tela da Referência mostra as
     duas; o original fica **só para consulta** (D18).
3. **Verificar veracidade** antes de virar referência utilizável. A IA **sinaliza** afirmações
   factuais duvidosas e **pergunta a mim**; a decisão é **humana** — nada de descarte automático.
   → **`validado`**.
4. **Classificar o Tema:** a IA **sugere** onde a referência se encaixa: novo Tema principal,
   subtema de um Tema existente, ou integração em Temas existentes. **Eu decido.**
   - **Novo Tema** → esta referência **vira o Texto Principal do novo Tema**.
   - **Tema existente** → segue para integração.
5. **Integrar ao Acervo** (quando eu mandar): análise de **lacunas** (o que a referência tem que o Tema ainda não tem) → o Claude edita os
   arquivos do Tema **sem commit** → o app mostra o **`git diff`** → **eu aprovo** (vira commit) ou
   **rejeito** (os arquivos voltam ao que eram) (D16).

> Com o tempo o Texto Principal fica mais completo, então cada nova Referência acrescenta menos —
> os detalhes finos se encaixam nos capítulos existentes em vez de criar tudo do zero.

Toda etapa que chama o Claude ou o `yt-dlp` entra numa **fila** e roda em segundo plano (D14).

---

## 4. Decisões tomadas

| # | Decisão | Escolha | Por quê |
|---|---------|---------|---------|
| D1 | Caminho geral | Pipeline próprio (DIY), **não** SurfSense | Controle total do prompt/regras, sem licença de plugin, transparente. Sou dev. |
| D2 | Extração | **`yt-dlp`** hoje; fontes **plugáveis** (só YouTube por ora) | Ferramentas prontas/gratuitas; arquitetura aberta a novas fontes. |
| D3 | Engine de IA | **Claude Code CLI headless** (`claude -p`) | Reaproveita minha **assinatura** + libera skills/subagents/MCP/hooks/CLAUDE.md. |
| D4 | Orquestração | **App web self-hosted** no homelab, acessada pelo navegador do celular | Dispensa app nativo; eu construo. |
| D5 | Validação | Validar as **regras de cada etapa em modo manual** antes de automatizar | Acerta o formato antes de investir na infra. |
| D6 | Nome | **Acervo** (base permanente) + **Referências** (efêmeros) | Descreve o que o sistema é. |
| D7 | Estrutura de dados | **Markdown por Tema**; **sem RAG** por ora. Formato da página: D23 | A estrutura resolve a escala. |
| D8 | Versionamento | **Git local no homelab**; **conteúdo NÃO sobe pro GitHub** | Privacidade; histórico/reversão de graça; *diff* = tela de revisão. |
| D9 | Reorganização na extração | **Sim** (consolidar complementos temporais) | Vídeo falado não é linear; o conteúdo precisa ser. |
| D10 | Checagem de veracidade | Estados + **revisão humana** (IA sinaliza, eu decido) | Evitar propagar informação falsa sem perder controle. |
| D12 | Stack | **Go** na API, **Vue** no PWA, **SQLite** como banco | Confirmado em 06/11/2026. O back-end orquestra programas externos (`yt-dlp`, `claude`, `git`), fila e arquivos; Go faz isso sem dependências e reaproveita o esqueleto da Biblioteca. |
| D11 | Infraestrutura | **A mesma da Biblioteca Digital:** VM `apps` (`192.168.1.20`), Docker Compose, Caddy com `tls internal`, `deploy.sh` com testes e volta à versão anterior. Endereço previsto: `https://192.168.1.20:8444`; dados em `/srv/data/acervo` | Já está provada e publicada; o Acervo só ganha uma porta nova no proxy. |
| D13 | Login do Claude no servidor | **Token** de longa duração (`claude setup-token`) numa variável de ambiente, **fora do git** | Não precisa refazer login a cada deploy ou reinício. |
| D14 | Fila de tarefas | Toda chamada ao Claude ou ao `yt-dlp` vira uma **tarefa na fila**, guardada no SQLite. **Uma por vez**, sobrevive a reinício, mostra posição e andamento, pode ser cancelada | As etapas vão crescer (skills, regras) e levar minutos; a fila evita sobrecarregar o servidor e a assinatura. |
| D15 | Onde fica cada coisa | **Arquivos** guardam o conteúdo: `cru`, `organizado`, tradução e os Temas. O **SQLite** guarda só metadados: Referências, estados, fonte, fila e a cópia das ligações entre Temas | O texto fica legível e versionável; o banco serve para listar e consultar. |
| D16 | Revisão da integração | O Claude edita sem commit; o app mostra o `git diff`; **aprovar = commit**, **rejeitar = restaurar** | O diff já é a melhor tela de revisão, e o git garante a volta. |
| D17 | Vídeo sem legenda | **Erro visível** na lista de Referências; o processo para. Sem Whisper por enquanto | Decisão do Lucas: ver o caso antes de escolher o que fazer. |
| D18 | Idioma | Fonte em outro idioma: traduz **só o organizado** para português. A tela mostra as duas; o original é **só para consulta** e **não entra no Acervo**, que é todo em português | Revisto em 06/11/2026: misturar idiomas no Acervo vira bagunça. |
| D19 | Ler e editar o Acervo | **No app:** Temas com sumário de capítulos navegável, e **edição manual do Markdown**. Edição salva vira commit | O Acervo é para consultar, não só para guardar. |
| D20 | Taxonomia | **Pastas** (subtema = pasta dentro do Tema) + **links** entre índices; onde cada Tema mora é decidido no cadastro; IA sugere. Uma Referência pode alimentar vários Temas. Detalhes na seção 2 | Revisto em 06/11/2026: mais simples que um grafo e cobre o caso "principal aqui, subtema ali" com links. |
| D21 | Offline | **PWA:** na rede, lê sempre a versão do servidor; fora dela, lê e **edita** os Temas que eu **escolher baixar, um por vez**. Ao voltar, sincroniza **por arquivo**: se o arquivo não mudou no servidor, a edição entra; se mudou, o app mostra as duas versões para eu escolher | Igual à Biblioteca, mas texto do Acervo não pode valer "a última edição": uma integração feita no servidor não pode ser apagada sem eu ver. |
| D23 | Página do Tema | **Uma página com capítulos** (`## `) e sumário gerado pelo app; o Lucas **separa manualmente** um capítulo grande em página própria (e pode trazer de volta) | Pedido do Lucas em 06/11/2026: os tópicos em arquivos separados ficavam curtos demais. |
| D25 | Instruções e sugestões | As **instruções da IA** (`regras/`) podem ser lidas e editadas no app, em **Ajustes**; a versão editada fica no servidor e vale no lugar da do repositório, com "Voltar ao padrão". **Sugestões de melhoria** ficam no app, e o Claude Code lê, faz e responde por `/api/sugestoes` | Pedido do Lucas em 06/11/2026: ver e ajustar pelo celular como o Claude organiza os textos, e anotar melhorias enquanto usa o app. |
| D24 | Visual e entrada pelo celular | **Design system comum** às apps ([DESIGN-SYSTEM.md](DESIGN-SYSTEM.md)): neutros, azul e terracota só como contraste, vidro com blur, listas finas. Vídeo entra por **Compartilhar** (Android: `share_target`; iPhone: atalho do app Atalhos) e vai direto para a fila | Pedido do Lucas em 06/11/2026: identidade clean e moderna igual na Biblioteca e no Acervo; mandar vídeo sem abrir tela. |
| D22 | Armazenamento | Só o **disco interno do M920q**. **SSD externo descontinuado** até o Lucas avisar que está disponível | Decisão de 05/11/2026. |

### Consequências do engine (D3)
- Sujeito aos **limites de uso da assinatura** (a fila de D14 ajuda a controlar).
- A imagem precisa do **Claude Code instalado**; o login vem do token de D13.
- Reserva: **API do Claude (Messages API)** se a assinatura não atender.
- O Claude roda com as **ferramentas limitadas** à pasta de trabalho da tarefa.

### Armazenamento (D22)
- Tudo no **disco interno do M920q**, em `/srv/data/acervo` na VM `apps`: o SQLite, as Referências e
  o git do conteúdo.
- **Backup:** sem destino ainda (o SSD externo saiu do plano). Mesmo risco aceito na Biblioteca
  Digital: fica para depois das funcionalidades.

### Regras das etapas
- As regras de cada etapa (organizar, traduzir, criar Tema, integrar...) ficam **neste repositório**,
  como `CLAUDE.md` e skills que o `claude -p` lê. São código, não conteúdo.
- **Transcrições de teste não entram no git:** esta repo é pública.

---

## 5. Pontos difíceis e riscos (honestidade)

- **Integração / análise de lacunas (o coração):** factível com Claude, mas depende da estrutura
  (capítulos = âncoras de encaixe) e tem o **gate humano** como salvaguarda. A IA pode
  propor redundância ou errar o encaixe — por isso eu aprovo.
- **Checagem de veracidade (a parte mais difícil):** *IA verificando com o próprio conhecimento é
  fraca* — é onde mora a alucinação. Checagem séria exige **fontes externas confiáveis** (bases
  públicas, livros) = problema de busca à parte, possível **app dedicado depois**.
  **Abordagem recomendada (faseada):** começar leve — a IA **marca** afirmações checáveis + **nível
  de confiança** + sugere **o que/onde conferir**; eu decido. A versão automática contra bases
  autoritativas fica adiada. Faz mais sentido para conteúdo **factual/histórico** (datas, nomes,
  eventos) do que para opinião/análise.
- **Backup:** git versiona o histórico, mas **versionar ≠ backup** e o conteúdo **não vai pro
  GitHub**. Hoje não há segunda cópia (D22). Urgência baixa até o Acervo acumular conteúdo real.
- **Reorganização:** força da IA, mas pode errar atribuição/perder nuance → manter o `cru`
  preservado permite re-derivar.
- **Escala:** a estrutura multi-arquivo adia o problema por muito tempo; só um Tema gigante traria
  de volta a necessidade de busca.
- **`yt-dlp`:** o YouTube muda e quebra o `yt-dlp` com frequência. O deploy precisa instalar sempre
  a versão mais recente.
- **Sincronização offline (D21):** editar fora da rede e integrar no servidor ao mesmo tempo gera
  conflito. A regra por arquivo evita perda, mas pede uma tela de escolha.

---

## 6. Adiado (decidido que fica para depois)

| Item | Observação |
|------|------------|
| Cenário "estudar um assunto do zero" (pesquisa automática → vídeos/livros/artigos) | Fora do foco atual. |
| **Whisper** (transcrição de alta qualidade) | Vídeo sem legenda hoje só mostra erro (D17). |
| Fontes além do YouTube (artigos etc.) | Arquitetura já prevê plugável. |
| App dedicado de verificação contra bases externas | Depois da versão leve de checagem. |
| **API do Claude** como engine | Reserva, se a assinatura não atender. |
| **SSD externo** | Descontinuado; volta quando o Lucas avisar (D22). |

---

## 7. Ordem de desenvolvimento

Definida em 05/11/2026. Itens 0 a 6 e 8 construídos em 06/11/2026 (ainda não publicados);
andamento em [`CONSTRUCAO.md`](CONSTRUCAO.md). **Próximo passo:** o Lucas revisar, publicar e
decidir a verificação (item 7).

0. ✅ **Validação manual (D5)**, feita em 06/11/2026 com 3 vídeos do canal Buenas Ideias e do
   Nostalgia (21 min, 42 min e 1h20). Regras em [`regras/`](../regras/). Resultado: `organizar` leva
   1 a 2,5 min por vídeo e mantém ~75% do texto (limpa sem resumir); `criar-tema` leva ~1,5 min.
   A seção "Dúvidas" pega erros de transcrição e de fala — semente da verificação. Nenhum dos três
   tinha capítulos na descrição. Falta testar um vídeo em inglês (regra de tradução).
1. **Esqueleto:** mesma base da Biblioteca (API, PWA, SQLite, `compose`, `deploy.sh`, testes), com
   Claude Code e `yt-dlp` na imagem, o token de D13 e a porta 8444 no Caddy.
2. **Fila + extração:** colar a URL, a tarefa entra na fila, o `yt-dlp` gera o `cru`. Sem legenda,
   erro visível na lista.
3. **Organizar + traduzir:** `claude -p` com as regras do item 0; tela da Referência com original e
   tradução.
4. **Primeiro Tema:** criar um Tema novo a partir de uma Referência; git do conteúdo; ler o Tema no
   app com sumário de capítulos.
5. **Edição manual** do Markdown e **painel de Temas** (papel e ligações, D20).
6. **Classificação e integração:** a IA sugere o Tema, eu escolho a versão, o Claude edita, eu reviso
   pelo diff.
7. **Verificação de veracidade** (versão leve, seção 5).
8. **Offline** com sincronização por arquivo (D21).
9. **Segurança, por último:** token de acesso e backup.

---

## 8. Em aberto (a definir antes/durante o desenvolvimento)

- [ ] **Regras de cada etapa** (organização, formato dos capítulos, o que o índice mostra, como citar
      referências/livros/autores sem inventar) → viram o `CLAUDE.md` e as skills. **É o que define a
      qualidade.** Começa no item 0 da seção 7.
- [ ] **Como fazer a checagem de veracidade** (fontes, método, interface de decisão).
- [ ] **Separação de repositórios:** código da app (GitHub) × conteúdo do Acervo (git local na VM).
      Confirmar.

---

## 9. Alternativas avaliadas e descartadas (para não reavaliar à toa)

- **SurfSense** (https://github.com/MODSetter/SurfSense) — self-hosted via Docker, cobre YouTube +
  pesquisa + respostas citadas. **Descartado** para este projeto: plugins oficiais exigem licença e
  dá menos controle sobre o prompt e sobre a estrutura do Acervo. Mantido como referência.
- **NotebookLM** (Google, grátis) — bom para validar qualidade com citações; não é self-hosted.
- **Fabric** (CLI, Daniel Miessler) — ótimo para extrair ideias/referências de um vídeo; não
  gerencia uma base de conhecimento.

---

## 10. Contexto do ambiente (homelab)

- **Proxmox VE 9.2** no **M920q** (i7-8700T, **sem GPU**) — IP `192.168.1.10`.
- **VM `apps`** (Debian 13 + Docker Compose) em `192.168.1.20`: roda a app web e o Claude Code
  headless, atrás do Caddy. Detalhes em `README.md` e `PLANO-BASE.md`, na pasta `homelab/`.
- **Mac mini M4** — bom para inferência local (Whisper/Ollama) quando necessário.
- **Armazenamento:** só o **disco interno do M920q** (D22).
- Serviços do homelab ficam **LAN-only, não expostos à internet**.
