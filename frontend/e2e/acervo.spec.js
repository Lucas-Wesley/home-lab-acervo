import { expect, test } from '@playwright/test'

// Um fluxo só, na ordem em que o Lucas usa: vídeo → Referência organizada → Tema revisado →
// leitura com navegação → edição → offline. Os testes dividem o mesmo servidor.
test.describe.configure({ mode: 'serial' })

test('do vídeo ao Tema, com revisão', async ({ page }) => {
  await page.goto('/#/referencias')
  // As entradas ficam no menu ⋯ (D34).
  await page.getByRole('button', { name: 'Mais opções' }).click()
  await page.getByRole('menuitem', { name: 'Link de vídeo' }).click()
  await page.getByLabel('Link do vídeo').fill('https://youtu.be/AAAAAAAAAAA?si=teste')
  await page.getByRole('button', { name: 'Adicionar' }).click()

  const cartao = page.locator('.referencia', { hasText: 'Vídeo AAAAAAAAAAA' })
  await expect(cartao.getByText('Organizado', { exact: true })).toBeVisible({ timeout: 15000 })
  await cartao.click()

  await expect(page.getByRole('heading', { name: 'Vídeo AAAAAAAAAAA' })).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Conteúdo.')
  await page.getByRole('button', { name: 'Transcrição' }).click()
  await expect(page.locator('article.texto')).toContainText('Tratado de Tordesilhas')

  await page.getByRole('button', { name: 'Mais ações' }).click()
  await page.getByRole('menuitem', { name: 'Sugerir Tema' }).click()
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

test('gera flashcards de um capítulo, escolhendo o baralho, e o link do cartão abre no capítulo', async ({ page, request }) => {
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.getByRole('button', { name: 'Opções do capítulo Segundo capítulo' }).click()
  await page.getByRole('dialog', { name: 'Capítulo Segundo capítulo' }).getByRole('button', { name: /Gerar flashcards/ }).click()

  // Começa no baralho que tem o nome do Tema; dá para trocar por um novo.
  const janela = page.getByRole('dialog', { name: /Gerar flashcards do capítulo/ })
  const baralho = janela.getByLabel('Baralho', { exact: true })
  await expect(baralho).toHaveValue('b1')
  await baralho.selectOption({ label: 'Baralho novo…' })
  await expect(janela.getByLabel('Nome do baralho novo')).toHaveValue('Tema de Teste')
  await janela.getByLabel('Nome do baralho novo').fill('Revisão do Tema')
  await janela.getByRole('button', { name: 'Gerar flashcards' }).click()
  await expect(page.getByText('Enviado ao Flashcards.')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Abrir no Flashcards' })).toHaveAttribute('href', 'https://flashcards.lan/#/geracao/7')

  const enviado = await (await request.get('http://localhost:8097/ultimo')).json()
  expect(enviado.baralho).toBe('Revisão do Tema')
  expect(enviado.baralho_id).toBeUndefined()
  expect(enviado.link).toBe('https://acervo.lan/#/ler/temas/tema-de-teste/indice.md?capitulo=Segundo%20cap%C3%ADtulo')

  // O link que o cartão guarda abre a página já no capítulo.
  await page.goto(`/${enviado.link.replace('https://acervo.lan/', '')}`)
  await expect(page.locator('#cap-segundo-capitulo')).toBeInViewport()
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
  await page.getByRole('button', { name: 'Mais opções' }).click()
  await page.getByRole('menuitem', { name: 'Link de vídeo' }).click()
  await page.getByLabel('Link do vídeo').fill('https://www.youtube.com/watch?v=SSSSSSSSSSS')
  await page.getByRole('button', { name: 'Adicionar' }).click()
  await expect(page.getByText('Não foi possível extrair a legenda deste vídeo. O processo não continuou.')).toBeVisible({ timeout: 15000 })
})

test('exclui uma Referência pelo menu', async ({ page }) => {
  await page.goto('/#/referencias')
  const cartao = page.locator('.referencia', { hasText: 'SSSSSSSSSSS' })
  await cartao.click()
  await page.getByRole('button', { name: 'Mais ações' }).click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('menuitem', { name: 'Excluir' }).click()
  await expect(page).toHaveURL(/#\/referencias$/)
  await expect(page.locator('.referencia', { hasText: 'SSSSSSSSSSS' })).toHaveCount(0)
})

test('link compartilhado pelo celular vai para a fila numa tela curta, sem abrir o app', async ({ page }) => {
  await page.goto('/compartilhar?title=Um%20v%C3%ADdeo&text=' + encodeURIComponent('Veja isto https://youtu.be/CCCCCCCCCCC?si=x'))
  await expect(page.getByRole('status')).toHaveText('Vídeo enviado para a fila.')
  await expect(page.locator('.menu')).toHaveCount(0)
  // O navegador do teste não deixa fechar a aba: fica o recado e o link.
  await expect(page.getByText('Pode voltar ao app de onde compartilhou.')).toBeVisible()
  await page.getByRole('link', { name: 'Abrir no Acervo' }).click()
  await expect(page.getByRole('heading', { name: 'Vídeo CCCCCCCCCCC' })).toBeVisible({ timeout: 15000 })

  // O endereço antigo do manifest ("/?text=") também cai na tela curta; o mesmo vídeo já estava lá.
  await page.goto('/?text=' + encodeURIComponent('https://www.youtube.com/watch?v=CCCCCCCCCCC'))
  await expect(page.getByRole('status')).toHaveText('Este vídeo já está nas Referências.')
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

// Seleciona a palavra no texto, como o dedo faria.
async function selecionar(page, palavra) {
  await page.evaluate((palavra) => {
    const andar = document.createTreeWalker(document.querySelector('.pagina-leitura'), NodeFilter.SHOW_TEXT)
    for (let no = andar.nextNode(); no; no = andar.nextNode()) {
      const i = no.data.indexOf(palavra)
      if (i < 0) continue
      no.parentElement.scrollIntoView({ block: 'center' })
      const intervalo = document.createRange()
      intervalo.setStart(no, i)
      intervalo.setEnd(no, i + palavra.length)
      getSelection().removeAllRanges()
      getSelection().addRange(intervalo)
      return
    }
    throw new Error(`não achei "${palavra}"`)
  }, palavra)
}

test('Mais informações: seleciona um trecho, a IA pesquisa e o trecho vira link para a explicação', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await expect(page.locator('article.texto').last()).toContainText('Tordesilhas')

  // Copiar.
  await selecionar(page, 'Tordesilhas')
  await page.getByRole('toolbar', { name: 'Seleção' }).getByRole('button', { name: 'Copiar' }).click()
  await expect(page.getByRole('status').filter({ hasText: 'Copiado.' })).toBeVisible()
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('Tordesilhas')

  // Mais informações: fica pontilhado enquanto pesquisa e sublinhado quando termina.
  await selecionar(page, 'Tordesilhas')
  await page.getByRole('button', { name: 'Mais informações' }).click()
  await expect(page.getByRole('toolbar', { name: 'Seleção' })).toBeHidden()
  const marca = page.locator('mark.detalhe', { hasText: 'Tordesilhas' })
  await expect(marca).toBeVisible()
  await expect(marca).toHaveClass(/pronto/, { timeout: 15000 })

  // Continua marcado ao abrir a página de novo; o toque abre o modal, sem sair do texto.
  await page.reload()
  await marca.click()
  const modal = page.getByRole('dialog', { name: '“Tordesilhas”' })
  await expect(modal).toContainText('Explicação de teste.')
  await expect(modal.getByRole('link', { name: 'Fonte A' })).toHaveAttribute('href', 'https://exemplo.com/a')
  await expect(page).toHaveURL(/#\/ler\/temas\/tema-de-teste\/indice\.md$/)
  await page.keyboard.press('Escape')
  await expect(modal).toBeHidden()

  // Na fila, a pesquisa leva de volta ao texto.
  await page.goto('/#/fila')
  await expect(page.getByRole('link', { name: '“Tordesilhas”' }).first()).toHaveAttribute('href', '#/ler/temas/tema-de-teste/indice.md')

  // A instrução nova aparece em Ajustes.
  await page.goto('/#/ajustes')
  await expect(page.getByRole('link', { name: /Mais informações/ })).toBeVisible()

  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await marca.click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Apagar marcação' }).click()
  await expect(page.locator('mark.detalhe')).toHaveCount(0)
})
