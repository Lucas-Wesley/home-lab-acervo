import { expect, test } from '@playwright/test'

// Um fluxo só, na ordem em que o Lucas usa: vídeo → Referência organizada → Tema revisado →
// leitura com navegação → edição → offline. Os testes dividem o mesmo servidor.
test.describe.configure({ mode: 'serial' })

test('do vídeo ao Tema, com revisão', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByLabel('Link do vídeo').fill('https://youtu.be/AAAAAAAAAAA?si=teste')
  await page.getByRole('button', { name: 'Adicionar' }).click()

  const cartao = page.locator('.referencia', { hasText: 'Vídeo AAAAAAAAAAA' })
  await expect(cartao.getByText('Organizado', { exact: true })).toBeVisible({ timeout: 15000 })
  await cartao.click()

  await expect(page.getByRole('heading', { name: 'Vídeo AAAAAAAAAAA' })).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Conteúdo.')
  await page.getByRole('button', { name: 'Transcrição' }).click()
  await expect(page.locator('article.texto')).toContainText('Tratado de Tordesilhas')

  await page.getByRole('button', { name: 'Sugerir Tema' }).click()
  await expect(page.getByText('Sugestão da IA:')).toBeVisible({ timeout: 15000 })
  await page.getByRole('button', { name: 'Usar sugestão' }).click()
  await expect(page.locator('.formulario input')).toHaveValue('Tema de Teste')
  await page.getByRole('button', { name: 'Pedir à IA' }).click()

  await page.getByRole('link', { name: 'Revisar proposta' }).click({ timeout: 15000 })
  await expect(page.getByRole('heading', { name: /Novo Tema: Tema de Teste/ })).toBeVisible()
  await expect(page.locator('.arquivo')).toHaveCount(1)
  await page.getByRole('button', { name: 'Aprovar e incluir no Acervo' }).click()

  // A aprovação leva direto ao Tema.
  await expect(page).toHaveURL(/#\/ler\/temas\/tema-de-teste\/indice\.md$/)
  await expect(page.locator('article.texto h1').first()).toHaveText('Tema de Teste')
  await expect(page.getByText('Referências deste Tema')).toBeVisible()
})

test('lê o Tema numa página só, com sumário de capítulos', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('2 capítulos')).toBeVisible()
  await page.getByText('Tema de Teste').first().click()
  await expect(page.locator('article.texto h1')).toHaveText('Tema de Teste')

  // Os capítulos estão na mesma página; o sumário leva até eles.
  const sumario = page.getByRole('navigation', { name: 'Capítulos' })
  await expect(sumario.getByRole('listitem')).toHaveText(['Primeiro capítulo', 'Segundo capítulo'])
  await expect(page.locator('article.texto')).toContainText(['Texto do primeiro capítulo.'])
  await sumario.getByRole('link', { name: 'Segundo capítulo' }).click()
  await expect(page.locator('#cap-segundo-capitulo')).toBeInViewport()
  // O botão do cabeçalho lista os capítulos de qualquer ponto do texto e marca o atual.
  await page.getByRole('button', { name: 'Capítulos', exact: true }).click()
  const painel = page.getByRole('navigation', { name: 'Lista de capítulos' })
  await expect(painel.locator('button.atual')).toHaveText('Segundo capítulo')
  await painel.getByRole('button', { name: 'Primeiro capítulo' }).click()
  await expect(painel).toBeHidden()
  await expect(page.locator('#cap-primeiro-capitulo')).toBeInViewport()
  // A fonte vira link para o momento do vídeo.
  await expect(page.locator('article.texto sub a', { hasText: '00:30' })).toHaveAttribute('href', /t=30s/)
})

test('separa um capítulo em página própria e traz de volta', async ({ page }) => {
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.getByRole('button', { name: 'Opções do capítulo Primeiro capítulo' }).click()
  await page.getByRole('button', { name: 'Separar em página própria' }).click()
  await expect(page.getByText('"Primeiro capítulo" agora tem página própria.')).toBeVisible()

  // Na página do Tema fica o título com o link; o texto saiu dela.
  await expect(page.locator('article.texto')).not.toContainText(['Texto do primeiro capítulo.'])
  await expect(page.getByRole('navigation', { name: 'Capítulos' })).toContainText('página própria')
  await page.getByRole('link', { name: 'Ler o capítulo →' }).click()
  await expect(page.locator('article.texto h1')).toHaveText('Primeiro capítulo')
  await expect(page.locator('article.texto').first()).toContainText('Texto do primeiro capítulo.')
  await expect(page.getByText('Próximo capítulo →')).toBeVisible()
  // Na página separada, o botão lista os capítulos do Tema e leva de volta a um deles.
  await page.getByRole('button', { name: 'Capítulos', exact: true }).click()
  await expect(page.getByRole('navigation', { name: 'Lista de capítulos' }).locator('button.atual')).toContainText('Primeiro capítulo')

  await page.keyboard.press('Escape')
  await page.locator('.veu-lista').click({ position: { x: 5, y: 400 } })
  // Voltar leva ao Tema.
  await page.getByRole('button', { name: 'Voltar' }).click()
  await expect(page.locator('article.texto h1').first()).toHaveText('Tema de Teste')

  await page.getByRole('button', { name: 'Opções do capítulo Primeiro capítulo' }).click()
  await page.getByRole('button', { name: 'Trazer de volta para esta página' }).click()
  await expect(page.getByText('"Primeiro capítulo" voltou para a página.')).toBeVisible()
  await expect(page.locator('article.texto').last()).toContainText('Texto do primeiro capítulo.')
})

test('edita só um capítulo, cria capítulo e subtema, liga Temas por link', async ({ page }) => {
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.getByRole('button', { name: 'Opções do capítulo Segundo capítulo' }).click()
  await page.getByRole('button', { name: 'Editar capítulo' }).click()
  const editor = page.locator('textarea.editor')
  await expect(editor).toHaveValue('## Segundo capítulo\n\nTexto do segundo capítulo.')
  await editor.fill('## Segundo capítulo\n\nTexto revisado por mim.')
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByText('Salvo.')).toBeVisible()
  const corpo = page.locator('article.texto').last()
  await expect(corpo).toContainText('Texto revisado por mim.')
  // O resto da página ficou igual.
  await expect(corpo).toContainText('Texto do primeiro capítulo.')
  await expect(corpo).toContainText('Linha do tempo')

  page.once('dialog', (d) => d.accept('Terceiro capítulo'))
  await page.getByRole('button', { name: 'Novo capítulo' }).click()
  await expect(editor).toHaveValue(/^## Terceiro capítulo/)
  await editor.fill('## Terceiro capítulo\n\nEscrito à mão.')
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByRole('navigation', { name: 'Capítulos' }).getByRole('listitem')).toHaveText(['Primeiro capítulo', 'Segundo capítulo', 'Terceiro capítulo'])

  // Subtema dentro do Tema, com link de volta para o Tema por "Inserir link".
  page.once('dialog', (d) => d.accept('Frente Leste'))
  await page.getByRole('button', { name: 'Novo subtema' }).click()
  await expect(editor).toBeVisible()
  await editor.press('End')
  await page.getByLabel('Inserir link para um Tema').selectOption({ label: 'Tema de Teste' })
  await expect(editor).toHaveValue(/\[Tema de Teste\]\(\.\.\/indice\.md\)/)
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.locator('.migalhas')).toContainText('Tema de Teste')

  // O Tema lista o subtema e mostra "citado em".
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await expect(page.locator('.rodape-tema', { hasText: 'Subtemas' })).toContainText('Frente Leste')
  await expect(page.locator('.rodape-tema', { hasText: 'Citado em' })).toContainText('Frente Leste')
})

test('baixa um Tema, edita fora da rede e sincroniza ao voltar', async ({ page, context }) => {
  await page.goto('/')
  await page.waitForFunction(() => navigator.serviceWorker?.controller)
  await page.getByRole('button', { name: 'Baixar para ler fora da rede' }).click()
  await expect(page.getByRole('button', { name: 'Remover do aparelho' })).toBeVisible()

  await context.setOffline(true)
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.reload()
  await expect(page.locator('article.texto').last()).toContainText('Texto revisado por mim.')
  await page.getByRole('button', { name: 'Opções do capítulo Terceiro capítulo' }).click()
  await page.getByRole('button', { name: 'Editar capítulo' }).click()
  await page.locator('textarea.editor').fill('## Terceiro capítulo\n\nEditado no avião.')
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByText(/ficou salva no aparelho/)).toBeVisible()

  await context.setOffline(false)
  await page.evaluate(() => window.dispatchEvent(new Event('online')))
  await expect(page.getByText(/salvas? no aparelho; sobe/)).toBeHidden({ timeout: 10000 })
  const servidor = await page.evaluate(() => fetch('/api/arquivos/temas/tema-de-teste/indice.md', { cache: 'no-store' }).then((r) => r.json()))
  expect(servidor.conteudo).toContain('Editado no avião.')
  expect(servidor.conteudo).toContain('Texto revisado por mim.')
})

test('vídeo sem legenda mostra o erro na lista', async ({ page }) => {
  await page.goto('/#/referencias')
  await page.getByLabel('Link do vídeo').fill('https://www.youtube.com/watch?v=SSSSSSSSSSS')
  await page.getByRole('button', { name: 'Adicionar' }).click()
  await expect(page.getByText('Não foi possível extrair a legenda deste vídeo. O processo não continuou.')).toBeVisible({ timeout: 15000 })
})

test('vídeo compartilhado pelo celular entra na fila sem passar por tela nenhuma', async ({ page }) => {
  await page.goto('/?title=Um%20v%C3%ADdeo&text=' + encodeURIComponent('Veja isto https://youtu.be/CCCCCCCCCCC?si=x'))
  await expect(page.getByRole('status')).toHaveText('Vídeo enviado para a fila.')
  await expect(page).toHaveURL(/\/#\/referencias$/)
  await expect(page.locator('.referencia', { hasText: 'Vídeo CCCCCCCCCCC' })).toBeVisible({ timeout: 15000 })
})

test('Ajustes: anota uma sugestão e edita a instrução que o Claude segue', async ({ page }) => {
  await page.goto('/#/fila')
  await page.getByRole('button', { name: 'Ajustes' }).click()
  await expect(page.getByRole('heading', { name: 'Ajustes' })).toBeVisible()

  // Sugestão: vai com a tela de onde se veio.
  await page.getByLabel('Sugestão de melhoria').fill('Mostrar o tempo de cada tarefa na fila')
  await page.getByRole('button', { name: 'Enviar sugestão' }).click()
  await expect(page.locator('.sugestao', { hasText: 'Mostrar o tempo de cada tarefa' })).toContainText('Nova')
  const [sugestao] = await (await page.request.get('/api/sugestoes')).json()
  expect(sugestao.tela).toBe('#/fila')

  // Instrução: abre o texto em vigor, edita e volta ao padrão.
  await page.getByRole('link', { name: /Organizar a transcrição/ }).click()
  await expect(page.locator('article.texto')).toContainText('Remover o que não é conteúdo')
  await expect(page.getByText('Texto original')).toBeVisible()
  await page.getByRole('button', { name: 'Editar' }).click()
  const editor = page.getByLabel('Texto da instrução')
  await editor.fill((await editor.inputValue()) + '\n- Nunca usar a palavra "basicamente".\n')
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByText(/Editada em/)).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Nunca usar a palavra')

  await page.getByRole('link', { name: 'Ajustes' }).first().click()
  await expect(page.locator('.instrucao', { hasText: 'Organizar a transcrição' })).toContainText('Editada')
  await page.getByRole('link', { name: /Organizar a transcrição/ }).click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Voltar ao padrão' }).click()
  await expect(page.getByText('Texto original')).toBeVisible()
  await expect(page.locator('article.texto')).not.toContainText('Nunca usar a palavra')
})
