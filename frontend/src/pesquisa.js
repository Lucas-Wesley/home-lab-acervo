// Pesquisador (D27): os nomes das fontes e das profundidades, na ordem em que aparecem.
export const FONTES = [
  { chave: 'livros', nome: 'Meus livros', curto: 'Livros' },
  { chave: 'wikipedia', nome: 'Wikipédia', curto: 'Wikipédia' },
  { chave: 'youtube', nome: 'YouTube', curto: 'YouTube' },
  { chave: 'sites', nome: 'Sites', curto: 'Sites' },
]

export const PROFUNDIDADES = [
  { chave: 'rapida', nome: 'Rápida', explicacao: 'De 1 a 3 perguntas e poucas fontes por pergunta.' },
  { chave: 'detalhada', nome: 'Detalhada', explicacao: 'De 4 a 8 perguntas, mais fontes, e cada afirmação conferida no fim.' },
]

// Uma linha sobre a pesquisa: "Pesquisa detalhada · Livros, YouTube" ou "Só a resposta da IA".
export function descreverPesquisa(pesquisa) {
  if (!pesquisa) return ''
  const rodada = pesquisa.rodada > 1 ? ` · ${pesquisa.rodada}ª rodada` : ''
  if (!pesquisa.fontes?.length) return `Só a resposta da IA, sem fontes${rodada}`
  const profundidade = pesquisa.profundidade === 'detalhada' ? 'detalhada' : 'rápida'
  const fontes = FONTES.filter((f) => pesquisa.fontes.includes(f.chave)).map((f) => f.curto)
  return `Pesquisa ${profundidade} · ${fontes.join(', ')}${rodada}`
}

// Uma linha sobre o resumo de um livro (D30): "Livro, de Autor · 2 capítulos".
export function descreverLivro(livro) {
  if (!livro) return ''
  const autor = livro.autor ? `, de ${livro.autor}` : ''
  const n = livro.capitulos?.length ?? 0
  const escolha = n === 0 ? `p. ${livro.de} a ${livro.ate}` : n === 1 ? `capítulo “${livro.capitulos[0].titulo}”` : `${n} capítulos`
  return `${livro.titulo}${autor} · ${escolha}`
}

// As buscas de uma pergunta do esboço que valem para as fontes escolhidas.
export function buscasDaPergunta(pesquisa) {
  return FONTES.filter((f) => f.chave !== 'sites' && pesquisa?.fontes?.includes(f.chave))
}
