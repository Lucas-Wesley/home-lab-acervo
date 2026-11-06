import { beforeEach, describe, expect, it, vi } from 'vitest'
import { conflitos, Conflito, lerArquivo, pendentes, salvar, sincronizar } from './offline.js'

function resposta(status, corpo) {
  return Promise.resolve(new Response(JSON.stringify(corpo), { status, headers: { 'Content-Type': 'application/json' } }))
}

describe('edição fora da rede', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
  })

  it('sem rede, guarda a edição no aparelho e mostra-a na leitura', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('offline'))
    const r = await salvar('temas/a/indice.md', 'meu texto', 'h1')
    expect(r.pendente).toBe(true)
    expect(pendentes()['temas/a/indice.md'].base).toBe('h1')

    fetch.mockImplementation(() => resposta(200, { caminho: 'temas/a/indice.md', conteudo: 'do servidor', hash: 'h1' }))
    const arquivo = await lerArquivo('temas/a/indice.md')
    expect(arquivo.conteudo).toBe('meu texto')
  })

  it('ao voltar, envia; se o arquivo mudou no servidor, marca conflito', async () => {
    vi.spyOn(globalThis, 'fetch').mockRejectedValue(new TypeError('offline'))
    await salvar('temas/a/um.md', 'edição 1', 'h1')
    await salvar('temas/a/dois.md', 'edição 2', 'h2')

    fetch.mockImplementation((url) =>
      url.includes('um.md')
        ? resposta(200, { hash: 'novo' })
        : resposta(409, { erro: 'mudou', atual: { conteudo: 'outra pessoa', hash: 'h3' } }),
    )
    const r = await sincronizar()
    expect(r.enviadas).toBe(1)
    expect(r.conflitos).toEqual(['temas/a/dois.md'])
    expect(conflitos()).toEqual(['temas/a/dois.md'])
    expect(pendentes()['temas/a/um.md']).toBeUndefined()
  })

  it('na rede, conflito vira erro com as duas versões', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(() => resposta(409, { erro: 'mudou', atual: { conteudo: 'servidor', hash: 'h9' } }))
    const erro = await salvar('temas/a/x.md', 'minha', 'h1').catch((e) => e)
    expect(erro).toBeInstanceOf(Conflito)
    expect(erro.atual.conteudo).toBe('servidor')
    expect(erro.minha).toBe('minha')
  })
})
