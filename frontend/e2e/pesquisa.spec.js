import { expect, test } from '@playwright/test'

// Pesquisador (D27), com o claude e o yt-dlp falsos: pedido → esboço revisado → coleta →
// texto conferido → sugestão de Tema.
test('pesquisa um tema no YouTube, com o esboço revisado antes de buscar', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByRole('button', { name: 'Pesquisar um tema' }).click()
  await page.getByLabel('Tema ou pergunta').fill('Qual é a origem da laranja?')

  // Os livros vêm marcados, mas este servidor não tem a Biblioteca: o erro aparece no formulário.
  await expect(page.getByRole('button', { name: 'Meus livros' })).toHaveAttribute('aria-pressed', 'true')
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()
  await expect(page.locator('.nova-pesquisa .aviso.erro')).toContainText('BIBLIOTECA_URL')

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
