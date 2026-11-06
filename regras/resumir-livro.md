# Regras: resumir capítulos de um livro

`cru.md` tem o texto de um ou mais capítulos de um livro da biblioteca do Lucas, tirado do próprio livro: cada capítulo é uma seção `## `, e cada página nova começa com o link dela (`[p. 45](...)`). Você escreve `organizado.md`: o **conteúdo do livro, sintetizado**, para entrar no Acervo, uma base de conhecimento permanente.

Não é a resenha do livro nem a sua opinião sobre ele. É o que o livro **ensina** — os fatos, as ideias, os argumentos, os dados, as técnicas —, sem o que é só embalagem.

## O que fica

- **A substância:** fatos, datas, nomes, lugares, números, definições, causas e consequências, conclusões.
- **O argumento do autor,** na ordem em que ele se sustenta: a tese, as razões, as provas que ele dá. Atribua: "segundo o autor, ...", "Fulano defende que ...". Fato e opinião não se misturam.
- **Exemplos que explicam** algo que, sem eles, ficaria abstrato. De vários exemplos para a mesma ideia, fique com **o mais claro**.
- **Técnica e cozinha:** quantidades, temperaturas, tempos, proporções e passos, **exatamente** como o livro diz, com a unidade. Nunca arredonde nem converta sem dizer.
- **Citações literais** só para uma formulação importante do autor, curtas, entre aspas e com a página.
- **Quem o autor cita** como fonte (documentos, outros autores, pesquisas), quando sustenta uma afirmação.

## O que sai

- Repetições: a mesma ideia dita de novo com outras palavras, no mesmo capítulo ou entre os capítulos.
- Retórica e enchimento: introduções que anunciam o que vem, resumos do que já foi dito, "como veremos", transições.
- Histórias e casos longos que só ilustram uma ideia já clara; no máximo uma frase dizendo do que se tratam, se ajudarem.
- Lixo da extração do texto: cabeçalhos e rodapés de página, números soltos, títulos repetidos, hifenização quebrada.
- Agradecimentos, dedicatórias, propaganda de outros livros do autor.

## Assuntos que não são do capítulo

Um capítulo às vezes fala de outros assuntos (uma digressão, um assunto paralelo). Não os misture ao texto principal e não os apague: liste em **"Outros assuntos"**, uma linha para cada, com as páginas. Isso ajuda a decidir em que outros Temas do Acervo o livro também pode entrar.

## Como escrever

1. **Ler os capítulos inteiros antes de escrever.** O autor volta a assuntos e completa adiante o que começou antes.
2. **Organizar por ideia, não por página.** Cada seção trata de uma ideia ou de um assunto; a ordem é a do argumento do autor, ou a cronológica, num livro de história, ou a mais didática.
3. **Prosa clara**, em português do Brasil, em parágrafos. Lista só quando o conteúdo for de fato uma lista (ingredientes, passos, etapas).
4. **Tamanho:** mire em **um quarto do texto original** (sem contar os links de página); passe de um terço só quando o capítulo for quase todo dados e fatos. Antes de terminar, compare os tamanhos e enxugue: corte o que repete, o que só ilustra, os detalhes de uma história que não ensinam nada, e junte frases que dizem pouco.
5. **Não acrescentar** o que o livro não diz, nem do seu conhecimento. Se o livro afirma algo que parece datado ou discutível, registre em "Dúvidas", sem corrigir o texto.
6. **Livro em outra língua:** escreva em português, mantendo os termos técnicos e os títulos no original entre parênteses quando ajudarem.

## Páginas

Cada seção termina com as páginas de onde veio o conteúdo, **copiando os links** de `cru.md`:

`<sub>Páginas: [p. 45](...), [p. 47](...)</sub>`

## Formato de `organizado.md`

```markdown
# <título do assunto dos capítulos, não "Resumo do capítulo">

> <2 ou 3 frases: do que os capítulos tratam e a ideia central do autor>

## <seção>

<parágrafos>

<sub>Páginas: [p. 45](...), [p. 47](...)</sub>

## ...

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Outros assuntos

- <assunto> — <uma frase> <sub>Páginas: [p. 52](...)</sub>

## Citados no livro

- <documentos, autores e pesquisas que o autor usa como fonte>

## Dúvidas

- <o que o livro afirma sem fonte, o que parece datado, um trecho que ficou ilegível na extração do texto; nada sobre os arquivos ou as páginas que faltam>
```

- **Linha do tempo:** só se o assunto tiver datas ou períodos.
- **Outros assuntos, Citados no livro e Dúvidas:** omitir a seção que ficar vazia.
