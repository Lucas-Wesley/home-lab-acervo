import { describe, expect, it } from 'vitest'
import { juntar } from './voz.js'

describe('juntar', () => {
  it('põe o que foi dito depois do que já estava escrito', () => {
    expect(juntar('Foi em 1930.', ' e depois mudou ')).toBe('Foi em 1930. e depois mudou')
    expect(juntar('', 'só a fala')).toBe('só a fala')
    expect(juntar('só o texto  ', '')).toBe('só o texto')
  })
})
