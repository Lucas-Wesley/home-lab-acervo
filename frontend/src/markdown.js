import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { slug } from './slug.js'

// Tira o cabeçalho YAML do indice.md; ele não é texto para ler.
export function semCabecalho(texto) {
  if (!texto.startsWith('---\n')) return texto
  const fim = texto.indexOf('\n---', 4)
  return fim < 0 ? texto : texto.slice(fim + 4).replace(/^\n+/, '')
}

// "05:59" ou "1:02:05" → segundos.
export function segundos(marca) {
  return marca.split(':').reduce((total, parte) => total * 60 + Number(parte), 0)
}

export function urlNoTempo(url, marca) {
  try {
    const u = new URL(url)
    u.searchParams.set('t', `${segundos(marca)}s`)
    return u.toString()
  } catch {
    return url
  }
}

const MARCA = /\[(\d{1,2}(?::\d{2}){1,2})\]/g

function escapar(texto) {
  return texto.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c])
}

function marcasComLink(texto, url) {
  return texto.replace(MARCA, (_, marca) => `<a href="${escapar(urlNoTempo(url, marca))}" target="_blank" rel="noopener">${marca}</a>`)
}

// Nos Temas, "<sub>Fonte: <id> [mm:ss]</sub>" vira o título da fonte com links para cada momento.
export function trocarFontes(texto, fontes = []) {
  return texto.replace(/<sub>Fontes?:\s*([^\s<]+)\s*([^<]*)<\/sub>/g, (todo, id, marcas) => {
    const fonte = fontes.find((f) => f.id === id)
    if (!fonte) return todo
    return `<sub>Fonte: <a href="${escapar(fonte.url)}" target="_blank" rel="noopener">${escapar(fonte.titulo || id)}</a> ${marcasComLink(marcas, fonte.url)}</sub>`
  })
}

// Na Referência, "<sub>Trechos: [mm:ss]</sub>" aponta para o próprio vídeo.
export function trocarTrechos(texto, url) {
  if (!url) return texto
  return texto.replace(/<sub>Trechos:([^<]*)<\/sub>/g, (_, marcas) => `<sub>Trechos: ${marcasComLink(marcas, url)}</sub>`)
}

// Resolve um link relativo a partir do arquivo atual: ("temas/a/b.md", "../c/indice.md") → "temas/c/indice.md".
export function resolver(origem, destino) {
  const partes = origem.split('/').slice(0, -1)
  for (const parte of destino.split('/')) {
    if (parte === '..') partes.pop()
    else if (parte !== '.' && parte !== '') partes.push(parte)
  }
  return partes.join('/')
}

// Caminho relativo de um arquivo para outro, para inserir links entre Temas.
export function relativo(origem, destino) {
  const de = origem.split('/').slice(0, -1)
  const para = destino.split('/')
  let i = 0
  while (i < de.length && i < para.length - 1 && de[i] === para[i]) i++
  return [...Array(de.length - i).fill('..'), ...para.slice(i)].join('/')
}

export function eLinkInterno(href) {
  return !!href && !/^[a-z]+:/i.test(href) && !href.startsWith('#') && /\.md(#.*)?$/.test(href)
}

export function renderizar(texto, { fontes = [], url = '' } = {}) {
  let md = semCabecalho(texto)
  md = trocarFontes(md, fontes)
  md = trocarTrechos(md, url)
  const html = DOMPurify.sanitize(marked.parse(md, { gfm: true }), { ADD_ATTR: ['target'] })
  // Links entre arquivos do Acervo ganham uma classe para o leitor interceptar o clique.
  const div = document.createElement('div')
  div.innerHTML = html
  // Âncoras nos títulos, para o sumário levar ao capítulo na mesma página.
  for (const h of div.querySelectorAll('h2, h3')) h.id = idDoTitulo(h.textContent)
  // Capítulo separado: na página, o corpo é só o link para o arquivo dele.
  for (const p of div.querySelectorAll('h2 + p')) {
    if (p.children.length === 1 && p.textContent.trim() === p.firstElementChild.textContent.trim() && eLinkInterno(p.firstElementChild.getAttribute('href') ?? '')) {
      p.classList.add('capitulo-separado')
    }
  }
  for (const a of div.querySelectorAll('a[href]')) {
    if (eLinkInterno(a.getAttribute('href'))) a.classList.add('interno')
    else if (!a.getAttribute('target')) {
      a.setAttribute('target', '_blank')
      a.setAttribute('rel', 'noopener')
    }
  }
  return div.innerHTML
}

// Separa a seção "## Dúvidas" do texto organizado.
export function duvidas(texto) {
  const achado = texto.match(/\n## (Dúvidas|Doubts|Uncertainties)\n([\s\S]*?)(?=\n## |\s*$)/)
  return achado ? achado[2].trim() : ''
}

// Os itens da seção "## Dúvidas" (um item pode ocupar várias linhas), igual ao servidor (duvidas.go).
export function listaDeDuvidas(texto) {
  const itens = []
  for (const bruta of duvidas(texto).split('\n')) {
    const l = bruta.trim()
    if (l.startsWith('- ') || l.startsWith('* ')) itens.push(l.slice(2).trim())
    else if (l && itens.length) itens[itens.length - 1] += ` ${l}`
  }
  return itens
}

// Título do Markdown: o "# " do texto ou o do cabeçalho.
export function tituloDoTexto(texto) {
  const cabecalho = texto.match(/^---\n(?:[^]*?\n)?titulo:[ \t]*(.+?)[ \t]*\n(?:[^]*?\n)?---/)
  if (cabecalho) return cabecalho[1].trim()
  const h1 = semCabecalho(texto).match(/^# (.+)$/m)
  return h1 ? h1[1].trim() : ''
}

export const idDoTitulo = (titulo) => `cap-${slug(titulo)}`

// Seções de apoio do Tema; aparecem no sumário, mas não contam como capítulos.
export const APENDICES = ['Linha do tempo', 'Citados', 'Divergências', 'Referências']

const LINK_DE_CAPITULO = /^\[[^\]]*\]\(([^)\s#/]+\.md)\)$/

// Capítulos do texto ("## "), com a posição de cada um, igual ao servidor (capitulos.go).
export function capitulos(texto) {
  const lista = []
  let emCodigo = false
  let pos = 0
  for (const linha of texto.split(/(?<=\n)/)) {
    const limpa = linha.replace(/\r?\n$/, '')
    if (limpa.startsWith('```')) emCodigo = !emCodigo
    if (!emCodigo && limpa.startsWith('## ')) {
      if (lista.length) lista.at(-1).fim = pos
      lista.push({ titulo: limpa.slice(3).trim(), ini: pos, fim: texto.length })
    }
    pos += linha.length
  }
  for (const c of lista) {
    const corpo = texto.slice(c.ini, c.fim).split('\n').slice(1).join('\n').trim()
    c.arquivo = corpo.match(LINK_DE_CAPITULO)?.[1] ?? null
    c.apendice = APENDICES.includes(c.titulo)
    c.id = idDoTitulo(c.titulo)
  }
  return lista
}

// Troca um capítulo pelo texto editado, mantendo o resto da página como estava.
export function trocarCapitulo(texto, capitulo, novo) {
  const fim = texto.slice(capitulo.fim)
  return texto.slice(0, capitulo.ini) + novo.replace(/\s*$/, fim ? '\n\n' : '\n') + fim
}

// Lugar de um capítulo novo: antes dos apêndices.
export function inserirCapitulo(texto, titulo) {
  const apendice = capitulos(texto).find((c) => c.apendice)
  const novo = `## ${titulo}\n\n`
  if (!apendice) return texto.replace(/\s*$/, '\n\n') + novo
  return texto.slice(0, apendice.ini) + novo + texto.slice(apendice.ini)
}
