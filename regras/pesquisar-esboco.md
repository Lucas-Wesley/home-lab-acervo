# Regras: esboço de uma pesquisa

O Lucas pediu uma pesquisa sobre um tema e escolheu as fontes. Antes de buscar, você monta o **esboço**: as perguntas que a pesquisa precisa responder e, para cada uma, o que buscar em cada fonte escolhida. Quem busca é o servidor, com o que você escolher; a qualidade do texto depende de você escolher bem.

## As perguntas

- Divida o tema nas perguntas que alguém que **não sabe nada** do assunto faria, na ordem em que fazem sentido: o que é, de onde vem, como funciona, como mudou com o tempo, por que importa.
- Cada pergunta cobre uma parte diferente do tema; não repita a mesma pergunta com outras palavras.
- Fique no tema pedido. Se o pedido for estreito ("a origem da laranja"), as perguntas também são: não transforme numa história de todas as frutas.
- Escreva as perguntas em português, curtas e diretas.

## O que já está escrito

O pedido traz a lista dos Temas do Acervo, e os arquivos deles podem ser lidos. Antes de montar as perguntas:

- Leia o `indice.md` dos Temas que tratam do assunto (o título sozinho engana) e indique as pastas deles em `relacionados` (no máximo 3; lista vazia se nenhum tratar do assunto).
- Se o Acervo já explica bem uma parte do tema, **não gaste perguntas com ela**: prefira o que o Tema não diz, o que ele diz sem fonte e o que está em "Divergências". Uma pergunta para confirmar o essencial pode valer a pena.
- **Quando o pedido diz que a pesquisa é para completar um Tema:** leia esse Tema inteiro (o `indice.md` e os capítulos separados). As perguntas vão atrás do que ele não tem ou trata por alto, e não do que ele já explica.
- **Ao aprofundar** (o pedido diz que é outra rodada): leia `anterior.md`, o texto atual da pesquisa, principalmente "Dúvidas". As perguntas novas vão atrás do que ficou sem resposta, do que as fontes contaram de jeitos diferentes, do que o Lucas pediu a mais e do que ainda falta para entender o assunto. **Não repita** as perguntas já respondidas.

## O que buscar em cada fonte

Deixe **vazio** o campo de uma fonte que não foi escolhida.

- **Livros (`livros`):** 2 a 5 palavras que apareceriam no texto de um livro sobre aquela pergunta, em português, sem artigos nem preposições. A busca acha o começo das palavras e ignora acentos ("laranja citros Ásia"). Nomes próprios funcionam bem.
- **Wikipédia (`wikipedia`):** o **nome exato do artigo**, como ele aparece na Wikipédia em português ("Laranja (fruta)", "João Ramalho", "Cinema Novo"). Não escreva palavras-chave nem frases: o servidor abre o artigo pelo nome e só busca se ele não existir, e uma busca vaga traz artigo errado. Na pesquisa detalhada, pode pedir **dois artigos** separados por `|` ("Glauber Rocha | Estética da fome"). Quando o artigo em português é curto ou não existe, o servidor busca o da Wikipédia em inglês sozinho.
- **YouTube (`youtube`):** como uma pessoa buscaria um vídeo explicativo e sério sobre a pergunta, em português; use inglês só quando o assunto quase só existir em inglês. Inclua palavras como "documentário", "aula" ou "história" quando ajudarem; evite as que tragam receitas rápidas, notícias, trailers ou cortes curtos.

### Capítulos dos livros (`capitulos`, quando o pedido traz `livros-marcados.md`)

`livros-marcados.md` é o mapa dos livros da Biblioteca do Lucas: o que cada capítulo trata, com as pessoas, os lugares, os períodos e os eventos. Leia e escolha os capítulos que tratam **de fato** do tema, para o servidor ler **inteiros** (com as marcações que o Lucas fez neles). Um capítulo inteiro sobre o assunto vale mais que muitos trechos soltos.

- Cada item: `livro` (o id, exatamente como no mapa), `n` (o número do capítulo) e `pergunta` (o número da pergunta a que ele mais serve, a partir de 1).
- Só capítulos que tratam do tema; não escolha um capítulo só porque cita um nome de passagem. Lista vazia se nenhum servir.
- Respeite o máximo que o pedido diz. Prefira os capítulos centrais; um livro pode dar mais de um.

### Páginas da internet (`paginas`, só quando o Lucas escolheu "Sites")

Busque com WebSearch e escolha as páginas que o servidor vai ler. Ele lê o texto da página como ela é, e a conferência confere o texto contra ele.

- Prefira, nesta ordem: universidades e instituições de pesquisa, órgãos oficiais e museus, enciclopédias e obras de referência, imprensa reconhecida, sites técnicos conhecidos da área. Blogs, fóruns e redes sociais, não.
- Escolha páginas que tratem **diretamente** de uma pergunta, com texto (artigo, verbete, página de instituição, PDF de texto). Evite páginas de busca, listas de links, vídeos (o YouTube tem fonte própria), páginas que pedem login e as da Wikipédia (ela também tem fonte própria).
- Cubra as perguntas que as outras fontes escolhidas devem responder pior; não gaste duas páginas que dizem a mesma coisa.
- Cada item: `url` (o endereço exato que a busca mostrou; nunca invente), `titulo` e `pergunta` (o número da pergunta, a partir de 1). Respeite o máximo que o pedido diz.

## O título

Um título curto para a pesquisa, como o título de um capítulo de livro, sem "Pesquisa sobre".
