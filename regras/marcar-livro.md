# Regras: marcar os capítulos de um livro por assunto

Você recebe `capitulos.md`: um ou mais capítulos de um livro da Biblioteca do Lucas, com a página de cada trecho (`[p. N]`). Marque cada capítulo para que o Acervo saiba **onde cada assunto está** nos livros dele.

Para cada capítulo (`## Capítulo <n>: ...`), devolva:

- `n`: o número do capítulo, como está no título.
- `resumo`: uma ou duas frases sobre o que o capítulo trata (não o que o autor opina sobre tudo; o assunto).
- `temas`: os assuntos do capítulo. Se um dos **Temas do Acervo** (lista no pedido) se aplica, use **a pasta dele exatamente** (ex.: `temas/guerra-de-canudos`); os outros assuntos, com nomes curtos e comuns ("Mineração de prata em Potosí", "Ciclo do açúcar").
- `pessoas`, `organizacoes`, `lugares`: os nomes próprios importantes no capítulo, como são conhecidos (sem repetir variações).
- `periodos`: séculos, anos ou intervalos de que o capítulo trata ("século XVI", "1930–1945").
- `eventos`: acontecimentos com nome ("Revolta de Túpac Amaru", "Guerra do Paraguai").
- `trechos`: até 6 passagens marcantes, com a página e o assunto em poucas palavras.

## Regras gerais

- Só o que o texto trata de fato; nada do seu conhecimento.
- Livro sem sumário: cada "capítulo" é um bloco de 20 páginas (`Páginas 21 a 40`), que pode começar e terminar no meio de um assunto. Marque o bloco do mesmo jeito, e no `resumo` diga o assunto das páginas.
- Capítulo quase sem texto (o título de uma parte, um prefácio curto): marque o que houver, mesmo vazio.
- Listas curtas e úteis: o importante, não cada nome citado de passagem.
- Não edite arquivos.
