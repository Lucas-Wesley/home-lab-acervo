import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { api, ErroApi } from './api.js'

// "Mais informações" (D26): o Lucas seleciona um trecho e a IA pesquisa sobre ele. A marcação não
// entra no texto: guarda o trecho e um pouco do que vem antes e depois, e o app acha o trecho de
// novo cada vez que a página abre. Se o texto mudar e o trecho sumir, a marcação fica "sem lugar".

const CONTEXTO = 300
const CHAVE_PENDENTES = 'acervo-detalhes-pendentes'
const prefixoCopia = (local) => `acervo-detalhes:${local}`

// ---------- O texto da página ----------

// O texto que conta: botões que o app põe no meio do texto (menu do capítulo) ficam de fora.
const FORA = 'button, .sem-detalhe'

// textoDe junta os nós de texto das raízes, na ordem, com a posição de cada um.
export function textoDe(raizes) {
  const nos = []
  let texto = ''
  for (const raiz of raizes) {
    if (!raiz) continue
    const andar = raiz.ownerDocument.createTreeWalker(raiz, NodeFilter.SHOW_TEXT, {
      acceptNode: (no) => (no.parentElement?.closest(FORA) ? NodeFilter.FILTER_REJECT : NodeFilter.FILTER_ACCEPT),
    })
    for (let no = andar.nextNode(); no; no = andar.nextNode()) {
      nos.push({ no, ini: texto.length })
      texto += no.data
    }
  }
  return { nos, texto }
}

// Posição de um ponto da seleção no texto juntado.
function posicao({ nos, texto }, container, offset) {
  const achado = nos.find((n) => n.no === container)
  if (achado) return achado.ini + offset
  const ponto = container.ownerDocument.createRange()
  ponto.setStart(container, offset)
  const depois = nos.find((n) => ponto.comparePoint(n.no, 0) >= 0)
  return depois ? depois.ini : texto.length
}

const dentro = (raizes, no) => raizes.some((r) => r?.contains(no))

// trechoDaSelecao devolve o trecho selecionado e o que vem antes e depois, ou null.
export function trechoDaSelecao(raizes, intervalo) {
  if (!intervalo || intervalo.collapsed) return null
  if (!dentro(raizes, intervalo.startContainer) || !dentro(raizes, intervalo.endContainer)) return null
  const junto = textoDe(raizes)
  let ini = posicao(junto, intervalo.startContainer, intervalo.startOffset)
  let fim = posicao(junto, intervalo.endContainer, intervalo.endOffset)
  // Sem os espaços das pontas, que o toque duplo costuma pegar.
  while (ini < fim && /\s/.test(junto.texto[ini])) ini++
  while (fim > ini && /\s/.test(junto.texto[fim - 1])) fim--
  if (fim <= ini) return null
  return {
    trecho: junto.texto.slice(ini, fim),
    antes: junto.texto.slice(Math.max(0, ini - CONTEXTO), ini),
    depois: junto.texto.slice(fim, fim + CONTEXTO),
  }
}

function emComum(a, b, doFim) {
  let n = 0
  const max = Math.min(a.length, b.length)
  while (n < max && (doFim ? a[a.length - 1 - n] === b[b.length - 1 - n] : a[n] === b[n])) n++
  return n
}

// acharTrecho escolhe, entre as ocorrências do trecho, a que tem mais do mesmo antes e depois.
export function acharTrecho(texto, { trecho, antes = '', depois = '' }) {
  let melhor = -1
  let nota = -1
  for (let i = texto.indexOf(trecho); i >= 0; i = texto.indexOf(trecho, i + 1)) {
    const n = emComum(texto.slice(Math.max(0, i - antes.length), i), antes, true) + emComum(texto.slice(i + trecho.length), depois, false)
    if (n > nota) {
      melhor = i
      nota = n
    }
  }
  return melhor
}

function envolver(junto, ini, fim, marcacao) {
  for (const { no, ini: comeco } of junto.nos) {
    const final = comeco + no.data.length
    if (final <= ini || comeco >= fim) continue
    let alvo = no
    if (ini > comeco) alvo = alvo.splitText(ini - comeco)
    if (fim < final) alvo.splitText(fim - Math.max(ini, comeco))
    if (!alvo.data.trim()) continue
    const marca = no.ownerDocument.createElement('mark')
    marca.className = `detalhe ${marcacao.estado}`
    marca.dataset.detalhe = marcacao.id
    alvo.replaceWith(marca)
    marca.append(alvo)
  }
}

// marcar sublinha as marcações em pedaços de HTML que aparecem juntos na página, na ordem.
export function marcar(htmls, lista) {
  if (!lista.length) return { htmls, semLugar: [] }
  const partes = htmls.map((html) => {
    const div = document.createElement('div')
    div.innerHTML = html
    return div
  })
  const semLugar = []
  for (const m of lista) {
    const junto = textoDe(partes)
    const ini = acharTrecho(junto.texto, m)
    if (ini < 0) semLugar.push(m)
    else envolver(junto, ini, ini + m.trecho.length, m)
  }
  return { htmls: partes.map((div) => div.innerHTML), semLugar }
}

// ---------- Servidor e aparelho ----------

function ler(chave, padrao) {
  try {
    return JSON.parse(localStorage.getItem(chave)) ?? padrao
  } catch {
    return padrao
  }
}

function gravar(chave, valor) {
  try {
    localStorage.setItem(chave, JSON.stringify(valor))
  } catch {
    // Sem armazenamento: só não fica cópia para ler fora da rede.
  }
}

const pendentes = () => ler(CHAVE_PENDENTES, {})

function gravarPendente(id, pedido) {
  const lista = pendentes()
  if (pedido) lista[id] = pedido
  else delete lista[id]
  gravar(CHAVE_PENDENTES, lista)
}

const comoPendente = (id, p) => ({ ...p, id, estado: 'pendente', explicacao: '', fontes: [] })

// lerDetalhes traz as marcações do texto; fora da rede, vale a última cópia vista.
export async function lerDetalhes(local) {
  let lista
  try {
    lista = await api(`/api/detalhes?local=${encodeURIComponent(local)}`, { tempo: 6000 })
    gravar(prefixoCopia(local), lista)
  } catch (erro) {
    if (!(erro instanceof ErroApi) || erro.status !== 0) throw erro
    lista = ler(prefixoCopia(local), [])
  }
  const naoEnviados = Object.entries(pendentes()).filter(([, p]) => p.local === local)
  return [...lista, ...naoEnviados.map(([id, p]) => comoPendente(id, p))]
}

// pedirDetalhe põe a pesquisa na fila; sem rede, o pedido fica no aparelho e sobe depois.
export async function pedirDetalhe(local, titulo, selecao) {
  const id = crypto.randomUUID()
  const pedido = { local, titulo, ...selecao }
  try {
    return await api(`/api/detalhes/${id}`, { metodo: 'PUT', corpo: pedido })
  } catch (erro) {
    if (!(erro instanceof ErroApi) || erro.status !== 0) throw erro
    gravarPendente(id, pedido)
    return comoPendente(id, pedido)
  }
}

export async function enviarDetalhesPendentes() {
  for (const [id, pedido] of Object.entries(pendentes())) {
    try {
      await api(`/api/detalhes/${id}`, { metodo: 'PUT', corpo: pedido })
      gravarPendente(id, null)
    } catch (erro) {
      if (erro instanceof ErroApi && erro.status === 0) return
      // Recusado pelo servidor (trecho inválido): não adianta tentar de novo.
      gravarPendente(id, null)
    }
  }
}

export async function apagarDetalhe(id) {
  if (pendentes()[id]) return gravarPendente(id, null)
  await api(`/api/detalhes/${id}`, { metodo: 'DELETE' })
}

export const refazerDetalhe = (id) => api(`/api/detalhes/${id}/refazer`, { metodo: 'POST' })

// ---------- Na tela ----------

// usarDetalhes guarda as marcações do texto aberto e acompanha as que ainda estão na fila.
export function usarDetalhes(local) {
  const lista = ref([])
  const aberto = ref(null) // id da marcação no modal
  const atual = computed(() => lista.value.find((m) => m.id === aberto.value) ?? null)

  async function carregar() {
    const l = local()
    if (!l) return
    try {
      const nova = await lerDetalhes(l)
      if (l === local()) lista.value = nova
    } catch {
      // Sem as marcações, o texto continua legível.
    }
  }

  async function pedir(titulo, selecao) {
    const m = await pedirDetalhe(local(), titulo, selecao)
    lista.value = [...lista.value.filter((x) => x.id !== m.id), m]
    return m
  }

  function trocar(m) {
    lista.value = lista.value.map((x) => (x.id === m.id ? m : x))
  }

  async function refazer(id) {
    trocar(await refazerDetalhe(id))
  }

  async function apagar(id) {
    await apagarDetalhe(id)
    lista.value = lista.value.filter((x) => x.id !== id)
    if (aberto.value === id) aberto.value = null
  }

  // Um toque no trecho sublinhado abre o modal; devolve true se o clique era dele.
  function aoClicar(evento) {
    const marca = evento.target.closest?.('mark.detalhe')
    if (!marca) return false
    evento.preventDefault()
    aberto.value = marca.dataset.detalhe
    return true
  }

  watch(local, carregar, { immediate: true })
  const relogio = setInterval(() => {
    if (document.visibilityState === 'visible' && lista.value.some((m) => m.estado === 'processando')) carregar()
  }, 4000)
  onBeforeUnmount(() => clearInterval(relogio))

  return { lista, aberto, atual, carregar, pedir, refazer, apagar, aoClicar }
}
