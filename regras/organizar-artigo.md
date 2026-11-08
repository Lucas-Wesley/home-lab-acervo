# Regras: organizar um artigo

Você recebe `cru.md`: o texto de uma página da web (artigo, post de blog, reportagem, verbete), tirado do HTML pelo servidor, com o título, o site e o link no topo. Escreva `organizado.md`: o mesmo conteúdo, organizado para alguém que **não sabe nada** do assunto entender.

## O que fazer

1. **Ler o texto inteiro antes de escrever.** A leitura automática da página pode ter trazido sobras do site no começo ou no fim: menus, "leia também", chamadas para newsletter, avisos de cookies, legendas soltas, comentários de leitores. Isso não é o artigo e sai.
2. **Manter a ordem do autor quando ela já é boa.** Um artigo costuma ter começo, meio e fim. Reorganize só quando ajudar: juntar num lugar um assunto espalhado, ou pôr os fatos em ordem cronológica numa narrativa histórica.
3. **Remover o que não é conteúdo:** propaganda, convites para assinar ou comprar, links para outros posts, agradecimentos, biografia do autor no rodapé.
4. **Manter tudo o que é conteúdo.** Isto não é um resumo: é a base de conhecimento. Datas, nomes, lugares, números, quantidades, passos de uma técnica, causas e consequências, citações e explicações ficam.
5. **Escrever em prosa clara**, em parágrafos, **no idioma do artigo** (um artigo em inglês é organizado em inglês; a tradução é outra etapa). Lista só quando o conteúdo for de fato uma lista (ingredientes, passos, itens).

## O que não fazer

- **Não acrescentar fatos** que não estão no texto, nem do seu conhecimento. O texto é o que o artigo diz.
- **Não apagar opiniões, mas atribuí-las:** "segundo o autor, ...", "para <nome>, ...". Fato e opinião não se misturam.
- **Texto truncado ou com pedaços que não fazem sentido** (a página não abriu inteira, uma tabela virou linhas soltas): não invente o que falta. Mantenha o que dá para entender e registre o problema em "Dúvidas".
- Se algo estiver ambíguo ou parecer errado, mantenha e marque com `[?]`.

## Formato de `organizado.md`

```markdown
# <título do assunto, não necessariamente o título da página>

> <2 ou 3 frases dizendo do que o texto trata>

## <seção>

<parágrafos>

## ...

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Citados no artigo

- <livros, autores, estudos, documentos, pessoas citadas como fonte; só os que o artigo menciona, com o link quando houver>

## Dúvidas

- <trechos marcados com [?], partes que vieram truncadas e o que ficou incerto; omitir a seção se não houver>
```

- Não há marcas de tempo: as seções **não** terminam com `<sub>Trechos: ...</sub>`.
- **Linha do tempo:** só se o assunto tiver datas ou períodos.
- **Citados no artigo:** omitir a seção se o artigo não citar nenhuma fonte.
