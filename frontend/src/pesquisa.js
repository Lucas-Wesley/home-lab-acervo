// Pesquisador (D27): os nomes das fontes e das profundidades, na ordem em que aparecem.
export const FONTES = [
  { chave: 'livros', nome: 'Meus livros', curto: 'Livros', explicacao: 'Trechos dos livros da Biblioteca, com o link da página.' },
  { chave: 'wikipedia', nome: 'Wikipédia', curto: 'Wikipédia', explicacao: 'Os artigos ficam guardados junto com a pesquisa.' },
  { chave: 'youtube', nome: 'YouTube', curto: 'YouTube', explicacao: 'A transcrição de vídeos com legenda; servem só de fonte e não viram Referência.' },
  { chave: 'sites', nome: 'Sites', curto: 'Sites', explicacao: 'A IA busca e lê páginas da internet na hora de escrever.' },
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

// O pedido que chega no endereço da página Pesquisar (D49): #/pesquisar?tema=…&para=temas/…
export function pedidoDoEndereco(hash) {
  const consulta = new URLSearchParams((hash ?? '').split('?').slice(1).join('?'))
  return { tema: (consulta.get('tema') ?? '').trim().slice(0, 500), para: (consulta.get('para') ?? '').trim() }
}

// O endereço da página Pesquisar com o tema (e o Tema a completar) já escolhidos.
export function enderecoDePesquisa(tema, para) {
  const consulta = new URLSearchParams()
  if (tema) consulta.set('tema', tema.trim().slice(0, 500))
  if (para) consulta.set('para', para)
  const q = consulta.toString()
  return `#/pesquisar${q ? `?${q}` : ''}`
}
