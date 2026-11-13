# Trilhas de aprendizado: plano

Pedido do Lucas em 13/11/2026. Este arquivo é o plano para continuar aos poucos. O que já foi feito está
marcado em cada fase.

## A ideia

Hoje o Acervo responde a **"o que eu sei"**: os Temas guardam o conhecimento organizado por assunto. A
trilha responde a outra pergunta: **"em que ordem eu aprendo isto, e quanto já domino"**.

- **Tema** guarda o conhecimento. É permanente e cresce com as Referências.
- **Trilha** é um percurso: etapas em ordem, cada uma apontando para Temas, Referências, capítulos de
  livros e atividades. Ela tem progresso.

Uma trilha **liga** os Temas e não copia o texto deles, como os links entre Temas (D20). O mesmo Tema
pode estar em várias trilhas. "Crase" pode estar em "Gramática normativa" e em "Português para
concurso". "Revolução Russa" pode estar em "Século XX" e em "História do socialismo".

### Por que faz sentido (e onde é preciso cuidado)

O pedido tem base no que se sabe sobre aprender:

- **Ordem e pré-requisitos.** Um conceito difícil fica fácil quando a base vem antes. A trilha torna a
  ordem explícita.
- **Domínio antes de avançar** (*mastery learning*). Cada etapa tem um critério, por exemplo 80% nos
  exercícios e no simulado. A trilha sugere o próximo passo, e o Lucas sempre pode pular.
- **Lembrar ativamente** (*retrieval practice*). Simulado, exercícios e flashcards ensinam mais do que
  reler.
- **Revisão espaçada.** O Flashcards já faz isso. A trilha traz de volta as etapas antigas.
- **Intercalar.** O simulado da trilha mistura etapas, porque só assim se aprende a escolher qual
  regra usar, e não só a aplicar a regra do capítulo do dia.
- **Feedback rápido.** Os exercícios corrigem na hora. O simulado é corrigido pela IA em minutos.

Os riscos, com honestidade:

1. **Construir a plataforma em vez de estudar.** É o maior risco. A trilha começa mínima e só ganha
   recursos depois de usada de verdade (como na regra "validar em modo manual antes de automatizar").
2. **A IA erra a ordem e a curadoria.** Ela sugere e o Lucas aprova, como na verificação de
   veracidade. A trilha é dele.
3. **Falta material.** Uma trilha de "Arquitetura de software" pede etapas que o Acervo ainda não tem.
   A etapa sem material aparece como lacuna, com sugestões (Pesquisador, livros da Biblioteca, vídeos),
   em vez de a IA inventar o conteúdo.
4. **Medir domínio é aproximado.** Uma nota de simulado indica, mas não prova. O progresso mostra os
   sinais (notas, exercícios, cartões) e não um número mágico.
5. **Custo da assinatura.** Preparar a trilha, os exercícios e os simulados gasta Claude. Tudo fica
   guardado e é refeito só quando o Lucas pede ou quando o texto muda (como no D41).

## Onde mora

**Dentro do Acervo**, como uma terceira área ao lado de Acervo e Referências. A trilha depende dos
Temas, da fila e do Claude, que estão aqui, e a aba Estudar já é a unidade de estudo de cada etapa.

Se as ferramentas por área (terminal, código) crescerem muito, elas podem virar um app à parte que o
Acervo chama por API, como o Flashcards.

- A **estrutura da trilha** é permanente e curada, como um Tema. Ela fica no conteúdo, em
  `trilhas/<slug>.md`, versionada no git do conteúdo (e fora do GitHub, como o resto do conteúdo).
- O **progresso** é derivado e muda o tempo todo. Ele fica no SQLite, como o estudo (D41).

Esboço do arquivo:

```markdown
---
titulo: Gramática normativa
objetivo: Escrever sem erros de norma e saber justificar cada regra.
---

## 1. Fonética e ortografia
Objetivo: separar sílabas, acentuar e saber por quê.
Pré-requisitos: —
Materiais: [Acentuação](../temas/acentuacao/indice.md) · Referência: Aula 3 (curso de gramática)
Concluir com: exercícios ≥ 80%

## 2. Crase
Pré-requisitos: 1, Artigos e preposições
...
```

## Fases

### Fase 0: a aba Estudar mais forte (13/11/2026, D52)

- [x] **Exercícios**, um cartão novo. São prática com resposta certa: completar a frase ou escolher entre
  opções ("Aqui vai crase?"). A correção é na hora, com a explicação do porquê. "Gerar mais" pede um
  lote novo, que pode ter um foco. Os errados voltam primeiro.
- [x] **Simulado dinâmico.** "Gerar mais perguntas" (com foco opcional) acrescenta ao banco sem repetir
  as que já existem. O sorteio prefere as perguntas nunca feitas e as erradas. As perguntas puxam mais
  o **porquê**.
- [x] **Resumo inteligente revisto.** Fala do conteúdo e não da fonte. Tem exemplos e um "Como aplicar"
  quando o assunto é regra ou técnica. Não cita nomes de arquivo.

### Fase 1: as aulas do Telegram viram Referências

O Streaming já baixa os vídeos (S42). Falta mandá-los ao Acervo, como os documentários (D50/S41):

- Em **Meus vídeos**: "Mandar ao Acervo" numa pasta (o curso inteiro, em ordem) ou num vídeo.
- O Streaming tira o áudio (mono, comprimido) e manda a `POST /api/aulas` do Acervo, com o curso (a
  pasta), o número e o título da aula.
- O Acervo cria a Referência do tipo **aula**:
  - O texto vem do Whisper do Mac mini.
  - Organiza com a regra nova `organizar-aula.md`, que guarda as regras, os exemplos e os exercícios
    que o professor faz, e tira as conversas de sala.
  - Liga as aulas do mesmo curso pela chave, como os episódios, e cada aula sabe o que as anteriores
    já explicaram.
- Na repo, nada do grupo nem do id do Telegram. A chave é o nome da pasta.

### Fase 2: a trilha mínima

- Criar uma trilha à mão: título, objetivo e etapas em ordem. Cada etapa aponta para Temas e
  Referências.
- A página da trilha mostra:
  - as etapas;
  - o **próximo passo**;
  - o progresso de cada etapa, tirado do que já existe (notas do simulado, acertos nos exercícios,
    resumo lido, cartões feitos).
- Marcar uma etapa como concluída, à mão.
- Abrir a etapa leva à aba Estudar do Tema dela.

O primeiro teste de verdade é com o curso de gramática do Telegram.

### Fase 3: a curadoria da IA

- **Sugerir uma trilha a partir de um objetivo** ("quero entender o século XX", "DDD"). A IA propõe as
  etapas em ordem, os pré-requisitos e o critério de cada uma. Ela marca o que já está no Acervo e o
  que é lacuna.
- **Sugerir trilhas a partir do que já existe.** A IA lê o mapa do Acervo (`acervo.md`, o mesmo do
  Conversar) e propõe agrupamentos ("estes 9 Temas formam uma trilha sobre o século XX, nesta ordem").
- **Lacunas.** Cada etapa sem material ganha "Pesquisar" (Pesquisador, D27), livros da Biblioteca
  (D38) ou vídeos sugeridos.
- Tudo volta como **proposta** para aprovar, com o diff do arquivo da trilha, como no Integrar.

### Fase 4: domínio e revisão

- O **critério de conclusão** de cada etapa é conferido sozinho, e a próxima etapa é liberada como
  sugestão.
- O **simulado da trilha** mistura as etapas feitas (intercalar).
- **Revisão espaçada das etapas.** Uma etapa concluída volta depois de alguns dias, com perguntas
  rápidas.
- **Pontos fracos.** Lista as regras e os conceitos que mais se erram, com o link para reler.
- **Explicar com as suas palavras** (Feynman, ESTUDAR.md item 1). O Lucas fala o que entendeu e a IA
  aponta o que faltou.
- Um **painel** com as trilhas em andamento, o tempo de estudo e as notas.

### Fase 5: ferramentas por área

- **Programação:**
  - exercícios de "qual a saída deste código", "ache o erro" e "complete a função";
  - depois, rodar o código num container sem rede e descartável, com testes que conferem.
- **Linux:** um terminal no navegador, ligado a um container efêmero, sem rede e com limite de CPU e
  memória. Desafios conferidos por script ("dê permissão 640 a este arquivo"). Dá para fazer no
  homelab, mas o isolamento é a parte séria. Fica para quando a trilha de Linux existir de verdade.
- **História:** linha do tempo da trilha e mapa de causa e consequência.
- **Arquitetura, DDD e dados:** estudos de caso ("modele este domínio"), corrigidos pela IA com
  critérios. Também a leitura guiada de livros, capítulo a capítulo, ligada à Biblioteca.

## Em aberto (perguntar ao Lucas antes da fase 2)

- O critério de conclusão é fixo (80%) ou escolhido por etapa?
- A trilha tem prazo e meta semanal, ou só ordem?
- Um capítulo de livro (Biblioteca) pode ser a única fonte de uma etapa, sem virar Tema?
