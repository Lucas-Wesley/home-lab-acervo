// Chamadas à API. Erros viram exceções com a mensagem que o servidor mandou.
export class ErroApi extends Error {
  constructor(mensagem, status, dados) {
    super(mensagem)
    this.status = status
    this.dados = dados
  }
}

export async function api(caminho, { metodo = 'GET', corpo, tempo = 15000 } = {}) {
  let resposta
  try {
    resposta = await fetch(caminho, {
      method: metodo,
      headers: corpo === undefined ? {} : { 'Content-Type': 'application/json' },
      body: corpo === undefined ? undefined : JSON.stringify(corpo),
      signal: AbortSignal.timeout(tempo),
    })
  } catch {
    throw new ErroApi('Sem conexão com o servidor.', 0)
  }
  if (resposta.status === 204) return null
  const dados = await resposta.json().catch(() => null)
  if (!resposta.ok) throw new ErroApi(dados?.erro || `Erro ${resposta.status}`, resposta.status, dados)
  return dados
}

export const caminhoApi = (caminho) => caminho.split('/').map(encodeURIComponent).join('/')

export const NOMES_TAREFA = {
  extrair: 'Extrair legenda',
  organizar: 'Organizar',
  traduzir: 'Traduzir',
  sugerir: 'Sugerir Tema',
  criar_tema: 'Criar Tema',
  integrar: 'Integrar ao Tema',
  detalhar: 'Mais informações',
  planejar: 'Montar o esboço',
  coletar: 'Coletar nas fontes',
  redigir: 'Escrever a pesquisa',
  conferir: 'Conferir a pesquisa',
}

export const NOMES_ESTADO = {
  novo: 'Novo',
  esboco: 'Esboço para revisar',
  cru: 'Extraído',
  organizado: 'Organizado',
  validado: 'Validado',
  integrado: 'Integrado',
  descartado: 'Descartado',
  erro: 'Erro',
}

export const NOMES_ESTADO_TAREFA = {
  na_fila: 'Na fila',
  rodando: 'Rodando',
  em_revisao: 'Aguardando revisão',
  concluida: 'Concluída',
  falhou: 'Falhou',
  cancelada: 'Cancelada',
  rejeitada: 'Rejeitada',
}

// O estado com o nome certo para o tipo: numa pesquisa, o material não é "extraído", é coletado.
export function nomeDoEstado(r) {
  if (r.tipo === 'pesquisa' && r.estado === 'cru') return 'Coletado'
  return NOMES_ESTADO[r.estado] || r.estado
}

// O que a lista mostra sobre a tarefa de agora.
export function situacao(ref) {
  const t = ref.tarefa
  if (!t) return ''
  const nome = NOMES_TAREFA[t.tipo] || t.tipo
  if (t.estado === 'rodando') return `${nome}…`
  if (t.estado === 'na_fila') return `${nome}: na fila${t.posicao > 1 ? ` (${t.posicao}ª)` : ''}`
  if (t.estado === 'em_revisao') return `${nome}: aguardando sua revisão`
  return ''
}
