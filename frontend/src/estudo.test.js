import { describe, expect, it } from 'vitest'
import { comparavel, confere, escolher, partesDoEnunciado } from './estudo.js'

describe('escolher', () => {
  const lista = ['a', 'b', 'c', 'd', 'e', 'f'].map((id) => ({ id }))
  const historico = {
    a: { vezes: 1, ultima: 'certo', quando: '2026-11-02' },
    b: { vezes: 2, ultima: 'errado', quando: '2026-11-03' },
    c: { vezes: 1, ultima: 'certo', quando: '2026-11-01' },
    d: { vezes: 1, ultima: 'parcial', quando: '2026-11-04' },
  }
  it('prefere os errados e os nunca feitos', () => {
    const ids = escolher(lista, 4, historico).map((x) => x.id).sort()
    expect(ids).toEqual(['b', 'd', 'e', 'f'])
  })
  it('depois, os acertados há mais tempo', () => {
    const ids = escolher(lista, 5, historico).map((x) => x.id)
    expect(ids).toContain('c')
    expect(ids).not.toContain('a')
  })
  it('não passa do que tem', () => {
    expect(escolher(lista, 20, historico)).toHaveLength(6)
  })
  it('intercala erros e novos quando faltam vagas', () => {
    const ids = escolher(lista, 2, historico).map((x) => x.id)
    expect(ids.filter((i) => ['b', 'd'].includes(i))).toHaveLength(1)
    expect(ids.filter((i) => ['e', 'f'].includes(i))).toHaveLength(1)
  })
})

describe('confere', () => {
  it('ignora caixa, espaços e pontuação, mas não o acento', () => {
    expect(confere(' À. ', { resposta: 'à' })).toBe(true)
    expect(confere('a', { resposta: 'à' })).toBe(false)
    expect(confere('em 1930', { resposta: '1930', aceitas: ['em 1930'] })).toBe(true)
    expect(confere('', { resposta: '1930' })).toBe(false)
  })
})

it('comparavel é o mesmo do servidor', () => {
  expect(comparavel('  Pergunta   1? ')).toBe('pergunta 1')
})

it('separa a lacuna do enunciado', () => {
  expect(partesDoEnunciado('Fui ___ feira.')).toEqual([{ texto: 'Fui ' }, { lacuna: true }, { texto: ' feira.' }])
})
