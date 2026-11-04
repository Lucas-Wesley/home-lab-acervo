# Acervo — Plano do Projeto

> Gerenciador de conhecimento pessoal. Registro das decisões até **03/11/2026**.
> Fonte da verdade do projeto enquanto o código não existe. Ao clonar/retomar em outra máquina,
> leia este arquivo inteiro antes de qualquer coisa.

---

## 1. O que é

Sistema pessoal para transformar conteúdos (hoje **vídeos do YouTube**; no futuro artigos e outras
fontes) em uma **base de conhecimento permanente** — organizada, verificada, cronológica e
versionada — construída incrementalmente ao longo do tempo.

**Não é um "resumidor de vídeo".** O vídeo é só matéria-prima. O produto é a base de conhecimento
que cresce e fica cada vez mais completa.

---

## 2. Vocabulário — as duas áreas

- **Acervo (base permanente):** o conhecimento consolidado, dividido por **Temas** (Segunda Guerra
  Mundial, Café, ...). Permanente, cumulativo, **nunca substituído — só enriquecido**.
- **Referências (efêmeras):** os conteúdos extraídos de fontes. Matéria-prima com ciclo de vida
  (estados). Eu gerencio: posso integrar, manter ou descartar.

### Estrutura de um Tema — multi-arquivo (NÃO um arquivo único)

Cada Tema é uma **pasta** em Markdown:
- um **`indice.md`** (índice/sumário) que começa pela introdução e **linka** para os demais;
- vários **arquivos de tópico** (`introducao.md`, `contexto.md`, `alemanha-relacoes.md`, ...),
  cada um referenciado pelo índice.

O índice se atualiza conforme novos conteúdos entram.

> **Por que multi-arquivo e não RAG:** quebrar o Tema em índice + tópicos mantém cada operação
> pequena (mexe só no índice + no arquivo do tópico certo), então não precisamos de RAG para lidar
> com "texto grande" por enquanto. A estrutura resolve a escala.

### Estados de uma Referência

```
cru ─> organizado ─> em verificação ─> validado ─> integrado
                                                 └─> descartado
```
- **cru:** texto extraído bruto da fonte (preservado, para poder reverter).
- **organizado:** reestruturado e com complementos temporais consolidados (ver fluxo, passo 2).
- **em verificação / validado:** passou pela checagem de veracidade com decisão humana.
- **integrado / descartado:** eu decido o destino.

---

## 3. Fluxo completo (da URL ao Acervo)

1. **Extrair** a transcrição da fonte. YouTube → `yt-dlp`. Futuro **breve**: **Whisper** quando a
   legenda automática for ruim ou ausente. → estado **`cru`**.
2. **Organizar / reorganizar:** estruturar o texto e **consolidar complementos temporais** — num
   podcast, algo dito no minuto 5 é complementado no minuto 20; a reorganização junta isso num
   fluxo lógico/temático, **não** na ordem falada. (O `cru` fica preservado.) → **`organizado`**.
   - *Observação:* são duas ordens diferentes — a **ordem do discurso** (como foi falado) vira a
     **ordem lógica/cronológica do conteúdo** (como o assunto se organiza de fato).
3. **Verificar veracidade** antes de virar referência utilizável. A IA **sinaliza** afirmações
   factuais duvidosas e **pergunta a mim**; a decisão é **humana** — nada de descarte automático.
   → **`validado`**.
4. **Classificar o Tema:** a IA analisa em qual Tema do Acervo a referência se encaixa.
   - **Não existe Tema** → esta referência **vira o Texto Principal de um novo Tema**.
   - **Existe** → segue para integração.
5. **Integrar ao Acervo** (quando eu mandar): análise de **lacunas** (o que a referência tem que o
   Tema ainda não tem) → **proposta visual** de onde encaixar (sob quais tópicos/arquivos) → **eu
   reviso e aprovo** → o Acervo incorpora **só o que falta**. O *diff* do git é a tela de revisão.

> Com o tempo o Texto Principal fica mais completo, então cada nova Referência acrescenta menos —
> os detalhes finos se encaixam nos tópicos existentes em vez de criar tudo do zero.

---

## 4. Decisões tomadas

| # | Decisão | Escolha | Por quê |
|---|---------|---------|---------|
| D1 | Caminho geral | Pipeline próprio (DIY), **não** SurfSense | Controle total do prompt/regras, sem licença de plugin, transparente. Sou dev. |
| D2 | Extração | **`yt-dlp`** hoje; **Whisper** em breve; fontes **plugáveis** (só YouTube por ora) | Ferramentas prontas/gratuitas; arquitetura aberta a novas fontes. |
| D3 | Engine de IA | **Claude Code CLI headless** (`claude -p`) | Reaproveita minha **assinatura** + libera skills/subagents/MCP/hooks/CLAUDE.md. |
| D4 | Orquestração | **App web self-hosted** no homelab, acessada pelo navegador do celular | Dispensa app nativo; eu construo. |
| D5 | Validação | Validar as **regras de cada etapa em modo manual** antes de automatizar | Acerta o formato antes de investir na infra. |
| D6 | Nome | **Acervo** (base permanente) + **Referências** (efêmeros) | Descreve o que o sistema é. |
| D7 | Estrutura de dados | **Markdown multi-arquivo por Tema** (índice + tópicos linkados); **sem RAG** por ora | A estrutura resolve a escala; mantém cada operação pequena. |
| D8 | Versionamento | **Git local no homelab**; **conteúdo NÃO sobe pro GitHub** | Privacidade; histórico/reversão de graça; *diff* = tela de revisão. |
| D9 | Reorganização na extração | **Sim** (consolidar complementos temporais) | Vídeo falado não é linear; o conteúdo precisa ser. |
| D10 | Checagem de veracidade | Estados + **revisão humana** (IA sinaliza, eu decido) | Evitar propagar informação falsa sem perder controle. |

### Consequências do engine (D3)
- Sujeito aos **limites de uso da assinatura**.
- O servidor precisa de **Node + Claude Code instalado e autenticado** (login OAuth uma vez).
- Reserva: **API do Claude (Messages API)** se a assinatura não atender.

### Armazenamento (decidido 03/11/2026)
- **Começar no disco interno do M920q** (tem espaço/RAM de sobra para começar) — evita montar storage
  cedo demais.
- O **SSD externo (ex-Mac)** fica reservado para **backup** (formatar em **ext4** pelo Proxmox — o
  formato do Mac, APFS/HFS+, não serve no Linux).
- **Migrar para storage dedicado quando os arquivos crescerem**, decidindo o filesystem (ext4 vs ZFS)
  com números reais. O texto do Acervo é leve; quem tende a crescer é arquivo binário (gatilho
  provável vem da Biblioteca Digital, não do Acervo).

---

## 5. Pontos difíceis e riscos (honestidade)

- **Integração / análise de lacunas (o coração):** factível com Claude, mas depende da estrutura
  (índice + tópicos = âncoras de encaixe) e tem o **gate humano** como salvaguarda. A IA pode
  propor redundância ou errar o encaixe — por isso eu aprovo.
- **Checagem de veracidade (a parte mais difícil):** *IA verificando com o próprio conhecimento é
  fraca* — é onde mora a alucinação. Checagem séria exige **fontes externas confiáveis** (bases
  públicas, livros) = problema de busca à parte, possível **app dedicado depois**.
  **Abordagem recomendada (faseada):** começar leve — a IA **marca** afirmações checáveis + **nível
  de confiança** + sugere **o que/onde conferir**; eu decido. A versão automática contra bases
  autoritativas fica adiada. Faz mais sentido para conteúdo **factual/histórico** (datas, nomes,
  eventos) do que para opinião/análise.
- **Backup:** git versiona o histórico, mas **versionar ≠ backup** e o conteúdo **não vai pro
  GitHub**, então precisa de uma **segunda cópia**. Plano atual: usar o **SSD externo como backup**
  do conteúdo que mora no disco interno do M920q. Atenção: backup **só vale se rodar de fato**
  (job periódico), não SSD guardado na gaveta. Urgência baixa até o Acervo acumular conteúdo real.
- **Reorganização:** força da IA, mas pode errar atribuição/perder nuance → manter o `cru`
  preservado permite re-derivar.
- **Escala:** a estrutura multi-arquivo adia o problema por muito tempo; só um Tema gigante traria
  de volta a necessidade de busca.

---

## 6. Adiado (decidido que fica para depois)

| Item | Observação |
|------|------------|
| Cenário "estudar um assunto do zero" (pesquisa automática → vídeos/livros/artigos) | Fora do foco atual. |
| **Whisper** (transcrição de alta qualidade) | Futuro **breve**; começar com legenda do YouTube. |
| Fontes além do YouTube (artigos etc.) | Arquitetura já prevê plugável. |
| App dedicado de verificação contra bases externas | Depois da versão leve de checagem. |
| **API do Claude** como engine | Reserva, se a assinatura não atender. |

---

## 7. Em aberto (a definir antes/durante o desenvolvimento)

- [ ] **Regras de cada etapa** (organização, formato dos tópicos, o que o índice mostra, como citar
      referências/livros/autores sem inventar) → viram o `CLAUDE.md` do projeto. **É o que define a
      qualidade.**
- [ ] **Como fazer a checagem de veracidade** (fontes, método, interface de decisão).
- [ ] **Separação de repositórios:** código da app (GitHub) × conteúdo do Acervo (git local no
      SSD). Confirmar.
- [ ] Stack da app web; LXC vs VM. (Storage inicial decidido: disco interno do M920q — ver §4.)
- [ ] **Taxonomia de Temas:** Tema grande vs sub-temas; múltiplos temas por referência;
      reclassificação.

---

## 8. Alternativas avaliadas e descartadas (para não reavaliar à toa)

- **SurfSense** (https://github.com/MODSetter/SurfSense) — self-hosted via Docker, cobre YouTube +
  pesquisa + respostas citadas. **Descartado** para este projeto: plugins oficiais exigem licença e
  dá menos controle sobre o prompt e sobre a estrutura do Acervo. Mantido como referência.
- **NotebookLM** (Google, grátis) — bom para validar qualidade com citações; não é self-hosted.
- **Fabric** (CLI, Daniel Miessler) — ótimo para extrair ideias/referências de um vídeo; não
  gerencia uma base de conhecimento.

---

## 9. Contexto do ambiente (homelab)

- **Proxmox VE 9.2** no **M920q** (i7-8700T, **sem GPU**) — IP `192.168.1.10`. Roda a app web e o
  Claude Code headless.
- **Mac mini M4** — bom para inferência local (Whisper/Ollama) quando necessário.
- **Armazenamento:** começa no **disco interno do M920q** (guarda os arquivos do Acervo + o git). O
  **SSD externo** fica reservado para backup futuro (formatar em ext4).
- Serviços do homelab ficam **LAN-only, não expostos à internet**.
