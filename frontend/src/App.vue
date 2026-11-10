<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Acervo from './Acervo.vue'
import Ajustes from './Ajustes.vue'
import Fila from './Fila.vue'
import Icone from './Icone.vue'
import Instrucao from './Instrucao.vue'
import Leitor from './Leitor.vue'
import Pesquisar from './Pesquisar.vue'
import Referencia from './Referencia.vue'
import Referencias from './Referencias.vue'
import Resumir from './Resumir.vue'
import Revisoes from './Revisoes.vue'
import { api } from './api.js'
import { conflitos, pendentes, sincronizar } from './offline.js'
import { enviarDetalhesPendentes } from './detalhes.js'

// Rotas no hash, como na Biblioteca: o "voltar" do celular volta para a tela anterior.
const hash = ref(location.hash)
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
  if (n === 'referencia' || n === 'resumir' || n === 'pesquisar') return 'referencias'
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

onMounted(async () => {
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
      <div class="marca"><span class="marca-icone"><Icone nome="acervo" /></span>Acervo</div>
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
      <Resumir v-else-if="rota.nome === 'resumir'" :livro="rota.param" />
      <Pesquisar v-else-if="rota.nome === 'pesquisar'" />
      <Fila v-else-if="rota.nome === 'fila'" />
      <Instrucao v-else-if="rota.nome === 'ajustes' && rota.param.startsWith('instrucoes/')" :key="rota.param" :arquivo="rota.param.slice('instrucoes/'.length)" />
      <Ajustes v-else-if="rota.nome === 'ajustes'" :tela="telaAnterior" :secao="rota.param" />
      <Revisoes v-else-if="rota.nome === 'revisoes' || rota.nome === 'revisao'" :key="rota.param" :id="rota.param" />
      <Acervo v-else :so-baixados="rota.nome === 'baixados'" />
    </main>

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
  background: var(--lateral);
}

/* A cor do app aparece aqui, no item ativo e no botão principal. */
.marca {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 8px 24px;
  font-size: 1.02rem;
  font-weight: 700;
  letter-spacing: -0.015em;
}

.marca-icone {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: var(--marca);
  color: var(--marca-texto);
}

.marca-icone svg {
  width: 17px;
  height: 17px;
  stroke-width: 1.9;
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
  background: var(--linha);
  color: var(--texto);
}

.item.ativo {
  background: var(--item-ativo);
  box-shadow: 0 0 0 1px var(--linha), 0 1px 2px rgb(23 25 30 / 5%);
  color: var(--texto);
  font-weight: 600;
}

.item.ativo svg {
  color: var(--marca);
}

.contador {
  margin-left: auto;
  min-width: 18px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--vermelho);
  color: #ffffff;
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
  background: var(--vermelho-fundo);
  color: var(--vermelho);
}

.faixa a {
  margin-left: 8px;
  color: inherit;
  font-weight: 600;
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
    box-shadow: none;
    font-weight: 600;
  }

  .item.ativo {
    color: var(--marca);
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
}
</style>
