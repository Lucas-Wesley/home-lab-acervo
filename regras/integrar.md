# Regras: integrar uma Referência a um Tema existente

O Tema já existe e é **permanente: só se acrescenta, nunca se substitui**. Você recebe a Referência organizada e a pasta do Tema. Edite o Tema para incorporar **só o que falta**.

## Estrutura do Tema

- O texto fica em `<pasta>/indice.md`, dividido em capítulos (`## `).
- Um capítulo pode ter sido **separado** pelo Lucas: na página fica só o título e um link (`## Título` seguido de `[Ler o capítulo →](arquivo.md)`), e o texto está em `<pasta>/arquivo.md`. Nesse caso, o que entra nesse capítulo vai **no arquivo dele**, não na página principal.
- Subpastas com `indice.md` são subtemas: não mexa nelas.

## Como fazer

1. Leia o `indice.md` e os capítulos separados. Depois leia a Referência inteira.
2. Liste para si as **lacunas**: fatos, explicações, datas, nomes ou nuances que a Referência traz e o Tema ainda não tem. O que o Tema já diz, mesmo com outras palavras, **não é lacuna**.
3. Encaixe cada lacuna **no capítulo certo**, no ponto em que ela faz sentido na leitura (ordem cronológica ou didática, como o resto do texto).
4. Se uma lacuna for um assunto novo que não cabe em nenhum capítulo, crie um capítulo novo (`## `) **na página principal**, na posição de leitura certa (antes de "Linha do tempo" e "Citados"). Não crie arquivos novos.
5. Se a Referência **contradiz** o Tema, não apague nada: acrescente a outra versão atribuída à fonte ("segundo <fonte>, ...") e registre num capítulo "Divergências", antes de "Linha do tempo".
6. Atualize "Linha do tempo" e "Citados", e acrescente a Referência em `referencias` no cabeçalho.

## Regras gerais

- **Não reescrever** o que já existe, só inserir. Correção de digitação óbvia pode.
- **Não acrescentar fatos** que não estejam na Referência.
- Cada trecho novo leva a fonte: `<sub>Fonte: <id> [mm:ss]</sub>`.
- Mexa só dentro da pasta do Tema. Nunca apague arquivos.
- Se a Referência não acrescentar nada, não mexa em nada e responda `nada novo`.
