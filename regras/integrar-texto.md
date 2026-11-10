# Regras: integrar ao texto uma resposta verificada ou um "Mais informações"

O Lucas perguntou sobre o texto (Perguntar) ou pediu "Mais informações" sobre um trecho, e agora quer **levar o que foi achado para dentro do texto**. Você recebe `material.md` (o que integrar e as escolhas dele) e o texto: a pasta de um Tema ou o `texto.md` de uma Referência.

## Estrutura

- **Tema:** o texto fica em `<pasta>/indice.md`, em capítulos (`## `). Um capítulo **separado** tem na página só o título e um link (`[Ler o capítulo →](arquivo.md)`), e o texto está em `<pasta>/arquivo.md`: o que entra nesse capítulo vai **no arquivo dele**. Subpastas com `indice.md` são subtemas: não mexa nelas.
- **Referência:** o texto inteiro está em `texto.md`. Edite só ele.

## Onde encaixar

1. Leia o texto inteiro antes de escrever.
2. Cada informação nova vai **no capítulo e no ponto certos**: na ordem cronológica dos fatos ou, se o texto segue outra ordem, na ordem didática dele. **Nunca** cole no fim do texto nem num capítulo "Complemento".
3. Só se for um assunto que não cabe em nenhum capítulo, crie um capítulo novo, na posição de leitura certa (antes de "Linha do tempo", "Citados" e das seções de referências).
4. Escreva no estilo do texto (prosa, mesmo tom, mesmo idioma) e **sem repetir** o que ele já diz com outras palavras.
5. Atualize a "Linha do tempo", se houver e a informação tiver data.

## As escolhas do Lucas (estão em `material.md`)

Cada divergência vem com a escolha dele. Siga exatamente:

- **Substituir:** corrija o texto no lugar: troque o trecho errado ou desatualizado pelo que as fontes dizem. Aqui (e só aqui) é permitido mudar o que já estava escrito.
- **Manter as duas versões:** não apague o que o texto diz; logo depois, acrescente a outra versão, atribuída ("Já segundo <fontes>, ..."), e registre no capítulo "Divergências" (crie antes de "Linha do tempo", se não houver).
- **Acrescentar:** é algo que falta ao texto: encaixe no ponto certo.
- **Ignorar:** não vem para cá; não mexa no que ela trata.

A **resposta verificada** só entra se `material.md` disser que sim.

## "Mais informações"

- **O texto completo:** a explicação entra **inteira, como está**, logo depois do parágrafo do trecho selecionado (ou no ponto mais próximo que faça sentido). Pode ajustar só os títulos (`#`) para o nível certo e as marcas de fonte.
- **Só o que serve ao texto:** use apenas o que acrescenta ao texto, reescrito no estilo dele, espalhado nos pontos certos.

## Dúvida resolvida

Quando `material.md` traz uma dúvida resolvida (pelo áudio ou pela internet): faça a correção indicada no lugar, tire a dúvida da seção "## Dúvidas" (e a seção, se ficar vazia) e a marca `[?]` do trecho corrigido. Uma correção vinda só do áudio não precisa de fonte da internet.

## Fontes

- Cada trecho novo ou corrigido leva as fontes da internet, curtas e com link: `<sub>Fontes: [Título](url), [Título](url)</sub>`.
- Ponha essas fontes também em "Citados" (num Tema) ou na seção de fontes/citados da Referência, se houver.
- **Não acrescente fatos** que não estejam em `material.md`.

## Regras gerais

- Mexa só no texto indicado. Nunca apague arquivos nem crie arquivos novos.
- Se, seguindo as escolhas, não houver nada a mudar, não mexa em nada e responda `nada novo`.
