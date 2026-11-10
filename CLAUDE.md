# Acervo

App do **ecossistema de apps pessoais do Lucas**. Gerenciador de conhecimento: transforma conteúdos
(vídeos do YouTube, Reels do Instagram, podcasts, artigos e páginas da web, pesquisas e capítulos de livros) numa **base de conhecimento permanente, organizada,
verificada, cronológica e versionada**. **Não é um "resumidor de vídeo"** — o vídeo é matéria-prima;
o produto é a base. É o **2º app a ser desenvolvido** (ver ROADMAP do Hub).

**Comece lendo:**
- [`docs/DECISOES.md`](docs/DECISOES.md) — **a fonte da verdade do projeto.** Conceitos, fluxo
  completo, decisões (D1–D40), pontos difíceis/riscos, o que foi adiado e o que está em aberto.

## Em uma frase
Duas áreas: **Acervo** (base permanente por **Temas**, Markdown multi-arquivo, cumulativa) e
**Referências** (conteúdos extraídos, efêmeros, com estados cru → organizado → validado →
integrado/descartado). Fluxo: extrair (`yt-dlp`) → reorganizar → verificar (humano decide) →
classificar Tema → integrar (lacunas + aprovação; *diff* do git é a revisão).

## Decisões-chave
- **Engine:** Claude Code CLI **headless** (`claude -p`) — reaproveita a assinatura.
- **Dados:** Markdown **multi-arquivo por Tema** (índice + tópicos linkados), **sem RAG**.
- **Sem RAG por ora** — a estrutura resolve a escala.

## Contexto do ecossistema
App **independente** (banco + API próprios) que conversa com os outros **por API**. Visão geral,
roadmap e contrato comum no repo **Hub**: `github.com/Lucas-Wesley/hub-home-lab-web` (pasta `docs/`).

## Ambiente
Homelab **LAN-only**: Proxmox no Lenovo M920q (sem GPU). App web + Claude Code headless rodam na
VM `apps` (`192.168.1.20`), com Docker Compose e Caddy, igual à Biblioteca Digital (D11). Mac mini M4
disponível pra inferência local (Whisper) quando precisar.

## Regras e convenções (IMPORTANTE)
- **O CONTEÚDO do Acervo (textos + git do conteúdo) NÃO sobe pro GitHub** — fica local no homelab.
  **Só o código da app** vai pra este repositório.
- **Armazenamento:** só o **disco interno do M920q**; **SSD externo descontinuado** (D22).
- **Verificação de veracidade é sempre com decisão humana** — a IA sinaliza, o Lucas decide. Nada de
  descarte automático.
- Antes de automatizar, **validar as regras de cada etapa em modo manual** (elas viram o futuro
  `CLAUDE.md` que define a qualidade — hoje ainda em aberto).
- Toda chamada ao Claude ou ao `yt-dlp` passa pela **fila** (uma tarefa por vez, D14).
- **Internet só em "Mais informações"** (D26), **em "Perguntar"** (D35, só leitura de uma cópia do texto) **e na escrita de uma pesquisa com "Sites"** (D27): as
  outras etapas continuam sem `WebSearch`/`WebFetch`. A coleta do Pesquisador (livros, Wikipédia,
  YouTube) é feita pelo servidor, sem a IA.
- **Podcasts** (D39): o áudio vem do feed RSS aberto do podcast (o do Spotify é protegido), sem IA; a
  transcrição é do Whisper large-v3 no Mac mini (`WHISPER_URL`), com o Whisper do servidor de reserva
  (`WHISPER_RESERVA_URL`). Nos testes, servidores falsos (`podcast_test.go`): nada do Spotify, da Apple ou do Whisper de verdade.
- **Reels do Instagram** (D40): yt-dlp (público sem login; com os cookies da conta secundária em
  `DATA_DIR/instagram-cookies.txt` quando o Instagram pede, ver `docs/INSTAGRAM.md`), Whisper para a fala
  e o Claude lendo os quadros do ffmpeg (`regras/ver-quadros.md`). Nos testes, o `yt-dlp-falso` gera o vídeo com o ffmpeg.
- **Artigos** (D34): o servidor lê a página sem IA (`artigo.go`), sem nada específico de um site;
  o que sobrar do site, a etapa de organizar tira (`regras/organizar-artigo.md`).
- **Flashcards** (D28, D31): o Acervo só manda o texto (com o baralho escolhido e o link de cada capítulo) ao app Flashcards (`FLASHCARDS_URL`); a IA e as regras dos
  cartões ficam lá (`flashcards/regras/gerar-do-texto.md`).
- **As instruções em vigor podem estar editadas no app** (D25). Antes de mudar `regras/`, ler as
  versões do servidor (`GET https://192.168.1.20:8444/api/instrucoes/<arquivo>`, com
  `--cacert ../certs/homelab-ca.crt`). Se uma estiver editada, levar a edição para `regras/` e só
  então voltar a do servidor ao padrão (`DELETE`), para nada se perder.
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
