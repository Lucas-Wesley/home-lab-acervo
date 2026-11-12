import { expect, test } from '@playwright/test'

// Documentário do Streaming (D50): dois episódios pela legenda viram um item só na lista, e a página
// de cada um leva aos outros.
function legenda(n) {
  let s = ''
  for (let i = 0; i < 30; i++) s += `${i + 1}\n00:00:${String(i * 2).padStart(2, '0')},000 --> 00:00:${String(i * 2 + 1).padStart(2, '0')},500\nFala ${i} do episódio ${n}.\n\n`
  return Buffer.from(s)
}

test('episódios de um documentário ficam juntos e ligados', async ({ page }) => {
  const ids = []
  for (const [n, nome] of [[2, 'O Delta'], [1, 'A Nascente']]) {
    const dados = {
      chave: 'serie/4242', nome: 'Segredos do Nilo', ano: '2020', temporada: 1, episodio: n, episodios: 3, titulo_episodio: nome,
      tmdb: `https://www.themoviedb.org/tv/4242/season/1/episode/${n}`, link: 'https://streaming.lan/#/titulo/serie/4242',
      texto: 'legenda', legenda: 'OpenSubtitles, pt-br: teste', idioma_texto: 'pt',
    }
    const r = await page.request.post('/api/documentarios', { multipart: { dados: JSON.stringify(dados), legenda: { name: 'l.srt', mimeType: 'application/x-subrip', buffer: legenda(n) } } })
    expect(r.status()).toBe(201)
    ids.push((await r.json()).id)
  }

  await page.goto('/#/referencias')
  const grupo = page.locator('.grupo-doc', { hasText: 'Segredos do Nilo' })
  await expect(grupo).toContainText('Documentário · 2020 · 2 episódios')
  await expect(grupo.locator('.episodio-doc')).toHaveCount(2)
  // Em ordem de episódio, não de chegada.
  await expect(grupo.locator('.episodio-doc').first()).toContainText(/T1 E1\s*A Nascente/)
  await expect(grupo.locator('.episodio-doc', { hasText: 'O Delta' }).getByText('Organizado', { exact: true })).toBeVisible({ timeout: 15000 })

  await grupo.locator('.episodio-doc', { hasText: 'O Delta' }).click()
  await expect(page.getByRole('heading', { name: 'Segredos do Nilo, episódio 2: O Delta' })).toBeVisible()
  await expect(page.locator('.meta')).toContainText('pela legenda')
  await expect(page.getByRole('link', { name: 'abrir no Streaming' })).toHaveAttribute('href', 'https://streaming.lan/#/titulo/serie/4242')
  const nav = page.getByRole('navigation', { name: 'Episódios deste documentário' })
  await expect(nav).toContainText('2 episódios no Acervo')
  await nav.getByRole('link', { name: '← Anterior' }).click()
  await expect(page.getByRole('heading', { name: 'Segredos do Nilo, episódio 1: A Nascente' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'Próximo →' })).toBeVisible()

  for (const id of ids) await page.request.delete(`/api/referencias/${id}`)
})
