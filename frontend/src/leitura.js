// Ouvir o resumo (D41): a voz do próprio aparelho (speechSynthesis) lê o texto em voz alta.
// O texto inteiro numa fala só não funciona no Chrome: as vozes do Google param depois de uns 15 s
// (ou nem começam com um texto longo), e um speak() logo depois de cancel() é descartado. Por isso a
// leitura vai em trechos curtos, um depois do outro, com uma voz em português escolhida.

export const temLeitura = typeof window !== 'undefined' && 'speechSynthesis' in window && typeof window.SpeechSynthesisUtterance !== 'undefined'

// No Chrome, a lista de vozes só carrega depois do primeiro pedido.
if (temLeitura) window.speechSynthesis.getVoices()

// paraFala tira do Markdown o que não se lê: marcas, links (fica o texto) e imagens. Um título vira
// uma frase, para a voz fazer a pausa.
export function paraFala(md) {
  return md
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/<[^>]+>/g, '')
    .replace(/^\s*#{1,6}\s+(.*)$/gm, (_, t) => (/[.!?:…]$/.test(t.trim()) ? t.trim() : `${t.trim()}.`))
    .replace(/^\s*(?:[-*+]|\d+[.)])\s+/gm, '')
    .replace(/^\s*>\s?/gm, '')
    .replace(/\*\*|__|[*`~]/g, '')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{2,}/g, '\n')
    .trim()
}

// trechos divide o texto em pedaços de uma ou poucas frases (até max caracteres), sem juntar
// parágrafos diferentes. Uma frase longa demais se divide nas vírgulas e, em último caso, nos espaços.
export function trechos(texto, max = 220) {
  const saida = []
  for (const paragrafo of texto.split(/\n+/)) {
    const frases = paragrafo.match(/[^.!?…]+(?:[.!?…]+["”»)]*|$)/g) ?? []
    let atual = ''
    for (const frase of frases.flatMap((f) => quebrar(f.trim(), max))) {
      if (!frase) continue
      if (atual && atual.length + 1 + frase.length > max) {
        saida.push(atual)
        atual = frase
      } else {
        atual = atual ? `${atual} ${frase}` : frase
      }
    }
    if (atual) saida.push(atual)
  }
  return saida
}

function quebrar(frase, max) {
  if (frase.length <= max) return [frase]
  const partes = []
  let resto = frase
  while (resto.length > max) {
    const janela = resto.slice(0, max)
    let corte = Math.max(janela.lastIndexOf(', '), janela.lastIndexOf('; '), janela.lastIndexOf(': '), janela.lastIndexOf(' — '))
    if (corte < max / 3) corte = janela.lastIndexOf(' ')
    if (corte <= 0) corte = max
    partes.push(resto.slice(0, corte + 1).trim())
    resto = resto.slice(corte + 1).trim()
  }
  if (resto) partes.push(resto)
  return partes
}

// vozEmPortugues: a do Brasil instalada no aparelho, depois qualquer do Brasil, depois qualquer
// português. Sem nenhuma, a fala vai com lang pt-BR e o aparelho escolhe.
export function vozEmPortugues(vozes) {
  const idioma = (v) => (v.lang || '').replace('_', '-').toLowerCase()
  const br = vozes.filter((v) => idioma(v) === 'pt-br')
  return br.find((v) => v.localService) ?? br[0] ?? vozes.find((v) => idioma(v).startsWith('pt')) ?? null
}

function vozes() {
  const s = window.speechSynthesis
  const lista = s.getVoices()
  if (lista.length) return lista
  return new Promise((ok) => {
    const fim = () => {
      clearTimeout(limite)
      s.removeEventListener?.('voiceschanged', fim)
      ok(s.getVoices())
    }
    const limite = setTimeout(fim, 1500)
    s.addEventListener?.('voiceschanged', fim)
  })
}

const MENSAGENS = {
  'not-allowed': 'O navegador não deixou a página falar. Toque em Ouvir de novo.',
  'synthesis-unavailable': 'Nenhuma voz disponível neste aparelho.',
  'language-unavailable': 'Não há voz em português instalada neste aparelho.',
  'voice-unavailable': 'A voz escolhida não está disponível.',
  network: 'A voz do aparelho precisa de internet.',
}

const SEM_VOZ = 'O aparelho não começou a ler. Confira se há uma voz em português instalada (no Android: Configurações › Conversão de texto em voz).'

// ler fala os trechos a partir de "inicio". aoMudar(i) vem a cada trecho que começa; aoTerminar(erro)
// quando acaba ou dá erro (erro vazio = leu até o fim). Devolve uma função que para e diz em que
// trecho parou, para continuar dali.
export function ler(lista, { inicio = 0, aoMudar, aoTerminar }) {
  const s = window.speechSynthesis
  let i = inicio
  let parado = false
  let comecou = false
  let voz = null
  let fala = null // guardada: no Chrome, uma fala sem referência é recolhida e o onend não vem
  let vigia

  function terminar(erro) {
    if (parado) return
    parado = true
    clearTimeout(vigia)
    aoTerminar?.(erro)
  }

  function falar() {
    if (parado) return
    if (i >= lista.length) return terminar('')
    aoMudar?.(i)
    fala = new SpeechSynthesisUtterance(lista[i])
    fala.lang = voz?.lang || 'pt-BR'
    if (voz) fala.voice = voz
    fala.onstart = () => {
      comecou = true
      clearTimeout(vigia)
    }
    fala.onend = () => {
      if (parado) return
      i++
      falar()
    }
    fala.onerror = (ev) => {
      if (parado || ev.error === 'interrupted' || ev.error === 'canceled') return
      terminar(MENSAGENS[ev.error] ?? `A voz do aparelho parou (${ev.error}).`)
    }
    s.speak(fala)
    if (s.paused) s.resume()
    if (!comecou) {
      clearTimeout(vigia)
      vigia = setTimeout(() => !comecou && terminar(SEM_VOZ), 5000)
    }
  }

  function comecar(lista) {
    voz = vozEmPortugues(lista)
    // Só cancela o que já estava falando, e espera um pouco antes do speak().
    if (s.speaking || s.pending) {
      s.cancel()
      setTimeout(falar, 150)
    } else {
      falar()
    }
  }

  const v = vozes()
  if (Array.isArray(v)) comecar(v)
  else v.then(comecar)

  return () => {
    if (!parado) {
      parado = true
      clearTimeout(vigia)
      s.cancel()
    }
    return i
  }
}
