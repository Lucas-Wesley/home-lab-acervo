# Regras: organizar um documentário

Você recebe `cru.md`: o texto de um documentário (um filme ou um episódio de uma série), com marcas de tempo `[mm:ss]` a cada ~30 s. O cabeçalho diz qual é o documentário, o episódio, as sinopses e de onde veio o texto: a **legenda** do vídeo ou a **transcrição do áudio**. Escreva `organizado.md`: o **conhecimento** que o documentário traz, organizado como material de estudo, para alguém que **não sabe nada** do assunto entender.

Um documentário não é uma aula: tem narração, entrevistas, cenas encenadas, música e recursos para prender quem assiste. O que fica é o conteúdo: o que aconteceu, quem, quando, onde, por quê, como funciona, o que se sabe e o que se discute.

## O que fazer

1. **Ler o texto inteiro antes de escrever.** O documentário volta aos assuntos e conta a história em idas e vindas. Junte cada assunto num lugar só.
2. **Organizar por assunto**, na ordem que faz o assunto ser entendido. Numa história, a ordem cronológica dos fatos (não a ordem das cenas). Num tema de ciência ou natureza, do geral para o particular.
3. **Manter tudo o que é conteúdo.** Não é um resumo: datas, nomes, lugares, números, causas, consequências, explicações e descobertas ficam, com o detalhe que o documentário dá.
4. **Escrever em prosa clara**, em parágrafos, **no idioma do texto** (um documentário em inglês é organizado em inglês; a tradução é outra etapa). Lista só quando o conteúdo for de fato uma lista.

## Quem diz o quê

- **A narração** é a voz do documentário. O que ela afirma de fato (datas, nomes, o que aconteceu) entra direto, sem "segundo a narração" ("A pirâmide foi construída por volta de 2560 a.C."), com a marca de tempo no fim da seção.
- **A interpretação do documentário** (um julgamento, uma causa disputada, o que alguém "quis" ou "sentiu") vai atribuída à **obra ou ao diretor**, e não ao tipo de material: "para o filme de Silvio Tendler, Vargas…", "o documentário sustenta que…" (o nome vem de `- Direção:` no cabeçalho). Nunca "a narração diz", "segundo a narração", "o vídeo mostra".
- **Os entrevistados** (historiadores, cientistas, testemunhas): o que é **opinião, interpretação ou lembrança pessoal** vai atribuído ("segundo o egiptólogo Zahi Hawass, …"; "um sobrevivente conta que …"). Use o nome só quando o texto o disser ou o próprio entrevistado se apresentar; sem nome, pela função ("um arqueólogo entrevistado"). A transcrição não diz quem fala: não adivinhe.
- **Hipóteses e teorias** ficam como hipóteses ("uma das hipóteses é…", "o documentário defende que…"), nunca como fato.
- **O ponto de vista do documentário:** se ele defende uma leitura (uma tese, um lado de uma polêmica, uma teoria fora do consenso), diga isso no resumo do topo e trate essa leitura como do documentário. Não esconda nem combata: registre. O que for discutível vai também para "Dúvidas".

## O que tirar

- **Abertura e encerramento:** vinhetas, "neste episódio…", "anteriormente…", o gancho para o próximo episódio, os créditos e agradecimentos.
- **Repetições** de antes e depois do intervalo comercial e os resumos do que já foi dito.
- **Cenas encenadas e diálogos dramatizados:** as falas dos atores numa reconstituição não são conteúdo. Fica só o fato que a cena mostra, se a narração o confirmar.
- **Narração de efeito** sem informação ("naquela noite fria, ninguém imaginava…", "o que eles encontraram mudaria tudo…"): fica a informação, sai o suspense.
- **Letras de música**, sons descritos e frases soltas sem relação com o resto (ruído da transcrição sobre a trilha).
- Comentários sobre a própria produção, a equipe filmando ou o canal.

## O que não fazer

- **Não acrescentar fatos** que não estão no texto, nem do seu conhecimento. O texto é o que o documentário diz. Se ele erra ou simplifica algo que você sabe, mantenha o que ele diz e registre em "Dúvidas".
- **Não usar a sinopse como fonte:** ela é só para entender do que o documentário trata e acertar nomes.
- **Erros de transcrição:** corrija só quando o contexto deixar claro (um nome conhecido escrito errado, um nome que aparece certo na sinopse). Se não tiver certeza, mantenha e marque com `[?]`.
- **Texto que veio da legenda:** a legenda resume a fala e quase nunca diz quem fala. Não trate a forma da frase como citação exata.

## Episódios de uma série

O cabeçalho diz o episódio (`- Episódio: T1 E2 de 6 — …`). Quando a instrução mencionar `outros-episodios.md`, ele tem o resumo e as seções dos episódios anteriores, já organizados:

- **Não explique de novo** o que um episódio anterior já explicou: retome em uma frase e siga para o que este episódio acrescenta.
- **Use os mesmos nomes** (pessoas, lugares, conceitos) que os anteriores usaram.
- O título do texto é o assunto **deste episódio**, não o nome da série.

## Formato de `organizado.md`

```markdown
# <o assunto deste filme ou episódio, não o título do documentário>

> <2 ou 3 frases dizendo do que o documentário trata e, se houver, a leitura que ele defende>

## <seção>

<parágrafos>

<sub>Trechos: [mm:ss], [mm:ss]</sub>

## ...

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Pessoas

- **<nome>** — <quem é e o papel no assunto, como o documentário apresenta>

## Citados no documentário

- <livros, documentos, estudos, arquivos e obras citados como fonte; só os que ele menciona>

## Dúvidas

- "<trecho como está>" [mm:ss]: <o que ficou incerto, discutível ou sem fonte>
```

- Cada seção termina com as marcas de tempo de onde veio o conteúdo.
- **Linha do tempo:** só se o assunto tiver datas ou períodos. Os itens em **ordem cronológica** e só com o que o texto diz.
- **Pessoas:** as pessoas que fazem parte do assunto (personagens históricos, cientistas cujo trabalho é explicado). Os entrevistados entram só se o texto disser quem são. Omitir a seção se houver menos de três.
- **Citados no documentário:** omitir a seção se ele não citar nenhuma fonte.
- **Dúvidas:** cada uma com o trecho entre aspas, como está no texto, e **sempre** a marca `[mm:ss]` do bloco onde está. Vão para cá as afirmações fortes sem fonte, as teorias apresentadas como fato, as versões que se contradizem e os nomes incertos. O app usa a marca para ouvir de novo o áudio daquele ponto, quando há áudio, e pesquisar na internet.
