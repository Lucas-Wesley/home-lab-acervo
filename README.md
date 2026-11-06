# Acervo

Gerenciador de conhecimento pessoal. Transforma conteúdos (hoje vídeos do YouTube; no futuro
artigos e outras fontes) em uma **base de conhecimento permanente** — organizada, verificada,
cronológica e versionada — construída de forma incremental ao longo do tempo.

> Não é um "resumidor de vídeo". O vídeo é só a matéria-prima; o produto é a base de conhecimento.

> **Status (06/11/2026):** primeira versão construída e testada localmente; ainda não publicada.
> Andamento em [`docs/CONSTRUCAO.md`](docs/CONSTRUCAO.md).

## As duas áreas

- **Acervo** — a base permanente, dividida em **Temas** (Segunda Guerra Mundial, Café, ...).
  Cumulativa, nunca substituída, só enriquecida. Subtema = pasta dentro do Tema; Temas se ligam
  por links.
- **Referências** — os conteúdos extraídos de fontes. Matéria-prima efêmera, com ciclo de vida.

## Fluxo em uma imagem

```
fonte (YouTube) ─> extrair ─> organizar ─> (traduzir) ─> sugerir Tema ─> criar/integrar ─> revisar diff
   (yt-dlp)        (cru)    (organizado)   (pt-BR)       (IA sugere)     (IA propõe)     (Lucas aprova)
```

Toda etapa com o `yt-dlp` ou o Claude entra numa **fila** e roda uma por vez.

## Documentação

- [`docs/DECISOES.md`](docs/DECISOES.md) — **comece por aqui.** Conceitos, fluxo, decisões, riscos,
  o que foi adiado e o que está em aberto.
- [`docs/CONSTRUCAO.md`](docs/CONSTRUCAO.md) — arquitetura e andamento da construção.
- [`docs/PESQUISADOR.md`](docs/PESQUISADOR.md) — caminhos para integrar a Biblioteca e para o
  módulo Pesquisador (planejamento, nada implementado).
- [`regras/`](regras/) — as regras que o Claude segue em cada etapa. **É o que define a qualidade.**
  O Lucas pode editá-las no app (Ajustes → Instruções da IA); a versão editada fica no servidor e
  vale no lugar destas.

## Rodar e testar

Precisa de Go, Node, `git`, `yt-dlp` e do Claude Code logado.

| Comando | O que faz |
|---------|-----------|
| `make dev-api` | API em `:8080`, dados em `./data` (fora do Git) |
| `make dev-web` | PWA no Vite, usando a API acima |
| `make build-web` | Gera o PWA dentro do binário Go (para rodar só com `go run`) |
| `make test` | Testes do Go, do Vue e de navegador (Playwright), com `yt-dlp` e `claude` falsos |

## Publicar no homelab

Igual à Biblioteca: `git push origin main` e `./deploy.sh`. Antes da primeira vez:

1. **Token do Claude (D13):** no Mac, `claude setup-token`; na VM, gravar
   `CLAUDE_CODE_OAUTH_TOKEN=<token>` em `/opt/apps/acervo.env` (fora da repo).
2. **Proxy:** o bloco do Acervo (porta 8444) já está em `homelab/proxy/`; copiar para
   `/opt/proxy` na VM e rodar `docker compose up -d` lá.
3. Endereço: `https://192.168.1.20:8444`.

## Mandar um vídeo do celular

Basta compartilhar o vídeo. A Referência é criada e entra na fila (extrair → organizar), e depois
espera a sua aprovação.

- **Android:** instale o PWA (Chrome → "Adicionar à tela inicial"). Depois, no YouTube, toque em
  Compartilhar → **Acervo**. O app abre por um instante, mostra "Vídeo enviado para a fila." e
  pronto. Isso usa o `share_target` do manifest (`frontend/vite.config.js`).
- **iPhone:** o Safari não deixa um PWA aparecer em Compartilhar (o WebKit não tem *Web Share
  Target*). A saída é um atalho do app **Atalhos**, que envia sem abrir tela nenhuma:
  1. Novo atalho → (i) → ligar **Mostrar na Folha de Compartilhamento**, aceitando **URLs** e
     **Texto**.
  2. Ação **Obter Conteúdo do URL**: URL `https://192.168.1.20:8444/api/referencias`, método
     **POST**, corpo **JSON** com o campo `url` = *Entrada do Atalho*.
  3. Ação **Mostrar Notificação**: "Enviado ao Acervo".
  4. Dê o nome **Acervo**. No YouTube: Compartilhar → Mais → **Acervo**.

  Funciona na rede de casa, com o certificado do homelab instalado no iPhone (o mesmo da
  Biblioteca). O back-end aceita o link sozinho ou junto com o título, e um vídeo repetido devolve
  "Este vídeo já está nas Referências."

## Ajustes

No fim do menu:

- **Instruções da IA:** o texto que o Claude segue em cada etapa (organizar, traduzir, sugerir
  Tema, criar Tema, integrar) e na pesquisa de "Mais informações". Dá para ler e editar pelo celular, e a mudança vale a partir da
  próxima tarefa. "Voltar ao padrão" volta ao texto de `regras/`.
- **Sugestões de melhoria:** anote o que pode melhorar no app. Depois, peça ao Claude Code para ver
  as sugestões: ele lê por `GET /api/sugestoes`, faz as mudanças e marca cada uma como feita, com
  uma resposta que aparece no app.

## Mais informações

Selecione um trecho de um Tema ou de uma Referência (um nome, um lugar, uma frase) e toque no ícone
**Mais informações**. A IA pesquisa na internet, em mais de uma fonte, e o trecho fica sublinhado:
toque nele para abrir a explicação, com as fontes. O mesmo menu tem **Copiar**. As instruções da
pesquisa estão em Ajustes → Instruções da IA → Mais informações.

## Identidade visual

Segue o design system comum às apps do homelab: [`docs/DESIGN-SYSTEM.md`](docs/DESIGN-SYSTEM.md).

## Dois repositórios

- **Código da aplicação** — este repo, no GitHub.
- **Conteúdo do Acervo** (os textos + o git do conteúdo) — **fica local no homelab**, em
  `/srv/data/acervo/conteudo`. **Não sobe para o GitHub.** Só o disco interno do M920q (D22).
