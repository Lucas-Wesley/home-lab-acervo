# Regras: exercícios de prática de um Tema

Você recebe os arquivos de um Tema do Acervo do Lucas. Monte **exercícios de prática**: treino com uma resposta certa, que o app confere na hora e explica. Não é o simulado, onde ele responde à teoria com as suas palavras. Aqui ele **aplica** o que o Tema ensina a um caso concreto: uma frase, uma situação, um dado.

Exemplo, num Tema sobre crase:

- escolha: "Entreguei o relatório ___ diretora." Opções: "a", "à". Resposta: "à". Explicação: "Entregar pede a preposição a, e diretora é feminina e aceita o artigo a: a + a = à. Troque por um masculino: 'ao diretor'."
- completar: "Fomos ___ pé até a praia." Resposta: "a". Explicação: "Não há crase antes de palavra masculina: 'a pé'."

Devolva de 10 a 15 exercícios em `exercicios`. Para cada um:

- `tipo`:
  - `escolha`: o Lucas escolhe entre 2 a 4 opções curtas.
  - `completar`: ele escreve o que falta na lacuna. Use só quando a resposta é curta e única: uma palavra, um número, uma data. Se mais de uma forma for certa, ponha as outras em `aceitas`.
- `enunciado`: a frase ou a situação, com `___` no lugar da lacuna (em `completar`, sempre; em `escolha`, quando houver lacuna). Se precisar de uma pergunta, ela vem antes, na mesma linha ("Vai crase? ___ partir de hoje").
- `opcoes`: na escolha, as opções, uma delas exatamente igual a `resposta`; no completar, lista vazia.
- `resposta`: a resposta certa, exatamente como deve ser escrita.
- `aceitas`: outras formas certas (ex.: "1789" e "em 1789"); em geral, vazia.
- `explicacao`: o **porquê**, em 1 a 3 frases. A regra aplicada a este caso e, quando ajuda, o macete para decidir sozinho (a troca pelo masculino, a pergunta que se faz ao verbo…). É o que ele lê depois de responder, certo ou errado.
- `assunto`: a regra ou a ideia que o exercício treina, em poucas palavras ("crase antes de palavra feminina", "causas da Revolução de 1917"). Exercícios do mesmo ponto usam o mesmo assunto.
- `capitulo`: o título do capítulo (`## `) ou do subtema de onde saiu, exatamente como está.

## Adapte ao conteúdo

- **Regras e normas** (gramática, ortografia, regras de um jogo, leis): frases novas para aplicar a regra, incluindo **os casos em que ela não vale** e as exceções. Misture certo e errado: nem sempre a resposta é "vai crase".
- **Técnico** (programação, matemática, ciências): qual a saída, qual o valor, qual o próximo passo, qual o erro, qual termo completa a definição. Código curto, em bloco de uma linha ou com `\n` no enunciado.
- **História e narrativa:** causa e consequência, ordem dos acontecimentos, quem fez o quê, a qual período um fato pertence, qual afirmação está certa. Evite decorar datas soltas: prefira datas que situam um porquê.
- **Conceitual:** qual conceito explica o caso descrito, qual exemplo pertence a qual categoria.

## Regras gerais

- **Casos novos**, não frases copiadas do Tema: o treino é aplicar, não reconhecer o texto.
- Uma só resposta certa, sem ambiguidade. Se um caso divide os especialistas, não use.
- Opções erradas plausíveis: os erros que alguém comete de verdade.
- Do mais simples ao mais difícil, cobrindo os capítulos e os subtemas.
- Só o que o Tema ensina: a regra e a resposta têm de sair dele. Nada que dependa de uma dúvida marcada.
- Se existir `ja-existem.md`, é a lista do que o banco já tem: não repita nada, nem com outras palavras; traga casos e pontos novos.
- Português do Brasil.
- Não edite arquivos.
