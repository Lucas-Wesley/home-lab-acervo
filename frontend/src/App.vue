<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Acervo from './Acervo.vue'
import Ajustes from './Ajustes.vue'
import Fila from './Fila.vue'
import Icone from './Icone.vue'
import Instrucao from './Instrucao.vue'
import Leitor from './Leitor.vue'
import Referencia from './Referencia.vue'
import Referencias from './Referencias.vue'
import Revisoes from './Revisoes.vue'
import { api } from './api.js'
import { conflitos, pendentes, sincronizar } from './offline.js'
import { enviarDetalhesPendentes } from './detalhes.js'

// Rotas no hash, como na Biblioteca: o "voltar" do celular volta para a tela anterior.
const hash = ref(location.hash)
const recado = ref(null) // { texto, erro } — aviso rápido, como o de um link compartilhado
const versao = ref('')
const contagem = ref({ fila: 0, revisoes: 0 })
const naoEnviadas = ref(0)
const emConflito = ref([])

const rota = computed(() => {
  const h = decodeURIComponent(hash.value.replace(/^#\/?/, ''))
  const [nome, ...resto] = h.split('/')
  return { nome: nome || 'acervo', param: resto.join('/') }
})

const MENU = [
  { rota: 'acervo', nome: 'Acervo', icone: 'acervo' },
  { rota: 'referencias', nome: 'Referências', icone: 'referencias' },
  { rota: 'revisoes', nome: 'Revisões', icone: 'revisao', contador: 'revisoes' },
  { rota: 'fila', nome: 'Fila', icone: 'fila', contador: 'fila' },
  { rota: 'baixados', nome: 'Baixados', icone: 'baixado' },
  { rota: 'ajustes', nome: 'Ajustes', icone: 'ajustes' },
]

const secao = computed(() => {
  const n = rota.value.nome
  if (n === 'ler') return 'acervo'
  if (n === 'referencia') return 'referencias'
  if (n === 'revisao') return 'revisoes'
  return n
})

function ir(destino) {
  location.hash = `#/${destino}`
}

// A tela de onde se foi para Ajustes: vai junto com a sugestão, para dar contexto.
const telaAnterior = ref('')

function aoMudarHash() {
  if (location.hash.startsWith('#/ajustes') && !hash.value.startsWith('#/ajustes')) telaAnterior.value = hash.value || '#/acervo'
  hash.value = location.hash
}

let relogio
async function atualizarContagem() {
  if (document.visibilityState !== 'visible') return
  try {
    const [tarefas, revisoes] = await Promise.all([api('/api/tarefas', { tempo: 5000 }), api('/api/revisoes', { tempo: 5000 })])
    contagem.value = {
      fila: tarefas.filter((t) => t.estado === 'na_fila' || t.estado === 'rodando').length,
      revisoes: revisoes.length,
    }
  } catch {
    // Sem rede: os contadores ficam como estavam.
  }
}

async function sincronizarEdicoes() {
  await sincronizar()
  await enviarDetalhesPendentes()
  atualizarPendentes()
}

function atualizarPendentes() {
  emConflito.value = conflitos()
  naoEnviadas.value = Object.keys(pendentes()).length - emConflito.value.length
}

function aoVoltarARede() {
  sincronizarEdicoes()
  atualizarContagem()
}

function aoMudarVisibilidade() {
  if (document.visibilityState !== 'visible') return
  sincronizarEdicoes()
  atualizarContagem()
}

// Compartilhar um vídeo com o app instalado (Android) abre "/?text=…&url=…": o link vai direto
// para a fila, como se tivesse sido colado em Referências.
let tempoDoRecado
function mostrarRecado(texto, erro = false) {
  recado.value = { texto, erro }
  clearTimeout(tempoDoRecado)
  tempoDoRecado = setTimeout(() => (recado.value = null), 4000)
}

async function receberCompartilhado() {
  const p = new URLSearchParams(location.search)
  const enviado = [p.get('url'), p.get('text'), p.get('title')].filter(Boolean).join(' ')
  if (!enviado) return
  history.replaceState(null, '', `${location.pathname}#/referencias`)
  hash.value = location.hash
  try {
    await api('/api/referencias', { metodo: 'POST', corpo: { url: enviado } })
    mostrarRecado('Vídeo enviado para a fila.')
  } catch (e) {
    mostrarRecado(e.message, true)
  }
}

onMounted(async () => {
  receberCompartilhado()
  window.addEventListener('hashchange', aoMudarHash)
  window.addEventListener('online', aoVoltarARede)
  window.addEventListener('acervo:pendentes', atualizarPendentes)
  document.addEventListener('visibilitychange', aoMudarVisibilidade)
  relogio = setInterval(atualizarContagem, 4000)
  atualizarPendentes()
  sincronizarEdicoes()
  atualizarContagem()
  try {
    versao.value = (await api('/api/health', { tempo: 4000 })).version
  } catch {
    versao.value = ''
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('hashchange', aoMudarHash)
  window.removeEventListener('online', aoVoltarARede)
  window.removeEventListener('acervo:pendentes', atualizarPendentes)
  document.removeEventListener('visibilitychange', aoMudarVisibilidade)
  clearInterval(relogio)
})
</script>


<template>
  <div class="app" :class="{ lendo: rota.nome === 'ler' }">
    <aside class="menu" aria-label="Menu">
      <div class="marca">Acervo</div>
      <nav>
        <button
          v-for="item in MENU"
          :key="item.rota"
          class="item"
          :class="{ ativo: secao === item.rota }"
          :aria-current="secao === item.rota ? 'page' : undefined"
          @click="ir(item.rota)"
        >
          <Icone :nome="item.icone" />
          <span class="nome">{{ item.nome }}</span>
          <span v-if="item.contador && contagem[item.contador]" class="contador">{{ contagem[item.contador] }}</span>
        </button>
      </nav>
      <div class="rodape">{{ versao ? `versão ${versao.slice(0, 7)}` : 'sem conexão' }}</div>
    </aside>

    <main class="conteudo">
      <div v-if="emConflito.length" class="faixa erro">
        Edição feita fora da rede em conflito com o servidor:
        <a v-for="c in emConflito" :key="c" :href="`#/ler/${c}`">{{ c.split('/').slice(-2).join('/') }}</a>
      </div>
      <div v-else-if="naoEnviadas" class="faixa">
        {{ naoEnviadas === 1 ? '1 edição salva no aparelho' : `${naoEnviadas} edições salvas no aparelho` }}; sobe quando voltar à rede.
      </div>

      <Leitor v-if="rota.nome === 'ler'" :key="rota.param" :caminho="rota.param" />
      <Referencias v-else-if="rota.nome === 'referencias'" />
      <Referencia v-else-if="rota.nome === 'referencia'" :key="rota.param" :id="rota.param" />
      <Fila v-else-if="rota.nome === 'fila'" />
      <Instrucao v-else-if="rota.nome === 'ajustes' && rota.param.startsWith('instrucoes/')" :key="rota.param" :arquivo="rota.param.slice('instrucoes/'.length)" />
      <Ajustes v-else-if="rota.nome === 'ajustes'" :tela="telaAnterior" />
      <Revisoes v-else-if="rota.nome === 'revisoes' || rota.nome === 'revisao'" :key="rota.param" :id="rota.param" />
      <Acervo v-else :so-baixados="rota.nome === 'baixados'" />
    </main>

    <Transition name="recado">
      <div v-if="recado" class="recado vidro" :class="{ erro: recado.erro }" role="status">{{ recado.texto }}</div>
    </Transition>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
}

.menu {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 20;
  display: flex;
  flex-direction: column;
  width: 216px;
  padding: max(28px, env(safe-area-inset-top)) 12px 16px;
  border-right: 1px solid var(--linha);
  background: var(--fundo);
}

.marca {
  padding: 0 10px 22px;
  font-family: var(--leitura);
  font-size: 1.2rem;
  font-weight: 600;
  letter-spacing: -0.01em;
}

nav {
  display: grid;
  gap: 1px;
}

.item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: var(--raio-p);
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.9rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.item:hover {
  background: var(--superficie);
  color: var(--texto);
}

.item.ativo {
  background: var(--superficie);
  color: var(--texto);
  font-weight: 600;
}

.contador {
  margin-left: auto;
  min-width: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--terracota);
  color: var(--destaque-texto);
  font-size: 0.7rem;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
}

.rodape {
  margin-top: auto;
  padding: 0 10px;
  color: var(--texto-fraco);
  font-size: 0.72rem;
}

.conteudo {
  margin-left: 216px;
  min-height: 100vh;
}

.faixa {
  padding: 9px 16px;
  background: var(--superficie);
  color: var(--texto-suave);
  font-size: 0.85rem;
  text-align: center;
}

.faixa.erro {
  background: var(--terracota-fundo);
  color: var(--terracota);
}

.faixa a {
  margin-left: 8px;
  color: inherit;
  font-weight: 600;
}

.recado {
  position: fixed;
  left: 50%;
  bottom: calc(24px + env(safe-area-inset-bottom));
  z-index: 40;
  max-width: calc(100% - 32px);
  padding: 10px 16px;
  border: 1px solid var(--vidro-borda);
  border-radius: 999px;
  box-shadow: var(--sombra);
  font-size: 0.88rem;
  transform: translateX(-50%);
}

.recado.erro {
  color: var(--terracota);
}

.recado-enter-active,
.recado-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.recado-enter-from,
.recado-leave-to {
  opacity: 0;
  transform: translate(-50%, 8px);
}

/* Celular: o menu vira uma barra de abas de vidro no pé da tela. No leitor ela some, para o
   texto ter a tela toda; a barra do próprio leitor tem o voltar. */
@media (max-width: 820px) {
  .menu {
    inset: auto 0 0 0;
    width: auto;
    padding: 6px 8px max(6px, env(safe-area-inset-bottom));
    border-right: 0;
    border-top: 1px solid var(--linha);
    background: var(--vidro);
    -webkit-backdrop-filter: var(--blur);
    backdrop-filter: var(--blur);
  }

  .lendo .menu {
    display: none;
  }

  .marca,
  .rodape {
    display: none;
  }

  nav {
    grid-template-columns: repeat(6, 1fr);
  }

  .item {
    position: relative;
    flex-direction: column;
    justify-content: center;
    gap: 2px;
    min-height: 48px;
    padding: 4px 0 2px;
    font-size: 0.66rem;
    font-weight: 500;
  }

  .item.ativo,
  .item:hover {
    background: transparent;
    font-weight: 600;
  }

  .contador {
    position: absolute;
    top: 2px;
    left: calc(50% + 6px);
    margin: 0;
  }

  .conteudo {
    margin-left: 0;
  }

  .recado {
    bottom: calc(80px + env(safe-area-inset-bottom));
  }

  .lendo .recado {
    bottom: calc(24px + env(safe-area-inset-bottom));
  }
}
</style>
