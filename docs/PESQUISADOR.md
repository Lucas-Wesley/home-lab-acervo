# Integração com a Biblioteca e módulo Pesquisador — caminhos possíveis

> Planejamento de 06/11/2026, **nada implementado**. Pedido do Lucas: pensar em como a Biblioteca
> Digital conversa com o Acervo e num "pesquisador" que **produz conteúdo** sobre um tema, em vez
> de organizar um vídeo. Aqui ficam as opções, com recomendação; a decisão é do Lucas.

---

## 1. O que já existe e facilita

- **Fila + etapas com o Claude** (D14): qualquer trabalho novo vira só mais um tipo de tarefa.
- **Referência com estados** e **revisão por diff** (D16): todo conteúdo novo, venha de onde vier,
  pode cair como Referência e passar pelo mesmo caminho: verificar → sugerir → criar/integrar.
- **Fragmento de Conteúdo** (contrato comum, ainda não decidido no Hub): texto + fonte + localizador.
- Na Biblioteca: **marcações já guardam o texto citado** e a posição (R23), e o sumário com
  capítulos já sai no servidor.

A peça que falta no Acervo para quase tudo abaixo é a mesma: **Referência que não é vídeo**
(fonte = livro, pesquisa, artigo). Hoje a tabela exige `video_id`; precisa de `fonte_tipo` e
`fonte_ref` (ex.: id do livro + localizador), e o `cru` passa a poder chegar pronto, sem `yt-dlp`.

---

## 2. Biblioteca → Acervo

| # | Caminho | Como funciona | Esforço | Valor |
|---|---------|---------------|---------|-------|
| B1 | **Marcação vira Referência** | Na barra de seleção da Biblioteca, botão "Acervo". Manda um Fragmento (`POST /api/v1/referencias`) com o trecho, livro, autor e localizador. No Acervo entra direto como `organizado` (trecho curto não precisa reorganizar) e segue para sugerir/integrar. | Baixo | Alto: é o uso diário de estudo. |
| B2 | **Capítulo vira Referência** | Botão no sumário: a Biblioteca extrai o texto do capítulo e manda. No Acervo, uma regra `organizar-livro` (sem filtro de propaganda; mantém a estrutura do autor; cita página). | Médio (extração de texto de PDF/EPUB na Biblioteca) | Alto para livros de estudo. |
| B3 | **Link de volta para o livro** | A fonte no Tema (`<sub>Fonte: ...</sub>`) aponta para a Biblioteca no ponto exato (`https://192.168.1.20:8443/#/livro/<id>?loc=...`), como hoje aponta para o minuto do vídeo. | Baixo (depois de B1) | Médio: fecha o ciclo leitura ↔ conhecimento. |
| B4 | **Livros relacionados no Tema** | O Tema consulta a busca da Biblioteca pelo título e mostra "na sua biblioteca". | Baixo | Médio. |
| B5 | **Livro inteiro** | Vários capítulos de uma vez, como pacote de Referências. | Alto | Só depois de B2 provar a regra. |

**Recomendação:** B1 → B3 → B2. B1 sozinho já entrega "li, marquei, foi pro Acervo" e obriga a
resolver o modelo de Referência genérica e o contrato, que o resto reaproveita.

**Decisões para o Lucas:** um trecho curto passa pela verificação ou vai direto? Várias marcações
do mesmo livro viram uma Referência só ("notas do livro X") ou uma por marcação?

---

## 3. Módulo Pesquisador

Ideia: na Biblioteca (ou no próprio Acervo), escolho um tema — digitado ou a partir de um trecho —
e mando para o Pesquisador. Ele **produz** um texto sobre o tema, com fontes, que entra como
Referência do tipo `pesquisa` e passa pela mesma revisão de sempre. Nada entra no Acervo sozinho.

### Caminhos

**P1 — Pesquisa na web com o Claude Code.** Uma tarefa `pesquisar` na fila, com uma regra própria e
as ferramentas `WebSearch`/`WebFetch` liberadas **só nessa tarefa** (hoje são proibidas). O Claude
busca, lê e escreve um texto organizado como os outros, com **fonte em cada parágrafo** (URL +
trecho citado). As páginas usadas ficam salvas na pasta da Referência, como prova.
- A favor: o mais rápido de construir (uma regra + um tipo de tarefa); serve para qualquer assunto.
- Contra: qualidade da web varia; é onde a alucinação aparece se a regra não obrigar citação.
  Consome bastante da assinatura: precisa de limite de tempo/turnos por pesquisa.

**P2 — Pesquisa nos livros da Biblioteca.** A Biblioteca ganha busca no texto dos livros (SQLite
FTS5, que já vem no SQLite) e um endpoint `GET /api/v1/busca?q=` que devolve trechos com
localizador. O Pesquisador consulta essa busca, lê os trechos e escreve citando **livro e página**.
- A favor: fontes que você escolheu e confia; funciona sem internet; **a mesma busca resolve a
  checagem de veracidade** (D10, "fontes externas confiáveis"), o ponto mais difícil do projeto.
- Contra: depende de extrair texto dos livros (EPUB é fácil, PDF escaneado não) e só cobre o que
  está na biblioteca.

**P3 — Pesquisador que acha vídeos.** O `yt-dlp` também busca (`ytsearch10:<tema>`). O
Pesquisador escolhe os vídeos mais relevantes e os põe na fila como Referências comuns; o pipeline
que já existe faz o resto. É o cenário adiado "estudar um assunto do zero".
- A favor: quase nada novo; reaproveita tudo.
- Contra: o Pesquisador não escreve, só seleciona; qualidade = a dos vídeos.

**P4 — Orquestrado (o "completo").** Primeiro um esboço do tema (subtópicos); para cada um, busca
nos livros (P2), depois vídeos (P3), e a web (P1) só para lacunas; no fim, um rascunho de Tema com
todas as fontes, revisado por diff. É o modelo de "deep research", com subagentes do Claude Code.
- A favor: o resultado mais completo e mais confiável.
- Contra: o mais caro em tempo e assinatura; só vale depois de P1/P2 testados.

### Como seria a experiência

1. Na Biblioteca, seleciono um trecho ou digito um tema → **"Pesquisar no Acervo"**.
2. Escolho o alcance: rápido (P1, ~5 min) ou com meus livros (P2), e se quero que vire Tema novo
   ou complemente um existente.
3. A pesquisa entra na **fila** do Acervo; acompanho em Referências.
4. Pronta, aparece como Referência `pesquisa`, com a lista de fontes e as **Dúvidas**.
5. Daí em diante é o fluxo de sempre: sugerir → criar/integrar → revisar o diff.

### Recomendação

**P1 primeiro**, com regra rígida: toda afirmação com fonte, sem fonte não entra, e as páginas
salvas. É pequeno, testável em dias e já entrega "produzir conteúdo sobre um tema". **Depois P2**,
que além do Pesquisador destrava a verificação de veracidade. **P3** é um extra barato. **P4** só no
fim, juntando os três.

### Decisões para o Lucas

- [ ] Pode usar a web, ou só fontes escolhidas (livros, vídeos)?
- [ ] Quanto tempo/assinatura uma pesquisa pode gastar (limite por pesquisa)?
- [ ] O resultado é Referência (revisada como as outras) ou já um rascunho de Tema?
- [ ] O Pesquisador fica dentro do Acervo (proposta) ou vira um app próprio no ecossistema?
- [ ] Contrato comum: é a hora de defini-lo (Biblioteca e Acervo vão conversar de verdade).

---

## 4. Ordem sugerida

1. Acervo: Referência genérica (`fonte_tipo`, `fonte_ref`) + `POST /api/v1/referencias` aceitando
   um Fragmento (+ token simples, conforme o contrato).
2. Biblioteca: botão "Acervo" na seleção (B1) e link de volta (B3).
3. Acervo: Pesquisador P1 (tarefa `pesquisar`, regra, limite, fontes salvas).
4. Biblioteca: texto dos livros + busca FTS5 → B2 e Pesquisador P2 → verificação com fontes.
5. P3 e P4.
