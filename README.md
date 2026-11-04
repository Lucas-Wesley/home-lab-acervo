# Acervo

Gerenciador de conhecimento pessoal. Transforma conteúdos (hoje vídeos do YouTube; no futuro
artigos e outras fontes) em uma **base de conhecimento permanente** — organizada, verificada,
cronológica e versionada — construída de forma incremental ao longo do tempo.

> Não é um "resumidor de vídeo". O vídeo é só a matéria-prima; o produto é a base de conhecimento.

> **Status:** fase de planejamento / documentação. Nada foi desenvolvido ainda.

## As duas áreas

- **Acervo** — a base permanente, dividida em **Temas** (Segunda Guerra Mundial, Café, ...).
  Cumulativa, nunca substituída, só enriquecida.
- **Referências** — os conteúdos extraídos de fontes. Matéria-prima efêmera, com ciclo de vida.

## Fluxo em uma imagem

```
fonte (YouTube) ─> extrair ─> reorganizar ─> verificar ─> classificar Tema ─> integrar ao Acervo
   (yt-dlp)        (cru)      (organizado)   (validado)   (novo? vira princ.)   (lacunas+aprovação)
```

## Documentação

- [`docs/DECISOES.md`](docs/DECISOES.md) — **comece por aqui.** Plano completo: conceitos, fluxo,
  decisões tomadas, pontos difíceis/riscos, o que foi adiado e o que está em aberto.

## Dois repositórios (previsto)

- **Código da aplicação** — pode ir para o GitHub (este repo).
- **Conteúdo do Acervo** (os textos + o git/versionamento) — **fica local no homelab**. **Não sobe
  para o GitHub.** Começa no **disco interno do M920q**; o **SSD externo** fica reservado para
  **backup futuro**.
