import { describe, expect, it } from 'vitest'
import { nomeDoEstado } from './api.js'
import { buscasDaPergunta, descreverLivro, descreverPesquisa } from './pesquisa.js'

describe('pesquisa', () => {
  it('descreve a profundidade e as fontes, na ordem de sempre', () => {
    expect(descreverPesquisa({ fontes: ['youtube', 'livros'], profundidade: 'detalhada' })).toBe('Pesquisa detalhada · Livros, YouTube')
    expect(descreverPesquisa({ fontes: ['sites'], profundidade: 'rapida' })).toBe('Pesquisa rápida · Sites')
    expect(descreverPesquisa({ fontes: [], profundidade: 'rapida' })).toBe('Só a resposta da IA, sem fontes')
    expect(descreverPesquisa(null)).toBe('')
    expect(descreverPesquisa({ fontes: ['wikipedia'], profundidade: 'rapida', rodada: 2 })).toBe('Pesquisa rápida · Wikipédia · 2ª rodada')
  })

  it('descreve o livro e o que foi escolhido dele', () => {
    const livro = { titulo: 'Frutas', autor: 'Autora', capitulos: [{ n: 2, titulo: 'O caju' }] }
    expect(descreverLivro(livro)).toBe('Frutas, de Autora · capítulo “O caju”')
    expect(descreverLivro({ ...livro, capitulos: [{}, {}, {}] })).toBe('Frutas, de Autora · 3 capítulos')
    expect(descreverLivro({ titulo: 'Manual', autor: '', capitulos: [], de: 4, ate: 9 })).toBe('Manual · p. 4 a 9')
  })

  it('o esboço só pede palavras de busca das fontes que buscam', () => {
    expect(buscasDaPergunta({ fontes: ['sites', 'wikipedia', 'livros'] }).map((f) => f.chave)).toEqual(['livros', 'wikipedia'])
  })

  it('numa pesquisa, o material é coletado, não extraído', () => {
    expect(nomeDoEstado({ tipo: 'pesquisa', estado: 'cru' })).toBe('Coletado')
    expect(nomeDoEstado({ tipo: 'video', estado: 'cru' })).toBe('Extraído')
    expect(nomeDoEstado({ tipo: 'pesquisa', estado: 'esboco' })).toBe('Esboço para revisar')
    expect(nomeDoEstado({ tipo: 'livro', estado: 'cru' })).toBe('Lido')
  })
})
