// Sorteio do estudo (D52): o simulado e os exercícios preferem o que nunca saiu e o que se errou,
// intercalados; depois, o que se acertou há mais tempo. O histórico vem do servidor, por item.

export function embaralhar(lista, aleatorio = Math.random) {
  const copia = [...lista]
  for (let i = copia.length - 1; i > 0; i--) {
    const j = Math.floor(aleatorio() * (i + 1))
    ;[copia[i], copia[j]] = [copia[j], copia[i]]
  }
  return copia
}

const ERROU = new Set(['errada', 'parcial', 'em_branco', 'errado'])

// escolher: n itens da lista. chave(item) dá a chave do histórico; historico[chave] = { vezes, ultima, quando }.
export function escolher(lista, n, historico = {}, chave = (x) => x.id, aleatorio = Math.random) {
  const erros = []
  const novos = []
  const resto = []
  for (const item of lista) {
    const h = historico[chave(item)]
    if (!h?.vezes) novos.push(item)
    else if (ERROU.has(h.ultima)) erros.push(item)
    else resto.push(item)
  }
  const fila = []
  const e = embaralhar(erros, aleatorio)
  const v = embaralhar(novos, aleatorio)
  while (fila.length < n && (e.length || v.length)) {
    if (e.length) fila.push(e.shift())
    if (v.length && fila.length < n) fila.push(v.shift())
  }
  resto.sort((a, b) => (historico[chave(a)].quando < historico[chave(b)].quando ? -1 : 1))
  while (fila.length < n && resto.length) fila.push(resto.shift())
  return embaralhar(fila, aleatorio)
}

// O texto comparável de uma pergunta: o mesmo que o servidor usa para o histórico.
export function comparavel(s) {
  return s.replace(/^[\s.?!…:;]+|[\s.?!…:;]+$/g, '').split(/\s+/).filter(Boolean).join(' ').toLowerCase()
}

// confere a resposta de um exercício de completar: sem caixa, espaços e pontuação nas pontas.
// Os acentos contam (à não é a).
export function confere(resposta, exercicio) {
  const limpa = (s) => comparavel(String(s ?? '')).replace(/^["'“”]+|["'“”]+$/g, '')
  const dada = limpa(resposta)
  if (!dada) return false
  return [exercicio.resposta, ...(exercicio.aceitas ?? [])].some((certa) => limpa(certa) === dada)
}

// partes do enunciado, com a lacuna separada: [{ texto }, { lacuna: true }, { texto }].
export function partesDoEnunciado(enunciado) {
  const saida = []
  enunciado.split(/(_{3,})/).forEach((p) => {
    if (/^_{3,}$/.test(p)) saida.push({ lacuna: true })
    else if (p) saida.push({ texto: p })
  })
  return saida
}
