# Regras: responder a uma pergunta sobre um texto

O Lucas está lendo um texto do Acervo (um Tema ou uma Referência) e fez uma pergunta. Você faz duas coisas: vê **se o texto dele responde**, e **verifica a resposta na internet**. O resultado mostra quanto o texto responde, a resposta verificada e onde o texto está errado, desatualizado ou incompleto.

Os arquivos do texto estão na pasta atual. Leia todos antes de responder.

## 1. O que o texto diz

- Procure a resposta no texto inteiro, em todos os capítulos.
- `cobertura`:
  - `responde`: o texto responde à pergunta por inteiro.
  - `em_parte`: responde uma parte; falta algo importante.
  - `toca`: fala do assunto, mas não responde à pergunta.
  - `nao_responde`: não fala disso.
- `no_texto`: o que o texto diz sobre a pergunta, em poucas frases e com as palavras dele, dizendo em que capítulo está. Vazio se não diz nada.

## 2. Verificar na internet

Pesquise sempre, mesmo quando o texto parece responder: é assim que se descobre texto errado ou desatualizado.

- **Ordem de confiança das fontes:**
  1. Primárias e oficiais: o site da própria instituição, clube, empresa ou órgão; documentos; a própria obra.
  2. Enciclopédias, universidades e obras de referência.
  3. Imprensa reconhecida.
  4. Blogs, fóruns e redes sociais servem só de pista, nunca de prova.
- **Fontes independentes:** dois sites que copiam o mesmo texto (por exemplo, a mesma Wikipédia) contam como uma fonte só.
- **Dado que muda com o tempo** (títulos, recordes, cargos, números, quem está vivo): procure a informação mais recente e diga a data dela.
- Liste em `fontes` só os sites que você realmente consultou.

## 3. O selo de cada afirmação

- `confirmado`: duas ou mais fontes independentes e confiáveis concordam.
- `uma_fonte`: só uma fonte confiável diz isso.
- `divergem`: as fontes contam de jeitos diferentes. Mostre as versões e quem diz cada uma.
- `nao_encontrado`: nenhuma fonte confiável trata disso.

`selo` é o selo da resposta como um todo: o da afirmação principal dela.

## 4. A resposta

- `resposta`: a resposta verificada, completa, em português do Brasil e em Markdown, com as fontes indicadas por número (`[1]`, `[2]`, na ordem de `fontes`). Escreva-a sempre, mesmo quando o texto responde por inteiro.
- Responda à pergunta que foi feita, no contexto do texto (se o texto é sobre um clube, "o primeiro título" é o do clube).

## 5. Divergências

Compare o que o texto diz com o que você verificou. Cada diferença vira um item de `divergencias`:

- `tipo`:
  - `erro`: o texto afirma algo que as fontes contradizem.
  - `desatualizado`: era verdade, mas mudou.
  - `omissao`: falta algo importante para a pergunta.
  - `melhoria`: está certo, mas pode ficar mais preciso ou mais claro.
- `capitulo`: o capítulo do texto em que isso está (ou estaria).
- `no_texto`: o trecho do texto, copiado como está (vazio numa omissão).
- `correto`: o que as fontes dizem.
- `selo`: o selo do que está em `correto`.
- `fontes`: os números das fontes que sustentam `correto`.

Não aponte como erro o que só uma fonte contradiz: nesse caso, use `divergem` ou deixe de fora. Diferença de estilo não é divergência. Se o texto está certo e completo para a pergunta, `divergencias` fica vazia.

## O que não fazer

- Não mexer em nenhum arquivo: só ler.
- Não inventar fonte nem link.
