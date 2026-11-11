import { expect, test } from '@playwright/test'

// Uma fonte na página Pesquisar um tema: a linha com o nome e a explicação.
const fonte = (page, nome) => page.getByRole('checkbox', { name: new RegExp(`^${nome}`) })

// Pesquisador (D27), com o claude e o yt-dlp falsos: pedido → esboço revisado → coleta →
// texto conferido → sugestão de Tema.
test('pesquisa um tema no YouTube, com o esboço revisado antes de buscar', async ({ page }) => {
  // Pesquisar um tema é uma página própria, aberta pelo menu ⋯ (D34).
  await page.goto('/#/referencias')
  await page.getByRole('button', { name: 'Mais opções' }).click()
  await page.getByRole('menuitem', { name: 'Pesquisar um tema' }).click()
  await expect(page).toHaveURL(/#\/pesquisar$/)
  await page.getByLabel('Tema ou pergunta').fill('Qual é a origem da laranja?')

  // Os livros vêm marcados; aqui, a pesquisa é só no YouTube.
  await expect(fonte(page, 'Meus livros')).toBeChecked()
  await fonte(page, 'Meus livros').uncheck()
  await fonte(page, 'Wikipédia').uncheck()
  // Sites também vem marcado (D49).
  await expect(fonte(page, 'Sites')).toBeChecked()
  await fonte(page, 'Sites').uncheck()
  await expect(page.getByText('Nenhuma fonte: só a resposta da IA')).toBeVisible()
  await fonte(page, 'YouTube').check()
  await page.getByRole('radio', { name: /^Detalhada/ }).check()
  await expect(page.locator('.barra-pesquisa')).toContainText('Detalhada · YouTube')
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
  await expect(page.locator('.duvida').first()).toContainText('Uma dúvida de teste')
  // Resolver (D37): numa pesquisa não há áudio; a falta no material vira "Pesquisar isto", que abre
  // o Aprofundar com a dúvida como foco.
  await page.getByRole('button', { name: 'Resolver as dúvidas' }).click()
  await expect(page.locator('.duvida .chip').first()).toHaveText('Falta no material', { timeout: 15000 })
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/duvidas.png` })
  await page.getByRole('button', { name: 'Pesquisar isto' }).first().click()
  await expect(page.getByPlaceholder('Ex.: as variedades e como escolher no mercado')).toHaveValue(/Uma dúvida de teste/)
  await page.getByRole('button', { name: 'Cancelar' }).first().click()
  // Corrigir no texto (D44): o botão some assim que a proposta existe, e o texto final aparece na
  // própria aba para aprovar, sem ir a Revisões.
  await page.getByRole('button', { name: 'Dúvidas' }).click()
  const duvida = page.locator('.duvida').first()
  await duvida.getByRole('button', { name: 'Corrigir no texto' }).click()
  await expect(duvida.getByRole('button', { name: 'Corrigir no texto' })).toHaveCount(0)
  await expect(duvida.getByRole('button', { name: 'Resolver de novo' })).toHaveCount(0)
  await expect(duvida.getByText('Como o texto fica')).toBeVisible({ timeout: 20000 })
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/duvidas-aprovar.png`, fullPage: true })
  await duvida.getByRole('button', { name: 'Aprovar' }).click()
  await expect(page.getByText('Corrigido no texto.')).toBeVisible()
  await page.getByRole('button', { name: 'Mais ações' }).click()
  await page.getByRole('menuitem', { name: 'Material coletado' }).click()
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
  await page.goto('/#/pesquisar')
  await page.getByLabel('Tema ou pergunta').fill('De onde vem o caju?')
  await fonte(page, 'Meus livros').setChecked(false)
  await fonte(page, 'Wikipédia').setChecked(false)
  await fonte(page, 'YouTube').setChecked(true)
  await fonte(page, 'Sites').setChecked(false)
  await page.getByRole('radio', { name: /^Rápida/ }).check()
  await page.getByLabel('Revisar o esboço antes de buscar').setChecked(false)
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })

  await page.getByRole('button', { name: 'Mais ações' }).click()
  await page.getByRole('menuitem', { name: 'Aprofundar' }).click()
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
  await expect(page.getByText('O texto antes da última mudança')).toBeVisible()
})

// Resumo de capítulos (D30, D32): escolher o livro na lista e os capítulos; a IA resume e sugere o Tema.
test('resume dois capítulos de um livro da Biblioteca', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByRole('button', { name: 'Mais opções' }).click()
  await page.getByRole('menuitem', { name: 'Resumir capítulos' }).click()
  await expect(page).toHaveURL(/#\/resumir$/)
  await expect(page.locator('.livro', { hasText: 'Pães de Fermentação Natural' })).toContainText('EPUB')
  await page.getByRole('link', { name: /Pães de Fermentação Natural/ }).first().click()
  await expect(page.getByText('O levain')).toBeVisible()
  await expect(page.getByText('2 marcações')).toBeVisible()
  await page.getByLabel(/O levain/).check()
  await page.getByLabel(/^A farinha/).check()
  await expect(page.getByText('2 capítulos · ≈ 20 páginas')).toBeVisible()
  await page.getByLabel('O que mais te interessa?').fill('a hidratação da massa')
  await page.getByRole('button', { name: 'Resumir' }).click()

  await expect(page.getByRole('heading', { name: 'Pães de Fermentação Natural: 2 capítulos' })).toBeVisible()
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 20000 })
  await expect(page.getByText('Pães de Fermentação Natural, de Autora · 2 capítulos · foco: “a hidratação da massa”')).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Foco recebido.')
  await expect(page.locator('article.texto')).toContainText('Marcações recebidas.')
  await expect(page.locator('article.texto').getByRole('link', { name: 'p. 1', exact: true })).toHaveAttribute('href', 'https://biblioteca.lan/#/livro/livro1?em=%7B%22pagina%22%3A1%7D')
  await page.getByRole('button', { name: 'Mais ações' }).click()
  await page.getByRole('menuitem', { name: 'Texto do livro' }).click()
  await expect(page.locator('article.texto')).toContainText('Texto do capítulo A farinha.')

  await page.goto('/#/referencias')
  await expect(page.locator('.referencia', { hasText: '2 capítulos' }).locator('.miniatura.pesquisa')).toBeVisible()
})
