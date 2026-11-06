import { expect, test } from '@playwright/test'

// Pesquisador (D27), com o claude e o yt-dlp falsos: pedido → esboço revisado → coleta →
// texto conferido → sugestão de Tema.
test('pesquisa um tema no YouTube, com o esboço revisado antes de buscar', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByRole('button', { name: 'Pesquisar um tema' }).click()
  await page.getByLabel('Tema ou pergunta').fill('Qual é a origem da laranja?')

  // Os livros vêm marcados; aqui, a pesquisa é só no YouTube.
  await expect(page.getByRole('button', { name: 'Meus livros' })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: 'Meus livros' }).click()
  await page.getByRole('button', { name: 'Wikipédia' }).click()
  await expect(page.getByText('Nenhuma fonte: só a resposta da IA')).toBeVisible()
  await page.getByRole('button', { name: 'YouTube' }).click()
  await page.getByRole('button', { name: 'Detalhada' }).click()
  await page.getByLabel('Revisar o esboço antes de buscar').check()
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()

  // O esboço para e espera.
  await expect(page.getByRole('heading', { name: 'A origem da laranja' })).toBeVisible()
  await expect(page.getByText('Pesquisa detalhada · YouTube')).toBeVisible()
  await expect(page.getByText('Esboço da pesquisa')).toBeVisible({ timeout: 15000 })
  await expect(page.getByLabel('Pergunta 1', { exact: true })).toHaveValue('De onde vem a laranja?')
  await expect(page.getByLabel('Busca em Livros da pergunta 1')).toHaveCount(0)
  await page.getByLabel('Busca em YouTube da pergunta 1').fill('laranja história')
  await page.getByRole('button', { name: 'Tirar a pergunta 2' }).click()
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()

  // Coleta, escrita e conferência; no fim, a sugestão de Tema já pedida.
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })
  await expect(page.locator('.cabeca .chip').first()).toHaveText('Organizado')
  await expect(page.locator('article.texto')).toContainText('Conferência: 1 afirmação')
  await page.getByRole('button', { name: 'Dúvidas' }).click()
  await expect(page.locator('article.texto')).toContainText('Uma dúvida de teste')
  await page.getByRole('button', { name: 'Material coletado' }).click()
  await expect(page.locator('article.texto')).toContainText('De onde vem a laranja?')
  await expect(page.locator('article.texto')).not.toContainText('Como chegou ao Brasil?')
  await expect(page.locator('article.texto a', { hasText: 'Vídeo AAAAAAAAAAA' })).toHaveAttribute('href', /youtube\.com\/watch\?v=AAAAAAAAAAA/)

  // Na lista, a pesquisa tem o seu ícone e a descrição.
  await page.goto('/#/referencias')
  const linha = page.locator('.referencia', { hasText: 'A origem da laranja' })
  await expect(linha.locator('.miniatura.pesquisa')).toBeVisible()
  await expect(linha).toContainText('Pesquisa detalhada · YouTube')
})

// Aprofundar (D29): outra rodada, com mais uma fonte e o que o Lucas quer a mais.
test('aprofunda uma pesquisa pronta, e o texto de antes fica na Versão anterior', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByRole('button', { name: 'Pesquisar um tema' }).click()
  await page.getByLabel('Tema ou pergunta').fill('De onde vem o caju?')
  for (const nome of ['Meus livros', 'Wikipédia']) {
    if ((await page.getByRole('button', { name: nome }).getAttribute('aria-pressed')) === 'true') await page.getByRole('button', { name: nome }).click()
  }
  for (const nome of ['YouTube', 'Detalhada']) await page.getByRole('button', { name: nome }).click()
  if ((await page.getByRole('button', { name: 'YouTube' }).getAttribute('aria-pressed')) !== 'true') await page.getByRole('button', { name: 'YouTube' }).click()
  await page.getByRole('button', { name: 'Rápida' }).click()
  if (await page.getByLabel('Revisar o esboço antes de buscar').isChecked()) await page.getByLabel('Revisar o esboço antes de buscar').uncheck()
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })

  await page.getByRole('button', { name: 'Aprofundar' }).click()
  await expect(page.getByRole('button', { name: 'YouTube' })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: 'Sites' }).click()
  await page.getByLabel('O que mais quer saber?').fill('as variedades')
  await page.locator('form').getByRole('button', { name: 'Aprofundar' }).click()

  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })
  await expect(page.getByText('Pesquisa rápida · YouTube, Sites · 2ª rodada')).toBeVisible()
  await expect(page.getByText('a mais: “as variedades”')).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Partiu de anterior.md.')
  await page.getByRole('button', { name: 'Versão anterior' }).click()
  await expect(page.locator('article.texto')).not.toContainText('Partiu de anterior.md.')
  await expect(page.getByText('O texto antes de aprofundar a pesquisa')).toBeVisible()
})

// Resumo de capítulos (D30, D32): escolher o livro na lista e os capítulos; a IA resume e sugere o Tema.
test('resume dois capítulos de um livro da Biblioteca', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByRole('link', { name: 'Resumir capítulos' }).click()
  await expect(page).toHaveURL(/#\/resumir$/)
  await expect(page.locator('.livro', { hasText: 'Pães de Fermentação Natural' })).toContainText('EPUB')
  await page.getByRole('link', { name: /Pães de Fermentação Natural/ }).first().click()
  await expect(page.getByText('O levain')).toBeVisible()
  await expect(page.getByText('p. 1–12 · ≈ 12 páginas')).toBeVisible()
  await page.getByLabel(/O levain/).check()
  await page.getByLabel(/^A farinha/).check()
  await expect(page.getByText('2 capítulos · ≈ 20 páginas')).toBeVisible()
  await page.getByLabel('O que mais te interessa?').fill('a hidratação da massa')
  await page.getByRole('button', { name: 'Resumir' }).click()

  await expect(page.getByRole('heading', { name: 'Pães de Fermentação Natural: 2 capítulos' })).toBeVisible()
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })
  await expect(page.getByText('Pães de Fermentação Natural, de Autora · 2 capítulos · foco: “a hidratação da massa”')).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Foco recebido.')
  await expect(page.locator('article.texto').getByRole('link', { name: 'p. 1', exact: true })).toHaveAttribute('href', 'https://biblioteca.lan/#/livro/livro1?em=%7B%22pagina%22%3A1%7D')
  await page.getByRole('button', { name: 'Texto do livro' }).click()
  await expect(page.locator('article.texto')).toContainText('Texto do capítulo A farinha.')

  await page.goto('/#/referencias')
  await expect(page.locator('.referencia', { hasText: '2 capítulos' }).locator('.miniatura.pesquisa')).toBeVisible()
})
