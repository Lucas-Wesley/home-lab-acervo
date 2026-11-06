import { expect, it } from 'vitest'
import { slug } from './slug.js'

it('gera o mesmo slug do servidor', () => {
  expect(slug('Segunda Guerra Mundial')).toBe('segunda-guerra-mundial')
  expect(slug('  Alemanha × Rússia (1941) ')).toBe('alemanha-russia-1941')
  expect(slug('Café: o pó preto!')).toBe('cafe-o-po-preto')
})
