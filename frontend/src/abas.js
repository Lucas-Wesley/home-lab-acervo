import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

// Abas da leitura (D53): Texto, Conversar, Estudar… numa Referência ou num Tema.
// - A barra de abas fica presa no topo (CSS, `position: sticky`); `marco` é um elemento logo antes
//   dela, para saber onde a barra começa.
// - Cada aba guarda a sua posição de rolagem: ao voltar, o texto está onde parou. Uma aba ainda não
//   vista abre com a barra no topo (o começo dela logo abaixo da barra), sem voltar ao topo da página.
// - Deslizar o dedo para o lado troca de aba. Só um gesto claramente horizontal, rápido e de um dedo
//   conta; perto das bordas (o voltar do Android), com texto selecionado ou em algo que rola de lado,
//   não.
export const DESLIZE = { minimo: 70, proporcao: 1.8, tempo: 650, borda: 28 }

// Decide se um toque (início e fim) foi um deslize de aba: -1 (anterior), 1 (próxima) ou 0.
export function direcaoDoDeslize(ini, fim, largura) {
  if (!ini || !fim) return 0
  if (ini.x < DESLIZE.borda || ini.x > largura - DESLIZE.borda) return 0
  const dx = fim.x - ini.x
  const dy = fim.y - ini.y
  if (fim.t - ini.t > DESLIZE.tempo) return 0
  if (Math.abs(dx) < DESLIZE.minimo || Math.abs(dx) < DESLIZE.proporcao * Math.abs(dy)) return 0
  return dx < 0 ? 1 : -1
}

const NAO_DESLIZA = 'input, textarea, select, [contenteditable="true"], .barra-selecao, .folha, .sem-deslizar, [role="dialog"]'

// Algo entre o alvo e a raiz que rola de lado (tabela larga, código, a própria barra de abas).
function rolaDeLado(el, raiz) {
  for (let n = el; n && n !== raiz && n.nodeType === 1; n = n.parentElement) {
    if (n.scrollWidth > n.clientWidth + 2) {
      const ox = getComputedStyle(n).overflowX
      if (ox === 'auto' || ox === 'scroll') return true
    }
  }
  return false
}

function haSelecao() {
  const s = typeof getSelection === 'function' ? getSelection() : null
  return !!s && !s.isCollapsed && !!s.toString().trim()
}

// atual: () => chave da aba na tela; lista: () => chaves na ordem da barra; ir: (chave) => troca;
// marco: ref do elemento logo antes da barra; barra: ref da barra (para mostrar a aba ativa nela);
// raiz: ref do elemento que recebe o gesto de deslizar.
export function usarAbas({ atual, lista, ir, marco, barra, raiz }) {
  const posicoes = new Map()
  const vistas = ref(new Set())
  const marcar = (a) => {
    if (a && !vistas.value.has(a)) vistas.value = new Set([...vistas.value, a])
  }

  // Onde a página fica com a barra presa no topo e o começo da aba logo abaixo dela.
  function inicioDasAbas() {
    const el = marco.value
    if (!el) return 0
    const topo = parseFloat(getComputedStyle(barra.value ?? el).top) || 0
    return Math.max(0, el.getBoundingClientRect().top + window.scrollY - topo)
  }

  // flush 'pre': roda antes de o DOM mudar, com a rolagem ainda da aba que sai.
  watch(atual, (nova, velha) => {
    marcar(nova)
    if (!velha || nova === velha) return
    const y = window.scrollY
    posicoes.set(velha, y)
    const inicio = inicioDasAbas()
    const destino = posicoes.has(nova) ? posicoes.get(nova) : Math.min(y, inicio)
    nextTick(() => {
      window.scrollTo(0, destino)
      // O conteúdo da aba pode crescer logo depois (texto com imagens, conversa); tenta de novo.
      requestAnimationFrame(() => {
        if (Math.abs(window.scrollY - destino) > 2) window.scrollTo(0, destino)
      })
      const ativa = barra.value?.querySelector('.ativa')
      if (ativa && barra.value.scrollWidth > barra.value.clientWidth) {
        barra.value.scrollLeft = ativa.offsetLeft - (barra.value.clientWidth - ativa.offsetWidth) / 2
      }
    })
  })

  let toque = null
  function aoTocar(e) {
    if (e.touches.length !== 1) {
      toque = null
      return
    }
    const t = e.touches[0]
    const alvo = e.target
    if (alvo.closest?.(NAO_DESLIZA) || rolaDeLado(alvo, e.currentTarget)) {
      toque = null
      return
    }
    toque = { x: t.clientX, y: t.clientY, t: Date.now() }
  }

  function aoSoltar(e) {
    const ini = toque
    toque = null
    if (!ini || e.touches.length) return
    const t = e.changedTouches[0]
    const dir = direcaoDoDeslize(ini, { x: t.clientX, y: t.clientY, t: Date.now() }, window.innerWidth)
    if (!dir || haSelecao()) return
    const chaves = lista()
    const i = chaves.indexOf(atual())
    if (i < 0) return
    const proxima = chaves[i + dir]
    if (proxima) ir(proxima)
  }

  function aoCancelar() {
    toque = null
  }

  // Ouvintes passivos: o gesto nunca atrasa a rolagem.
  const ouvintes = { touchstart: aoTocar, touchend: aoSoltar, touchcancel: aoCancelar }
  let el = null
  onMounted(() => {
    marcar(atual())
    el = raiz.value
    for (const [nome, fn] of Object.entries(ouvintes)) el?.addEventListener(nome, fn, { passive: true })
  })
  onBeforeUnmount(() => {
    for (const [nome, fn] of Object.entries(ouvintes)) el?.removeEventListener(nome, fn)
    posicoes.clear()
  })

  return { vistas, posicoes }
}
