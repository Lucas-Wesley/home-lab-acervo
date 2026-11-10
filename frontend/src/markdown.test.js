import { describe, expect, it } from 'vitest'
import { duvidas, eLinkInterno, relativo, renderizar, resolver, semCabecalho, tituloDoTexto, trocarFontes, urlNoTempo, listaDeDuvidas } from './markdown.js'

describe('links entre arquivos', () => {
  it('resolve caminhos relativos a partir do arquivo atual', () => {
    expect(resolver('temas/guerra/indice.md', 'frente-leste.md')).toBe('temas/guerra/frente-leste.md')
    expect(resolver('temas/guerra/frente/indice.md', '../indice.md')).toBe('temas/guerra/indice.md')
    expect(resolver('temas/a/indice.md', '../b/./indice.md')).toBe('temas/b/indice.md')
  })

  it('monta o caminho relativo para linkar outro Tema', () => {
    expect(relativo('temas/guerra/indice.md', 'temas/alemanha-russia/indice.md')).toBe('../alemanha-russia/indice.md')
    expect(relativo('temas/guerra/indice.md', 'temas/guerra/frente/indice.md')).toBe('frente/indice.md')
    expect(relativo('temas/a/b/topico.md', 'temas/c/indice.md')).toBe('../../c/indice.md')
    expect(resolver('temas/a/b/topico.md', relativo('temas/a/b/topico.md', 'temas/c/indice.md'))).toBe('temas/c/indice.md')
  })

  it('reconhece só links internos para Markdown', () => {
    expect(eLinkInterno('topico.md')).toBe(true)
    expect(eLinkInterno('../x/indice.md#parte')).toBe(true)
    expect(eLinkInterno('https://exemplo.com/a.md')).toBe(false)
    expect(eLinkInterno('#ancora')).toBe(false)
    expect(eLinkInterno('imagem.png')).toBe(false)
  })
})

describe('fontes e marcas de tempo', () => {
  const fontes = [{ id: 'yAgLfb-e8sY', titulo: 'As origens do gaúcho', url: 'https://www.youtube.com/watch?v=yAgLfb-e8sY' }]

  it('converte a marca em segundos na URL', () => {
    expect(urlNoTempo('https://www.youtube.com/watch?v=abc', '1:02:05')).toBe('https://www.youtube.com/watch?v=abc&t=3725s')
  })

  it('troca o id da fonte pelo título com links para cada momento', () => {
    const html = trocarFontes('<sub>Fonte: yAgLfb-e8sY [00:35], [05:59]</sub>', fontes)
    expect(html).toContain('>As origens do gaúcho</a>')
    expect(html).toContain('t=359s')
    expect(html).toContain('>05:59</a>')
  })

  it('deixa como está quando a fonte não é conhecida', () => {
    expect(trocarFontes('<sub>Fonte: outra [00:01]</sub>', fontes)).toBe('<sub>Fonte: outra [00:01]</sub>')
  })
})

describe('renderizar', () => {
  it('tira o cabeçalho, marca links internos e limpa HTML perigoso', () => {
    const html = renderizar('---\ntitulo: X\n---\n\n# X\n\n[Tópico](topico.md) [site](https://a.b) <img src=x onerror=alert(1)><script>alert(1)</script>')
    expect(html).not.toContain('titulo:')
    expect(html).toContain('class="interno"')
    expect(html).toContain('target="_blank"')
    expect(html).not.toContain('onerror')
    expect(html).not.toContain('<script')
  })

  it('separa a seção de dúvidas e acha o título', () => {
    const texto = '# Café\n\n## História\n\nTexto.\n\n## Dúvidas\n\n- nome incerto\n'
    expect(listaDeDuvidas(texto + '- outro, em\n  duas linhas\n')).toEqual(['nome incerto', 'outro, em duas linhas'])
    expect(duvidas(texto)).toBe('- nome incerto')
    expect(duvidas('# Sem\n\n## Outra\n')).toBe('')
    expect(tituloDoTexto(texto)).toBe('Café')
    expect(tituloDoTexto('---\ntitulo: Do cabeçalho\n---\n# Outro')).toBe('Do cabeçalho')
    expect(semCabecalho('# Sem cabeçalho')).toBe('# Sem cabeçalho')
  })
})

describe('capítulos', () => {
  const texto = '# T\n\nIntro.\n\n## Um\n\nA\n\n```\n## falso\n```\n\n## Dois\n\n[Ler o capítulo →](dois.md)\n\n## Linha do tempo\n\n- x\n'

  it('acha os capítulos como o servidor, inclusive os separados', async () => {
    const { capitulos } = await import('./markdown.js')
    const lista = capitulos(texto)
    expect(lista.map((c) => c.titulo)).toEqual(['Um', 'Dois', 'Linha do tempo'])
    expect(lista[0].arquivo).toBe(null)
    expect(lista[1].arquivo).toBe('dois.md')
    expect(lista[2].apendice).toBe(true)
    expect(texto.slice(lista[0].ini, lista[0].fim)).toContain('## falso')
  })

  it('troca só o capítulo editado e insere capítulo novo antes dos apêndices', async () => {
    const { capitulos, trocarCapitulo, inserirCapitulo } = await import('./markdown.js')
    const [um] = capitulos(texto)
    const novo = trocarCapitulo(texto, um, '## Um\n\nA revisado')
    expect(novo).toContain('## Um\n\nA revisado\n\n## Dois')
    expect(novo.startsWith('# T\n\nIntro.\n\n')).toBe(true)
    expect(inserirCapitulo(texto, 'Três')).toContain('[Ler o capítulo →](dois.md)\n\n## Três\n\n## Linha do tempo')
  })

  it('dá âncora aos títulos e marca o capítulo separado', () => {
    const html = renderizar(texto)
    expect(html).toContain('id="cap-um"')
    expect(html).toContain('class="capitulo-separado"')
  })
})
