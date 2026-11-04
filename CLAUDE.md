# Acervo

App do **ecossistema de apps pessoais do Lucas**. Gerenciador de conhecimento: transforma conteúdos
(hoje YouTube; futuro: artigos e outras fontes) numa **base de conhecimento permanente, organizada,
verificada, cronológica e versionada**. **Não é um "resumidor de vídeo"** — o vídeo é matéria-prima;
o produto é a base. É o **2º app a ser desenvolvido** (ver ROADMAP do Hub).

**Comece lendo:**
- [`docs/DECISOES.md`](docs/DECISOES.md) — **a fonte da verdade do projeto.** Conceitos, fluxo
  completo, decisões (D1–D10), pontos difíceis/riscos, o que foi adiado e o que está em aberto.

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
Homelab **LAN-only**: Proxmox no Lenovo M920q (`192.168.1.10`, sem GPU). App web + Claude Code
headless rodam aí; Mac mini M4 disponível pra inferência local (Whisper) quando precisar.

## Regras e convenções (IMPORTANTE)
- **O CONTEÚDO do Acervo (textos + git do conteúdo) NÃO sobe pro GitHub** — fica local no homelab.
  **Só o código da app** vai pra este repositório.
- **Armazenamento:** conteúdo começa no **disco interno do M920q**; **SSD externo = backup futuro**
  (ext4). Ver `docs/DECISOES.md` §4.
- **Verificação de veracidade é sempre com decisão humana** — a IA sinaliza, o Lucas decide. Nada de
  descarte automático.
- Antes de automatizar, **validar as regras de cada etapa em modo manual** (elas viram o futuro
  `CLAUDE.md` que define a qualidade — hoje ainda em aberto).
- **Commits:** autoria do Lucas; **sem linha `Co-Authored-By`**.
