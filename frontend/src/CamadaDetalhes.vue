<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icone from './Icone.vue'
import { renderizar } from './markdown.js'
import { trechoDaSelecao } from './detalhes.js'

// Tudo de "Mais informações" (D26) que fica por cima do texto: a barra que aparece ao selecionar,
// o modal com a explicação e, no fim da página, as marcações que o texto não tem mais.
const props = defineProps({
  detalhes: { type: Object, required: true }, // de usarDetalhes()
  raizes: { type: Function, required: true }, // os elementos do texto onde se pode selecionar
  titulo: { type: String, default: '' },
  semLugar: { type: Array, default: () => [] },
})

const MARGEM = 16
const LARGURA_BARRA = 104

const barra = ref(null) // { x, y, acima }
const recado = ref(null)
const ocupado = ref(false)
const caixa = ref(null)
let selecao = null
let intervalo = null
let espera
let tempoDoRecado

const m = computed(() => props.detalhes.atual.value)
const explicacao = computed(() => (m.value?.explicacao ? renderizar(m.value.explicacao) : ''))

function mostrarRecado(texto, erro = false) {
  recado.value = { texto, erro }
  clearTimeout(tempoDoRecado)
  tempoDoRecado = setTimeout(() => (recado.value = null), 3500)
}

// No celular a barra vai abaixo da seleção: em cima fica o menu do próprio sistema.
function posicionar() {
  if (!intervalo) return
  const r = intervalo.getBoundingClientRect()
  if (!r.width && !r.height) return
  const toque = matchMedia('(pointer: coarse)').matches
  const folga = toque ? 30 : 10
  const cabeAbaixo = r.bottom + folga + 48 < innerHeight
  const meio = r.left + r.width / 2
  barra.value = {
    x: Math.min(Math.max(meio - LARGURA_BARRA / 2, MARGEM), innerWidth - LARGURA_BARRA - MARGEM),
    y: cabeAbaixo ? r.bottom + folga : Math.max(r.top - folga - 44, MARGEM),
  }
}

function aoMudarSelecao() {
  clearTimeout(espera)
  espera = setTimeout(() => {
    const sel = getSelection()
    const atual = sel?.rangeCount ? sel.getRangeAt(0) : null
    const achado = m.value ? null : trechoDaSelecao(props.raizes(), atual)
    if (!achado) {
      barra.value = null
      return
    }
    selecao = achado
    intervalo = atual
    posicionar()
  }, 120)
}

function esconder() {
  barra.value = null
  intervalo = null
  getSelection()?.removeAllRanges()
}

async function pedirMais() {
  const s = selecao
  esconder()
  if (!s) return
  if (s.trecho.length > 1000) return mostrarRecado('Trecho longo demais: selecione até 1000 caracteres.', true)
  try {
    const nova = await props.detalhes.pedir(props.titulo, s)
    mostrarRecado(nova.estado === 'pendente' ? 'Sem conexão: o pedido sobe quando voltar à rede de casa.' : 'Pesquisando. O trecho fica sublinhado; toque nele para ver.')
  } catch (e) {
    mostrarRecado(e.message, true)
  }
}

async function copiar() {
  const s = selecao
  esconder()
  if (!s) return
  try {
    await navigator.clipboard.writeText(s.trecho)
    mostrarRecado('Copiado.')
  } catch {
    mostrarRecado('Não foi possível copiar.', true)
  }
}

async function agir(acao) {
  ocupado.value = true
  try {
    await acao()
  } catch (e) {
    mostrarRecado(e.status === 0 ? 'Sem conexão com a rede de casa.' : e.message, true)
  } finally {
    ocupado.value = false
  }
}

function refazer() {
  agir(() => props.detalhes.refazer(m.value.id))
}

function apagar() {
  if (confirm('Apagar esta marcação e a explicação?')) agir(() => props.detalhes.apagar(m.value.id))
}

function fecharModal() {
  props.detalhes.aberto.value = null
}

function aoTeclar(evento) {
  if (evento.key === 'Escape' && m.value) fecharModal()
}

watch(m, async (aberta, antes) => {
  if (aberta && !antes) {
    barra.value = null
    recado.value = null
    await nextTick()
    caixa.value?.focus()
  }
})

const data = (iso) => new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' })
const site = (url) => {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

onMounted(() => {
  document.addEventListener('selectionchange', aoMudarSelecao)
  addEventListener('scroll', posicionar, { passive: true })
  addEventListener('resize', posicionar)
  addEventListener('keydown', aoTeclar)
})
onBeforeUnmount(() => {
  document.removeEventListener('selectionchange', aoMudarSelecao)
  removeEventListener('scroll', posicionar)
  removeEventListener('resize', posicionar)
  removeEventListener('keydown', aoTeclar)
  clearTimeout(espera)
  clearTimeout(tempoDoRecado)
})
</script>

<template>
  <section v-if="semLugar.length" class="sem-lugar">
    <h3>Marcações que não estão mais no texto</h3>
    <p class="suave">O texto mudou e o trecho não foi encontrado. A explicação continua aqui.</p>
    <div class="linha-acoes">
      <button v-for="s in semLugar" :key="s.id" class="botao" @click="detalhes.aberto.value = s.id">“{{ s.trecho }}”</button>
    </div>
  </section>

  <Teleport to="body">
    <div v-if="barra" class="barra-selecao" role="toolbar" aria-label="Seleção" :style="{ left: `${barra.x}px`, top: `${barra.y}px` }" @mousedown.prevent>
      <button type="button" aria-label="Mais informações" title="Mais informações" @click="pedirMais"><Icone nome="info" /></button>
      <button type="button" aria-label="Copiar" title="Copiar" @click="copiar"><Icone nome="copiar" /></button>
    </div>

    <div v-if="m" class="veu-detalhe" @click="fecharModal">
      <div ref="caixa" class="modal-detalhe" tabindex="-1" role="dialog" aria-modal="true" aria-labelledby="titulo-detalhe" @click.stop>
        <header>
          <span class="rotulo">Mais informações</span>
          <button class="botao icone" aria-label="Fechar" @click="fecharModal"><Icone nome="fechar" /></button>
        </header>
        <h2 id="titulo-detalhe">“{{ m.trecho }}”</h2>

        <div class="corpo">
          <p v-if="m.estado === 'pendente'" class="aviso">Sem conexão: o pedido está guardado no aparelho e a pesquisa começa quando você voltar à rede de casa.</p>
          <p v-else-if="m.estado === 'processando'" class="aviso andando"><span class="ponto" aria-hidden="true" /> Pesquisando em mais de uma fonte. Leva um ou dois minutos, ou mais se houver outra tarefa na fila; pode continuar lendo.</p>
          <template v-else-if="m.estado === 'falhou'">
            <p class="aviso erro">A pesquisa não deu certo: {{ m.erro }}</p>
          </template>
          <template v-else>
            <article class="texto explicacao" v-html="explicacao" />
            <section v-if="m.fontes.length" class="fontes">
              <h3>Fontes consultadas</h3>
              <ol>
                <li v-for="f in m.fontes" :key="f.url">
                  <a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || site(f.url) }}</a>
                  <span class="suave"> · {{ site(f.url) }}</span>
                </li>
              </ol>
            </section>
            <p class="suave quando">Pesquisado em {{ data(m.atualizado_em) }}.</p>
          </template>
        </div>

        <footer class="linha-acoes">
          <button v-if="m.estado === 'pronto' || m.estado === 'falhou'" class="botao" :disabled="ocupado" @click="refazer">{{ m.estado === 'falhou' ? 'Tentar de novo' : 'Pedir de novo' }}</button>
          <button class="botao perigo" :disabled="ocupado" @click="apagar">Apagar marcação</button>
        </footer>
      </div>
    </div>

    <div v-if="recado" class="recado-detalhe" :class="{ erro: recado.erro }" role="status">{{ recado.texto }}</div>
  </Teleport>
</template>

<style scoped>
.barra-selecao {
  position: fixed;
  z-index: 40;
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--destaque);
  box-shadow: var(--sombra);
}

.barra-selecao button {
  display: grid;
  place-items: center;
  width: 44px;
  height: 36px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--destaque-texto);
  cursor: pointer;
}

.barra-selecao button:hover {
  background: color-mix(in srgb, var(--destaque-texto) 16%, transparent);
}

.barra-selecao :deep(svg) {
  width: 20px;
  height: 20px;
}

.veu-detalhe {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 22%);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
}

.modal-detalhe {
  display: flex;
  flex-direction: column;
  width: min(620px, 100%);
  max-height: 86vh;
  padding: 14px 20px max(16px, env(safe-area-inset-bottom));
  border-radius: var(--raio-g) var(--raio-g) 0 0;
  background: var(--fundo);
  box-shadow: var(--sombra);
  outline: none;
}

.modal-detalhe header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-right: -8px;
}

.rotulo {
  margin: 0;
  color: var(--texto-fraco);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0;
}

.modal-detalhe h2 {
  margin: 2px 0 12px;
  font-family: var(--leitura);
  font-size: 1.25rem;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.corpo {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.corpo .aviso {
  margin: 0 0 12px;
}

.aviso.andando {
  display: flex;
  gap: 10px;
  align-items: baseline;
}

.ponto {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--azul);
  animation: pulsar 1.2s ease-in-out infinite;
}

@keyframes pulsar {
  50% {
    opacity: 0.25;
  }
}

/* A explicação é texto de leitura, um pouco menor que o do Tema. */
.explicacao {
  font-size: 1rem;
}

.explicacao :deep(> :first-child) {
  margin-top: 0;
}

.explicacao :deep(h1),
.explicacao :deep(h2) {
  font-size: 1.1rem;
}

.fontes {
  margin-top: 18px;
  padding-top: 12px;
  border-top: 1px solid var(--linha);
}

.fontes h3 {
  margin: 0 0 6px;
  color: var(--texto-fraco);
  font-size: 0.8rem;
  font-weight: 600;
}

.fontes ol {
  margin: 0;
  padding-left: 20px;
  font-size: 0.9rem;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.quando {
  margin: 12px 0 0;
  font-size: 0.8rem;
}

.modal-detalhe footer {
  padding-top: 12px;
  border-top: 1px solid var(--linha);
  margin-top: 12px;
}

.sem-lugar {
  margin-top: 36px;
  padding-top: 16px;
  border-top: 1px solid var(--linha);
}

.sem-lugar h3 {
  margin: 0 0 4px;
  color: var(--texto-suave);
  font-size: 0.82rem;
  font-weight: 600;
}

.sem-lugar p {
  margin: 0 0 10px;
  font-size: 0.88rem;
}

.sem-lugar .botao {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}

.recado-detalhe {
  position: fixed;
  left: 50%;
  bottom: calc(84px + env(safe-area-inset-bottom));
  z-index: 60;
  max-width: calc(100% - 32px);
  padding: 10px 16px;
  border-radius: var(--raio);
  background: var(--destaque);
  color: var(--destaque-texto);
  font-size: 0.88rem;
  box-shadow: var(--sombra);
  transform: translateX(-50%);
}

.recado-detalhe.erro {
  background: var(--vermelho);
  color: #fff;
}

@media (min-width: 821px) {
  .veu-detalhe {
    align-items: center;
  }

  .modal-detalhe {
    border-radius: var(--raio-g);
    max-height: 80vh;
  }

  .recado-detalhe {
    bottom: 24px;
  }
}
</style>
