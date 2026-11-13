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
  await page.getByRole('button', { name: 'Mais ações' }).click()
  await page.getByRole('menuitem', { name: 'Transcrição' }).click()
  await expect(page.locator('article.texto')).toContainText('Tratado de Tordesilhas')
  await page.getByRole('button', { name: 'Voltar ao texto' }).click()
  // Conversar (D48) e Estudar ficam nas abas, ao lado de Dúvidas. A IA devolve a mudança pedida, e
  // Aplicar a leva ao texto.
  await expect(page.locator('.abas')).toContainText('Conversar')
  await page.locator('.abas').getByRole('button', { name: 'Conversar' }).click()
  await expect(page).toHaveURL(/\?conversar$/)
  await page.getByLabel('Mensagem').fill('Melhore o título')
  await page.getByRole('button', { name: 'Enviar' }).click()
  await expect(page.getByText(/Resposta de teste/)).toBeVisible({ timeout: 15000 })
  await expect(page.locator('.alteracao')).toContainText('Cidade da fundação')
  await page.getByRole('button', { name: 'Aplicar' }).click()
  await expect(page.locator('.alteracao')).toContainText('Aplicada no texto')
  if (process.env.CAPTURAS) {
    await page.getByRole('button', { name: 'Mais ações' }).click()
    await page.screenshot({ path: `${process.env.CAPTURAS}/referencia.png` })
    await page.keyboard.press('Escape')
  }
  await page.locator('.abas').getByRole('button', { name: 'Organizado' }).click()
  await expect(page.locator('article.texto')).toContainText('(revisto)')

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

  // Conversar sobre um trecho (D48): selecionar e tocar em Conversar abre a aba com o trecho; a
  // resposta lembra a conversa, e a mudança proposta pode ser descartada.
  await selecionar(page, 'Tema de Teste')
  await page.getByRole('toolbar', { name: 'Seleção' }).getByRole('button', { name: 'Conversar sobre o trecho' }).click()
  await expect(page).toHaveURL(/indice\.md\?conversar$/)
  await expect(page.locator('.trecho-escolhido')).toContainText('Tema de Teste')
  await page.getByLabel('Mensagem').fill('Onde foi fundado?')
  await page.getByRole('button', { name: 'Enviar' }).click()
  await expect(page.locator('.trecho-escolhido')).toHaveCount(0)
  const mensagem = page.locator('.mensagem').last()
  await expect(mensagem.locator('.citado')).toContainText('Tema de Teste')
  await expect(mensagem.getByText(/Resposta de teste/)).toBeVisible({ timeout: 15000 })
  await mensagem.getByText('Fontes (1)').click()
  await expect(mensagem.getByRole('link', { name: 'Fonte A' })).toBeVisible()
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/conversar.png` })
  await mensagem.getByRole('button', { name: 'Descartar' }).click()
  await expect(mensagem.locator('.alteracao')).toContainText('Descartada')
  await page.locator('.abas').getByRole('button', { name: 'Ler' }).click()
  await expect(page).toHaveURL(/#\/ler\/temas\/tema-de-teste\/indice\.md$/)
})

// Estudar (D41): a aba abre só com os cartões; resumo (ouvido pela voz do aparelho, falsa
// aqui), conceitos anteriores e o simulado em tela própria, respondido por voz (o ditado do
// navegador também é falso: devolve uma frase fixa).
async function vozesFalsas(page) {
  await page.addInitScript(() => {
    class Falso {
      start() {
        setTimeout(() => this.onresult?.({ resultIndex: 0, results: [Object.assign([{ transcript: 'foi fundado em 1930' }], { isFinal: true })] }), 50)
      }
      stop() {
        setTimeout(() => this.onend?.(), 10)
      }
    }
    window.SpeechRecognition = window.webkitSpeechRecognition = Falso
    // A voz que lê: guarda cada trecho falado, um de cada vez.
    window.falas = []
    const voz = { lang: 'pt-BR', localService: true, name: 'Falsa' }
    const sintese = {
      speaking: false,
      pending: false,
      paused: false,
      getVoices: () => [voz],
      speak(u) {
        window.falas.push({ texto: u.text, voz: u.voice?.name })
        setTimeout(() => {
          u.onstart?.()
          setTimeout(() => u.onend?.(), 150)
        }, 10)
      },
      cancel() {},
      resume() {},
    }
    Object.defineProperty(window, 'speechSynthesis', { value: sintese, configurable: true })
    window.SpeechSynthesisUtterance = class {
      constructor(texto) {
        this.text = texto
      }
    }
  })
}

const cartaoDoEstudo = (page, nome) => page.getByRole('group', { name: 'Partes do estudo' }).getByRole('button', { name: new RegExp(`^${nome}`) })

test('estuda o Tema: resumo, conceitos anteriores e simulado por voz', async ({ page }) => {
  await vozesFalsas(page)
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.locator('.abas').getByRole('button', { name: 'Estudar' }).click()
  await expect(page).toHaveURL(/indice\.md\?estudar$/)

  // Só os cartões, nada aberto.
  await expect(page.getByRole('group', { name: 'Partes do estudo' }).getByRole('button')).toHaveCount(5)
  await expect(cartaoDoEstudo(page, 'Resumo inteligente')).toContainText('Ainda não preparado')
  await expect(page.getByRole('button', { name: 'Preparar o resumo' })).toHaveCount(0)

  await cartaoDoEstudo(page, 'Resumo inteligente').click()
  await page.getByRole('button', { name: 'Preparar o resumo' }).click()
  await expect(page.locator('.resumo')).toContainText('Resumo de teste.', { timeout: 15000 })
  await expect(cartaoDoEstudo(page, 'Resumo inteligente')).toContainText('min de leitura')

  // Ouvir: um trecho de cada vez, com a voz em português, até o fim.
  await page.getByRole('button', { name: 'Ouvir o resumo' }).click()
  await expect(page.getByRole('button', { name: 'Pausar' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Ouvir o resumo' })).toBeVisible({ timeout: 10000 })
  const falas = await page.evaluate(() => window.falas)
  expect(falas.length).toBeGreaterThan(1)
  expect(falas[0]).toEqual({ texto: 'Em uma frase.', voz: 'Falsa' })
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/estudar-resumo.png`, fullPage: true })

  // Outro cartão troca o que fica aberto embaixo.
  await cartaoDoEstudo(page, 'Conceitos anteriores').click()
  await expect(page.locator('.resumo')).toHaveCount(0)
  await page.getByRole('button', { name: 'Achar os conceitos anteriores' }).click()
  await expect(page.getByRole('heading', { name: 'Conceito base' })).toBeVisible({ timeout: 15000 })
  await page.getByText('Entender o conceito').first().click()
  await expect(page.getByText(/Explicação base/)).toBeVisible()
  // D49: cada conceito gera os próprios flashcards e, fora do Acervo, leva a Pesquisar.
  await page.locator('.conceito').first().getByRole('button', { name: 'Gerar flashcards', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Gerar flashcards de “Conceito base”' })).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancelar' }).click()
  await expect(page.getByRole('link', { name: 'Pesquisar este conceito' }).last()).toHaveAttribute('href', '#/pesquisar?tema=Vizinho')

  // Flashcards do Tema inteiro: padrão ou de leitura; os dos conceitos anteriores saíram daqui.
  await cartaoDoEstudo(page, 'Flashcards').click()
  await expect(page.getByRole('button', { name: 'Dos conceitos anteriores' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Padrão', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Leitura', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Gerar flashcards de leitura' })).toBeVisible()
  await page.getByRole('dialog').getByRole('button', { name: 'Cancelar' }).click()

  // O simulado abre em tela própria, sem os cartões.
  await cartaoDoEstudo(page, 'Simulado').click()
  await expect(page.getByRole('group', { name: 'Partes do estudo' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Preparar as perguntas' }).click()
  await page.getByRole('button', { name: 'Começar o simulado' }).click({ timeout: 15000 })
  await page.getByRole('button', { name: 'Falar a resposta' }).click()
  await expect(page.locator('.resposta textarea')).toHaveValue('foi fundado em 1930')
  await page.getByRole('button', { name: 'Parar' }).click()
  for (let i = 0; i < 4; i++) await page.getByRole('button', { name: 'Próxima' }).click()
  await page.getByRole('button', { name: /Entregar \(1 de 5\)/ }).click()
  await expect(page.locator('.nota strong')).toHaveText('20', { timeout: 15000 })
  await expect(page.getByText('Teve resposta falada.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Flashcards do que errei' })).toBeVisible()
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/estudar.png`, fullPage: true })

  // Voltar leva aos cartões, que já mostram a nota.
  await page.getByRole('button', { name: 'Estudar', exact: true }).last().click()
  await expect(cartaoDoEstudo(page, 'Simulado')).toContainText('última nota 20')
  if (process.env.CAPTURAS) {
    await page.screenshot({ path: `${process.env.CAPTURAS}/estudar-cartoes.png` })
    await page.setViewportSize({ width: 390, height: 844 })
    await cartaoDoEstudo(page, 'Resumo inteligente').click()
    await page.screenshot({ path: `${process.env.CAPTURAS}/estudar-celular.png` })
    await page.setViewportSize({ width: 1280, height: 720 })
  }

  // Recarregar volta para a aba, e o simulado guarda o histórico.
  await page.reload()
  await cartaoDoEstudo(page, 'Simulado').click()
  await expect(page.getByText(/5 perguntas/)).toBeVisible()

  // D52: gerar mais acrescenta ao banco, com um foco; as perguntas que saíram contam.
  await expect(page.getByText(/6 perguntas no banco: 1 nunca saíram/)).toBeVisible()
  await page.getByRole('button', { name: 'Gerar mais perguntas' }).click()
  await page.getByLabel('Foco do lote (opcional)').fill('só a fundação')
  await page.getByRole('button', { name: 'Gerar', exact: true }).click()
  await expect(page.getByText(/12 perguntas no banco: 7 nunca saíram/)).toBeVisible({ timeout: 15000 })
})

// Exercícios (D52): prática com resposta certa, conferida na hora, com o porquê; o resultado
// mostra o que rever, e o cartão, quantos acertou. "Gerar mais" acrescenta sem repetir.
test('pratica os exercícios do Tema', async ({ page }) => {
  await page.goto('/#/ler/temas/tema-de-teste/indice.md?estudar')
  await cartaoDoEstudo(page, 'Exercícios').click()
  await expect(page.getByRole('group', { name: 'Partes do estudo' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Preparar os exercícios' }).click()
  await expect(page.getByText(/2 exercícios no banco/)).toBeVisible({ timeout: 15000 })
  await page.getByRole('button', { name: 'Praticar', exact: true }).click()

  // A ordem é sorteada: responde certo o primeiro e errado o segundo, seja qual for o tipo.
  async function responder(certo) {
    const campo = page.getByPlaceholder('O que vai na lacuna')
    if (await campo.count()) {
      await campo.fill(certo ? 'em 1930' : '1920')
      await page.getByRole('button', { name: 'Conferir' }).click()
    } else {
      await page.locator('.opcoes').getByRole('button', { name: certo ? '1930' : '1940' }).click()
    }
  }
  await responder(true)
  await expect(page.locator('.retorno')).toContainText('Certo')
  await page.getByRole('button', { name: 'Próximo' }).click()
  await responder(false)
  await expect(page.locator('.retorno')).toContainText('A resposta é 1930.')
  await page.getByRole('button', { name: 'Ver o resultado' }).click()
  await expect(page.locator('.nota strong')).toHaveText('1')
  await expect(page.getByRole('button', { name: 'Refazer os que errei' })).toBeVisible()
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/exercicios.png`, fullPage: true })
  await page.locator('.exercicios').getByRole('button', { name: 'Voltar', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Os que errei (1)' })).toBeVisible()

  await page.getByRole('button', { name: 'Gerar mais exercícios' }).click()
  await page.getByRole('button', { name: 'Gerar', exact: true }).click()
  await expect(page.getByText(/3 exercícios no banco/)).toBeVisible({ timeout: 15000 })

  await page.getByRole('button', { name: 'Estudar', exact: true }).last().click()
  await expect(cartaoDoEstudo(page, 'Exercícios')).toContainText('3 exercícios · 1 de 2 certos')
  if (process.env.CAPTURAS) {
    await page.setViewportSize({ width: 390, height: 844 })
    await page.screenshot({ path: `${process.env.CAPTURAS}/estudar-cinco-cartoes.png` })
  }
})

// D49: do Tema, a pesquisa já vem com o assunto e o Tema a completar; as buscas sem resultado
// aparecem na Referência, com Buscar de novo.
test('pesquisa para completar um Tema', async ({ page }) => {
  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await page.getByRole('link', { name: 'Pesquisar para este Tema' }).click()
  await expect(page.getByRole('heading', { name: 'Pesquisar para este Tema' })).toBeVisible()
  await expect(page.locator('.nova-pesquisa textarea')).toHaveValue('Tema de Teste')
  // Sites vem marcado por padrão; a Wikipédia de verdade fica de fora do teste.
  await expect(page.getByRole('checkbox', { name: /Sites/ })).toBeChecked()
  await page.getByRole('checkbox', { name: /Wikipédia/ }).uncheck()
  await page.getByRole('checkbox', { name: /Gerar flashcards/ }).check()
  await expect(page.getByRole('group', { name: 'Tipo de cartão' }).getByRole('button', { name: 'Leitura' })).toBeVisible()
  await page.getByRole('checkbox', { name: /Gerar flashcards/ }).uncheck()
  await page.getByRole('button', { name: 'Pesquisar', exact: true }).click()

  await expect(page).toHaveURL(/#\/referencia\//)
  await expect(page.locator('.meta')).toContainText('para o Tema Tema de Teste')
  await expect(page.getByText('2 buscas ficaram sem resultado')).toBeVisible({ timeout: 30000 })
  await page.getByText('Ver quais').click()
  await expect(page.locator('.faltou li').first()).toContainText('nenhum trecho nos livros')
  await page.getByRole('button', { name: 'Buscar de novo o que faltou' }).click()
  // A busca de novo também não acha nada: o texto fica, e o aviso continua.
  await expect(page.getByText('2 buscas ficaram sem resultado')).toBeVisible({ timeout: 30000 })
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/pesquisa-faltou.png`, fullPage: true })
  // Sai do banco do teste, para não confundir os testes da pesquisa.
  await page.getByRole('button', { name: 'Mais ações' }).click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('menuitem', { name: 'Excluir' }).click()
  await expect(page).toHaveURL(/#\/referencias$/)
})

// A Referência tem a mesma aba Estudar, a partir do texto organizado dela.
test('estuda uma Referência', async ({ page }) => {
  await vozesFalsas(page)
  await page.goto('/#/referencias')
  await page.locator('.referencia', { hasText: 'Vídeo AAAAAAAAAAA' }).click()
  await page.locator('.abas').getByRole('button', { name: 'Estudar' }).click()
  await expect(page).toHaveURL(/#\/referencia\/[^?]+\?estudar$/)
  await expect(page.getByRole('group', { name: 'Partes do estudo' }).getByRole('button')).toHaveCount(5)
  await expect(cartaoDoEstudo(page, 'Flashcards')).toBeVisible()

  await cartaoDoEstudo(page, 'Resumo inteligente').click()
  // A explicação fica no "i", não na tela.
  await expect(page.getByText(/O texto inteiro em poucos minutos/)).toHaveCount(0)
  await page.getByRole('button', { name: 'Mais informações' }).click()
  await expect(page.getByText(/O texto inteiro em poucos minutos/)).toBeVisible()
  await page.getByRole('button', { name: 'Preparar o resumo' }).click()
  await expect(page.locator('.resumo')).toContainText('Arquivos: ./indice.md', { timeout: 15000 })

  // Recarregar volta para a aba Estudar da Referência.
  await page.reload()
  await expect(cartaoDoEstudo(page, 'Resumo inteligente')).toContainText('min de leitura')
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
  // O foco (D51): escrito ou falado; o microfone fica no campo.
  await expect(janela.getByRole('button', { name: 'Falar em vez de escrever' })).toBeVisible()
  await janela.getByLabel('Foco e jeito dos cartões').fill('  só as datas, o nome na frente  ')
  await janela.getByRole('button', { name: 'Gerar flashcards' }).click()
  await expect(page.getByText('Enviado ao Flashcards.')).toBeVisible()
  await expect(page.getByRole('link', { name: 'Abrir no Flashcards' })).toHaveAttribute('href', 'https://flashcards.lan/#/geracao/7')

  const enviado = await (await request.get('http://localhost:8097/ultimo')).json()
  expect(enviado.baralho).toBe('Revisão do Tema')
  expect(enviado.baralho_id).toBeUndefined()
  expect(enviado.instrucao).toBe('só as datas, o nome na frente')
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
  // A primeira tela só lista as seções (em níveis, design system).
  await expect(page.getByLabel('Sugestão de melhoria')).toHaveCount(0)
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/ajustes.png` })
  await page.getByRole('link', { name: /Sugestões de melhoria/ }).click()

  // Sugestão: vai com a tela de onde se veio.
  await page.getByLabel('Sugestão de melhoria').fill('Mostrar o tempo de cada tarefa na fila')
  await page.getByRole('button', { name: 'Enviar sugestão' }).click()
  await expect(page.locator('.sugestao', { hasText: 'Mostrar o tempo de cada tarefa' })).toContainText('Nova')
  const [sugestao] = await (await page.request.get('/api/sugestoes')).json()
  expect(sugestao.tela).toBe('#/fila')

  // Instrução: abre o texto em vigor, edita e volta ao padrão.
  await page.getByRole('link', { name: 'Ajustes' }).first().click()
  await page.getByRole('link', { name: /Instruções da IA/ }).click()
  if (process.env.CAPTURAS) await page.screenshot({ path: `${process.env.CAPTURAS}/instrucoes.png` })
  await page.getByRole('link', { name: /Organizar a transcrição/ }).click()
  await expect(page.locator('article.texto')).toContainText('Remover o que não é conteúdo')
  await expect(page.getByText('Texto original')).toBeVisible()
  await page.getByRole('button', { name: 'Editar' }).click()
  const editor = page.getByLabel('Texto da instrução')
  await editor.fill((await editor.inputValue()) + '\n- Nunca usar a palavra "basicamente".\n')
  await page.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByText(/Editada em/)).toBeVisible()
  await expect(page.locator('article.texto')).toContainText('Nunca usar a palavra')

  await page.getByRole('link', { name: 'Instruções da IA' }).first().click()
  await expect(page.getByRole('link', { name: /Organizar a transcrição/ })).toContainText('Editada')
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
  await page.goto('/#/ajustes/instrucoes')
  await expect(page.getByRole('link', { name: /^Mais informações/ })).toBeVisible()

  await page.goto('/#/ler/temas/tema-de-teste/indice.md')
  await marca.click()
  page.once('dialog', (d) => d.accept())
  await page.getByRole('button', { name: 'Apagar marcação' }).click()
  await expect(page.locator('mark.detalhe')).toHaveCount(0)
})
