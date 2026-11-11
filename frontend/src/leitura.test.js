import { describe, expect, it } from 'vitest'
import { paraFala, trechos, vozEmPortugues } from './leitura.js'

describe('paraFala', () => {
  it('tira as marcas do Markdown e faz do título uma frase', () => {
    const md = '## Em uma frase\n\nO café veio da **Etiópia** ([ver](cap.md)).\n\n- Primeiro ponto\n- *Segundo* ponto\n\n1. Item'
    expect(paraFala(md)).toBe('Em uma frase.\nO café veio da Etiópia (ver).\nPrimeiro ponto\nSegundo ponto\nItem')
  })

  it('não põe outro ponto num título que já termina com pontuação', () => {
    expect(paraFala('## O que confunde?')).toBe('O que confunde?')
  })
})

describe('trechos', () => {
  it('junta frases curtas do mesmo parágrafo até o limite', () => {
    expect(trechos('Uma frase. Outra frase. Mais uma.', 30)).toEqual(['Uma frase. Outra frase.', 'Mais uma.'])
  })

  it('não junta parágrafos diferentes', () => {
    expect(trechos('Título.\nTexto do parágrafo.')).toEqual(['Título.', 'Texto do parágrafo.'])
  })

  it('divide a frase longa nas vírgulas', () => {
    const t = trechos('Os árabes guardavam a planta em segredo, os europeus só recebiam o pó, e o roubo das mudas mudou tudo.', 50)
    expect(t.every((x) => x.length <= 50)).toBe(true)
    expect(t.join(' ')).toBe('Os árabes guardavam a planta em segredo, os europeus só recebiam o pó, e o roubo das mudas mudou tudo.')
  })

  it('guarda o texto sem pontuação no fim', () => {
    expect(trechos('Sem ponto final')).toEqual(['Sem ponto final'])
  })
})

describe('vozEmPortugues', () => {
  const v = (lang, localService = false, name = lang) => ({ lang, localService, name })

  it('prefere a voz do Brasil instalada no aparelho', () => {
    expect(vozEmPortugues([v('en-US', true), v('pt-BR', false, 'Google'), v('pt_BR', true, 'Luciana')]).name).toBe('Luciana')
  })

  it('cai para qualquer português, ou nenhuma', () => {
    expect(vozEmPortugues([v('pt-PT', true)]).lang).toBe('pt-PT')
    expect(vozEmPortugues([v('en-US', true)])).toBe(null)
  })
})
