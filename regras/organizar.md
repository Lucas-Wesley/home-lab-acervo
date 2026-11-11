# Regras: organizar uma Referência

Você recebe `cru.md`: a transcrição bruta de um vídeo, de um episódio de podcast ou de um Reel do Instagram, com marcas de tempo `[mm:ss]` a cada ~30 s. Escreva `organizado.md`: o mesmo conteúdo, organizado para alguém que **não sabe nada** do assunto entender.

## O que fazer

1. **Ler a transcrição inteira antes de escrever.** Quem fala num vídeo volta a assuntos, completa no minuto 20 o que começou no minuto 5. Junte cada assunto num lugar só.
2. **Ordem cronológica dos fatos**, não a ordem em que foram falados. Se a ordem cronológica atrapalhar o entendimento (ex.: um conceito precisa vir antes da história), use a ordem mais didática.
3. **Remover o que não é conteúdo:** publicidade, patrocinadores, pedidos de inscrição/like/sininho, agradecimentos, piadas e conversa paralela, comentários sobre a gravação ou o canal.
4. **Manter tudo o que é conteúdo.** Isto não é um resumo: é a base de conhecimento. Datas, nomes, lugares, números, causas e consequências, citações e explicações ficam.
5. **Escrever em prosa clara**, em parágrafos, **no idioma da transcrição** (um vídeo em inglês é organizado em inglês; a tradução é outra etapa). Lista só quando o conteúdo for de fato uma lista.

## O que não fazer

- **Não acrescentar fatos** que não estão na transcrição, nem do seu conhecimento. O texto é o que o vídeo diz.
- **Não apagar opiniões, mas atribuí-las:** "segundo Eduardo Bueno, ...". Fato e opinião não se misturam.
- **Erros de transcrição automática:** corrija só quando o contexto deixar claro (nome conhecido escrito errado, palavra trocada). Se não tiver certeza, mantenha e marque com `[?]`. Uma correção clara ("BigCarrie" numa lista de data warehouses é o BigQuery) entra corrigida, sem `[?]` e sem ir para Dúvidas.

## Formato de `organizado.md`

```markdown
# <título do assunto, não o título do vídeo>

> <2 ou 3 frases dizendo do que o texto trata>

## <seção>

<parágrafos>

<sub>Trechos: [mm:ss], [mm:ss]</sub>

## ...

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Citados no vídeo

- <livros, autores, documentos, pessoas citadas como fonte; só os que o vídeo menciona>

## Referências do episódio

- <só num podcast: o que veio na descrição do episódio; ver "Podcast" abaixo>

## Dúvidas

- "<trecho como está>" [mm:ss]: <o que ficou incerto; omitir a seção se não houver>
```

- Cada seção termina com as marcas de tempo de onde veio o conteúdo, para conferir no vídeo.
- **Linha do tempo:** só se o assunto tiver datas ou períodos. Os itens em **ordem cronológica** (do mais antigo ao mais recente, não na ordem do texto) e só com o que o texto diz.
- **Citados no vídeo:** omitir a seção se o vídeo não citar nenhuma fonte.
- **Dúvidas:** cada uma com o trecho entre aspas, como está na transcrição, e **sempre** a marca `[mm:ss]` do momento da fala (a do bloco onde está). O app usa a marca para ouvir de novo o áudio daquele ponto e resolver a dúvida.

## Podcast

Quando o cabeçalho de `cru.md` tem `- Podcast:`, é um episódio de podcast transcrito do áudio (Whisper):

- Num podcast quase sempre há mais de uma pessoa falando, e a transcrição não diz quem fala. Atribua opiniões e histórias pessoais pelo nome só quando a conversa deixar claro quem é; senão, "um dos apresentadores".
- Troque "Citados no vídeo" por **"Citados no episódio"**.
- A seção `## Descrição do episódio`, no fim de `cru.md`, não é fala: é o texto que o podcast publicou com o episódio. Não a misture ao conteúdo. Ela vira a **última seção do texto**, `## Referências do episódio` (só `## Dúvidas`, que o app mostra numa aba à parte, vem depois), com:
  - os **livros indicados**, com título e autor como aparecem (não invente o autor que não está lá) e o link, se houver;
  - os **links** para fontes, documentos, filmes, artigos e outros episódios citados, com o texto do link;
  - as **imagens** da descrição, no mesmo Markdown (`![...](...)`), e a capa do episódio (`- Capa:` no cabeçalho) como `![Capa do episódio](<endereço>)`.
  - Fora: pedidos de apoio, patrocínio, cupons e redes sociais dos apresentadores.

## Reel do Instagram

Quando o cabeçalho de `cru.md` tem `- Reel:`, é um vídeo curto do Instagram. O `cru.md` tem três partes, e o texto organizado junta as três num conteúdo só:

- `## Fala`: o que é dito, transcrito do áudio (Whisper). Pode não haver fala (só música). Se houver música com letra, ou frases soltas sem relação com o resto, é ruído da transcrição: fica de fora.
- `## Na tela`: o que aparece no vídeo e não está na fala (texto sobreposto, listas, código, o que é mostrado), lido dos quadros, com a marca `[mm:ss]` de cada um. **É conteúdo como a fala**: entra no texto, no lugar do assunto a que pertence. Um texto copiado da tela continua entre aspas quando a forma exata importa (código, nome, citação).
  - **A tela é uma descrição de imagens, feita por IA: vale menos que a fala e a legenda.** Use o texto copiado da tela (títulos, números, nomes, listas) e o que é mostrado, mas não transforme em afirmação a interpretação de uma imagem (o que uma seta, uma cor ou um destaque "significa") se a fala ou a legenda não disserem o mesmo.
  - **Listas de etiquetas lidas da tela** (os nomes num mapa, os itens de um painel): entram só as que explicam o assunto, como os exemplos de cada grupo. As de uma imagem geral ("o mapa mostra BR-210, BR-174, BR-319…" na abertura) ficam de fora.
  - **Ícones, logotipos e imagens sem texto** não entram no texto, nem comentários sobre o que a tela não mostra.
  - **A mecânica do vídeo** (o que aparece, some ou se acende na tela, a animação) não é conteúdo: escreva o que ela mostra, não como mostra.
  - **Fala, tela e legenda em conflito** (um número, um nome, uma regra): siga a fala; se a legenda ou a tela disserem outra coisa, ponha em `## Dúvidas`, com a marca do momento e as duas versões. **Exceção:** quando a legenda corrige a fala de propósito ("Obs: o sapo COAXA", "correção:", "na verdade é…"), vale a legenda, e o texto diz que o autor corrigiu. Correção que já está resolvida assim não vai para Dúvidas.
- `## Legenda do post`: o texto que o autor publicou com o Reel. O que nela for conteúdo (uma explicação, uma lista, a fonte de um dado) entra no texto; hashtags, pedidos de seguir/salvar/comentar, "link na bio", marcações de contas e iscas como "comenta GOAT que eu te mando o kit" (na legenda ou na tela) ficam de fora. Livros, autores, estudos, sites e outras **fontes** citadas vão para **"Citados no Reel"** (no lugar de "Citados no vídeo"). Os exemplos do próprio assunto (as rodovias de um Reel sobre rodovias, os pratos de um Reel de receita, as ferramentas de um Reel de programação) não são fontes: ficam no texto. Sem fonte citada, a seção não existe.

Um Reel é curto: o texto organizado também é, e pode ter uma seção só. As marcas `[mm:ss]` no fim das seções valem para a fala e para a tela.
