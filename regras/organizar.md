# Regras: organizar uma Referência

Você recebe `cru.md`: a transcrição bruta de um vídeo, com marcas de tempo `[mm:ss]` a cada ~30 s. Escreva `organizado.md`: o mesmo conteúdo, organizado para alguém que **não sabe nada** do assunto entender.

## O que fazer

1. **Ler a transcrição inteira antes de escrever.** Quem fala num vídeo volta a assuntos, completa no minuto 20 o que começou no minuto 5. Junte cada assunto num lugar só.
2. **Ordem cronológica dos fatos**, não a ordem em que foram falados. Se a ordem cronológica atrapalhar o entendimento (ex.: um conceito precisa vir antes da história), use a ordem mais didática.
3. **Remover o que não é conteúdo:** publicidade, patrocinadores, pedidos de inscrição/like/sininho, agradecimentos, piadas e conversa paralela, comentários sobre a gravação ou o canal.
4. **Manter tudo o que é conteúdo.** Isto não é um resumo: é a base de conhecimento. Datas, nomes, lugares, números, causas e consequências, citações e explicações ficam.
5. **Escrever em prosa clara**, em parágrafos, **no idioma da transcrição** (um vídeo em inglês é organizado em inglês; a tradução é outra etapa). Lista só quando o conteúdo for de fato uma lista.

## O que não fazer

- **Não acrescentar fatos** que não estão na transcrição, nem do seu conhecimento. O texto é o que o vídeo diz.
- **Não apagar opiniões, mas atribuí-las:** "segundo Eduardo Bueno, ...". Fato e opinião não se misturam.
- **Erros de transcrição automática:** corrija só quando o contexto deixar claro (nome conhecido escrito errado, palavra trocada). Se não tiver certeza, mantenha e marque com `[?]`.

## Formato de `organizado.md`

```markdown
# <título do assunto, não o título do vídeo>

> <2 ou 3 frases dizendo do que o texto trata>

## <seção>

<parágrafos>

<sub>Trechos: [mm:ss], [mm:ss]</sub>

## ...

## Linha do tempo

- **<data ou período>** — <acontecimento>

## Citados no vídeo

- <livros, autores, documentos, pessoas citadas como fonte; só os que o vídeo menciona>

## Dúvidas

- <trechos marcados com [?] e o que ficou incerto; omitir a seção se não houver>
```

- Cada seção termina com as marcas de tempo de onde veio o conteúdo, para conferir no vídeo.
- **Linha do tempo:** só se o assunto tiver datas ou períodos.
- **Citados no vídeo:** omitir a seção se o vídeo não citar nenhuma fonte.
