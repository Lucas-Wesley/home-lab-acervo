# Regras: criar um Tema a partir de uma Referência

Você recebe `organizado.md` (uma Referência já organizada) e a pasta onde o Tema novo deve nascer. Essa pasta é decidida pelo Lucas: `temas/<slug>/` para um Tema principal, ou `temas/<pai>/<slug>/` para um subtema. Você escreve só dentro dela.

## Estrutura: uma página só

O Tema é **um arquivo só**, `<pasta>/indice.md`, com o texto inteiro dividido em **capítulos** (`## `). Não crie outros arquivos: o app monta o sumário a partir dos capítulos, e o Lucas separa um capítulo em página própria quando ele crescer.

```markdown
---
titulo: <título do Tema>
criado: <AAAA-MM-DD>
referencias:
  - id: <id da Referência>
    titulo: <título da fonte>
    url: <url da fonte>
---

# <título do Tema>

<introdução: 1 a 3 parágrafos que situam o assunto para quem não sabe nada>

## <capítulo>

<parágrafos, copiados do organizado.md sem resumir>

<sub>Fonte: <id da Referência> [mm:ss], [mm:ss]</sub>

## <capítulo>

...

## Linha do tempo

<a linha do tempo do organizado.md, se houver>

## Citados

<as fontes citadas, se houver>
```

## Como dividir em capítulos

- Cada **seção** do `organizado.md` vira um capítulo, **na mesma ordem** (cronológica ou didática).
- Seções curtas que só fazem sentido juntas viram um capítulo só.
- Dentro de um capítulo longo, use `### ` para subdividir, se ajudar a leitura.
- Não escreva uma lista de capítulos no texto: o app gera o sumário.
- A seção "Dúvidas" do `organizado.md` **não** entra no Tema: ela fica na Referência.

## Regras gerais

- **Não reescrever o conteúdo**: o texto já foi organizado; aqui só se organiza em capítulos.
- **Não acrescentar fatos.**
- **Links para outros Temas são só links**: `[texto](caminho/relativo/indice.md)`, sempre para o `indice.md` do outro Tema, nunca copiando o texto dele.
