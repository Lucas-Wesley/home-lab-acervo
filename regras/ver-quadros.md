# Regras: ler a tela de um Reel

Você recebe os quadros de um Reel do Instagram em `quadros/`, um a cada mudança de cena (e, sem mudança, um a cada 4 s). O nome de cada arquivo é o momento do vídeo: `00-04.jpg` é `[00:04]`. Em `contexto.md` estão a fala transcrita (com marcas `[mm:ss]`) e a legenda do post. Escreva `tela.md`: **o que aparece na tela e não está na fala**, para quem não viu o vídeo.

Muitos Reels passam o conteúdo pela tela: texto sobreposto, listas, passos, slides, código, gráficos, telas de um programa, um produto, um lugar, uma demonstração. Às vezes a fala é só música e todo o conteúdo está na tela.

## O que fazer

1. **Abrir todos os quadros, em ordem**, antes de escrever.
2. **Copiar o texto da tela como está**, entre aspas, no idioma em que aparece: títulos, legendas sobrepostas, itens de lista, números, nomes, código (em bloco de código). Um texto que continua em vários quadros é escrito uma vez só, no momento em que aparece completo.
3. **Descrever o que é mostrado quando isso é conteúdo:** o diagrama e o que ele liga, o gráfico e o que ele mostra, o passo que a pessoa faz na tela, o objeto ou lugar de que se fala. Uma ou duas frases, sem enfeite.
4. **Ligar à fala quando ajudar:** "a lista que ela lê em [00:12]".

## O que não fazer

- **Não repetir o que a fala já diz.** Legenda automática que só transcreve a fala (as palavras aparecendo enquanto a pessoa fala) não entra; diga uma vez que o vídeo tem legenda da fala.
- **Não descrever o que não é conteúdo:** a pessoa falando para a câmera, o cenário, roupa, filtros, emojis, adesivos, "siga para mais", o @ da conta, marcas d'água, a interface do Instagram.
- **Não acrescentar nada que não está nos quadros**, nem do seu conhecimento. Texto ilegível ou cortado: escreva o que dá para ler e marque com `[?]`.

## Formato de `tela.md`

Uma entrada por momento em que algo novo aparece, com a marca do quadro:

```markdown
[00:00] Texto na tela: "3 erros ao estudar inglês".

[00:04] Lista que aparece item a item:
1. "Decorar listas de palavras"
2. "Estudar só gramática"

[00:12] Mostra a tela do Anki com um cartão: frente "to look forward to", verso "aguardar com expectativa".
```

Sem títulos nem introdução: só as entradas. Se nenhum quadro tiver conteúdo além da fala, escreva uma linha só: `Nada além da fala: <o que se vê, em poucas palavras>.`
