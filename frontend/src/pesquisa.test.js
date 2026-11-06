import { describe, expect, it } from 'vitest'
import { nomeDoEstado } from './api.js'
import { buscasDaPergunta, descreverPesquisa } from './pesquisa.js'

describe('pesquisa', () => {
  it('descreve a profundidade e as fontes, na ordem de sempre', () => {
    expect(descreverPesquisa({ fontes: ['youtube', 'livros'], profundidade: 'detalhada' })).toBe('Pesquisa detalhada · Livros, YouTube')
    expect(descreverPesquisa({ fontes: ['sites'], profundidade: 'rapida' })).toBe('Pesquisa rápida · Sites')
    expect(descreverPesquisa({ fontes: [], profundidade: 'rapida' })).toBe('Só a resposta da IA, sem fontes')
    expect(descreverPesquisa(null)).toBe('')
  })

  it('o esboço só pede palavras de busca das fontes que buscam', () => {
    expect(buscasDaPergunta({ fontes: ['sites', 'wikipedia', 'livros'] }).map((f) => f.chave)).toEqual(['livros', 'wikipedia'])
  })

  it('numa pesquisa, o material é coletado, não extraído', () => {
    expect(nomeDoEstado({ tipo: 'pesquisa', estado: 'cru' })).toBe('Coletado')
    expect(nomeDoEstado({ tipo: 'video', estado: 'cru' })).toBe('Extraído')
    expect(nomeDoEstado({ tipo: 'pesquisa', estado: 'esboco' })).toBe('Esboço para revisar')
  })
})
