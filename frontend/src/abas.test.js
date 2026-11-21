import { describe, expect, it } from 'vitest'
import { direcaoDoDeslize } from './abas.js'

const ini = (x, y = 300, t = 0) => ({ x, y, t })

describe('direcaoDoDeslize', () => {
  it('para a esquerda vai para a próxima aba; para a direita, para a anterior', () => {
    expect(direcaoDoDeslize(ini(300), ini(150, 310, 250), 400)).toBe(1)
    expect(direcaoDoDeslize(ini(100), ini(260, 290, 250), 400)).toBe(-1)
  })

  it('ignora o movimento curto, o mais vertical que horizontal e o lento', () => {
    expect(direcaoDoDeslize(ini(300), ini(250, 300, 100), 400)).toBe(0)
    expect(direcaoDoDeslize(ini(300), ini(200, 380, 200), 400)).toBe(0)
    expect(direcaoDoDeslize(ini(300), ini(100, 300, 1200), 400)).toBe(0)
  })

  it('ignora o gesto que começa na borda (o voltar do Android)', () => {
    expect(direcaoDoDeslize(ini(10), ini(200, 300, 200), 400)).toBe(0)
    expect(direcaoDoDeslize(ini(390), ini(150, 300, 200), 400)).toBe(0)
  })
})
