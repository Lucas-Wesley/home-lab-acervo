# Regras: resolver as dúvidas de uma Referência

Você recebe `duvidas.md` e `texto.md`. Em `duvidas.md`, cada dúvida (`## Dúvida <n>`) é um item da seção "Dúvidas" do texto organizado e pode trazer:

- **o momento** da fala e **o que a transcrição tem ali** (a legenda, muitas vezes automática, com erros de nomes e números);
- **o que o Whisper large-v3 ouviu no áudio** do mesmo trecho: uma segunda transcrição, bem mais fiel, feita agora a partir do áudio.

Para cada dúvida, decida o **estado**:

- `resolvida_audio`: o áudio deixa claro o que foi dito (o nome, o número, a frase). Diga em `no_audio` o que se ouve, com as palavras do Whisper.
- `confirmada_internet`: o áudio não basta (ou não há áudio), mas a pesquisa na internet resolve: o nome certo, a data, o fato. Pesquise **sempre** que a dúvida for sobre um fato, nome ou número, mesmo resolvida pelo áudio, para conferir a grafia.
- `incerta`: nem o áudio nem as fontes resolvem.
- `lacuna`: não é erro de transcrição, é **algo que falta no material** (o vídeo não diz o ano, não conclui a explicação, a pesquisa não achou). Explique o que falta; o Lucas pode mandar pesquisar.

## O que escrever

- `explicacao`: curta e direta: o que era a dúvida e o que se apurou.
- `trecho` e `correto`: se o texto organizado (`texto.md`) tem algo a corrigir, `trecho` é o pedaço **exatamente como está em texto.md** e `correto` como deve ficar. Vazios se não há o que corrigir.
- `fontes`: as páginas da internet usadas (título e endereço).

## Confiança nas fontes

Como no Perguntar: fontes primárias e oficiais; depois enciclopédias e universidades; imprensa reconhecida; blogs e fóruns só como pista. Fontes que copiam a mesma origem contam como uma.

## Regras gerais

- Não invente: se o áudio e as fontes não resolvem, é `incerta`.
- Não edite arquivos.
- Responda todas as dúvidas, com o `indice` de cada uma (o número em `## Dúvida <n>`).
