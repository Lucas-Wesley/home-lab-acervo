import { beforeEach, describe, expect, it, vi } from 'vitest'
import { acharTrecho, enviarDetalhesPendentes, lerDetalhes, marcar, pedirDetalhe, textoDe, trechoDaSelecao } from './detalhes.js'

function montar(...htmls) {
  return htmls.map((html) => {
    const el = document.createElement('article')
    el.innerHTML = html
    document.body.append(el)
    return el
  })
}

describe('achar e sublinhar o trecho', () => {
  beforeEach(() => (document.body.innerHTML = ''))

  it('lê a seleção com o que vem antes e depois, sem os espaços das pontas e sem botões', () => {
    const raizes = montar('<h2>Café <button>⋯</button></h2>', '<p>O café veio da <strong>Etiópia</strong>, segundo a lenda.</p>')
    const intervalo = document.createRange()
    const forte = raizes[1].querySelector('strong').firstChild
    intervalo.setStart(raizes[1].querySelector('p').firstChild, 14) // o espaço antes de "Etiópia"
    intervalo.setEnd(forte, forte.length)
    expect(trechoDaSelecao(raizes, intervalo)).toEqual({ trecho: 'Etiópia', antes: 'Café O café veio da ', depois: ', segundo a lenda.' })
    expect(textoDe(raizes).texto).not.toContain('⋯')
  })

  it('ignora a seleção fora do texto', () => {
    const [texto] = montar('<p>Texto</p>')
    const fora = document.createElement('p')
    fora.textContent = 'Fora'
    document.body.append(fora)
    const intervalo = document.createRange()
    intervalo.selectNodeContents(fora)
    expect(trechoDaSelecao([texto], intervalo)).toBeNull()
  })

  it('escolhe a ocorrência com o mesmo contexto', () => {
    const texto = 'Roma caiu. Depois, Roma renasceu.'
    expect(acharTrecho(texto, { trecho: 'Roma', antes: 'Depois, ', depois: ' renasceu' })).toBe(19)
    expect(acharTrecho(texto, { trecho: 'Roma' })).toBe(0)
    expect(acharTrecho(texto, { trecho: 'Atenas' })).toBe(-1)
  })

  it('sublinha atravessando negrito e partes da página, e separa o que sumiu do texto', () => {
    const { htmls, semLugar } = marcar(
      ['<p>Intro sobre <em>café</em> e</p>', '<p>Etiópia no corpo.</p>'],
      [
        { id: 'a', estado: 'pronto', trecho: 'café e', antes: 'Intro sobre ', depois: '' },
        { id: 'b', estado: 'processando', trecho: 'Etiópia' },
        { id: 'c', estado: 'pronto', trecho: 'Iêmen' },
      ],
    )
    expect(htmls[0]).toBe('<p>Intro sobre <em><mark class="detalhe pronto" data-detalhe="a">café</mark></em><mark class="detalhe pronto" data-detalhe="a"> e</mark></p>')
    expect(htmls[1]).toBe('<p><mark class="detalhe processando" data-detalhe="b">Etiópia</mark> no corpo.</p>')
    expect(semLugar.map((m) => m.id)).toEqual(['c'])
  })
})

function resposta(status, corpo) {
  return Promise.resolve(new Response(JSON.stringify(corpo), { status, headers: { 'Content-Type': 'application/json' } }))
}

describe('pedido fora da rede', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('guarda o pedido no aparelho e envia quando a rede volta', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('sem rede'))
    const m = await pedirDetalhe('temas/a/indice.md', 'A', { trecho: 'Etiópia', antes: '', depois: '' })
    expect(m.estado).toBe('pendente')
    expect((await lerDetalhes('temas/a/indice.md')).map((x) => x.trecho)).toEqual(['Etiópia'])

    const envio = vi.spyOn(globalThis, 'fetch').mockImplementation(() => resposta(201, {}))
    await enviarDetalhesPendentes()
    expect(envio).toHaveBeenCalledWith(`/api/detalhes/${m.id}`, expect.objectContaining({ method: 'PUT' }))
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => resposta(200, []))
    expect(await lerDetalhes('temas/a/indice.md')).toEqual([])
  })
})
