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
| D26 | Mais informações | Selecionar um trecho de um Tema ou de uma Referência mostra uma barra com **Mais informações** e **Copiar**. A IA **pesquisa na internet** sobre o trecho, no sentido em que ele aparece no texto, seguindo `regras/detalhar.md` (editável em Ajustes); o trecho fica **sublinhado** e o toque abre um **modal** com a explicação e as fontes. A marcação fica **fora do texto** (tabela `detalhes`) | Pedido do Lucas em 06/11/2026: saber mais sobre um autor, um lugar ou um conceito citado por cima, sem sair da leitura. |
| D27 | Pesquisador | Em Referências, **Pesquisar um tema**: o Lucas escreve o tema e escolhe as **fontes** (Meus livros, Wikipédia, YouTube, Sites; nenhuma = só a resposta da IA) e a **profundidade** (rápida: 1 a 3 perguntas; detalhada: 4 a 8 e conferência), e se quer **revisar o esboço** antes de buscar. Etapas na fila: `planejar` (esboço com as perguntas e as palavras de busca de cada fonte) → `coletar` (o servidor busca nos livros pela API da Biblioteca, na Wikipédia pela API dela e no YouTube pelo `yt-dlp`, e guarda tudo em `fontes/`, sem IA) → `redigir` (com fonte em cada seção; só com "Sites" a IA tem `WebSearch`/`WebFetch`, e guarda em `fontes/` o que usou) → `conferir` (só na detalhada, sem internet: cada afirmação contra o material guardado) → `sugerir`. O resultado é uma **Referência do tipo pesquisa**, que segue o caminho de sempre (criar ou integrar, revisão pelo diff). Os vídeos servem só de fonte e não viram Referência. Instruções em `regras/pesquisar-esboco.md`, `pesquisar.md` e `conferir.md`, editáveis em Ajustes | Pedido do Lucas em 06/11/2026: produzir conhecimento sobre um tema (cozinha e outros assuntos), escolhendo na hora até onde a pesquisa vai. A coleta sem IA não gasta a assinatura e deixa o material guardado como prova; a conferência só checa o que está guardado. Limite de tempo da IA: 20 min na rápida e 50 na detalhada, por etapa. |
| D28 | Flashcards de um texto | **Gerar flashcards** no Leitor (botão no cabeçalho: o Tema inteiro, ou a página do capítulo separado; e no menu ⋯ de cada capítulo) e na página da Referência (organizada, validada ou integrada; nunca o texto cru, D31). A pesquisa ganha a opção **Gerar flashcards do resultado**: pronta, o texto vai sozinho. O Acervo só monta e manda o texto ao Flashcards (`POST /api/textos`, pela rede interna, `FLASHCARDS_URL`): o Tema inteiro leva os capítulos separados no lugar do link e deixa de fora "Citados" e "Referências"; vão junto o caminho no Acervo, o resumo do Tema, o link de volta e o nome do baralho (o do Tema principal; numa Referência integrada, o do Tema onde ela entrou). A IA roda na fila do Flashcards, sem internet; nada entra num baralho sem aprovação. Revisão: no Flashcards, em Pedidos à IA; a de uma Referência também na página dela, com aprovar (escolhendo os cartões) e descartar. Descartar a Referência descarta a proposta. Cada envio fica na tabela `flashcards` (mostra "enviado em"); o Acervo só repassa ao Flashcards o que foi enviado daqui. Até 150 mil caracteres por envio: acima disso, por capítulo | Pedido do Lucas em 06/11/2026: ler no Acervo e estudar no Flashcards sem montar cartão à mão. A IA fica no Flashcards (dono dos cartões) e não ocupa a fila do Acervo. Só texto conferido vira cartão, para não decorar erro. |
| D29 | Aprofundar uma pesquisa | Na pesquisa pronta, **Aprofundar** abre o mesmo formulário com as escolhas da última rodada (fontes, profundidade, revisar o esboço) e um campo opcional **"O que mais quer saber?"**. É outra **rodada** na mesma Referência: o texto atual vira `anterior.md` (aba **Versão anterior**), as perguntas já respondidas vão para `anteriores`, o material das rodadas anteriores fica em `fontes/` e **não é buscado de novo** (artigos e vídeos já vistos são pulados). Em toda pesquisa, o esboço recebe a lista dos Temas do Acervo, pode ler os arquivos deles e aponta até 3 **relacionados** (mais os Temas em que a Referência já entrou); esses Temas são copiados para `ja-escrito/` antes de escrever. O texto novo é completo e ganha a seção **"O que há de novo"** (Novo, Confirmado, Diferente) contra `anterior.md` e `ja-escrito/`, que não contam como fonte. A sugestão de Tema é pedida de novo | Pedido do Lucas em 06/11/2026, depois de testar o Pesquisador: começar simples e aprofundar depois, acrescentando uma fonte, e levar em conta o que já foi escrito e pesquisado, para validar o que há de novo e ir atrás do que falta. Mesma Referência, e não uma nova, para o caminho de sempre (sugerir → integrar) receber o texto completo; a integração já só acrescenta as lacunas. |
| D30 | Resumo de capítulos | Em Referências, **Resumir capítulos**: o Lucas escolhe um livro da Biblioteca e um ou mais capítulos (no PDF sem sumário, as páginas) e, se quiser, o que mais interessa nele. O servidor lê o texto pela API da Biblioteca (R33 de lá: `/api/books/{id}/chapters` e `/text`, pelo índice da busca), sem IA, e guarda em `cru.md` com o link de cada página (etapa `ler_livro`). A IA resume (etapa `resumir`, até 30 min) seguindo `regras/resumir-livro.md`, editável em Ajustes: fica o que o livro **ensina** (fatos, argumento do autor atribuído, dados, técnica com as quantidades exatas, um exemplo por ideia), saem repetição, retórica, histórias que só ilustram e o lixo da extração; assuntos fora do capítulo vão para **"Outros assuntos"**; cada seção leva `<sub>Páginas: [p. N](link)</sub>`; mira em um quarto do original. Vira uma Referência `tipo = 'livro'` (coluna `livro`, JSON) que segue o caminho de sempre, com a sugestão de Tema já pedida; ao criar ou integrar, a fonte no Tema é o livro e a página | Pedido do Lucas em 06/11/2026: tirar de um ou mais capítulos o conteúdo resumido, porém preciso, e sugerir o Tema existente ou novo. Uma instrução própria porque um livro não é uma transcrição: o autor repete, faz digressões e argumenta, e o que importa é a substância, com a página para conferir. Teste real (capítulo 4 de "Uma breve história da religião", 12 mil caracteres): resumo fiel com citações e páginas, 63% do tamanho na segunda versão da instrução, dúvidas sobre o que o autor afirma sem fonte e a sugestão "criar o Tema Hinduísmo". |
| D31 | Flashcards: baralho e link de volta | **Gerar flashcards** abre uma janela para escolher o baralho do Flashcards: um que já existe (começa no que tem o nome do Tema principal, se houver; os cartões de mesma pergunta são atualizados) ou **Baralho novo**, com o nome que o Lucas der (sugerido: o do Tema). Os baralhos vêm de `GET /api/flashcards/baralhos`, que repassa os do Flashcards e dá o nome sugerido. Na pesquisa, a opção **Gerar flashcards do resultado** começa sempre desmarcada e, marcada, pede o baralho (novo sem nome = o título da pesquisa). Uma Referência vira flashcards já **organizada** (vídeo também, sem precisar validar). Cada capítulo do texto vai com o link dele (`secoes`): o capítulo na página do Tema abre com `#/ler/<arquivo>?capitulo=<título>` (o Leitor rola até ele), o separado abre a página própria; o cartão guarda o link do capítulo de onde saiu | Respostas do Lucas em 06/11/2026 às perguntas da D28. A conferência do texto passa a ser a revisão dos cartões, que continua obrigatória. |
| D32 | Página de resumir capítulos | **Resumir capítulos** deixou de ser um formulário com `select` e virou uma página (`#/resumir`): a lista dos livros da Biblioteca com capa pequena, título, autor, formato (EPUB/PDF) e quanto foi lido, os **lidos por último primeiro** ("Lendo agora") e busca por título ou autor; tocar num livro abre `#/resumir/{id}`, com o sumário em linhas marcáveis (Marcar todos), o foco opcional e uma barra de vidro fixa no pé com o que foi marcado e **Resumir**. A capa passa pelo Acervo (`GET /api/livros/{id}/capa`, que repassa a da Biblioteca) | Pedido do Lucas em 06/11/2026: o `select` era ruim de usar. |
| D33 | Ações da Referência num menu ⋯ e Excluir no lugar de Descartar | O topo da Referência fica com título, metadados e estado; as ações (**Gerar flashcards**, **Sugerir Tema**, **Criar Tema**, **Integrar em Tema**, **Aprofundar** na pesquisa e **Excluir**) vão para o menu ⋯ ao lado do título. Fica à vista só **Tentar de novo**, quando algo falhou, e o cartão de Flashcards aparece só depois de um envio. **Marcar como validada** e **Descartar** saíram (as ações `validar` e `descartar` da API também); **Excluir** (`DELETE /api/referencias/{id}`, com confirmação) apaga a Referência, a pasta dela, o histórico, as marcações de "Mais informações" e os envios ao Flashcards, descarta as propostas pendentes (do Acervo e de cartões) e recusa enquanto houver tarefa na fila ou rodando. O que já entrou nos Temas fica. As Referências já descartadas continuam no filtro Descartadas e podem ser excluídas | Pedido do Lucas em 07/11/2026: o começo da página estava poluído, e validar e descartar não estavam sendo úteis. |
| D34 | Artigos e páginas da web; Referências mais limpa; Pesquisar numa página; Compartilhar sem abrir o app | **Qualquer link** vira Referência: YouTube é vídeo; qualquer outra página é **artigo** (`tipo = 'artigo'`). A etapa `extrair` baixa a página pelo servidor, sem IA (`artigo.go`): os metadados (título, site, autor, data, idioma do `<html lang>`), e o texto principal escolhido como no "modo leitura" dos navegadores (o elemento com mais texto em parágrafos, subindo até juntar a maior parte; ficam de fora menus, rodapé, comentários, compartilhar, propaganda), em Markdown simples, em `cru.md`. O link vai sem `#`, `utm_…`, `fbclid` e `gclid`, e a mesma página não entra duas vezes. Página que pede login ou monta o texto por JavaScript, PDF ou erro do site: erro visível na Referência. Daí em diante, o caminho do vídeo: organizar com `regras/organizar-artigo.md` (editável em Ajustes; sem marcas de tempo), tradução se for outra língua, sugerir, criar ou integrar (a fonte no Tema é `<sub>Fonte: id</sub>` e a página vai em "Citados"), flashcards. Na tela **Referências**, só o título e o menu **⋯**: **Link de vídeo**, **Link de artigo ou site** (abrem o campo do link), **Pesquisar um tema** e **Resumir capítulos**. **Pesquisar um tema** virou a página `#/pesquisar`, como Resumir capítulos: cada fonte e cada profundidade numa linha com a explicação, as opções e a barra fixa no pé com a escolha e **Pesquisar**. **Compartilhar** no Android abre `/compartilhar`, uma tela curta (sem o app) que manda o link (vídeo ou página) e se fecha, voltando ao app de onde se compartilhou; o endereço antigo (`/?text=`) cai na mesma tela. Na **extensão do Chrome** (E10 de lá), qualquer página pode ir ao Acervo ou virar flashcards | Pedido do Lucas em 08/11/2026: mandar blogs e artigos para o Acervo e gerar flashcards deles; a tela de Referências estava ficando carregada; o Pesquisar precisava de espaço; compartilhar não devia abrir o app. O navegador só deixa uma página fechar a própria janela quando ela foi aberta só para isso (histórico de uma página): se o Acervo já estava aberto no celular, a tela fica com o resultado e o "voltar". |
| D35 | Perguntar e verificar | Num **Tema** (botão ? no cabeçalho do Leitor) ou numa **Referência** (aba **Perguntar**, ao lado de Dúvidas), **Perguntar** mostra o campo da pergunta e a lista das já feitas (no Tema, numa janela). Na Referência, as abas ficam Organizado (ou Português e Original), Dúvidas, Perguntar e **Estudar** (reservada, a definir); a transcrição (ou o material, o texto do livro ou da página) e o Histórico abrem pelo menu ⋯, e o menu ⋯ deixou de ser translúcido, para ter contraste. A tarefa `perguntar` (fila) copia o texto para uma pasta à parte (a página do Tema e os capítulos separados, sem os subtemas; na Referência, o texto em português) e a IA, com `Read` e `WebSearch`/`WebFetch` e sem escrever nada, segue `regras/perguntar.md` (editável em Ajustes): **quanto o texto responde** (responde, em parte, toca no assunto, não responde) e o que ele diz; a **resposta verificada**, sempre pesquisada, mesmo quando o texto responde, porque é assim que aparece o que está errado ou desatualizado; e as **divergências** com o texto (erro, desatualizado, falta no texto, pode melhorar), com o capítulo, o trecho e o que as fontes dizem. **O verificador:** fontes por ordem de confiança (primárias e oficiais; enciclopédias e universidades; imprensa reconhecida; blogs e fóruns só como pista), fontes que copiam a mesma origem contam como uma, dado que muda com o tempo busca a fonte mais recente, e cada afirmação ganha um **selo**: confirmado (duas fontes independentes), uma fonte só, as fontes divergem, não encontrado. Com o texto respondendo por inteiro e sem divergência, a resposta da IA fica recolhida. As perguntas ficam na tabela `perguntas` (fora do texto) e saem com a Referência excluída | Pedido do Lucas em 10/11/2026: perguntar sobre o que já tem no Acervo e conferir com a internet se o texto está certo, completo e atualizado. Uma tarefa só (ler e pesquisar na mesma chamada) gasta menos da assinatura e compara com a mesma leitura dos dois lados. Grau em níveis, não em %, porque a IA não tem essa precisão. |
| D36 | Integrar ao texto | Numa pergunta respondida (D35), **Integrar ao texto** abre as escolhas: levar ou não a **resposta verificada** e, em cada divergência, **Substituir**, **Manter as duas versões** ou **Ignorar** (no que é erro ou desatualizado) ou **Acrescentar** ou **Ignorar** (no que falta ou pode melhorar). A escolha vem marcada com o mais comum, mas é sempre do Lucas. No modal de **Mais informações** (D26), **Integrar ao texto** tem **o texto completo, como está aqui** ou **só o que serve ao texto**. A tarefa `integrar_texto` (fila) escreve `material.md` com o que integrar e as escolhas, e a IA segue `regras/integrar-texto.md` (editável em Ajustes): encaixa no capítulo e no ponto certos (ordem cronológica ou didática), nunca colado no fim; **substituir o que já estava escrito só quando o Lucas escolheu Substituir**; fontes da internet com link em `<sub>Fontes: …</sub>` e em Citados. Tudo vira **proposta para revisar pelo diff** (D16): num Tema, na worktree do git (só dentro da pasta do Tema); numa Referência, que fica fora do git, numa cópia do texto em `revisoes/texto-<tarefa>/`, com o diff no mesmo formato, e, ao aprovar, o texto de antes vira `anterior.md` (aba Versão anterior) | Pedido do Lucas em 10/11/2026: "mostra para mim e me dê essa opção de realmente substituir e trocar", não uma regra fixa. A proposta revisada mantém a regra de que a verificação termina em decisão humana. |
| D37 | Dúvidas resolvidas pelo áudio e pela internet | A aba **Dúvidas** lista cada item de "## Dúvidas" com **Resolver** (um) e **Resolver as dúvidas** (todos os que faltam; serve às Referências antigas). A tarefa `resolver_duvidas` (fila) acha o **momento** de cada dúvida: a marca `[mm:ss]` escrita nela (a regra de organizar passou a exigir) ou, nas antigas, o bloco da transcrição que contém o que ela cita entre aspas. Corta o áudio ali (−20 s a +25 s, até 2 min; vídeo pelo `yt-dlp --download-sections`, podcast pelo ffmpeg no endereço guardado) e manda ao **Whisper large-v3 do Mac mini** (o do servidor de reserva), com o título e a dúvida como contexto. A IA (`regras/resolver-duvidas.md`, com `WebSearch`/`WebFetch`, sem escrever) compara a transcrição, o áudio e a internet: **resolvida pelo áudio**, **confirmada na internet**, **continua incerta** ou **falta no material**, com o que se ouve, a explicação, a correção (o trecho como está e como fica) e as fontes. O Lucas decide: **Corrigir no texto** vira proposta revisada (D36, e a dúvida sai da lista), **Pesquisar isto** vira pergunta (D35) ou, numa pesquisa, abre o Aprofundar com a dúvida como foco. Resultados na tabela `duvidas`. **Vídeo sem legenda** (D17) não para mais com erro: o yt-dlp baixa o áudio e o Whisper transcreve o vídeo inteiro. O Parakeet como segunda leitura ficou de fora: o large-v3 do Mac erra bem menos (medido na D39) | Pedido do Lucas em 10/11/2026: a maioria das dúvidas é erro da legenda automática. |
| D38 | Livros da Biblioteca marcados por assunto | Com a fila parada (conferido a cada 10 min), o Acervo pega da Biblioteca um livro ainda não marcado (`marcarAosPoucos`): os novos entram sozinhos e os antigos aos poucos, sem mudar nada na Biblioteca. A tarefa `marcar_livro` divide os capítulos em lotes de ~60 mil caracteres e marca **um lote por tarefa**, pondo o próximo no fim da fila, para um livro grande nunca segurar o que o Lucas pede. Um **modelo menor** (Haiku, `pedidoClaude.modelo`), seguindo `regras/marcar-livro.md` (editável em Ajustes), devolve para cada capítulo o resumo do assunto, os **Temas** (a pasta exata quando é um Tema do Acervo), pessoas, organizações, lugares, períodos, eventos e trechos marcantes com a página. As marcas ficam na tabela `livros_marcados`. Usos: no Tema, **Livros da sua biblioteca que falam disto** (capítulo, páginas e link para a Biblioteca); no **Perguntar** sobre um Tema, os capítulos entram como pista de fonte; em Ajustes, **Livros marcados** mostra o andamento. Livro **sem sumário** (PDF sem índice) é lido em blocos de 20 páginas, cada um marcado como um capítulo ("Páginas 21 a 40"). Livro não indexado (PDF escaneado) falha e tenta de novo no dia seguinte | Pedido do Lucas em 10/11/2026. Sem aviso da Biblioteca (webhook): a verificação com a fila parada cobre os livros novos e os antigos com uma peça só. |
| D39 | Podcasts como Referência | Um link de episódio do **Spotify** ou do **Apple Podcasts** (colado, pelo menu ⋯ "Episódio de podcast", ou pelo Compartilhar do celular) vira Referência do tipo `podcast`. O Spotify protege o áudio, mas quase todo podcast publica os episódios num **feed RSS aberto**: o servidor pega o título pelo **oEmbed** do Spotify (a API pública de prévia de links; a página do episódio é montada no navegador e não traz os dados) ou pelo lookup da Apple, acha o podcast e o feed pela **busca de episódios do Apple Podcasts** (grátis, sem chave), acha o episódio no feed pelo título, baixa o MP3 e corta em pedaços de 10 min (ffmpeg). Cada pedaço vai ao **Whisper large-v3 no Mac mini** (`homelab/whisper-mac`, MLX no chip M4, `WHISPER_URL`), com o podcast, o título e o começo da descrição como contexto para os nomes; com o Mac fora do ar (pergunta rápida em `/saude`), vai ao **Whisper do servidor com o large-v3-turbo** (`WHISPER_RESERVA_URL`). O `cru.md` tem as marcas `[mm:ss]` de 30 em 30 s, como o de um vídeo, e termina com a **descrição do episódio** em Markdown (links, listas, imagens); a regra de organizar a transforma na última seção, **Referências do episódio** (livros indicados, links, imagens e a capa), sem apoio, patrocínio e redes sociais. O áudio não fica guardado: só o endereço, em `info.json`. Episódio exclusivo do Spotify (sem feed aberto) para com o aviso | Pedido do Lucas em 10/11/2026. **A escolha do modelo foi medida** com um podcast em português (3 trechos de 2 min de conversa, com nomes próprios): o Whisper small (o que o servidor usava) foi o pior ("Zeferin Ferès", "sem baixão salgado"); o Parakeet v3 acerta os nomes mas põe palavras em inglês no meio ("Yeah", "I'm just uh"); o large-v3-turbo e o large-v3 foram os melhores, e só o large-v3 acertou "Marc Ferrez" e "a foto da minha vida". No servidor (só processador) o turbo leva ~1,2× a duração do áudio; no Mac, o large-v3 leva ~1/6 (38 min de episódio em ~6 min). Benchmarks publicados divergem para português (o Voxtral, o Qwen3-ASR e o Canary ficaram abaixo do Whisper large-v3 num estudo de 2026 com línguas ibéricas). |
| D40 | Reels do Instagram como Referência | Um link de **Reel** (`instagram.com/reel/…`, colado pelo menu ⋯ "Reel do Instagram" ou pelo Compartilhar do celular) vira Referência do tipo `reel`; só Reels (carrossel e post com fotos ficam para depois). O **yt-dlp** baixa o vídeo (até 720p) e o `info.json` (conta, legenda do post, data); Reel público vem **sem login**. Quando o Instagram pede login, o yt-dlp usa os **cookies de uma conta secundária** (`DATA_DIR/instagram-cookies.txt`, levados do navegador do Mac por `scripts/cookies-instagram.sh`, passo a passo em [INSTAGRAM.md](INSTAGRAM.md)), e o erro diz se faltam ou se venceram. A fala vai ao **Whisper large-v3 do Mac** (D39), sem idioma marcado (o do Reel vem do que o Whisper reconhece); os trechos que o próprio Whisper marca como **sem fala** (`no_speech_prob` ≥ 0,5) ou repetidos demais (`compression_ratio` > 2,4) saem, porque sobre música ele inventa texto ("ご視聴ありがとうございました", "stressstress…"); o `whisper-mac` passou a devolver esses campos, como a API da OpenAI. O **ffmpeg** tira um quadro a cada mudança de cena (no máximo um por segundo) e, sem mudança, um a cada 2 s (até 60), e o **Claude lê os quadros** (`regras/ver-quadros.md`, editável em Ajustes): texto da tela como está, listas, código e o que é mostrado, sem repetir a fala. O `cru.md` tem **Fala**, **Na tela** (com as marcas `[mm:ss]` dos quadros) e **Legenda do post**; a regra de organizar junta as três num texto só e tira letra de música, hashtags e pedidos de seguir. O vídeo e os quadros não ficam guardados. Dúvidas ouvem de novo o trecho como num vídeo do YouTube (`--download-sections`). O **título** da Referência é o do texto organizado (antes dele, o começo da legenda). Depois do primeiro Reel real (rodovias BR, 10/11/2026): a leitura da tela **descreve sem concluir** (o Claude tinha escrito que a BR-369 era "par"), e no organizar a tela **vale menos que a fala e a legenda**: conflito vai para Dúvidas (menos quando a legenda corrige a fala de propósito: aí vale a legenda), e "Citados no Reel" é só para fontes. Depois de mais três Reels (música erudita, OLTP/OLAP, sons dos animais): a tela não descreve a interface de sites mostrados (abas, créditos de foto), ícones sem nome nem o que não aparece, e correção clara de transcrição não vira Dúvida. Na última rodada (dois quizzes, cinco livros, Osasco, 11/11/2026), a amostra passou de 4 s para **2 s** (até 60 quadros): o cartão "5/5 · Gergely Orosz", 3 s sobre a pessoa falando, caiu entre dois quadros; e a linha do tempo fica em ordem cronológica (a de Osasco saiu fora de ordem); a isca "comenta X que eu te mando o kit" fica de fora, como os pedidos de seguir | Pedido do Lucas em 10/11/2026: muito conteúdo de estudo está em Reels, e boa parte dele só aparece na tela. Projetos parecidos (claude-real-video, instagram-reel-extractor) usam o mesmo caminho (yt-dlp, ffmpeg por cena, Whisper); a regra de "cena ou intervalo fixo" é porque só a cena perde texto que entra sem corte (um Reel da NASA de 42 s, em plano contínuo, deu 1 quadro só por cena). |
| D41 | Aba Estudar no Tema | A página do Tema ganha as abas **Ler** e **Estudar** (`?estudar` no endereço). Estudar tem **Resumo inteligente**, **Conceitos anteriores** (o que ajuda a entender o Tema e não está nele: base ou ao redor, com explicação, fontes conferidas na internet e o link do Tema do Acervo que trata dele), **Simulado** (banco de perguntas abertas; cada simulado sorteia algumas; responde-se digitando ou **falando**, pelo ditado do navegador; a IA corrige na fila com nota, o que faltou e o que reler; fica o histórico) e **Flashcards** (do Tema, dos conceitos anteriores ou do que se errou no simulado, pelo D28). Cada parte é uma tarefa `estudar` (ou `corrigir_simulado`) que lê uma cópia do Tema inteiro, com os subtemas, e guarda o resultado nas tabelas `estudos` e `simulados` com a versão do Tema: se o texto muda, a parte avisa. Regras: `estudar-resumo.md`, `estudar-anteriores.md`, `estudar-simulado.md`, `corrigir-simulado.md`. O que pode vir depois: [ESTUDAR.md](ESTUDAR.md) | Pedido do Lucas em 10/11/2026. O estudo é derivado e se refaz a qualquer hora, por isso fica no SQLite, não no conteúdo do Acervo. Ditado do navegador (sem Whisper) porque responde na hora, enquanto se fala. |
| D42 | Avisos no celular | Pelo **ntfy** do homelab (`NTFY_URL`, tópico `acervo`; sem a variável, não avisa). Avisa só três coisas: a tarefa que **falhou** (prioridade alta, com o erro), a proposta que espera **revisão** e o trabalho de uma Referência que **parou esperando o Lucas** (a última etapa da cadeia: extrair → organizar avisa uma vez, no fim). Tarefas sem Referência (Perguntar, Estudar…) avisam ao terminar. Tocar abre a Referência, as Revisões ou a Fila (`ACERVO_LINK`) | Pedido do Lucas em 10/11/2026: saber quando o Acervo ficou pronto sem abrir o app. Aviso que não pede ação vira ruído, então nada de "começou" nem de etapa no meio. Nos testes, um ntfy falso (`ntfy_test.go`). |
| D43 | Marcações no resumo de capítulos | Ao ler os capítulos escolhidos (D30), o Acervo recebe também as **marcações** que o Lucas fez neles na Biblioteca (R34 de lá) e as grava em `marcacoes.md`, ao lado do `cru.md`: o trecho, a página com o link e a nota dele. A instrução do resumo (`resumir-livro.md`) trata as marcações como os **pontos-chave do Lucas**: toda marcação entra, com mais espaço; a nota diz o que ele quer lembrar; o resto do capítulo continua, mais enxuto; e o resumo abre com a seção **"Pontos-chave"**, de onde saem primeiro os cartões quando ele pede flashcards (`gerar-do-texto.md` do Flashcards). Sem marcação, nada muda: o resumo segue o peso que o próprio capítulo dá a cada assunto. Na escolha dos capítulos, cada um mostra quantas marcações tem. | Pedido do Lucas em 11/11/2026: ele marca o que acha fundamental ao ler e quer que o resumo use isso como referência. |
| D44 | Pedidos que viram proposta não se repetem | Tudo o que vira proposta para o Lucas decidir é **idempotente**: enquanto a proposta está na fila, rodando ou esperando decisão, pedir de novo devolve a mesma (a resposta traz `ja_pedida`) e o botão some. Vale para **Corrigir no texto** (Dúvidas), **Integrar ao texto** (Perguntar e Mais informações), **Criar/Integrar em Tema** (uma proposta de Tema por Referência em Revisões) e **Gerar flashcards** (um pedido em aberto por texto, conferido no Flashcards). Enquanto a proposta espera, refazer o que a originou (Verificar de novo, Pedir de novo, Resolver de novo) fica bloqueado, porque mudaria o material da proposta. As listas de dúvidas, perguntas e marcações trazem a proposta em aberto (`correcao`/`integracao`: tarefa e estado), e a tela mostra o estado em vez do botão. A **correção de uma dúvida** é pequena: o texto corrigido (só os parágrafos que mudam, sem a seção de Dúvidas) aparece na própria aba, com **Aprovar** e **Descartar**; Revisões continua tendo o diff, para as mudanças grandes. Várias propostas sobre o mesmo texto de uma Referência (três dúvidas corrigidas de uma vez) se **juntam** ao aprovar, como um merge do git (sobre `base.md`, o texto de quando a proposta foi feita): linha por linha e, onde as duas mexem na mesma linha, palavra por palavra, então duas correções no mesmo parágrafo e duas dúvidas vizinhas tiradas da lista entram juntas; só mudanças nas mesmas palavras são recusadas, e a proposta fica para descartar. | Pedido do Lucas em 11/11/2026: na aba Dúvidas do Reel dos sons dos animais, mandar corrigir de novo criava outra revisão igual, e aprovar uma proposta podia apagar a outra (cada uma era uma cópia do texto de antes). |
| D46 | Capa do Reel e do podcast; ícone de flashcard | Na lista das Referências, o **Reel** mostra a capa do post (a miniatura do yt-dlp, `--write-thumbnail`; sem ela, um quadro do vídeo) e o **podcast**, a capa do episódio no feed, as duas reduzidas pelo ffmpeg para `capa.jpg` (320 px) na pasta da Referência e servidas em `GET /api/referencias/{id}/capa`, com um selo do Instagram ou do Spotify. Sem capa, o símbolo do Instagram ou do Spotify nas cores da marca. As Referências de antes ganham a capa numa tarefa só da fila (`buscar_capas`, sem aviso no celular, só a miniatura, sem baixar o vídeo); quem não tem capa fica com `capa.falta` e não é tentado de novo. A capa é um enfeite: se falhar, a extração segue. O ícone de **flashcard** deixa de ser as duas folhas (que é o "copiar") e vira um cartão com a pergunta em cima e a resposta embaixo, no Acervo, no app Flashcards (inclusive o ícone do app) e na extensão. | Pedido do Lucas em 11/11/2026: os Reels e podcasts tinham um ícone genérico, e o de flashcard se confundia com o de copiar. |
| D45 | Estudar em cartões, também nas Referências | A aba **Estudar** (D41) abre só com **quatro cartões** (Resumo inteligente, Conceitos anteriores, Simulado, Flashcards), cada um com um ícone e a situação da parte (pronta e quanto tem, preparando, por fazer, "o texto mudou"). O cartão escolhido abre **embaixo**; o **Simulado** abre em **tela própria** (com "← Estudar"), porque responder pede foco. A mesma aba existe na **Referência** (`#/referencia/<id>?estudar`): as partes saem do texto organizado dela (em português quando há tradução), guardadas com `referencias/<id>` no lugar do Tema; ao excluir a Referência, o estudo vai junto. **Ouvir o resumo** lê em trechos curtos, um depois do outro, com uma voz em português do aparelho, e tem Pausar/Continuar/Parar (`leitura.js`). | Pedido do Lucas em 11/11/2026: a aba mostrava tudo de uma vez, o Ouvir não funcionava (no Chrome, o texto inteiro numa fala só não sai, e um `speak()` logo depois de `cancel()` é descartado) e as Referências tinham só "Em breve". |
| D24 | Visual e entrada pelo celular | **Design system comum** às apps ([DESIGN-SYSTEM.md](DESIGN-SYSTEM.md)): neutros, azul e terracota só como contraste, vidro com blur, listas finas. Vídeo entra por **Compartilhar** (Android: `share_target`; iPhone: atalho do app Atalhos) e vai direto para a fila | Pedido do Lucas em 06/11/2026: identidade clean e moderna igual na Biblioteca e no Acervo; mandar vídeo sem abrir tela. |
| D22 | Armazenamento | Só o **disco interno do M920q**. **SSD externo descontinuado** até o Lucas avisar que está disponível | Decisão de 05/11/2026. |

### Consequências do engine (D3)
- Sujeito aos **limites de uso da assinatura** (a fila de D14 ajuda a controlar).
- A imagem precisa do **Claude Code instalado**; o login vem do token de D13.
- Reserva: **API do Claude (Messages API)** se a assinatura não atender.
- O Claude roda com as **ferramentas limitadas** à pasta de trabalho da tarefa. A internet
  (`WebSearch`, `WebFetch`) só é liberada para "Mais informações" (D26), que roda numa pasta vazia
  e não mexe em arquivo nenhum.

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
| **Whisper** (transcrição de alta qualidade) | Feito na D37 e na D39: Whisper large-v3 no Mac mini. |
| Fontes além do YouTube e páginas da web (PDF, podcast) | Artigos e páginas entraram em D34. |
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
