// Ditado pela voz (D41): o reconhecimento de fala do próprio navegador (Web Speech API), em
// português. No Chrome o áudio passa pelo serviço do Google; no Safari, pelo da Apple. Sem suporte,
// o botão não aparece e o teclado do celular continua com o ditado dele.

const Reconhecimento = typeof window !== 'undefined' ? window.SpeechRecognition || window.webkitSpeechRecognition : undefined

export const temVoz = !!Reconhecimento

// ouvir começa a escutar e chama aoMudar(texto) com o que foi dito até agora (o final e o provisório).
// Devolve uma função que para. aoTerminar(erro) vem quando o navegador para sozinho ou por erro.
export function ouvir({ aoMudar, aoTerminar }) {
  const r = new Reconhecimento()
  r.lang = 'pt-BR'
  r.continuous = true
  r.interimResults = true
  let final = ''
  let erro = ''
  r.onresult = (ev) => {
    let provisorio = ''
    for (let i = ev.resultIndex; i < ev.results.length; i++) {
      const trecho = ev.results[i][0].transcript
      if (ev.results[i].isFinal) final += trecho
      else provisorio += trecho
    }
    aoMudar(juntar(final, provisorio))
  }
  r.onerror = (ev) => {
    erro = MENSAGENS[ev.error] ?? (ev.error === 'no-speech' || ev.error === 'aborted' ? '' : `O ditado parou (${ev.error}).`)
  }
  r.onend = () => aoTerminar(erro)
  r.start()
  return () => r.stop()
}

const MENSAGENS = {
  'not-allowed': 'O navegador não deixou usar o microfone. Libere nas permissões do site.',
  'service-not-allowed': 'O navegador não deixou usar o ditado neste site.',
  'audio-capture': 'Nenhum microfone encontrado.',
  network: 'O ditado do navegador precisa de internet.',
}

// juntar põe o que foi dito depois do que já estava escrito, com um espaço entre os dois.
export function juntar(antes, novo) {
  const a = antes.trimEnd()
  const n = novo.trim()
  if (!a) return n
  if (!n) return a
  return `${a} ${n}`
}
