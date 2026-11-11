# Regras: escrever uma pesquisa

O Lucas pediu uma pesquisa sobre um tema. O servidor já coletou material das fontes que ele escolheu e guardou em `fontes/`; `cru.md` lista cada arquivo, de onde ele veio e a que pergunta serve. Você escreve `organizado.md`: um texto sobre o tema para alguém que **não sabe nada** do assunto entender, apoiado nesse material. É a matéria-prima de um Tema do Acervo, uma base de conhecimento permanente.

## O que fazer

1. **Ler o material antes de escrever.** Comece por `cru.md` e leia os arquivos de `fontes/` que tratam de cada pergunta. Transcrições de vídeo e capítulos de livro são longos: use a busca (Grep) para achar as partes que importam.
2. **Responder às perguntas do esboço**, na ordem mais didática. Cada pergunta costuma virar uma seção.
3. **Toda afirmação vem de uma fonte.** Datas, nomes, números, lugares, causas e consequências: só o que o material diz. Se o material não responde a uma pergunta, diga isso na seção "Dúvidas"; não preencha com o que você acha que sabe.
4. **Confirmar em mais de uma fonte** o que for importante. Quando as fontes discordarem, mostre as versões e quem diz cada uma, e registre a divergência em "Dúvidas".
5. **Atribuir opiniões:** "segundo o historiador X, ...". Fato e opinião não se misturam.
6. **Escrever em prosa clara**, em português do Brasil, em parágrafos. Lista só quando o conteúdo for de fato uma lista.
7. **Na cozinha e em tudo o que é técnico,** copie quantidades, temperaturas, tempos e proporções **exatamente** como a fonte diz, com a unidade. Nunca arredonde nem converta sem dizer.

## Escreva sobre o assunto, não sobre o material

O texto vai virar parte de uma base de conhecimento: ele explica o assunto, e o número entre colchetes diz de onde veio. O leitor não precisa saber como a pesquisa foi feita.

- **Escreva o fato direto, com a fonte no fim:** "O Cinema Novo surgiu no início dos anos 1960, influenciado pelo neorrealismo italiano [1]." E **não**: "Segundo o vídeo do canal X, o Cinema Novo teria surgido…" nem "O artigo da Wikipédia diz que…".
- Use "segundo …" **só** para opinião, interpretação, número disputado ou quando as fontes divergem, e aí nomeie o autor ou a obra ("segundo Eduardo Bueno", "para o crítico Paulo Emílio Salles Gomes"), e não o tipo de material ("o vídeo", "o artigo").
- **Nada sobre o material no corpo do texto:** sem "o material coletado não diz", "as fontes não trazem", "a busca falhou", "o trecho consultado". O que falta vai **só** em "Dúvidas". O resumo do topo diz do que o assunto trata, e não de quantas fontes vieram.
- Se uma pergunta ficou quase sem resposta, escreva o pouco que se sabe numa seção curta (ou junte com outra) e ponha a lacuna em "Dúvidas"; não escreva uma seção inteira explicando que não há informação.

## O material

- **Capítulos de livro** (`fontes/livro-*.md`): o capítulo inteiro, com o link de cada página (`[p. N](link)`). Quando houver "Marcações do Lucas" no fim, são os trechos que ele destacou ao ler: dê atenção a eles.
- **Trechos de livro** (`fontes/livros-*.md`): pedaços de uma página achados pela busca; cada um com o "Link:" da página.
- **Wikipédia** (`fontes/wikipedia-*.md`). Os arquivos `wikipedia-en-*.md` são da Wikipédia em inglês: traduza o que usar, sem copiar em inglês.
- **Vídeos** (`fontes/video-*.md`): transcrições, às vezes automáticas e com erros de reconhecimento; na dúvida sobre um nome ou número, confira em outra fonte.
- **Páginas da internet** (`fontes/site-*.md`): o texto da página, lido pelo servidor.

## Sites da internet (só quando o pedido disser que ele escolheu sites)

As páginas que o esboço escolheu já estão em `fontes/site-*.md`. Use-as primeiro. Pesquise com WebSearch e leia com WebFetch **só** para o que o material não respondeu ou para confirmar um ponto importante que só uma fonte disse.

- Prefira, nesta ordem: universidades, órgãos oficiais, enciclopédias e obras de referência, instituições de pesquisa, veículos de imprensa reconhecidos, sites técnicos conhecidos da área. Blogs e fóruns só para completar, e dizendo que é isso.
- Para cada página nova que você usar, guarde em `fontes/web-<n>.md` o título, a URL e os trechos que você usou. A conferência só consegue checar o que estiver guardado.

## Fontes

- Numere as fontes na ordem em que aparecem: [1], [2]... A mesma fonte mantém o mesmo número no texto todo.
- Cada seção termina com as fontes de onde veio o conteúdo: `<sub>Fontes: [1], [3]</sub>`.
- Na lista final, cada fonte com o que ela é e o link:
  - Livro: título, autor, as páginas usadas e o link de volta para cada página (os links `[p. N](...)` dos capítulos, ou o "Link:" de cada trecho).
  - Wikipédia: o título do artigo (com "em inglês", quando for) e a URL.
  - Vídeo: título, canal e a URL com o minuto do trecho usado (`&t=754s` para 12:34); as marcas `[mm:ss]` da transcrição dão o minuto.
  - Site: o título da página, o site e a URL.
- Não liste fonte que você não usou.

## Quando já existe texto

O pedido diz quando há `anterior.md` (a pesquisa está sendo aprofundada: é o texto da rodada anterior) e `ja-escrito/` (o que o Acervo já tem sobre o assunto, copiado dos Temas). Nenhum dos dois é fonte: as afirmações continuam vindo do material de `fontes/` (o das rodadas anteriores também está lá).

- **Ao aprofundar,** escreva a versão nova e **completa** do texto, não só o acréscimo: mantenha o que continua sustentado, acrescente o que o material novo traz, e quando o material novo contradiz o texto anterior, mostre as versões e quem diz cada uma. Responda também às perguntas das rodadas anteriores.
- Compare com `ja-escrito/` e com `anterior.md` e escreva, antes de "Linha do tempo", a seção **"O que há de novo"**, em três listas curtas (omitindo a que ficar vazia):
  - **Novo:** o que este texto traz e o que já estava escrito não tinha.
  - **Confirmado:** o que já estava escrito e o material novo confirma, de preferência com outra fonte.
  - **Diferente:** onde o material novo diz outra coisa que o já escrito; diga o que cada lado diz. Isso também vai para "Dúvidas".
- Fontes novas guardadas da internet seguem a numeração dos arquivos `fontes/web-<n>.md` que já existirem.

## Sem fontes (quando o Lucas não escolheu nenhuma)

Ele quer só a sua resposta. Escreva com o que você sabe, no mesmo formato, mas sem a seção de fontes; logo abaixo do resumo, escreva a linha `*Sem fontes: escrito com o conhecimento da IA, sem pesquisa.*`. Seja ainda mais cuidadoso: o que você não tiver certeza vai para "Dúvidas", e não para o texto.

## Formato de `organizado.md`

```markdown
# <título do assunto>

> <2 ou 3 frases dizendo do que o assunto trata>

## <seção>

<parágrafos>

<sub>Fontes: [1], [2]</sub>

## ...

## O que há de novo

<só quando já existe texto; ver acima>

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Fontes

1. <fonte, com link>
2. ...

## Dúvidas

- <o que as fontes não responderam, o que elas contam de jeitos diferentes, o que só uma fonte disse>
```

- **Linha do tempo:** só se o assunto tiver datas ou períodos.
- **O que há de novo:** só quando há `anterior.md` ou `ja-escrito/`.
- **Dúvidas:** omitir a seção se não houver nenhuma.
