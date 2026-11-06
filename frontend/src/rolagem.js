// Histórico de navegação do leitor. Cada entrada ganha um número em history.state; ao chegar numa
// entrada que já tem número, é o "voltar" (ou "avançar") do navegador, e a leitura volta para a
// posição em que estava.
let atual = 1
let voltando = false

export function iniciarHistorico() {
  if (history.state?.n) atual = history.state.n
  else history.replaceState({ n: 1 }, '')
  window.addEventListener('hashchange', () => {
    if (history.state?.n) {
      voltando = true
      atual = history.state.n
    } else {
      voltando = false
      atual += 1
      history.replaceState({ n: atual }, '')
    }
  })
}

export const voltandoNoHistorico = () => voltando
export const podeVoltar = () => atual > 1

// Troca o endereço sem criar entrada nova (ex.: tirar o "?editar").
export function trocarEndereco(hash) {
  history.replaceState(history.state, '', hash)
}

const CHAVE = 'acervo-rolagem'

export function guardarPosicao(caminho, y) {
  try {
    const mapa = JSON.parse(sessionStorage.getItem(CHAVE) || '{}')
    mapa[caminho] = Math.round(y)
    sessionStorage.setItem(CHAVE, JSON.stringify(mapa))
  } catch {
    // Sem armazenamento: só não restaura a posição.
  }
}

export function posicaoGuardada(caminho) {
  try {
    return JSON.parse(sessionStorage.getItem(CHAVE) || '{}')[caminho] ?? 0
  } catch {
    return 0
  }
}
