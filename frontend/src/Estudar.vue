<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import Simulado from './Simulado.vue'
import { api } from './api.js'
import { renderizar } from './markdown.js'

// Aba Estudar de um Tema (D41): resumo inteligente, conceitos anteriores, simulado e flashcards.
// Cada parte é preparada pela IA na fila, a partir do Tema inteiro (com os subtemas), e fica
// guardada; quando o Tema muda, a parte avisa que pode ser refeita.
const props = defineProps({
  tema: { type: String, required: true }, // temas/<pasta>
  titulo: { type: String, required: true },
  flashcards: { type: Boolean, default: false }, // o Flashcards está no ar
})
const emit = defineEmits(['flashcards'])

const SECOES = [
  ['resumo', 'Resumo inteligente', 'O Tema inteiro em poucos minutos: a ideia central, o essencial, como as partes se ligam e o que costuma confundir.'],
  ['anteriores', 'Conceitos anteriores', 'O que ajuda a entender este Tema e não está explicado nele: a base que ele supõe e os assuntos ao redor.'],
  ['simulado', 'Simulado', 'Perguntas sorteadas do Tema para ver se você entendeu. Responda escrevendo ou falando; a IA corrige.'],
]
const PREPARAR = { resumo: 'Preparar o resumo', anteriores: 'Achar os conceitos anteriores', simulado: 'Preparar as perguntas' }

const dados = ref(null) // { partes: { resumo, anteriores, simulado }, simulados }
const erro = ref('')
const pedindo = ref('')
const falando = ref(false)
let espera = null

const partes = computed(() => dados.value?.partes ?? {})
const processando = computed(() => Object.values(partes.value).some((p) => p.estado === 'processando') || dados.value?.simulados?.some((s) => s.estado === 'processando'))
const conceitos = computed(() => partes.value.anteriores?.resultado?.conceitos ?? [])
const temVozDeLeitura = typeof window !== 'undefined' && 'speechSynthesis' in window

function mensagem(e) {
  return e.status === 0 ? 'Estudar precisa da rede de casa.' : e.message
}

async function carregar() {
  try {
    dados.value = await api(`/api/estudos?tema=${encodeURIComponent(props.tema)}`)
    erro.value = ''
  } catch (e) {
    erro.value = mensagem(e)
  }
  clearTimeout(espera)
  if (processando.value) espera = setTimeout(carregar, 4000)
}

async function preparar(parte) {
  pedindo.value = parte
  erro.value = ''
  try {
    await api('/api/estudos', { metodo: 'POST', corpo: { tema: props.tema, parte } })
    await carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    pedindo.value = ''
  }
}

// Ouvir o resumo: a voz do próprio aparelho lê o texto, sem as marcas do Markdown.
function ouvirResumo() {
  if (falando.value) {
    speechSynthesis.cancel()
    falando.value = false
    return
  }
  const texto = partes.value.resumo.resultado.texto.replace(/[#*_>`]/g, '').replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
  const fala = new SpeechSynthesisUtterance(texto)
  fala.lang = 'pt-BR'
  fala.onend = fala.onerror = () => (falando.value = false)
  speechSynthesis.cancel()
  speechSynthesis.speak(fala)
  falando.value = true
}

function quando(iso) {
  return iso ? new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' }) : ''
}

function ir(id) {
  document.getElementById(`estudar-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

onMounted(carregar)
onBeforeUnmount(() => {
  clearTimeout(espera)
  if (falando.value) speechSynthesis.cancel()
})
</script>

<template>
  <div class="estudar">
    <nav class="atalhos" aria-label="Partes do estudo">
      <button v-for="[id, nome] in SECOES" :key="id" class="chip" @click="ir(id)">{{ nome }}</button>
      <button class="chip" @click="ir('flashcards')">Flashcards</button>
    </nav>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <p v-if="!dados && !erro" class="suave">Carregando…</p>

    <template v-if="dados">
      <section v-for="[id, nome, descricao] in SECOES" :id="`estudar-${id}`" :key="id" class="parte">
        <header>
          <h2>{{ nome }}</h2>
          <p class="descricao">{{ descricao }}</p>
        </header>

        <p v-if="partes[id].estado === 'processando'" class="chip andando">{{ partes[id].resultado ? 'Refazendo…' : 'Preparando…' }} (na fila)</p>
        <p v-else-if="partes[id].estado === 'falhou'" class="aviso erro">{{ partes[id].erro }}</p>
        <p v-if="partes[id].desatualizado && partes[id].estado !== 'processando'" class="aviso">O Tema mudou desde que isto foi preparado. Refaça para valer o texto de agora.</p>

        <!-- Resumo -->
        <template v-if="id === 'resumo' && partes.resumo.resultado">
          <button v-if="temVozDeLeitura" class="botao ouvir" @click="ouvirResumo"><Icone :nome="falando ? 'parar' : 'ouvir'" /> {{ falando ? 'Parar' : 'Ouvir o resumo' }}</button>
          <article class="texto resumo" v-html="renderizar(partes.resumo.resultado.texto)" />
        </template>

        <!-- Conceitos anteriores -->
        <template v-if="id === 'anteriores' && conceitos.length">
          <article v-for="c in conceitos" :key="c.nome" class="conceito">
            <div class="cabeca-conceito">
              <h3>{{ c.nome }}</h3>
              <span class="chip" :class="{ base: c.tipo === 'base' }">{{ c.tipo === 'base' ? 'Base' : 'Ao redor' }}</span>
            </div>
            <p class="por-que">{{ c.por_que }}</p>
            <details>
              <summary>Entender o conceito</summary>
              <div class="texto explicacao" v-html="renderizar(c.explicacao)" />
              <p v-if="c.fontes.length" class="fontes suave">
                Fontes:
                <template v-for="(f, i) in c.fontes" :key="f.url"><template v-if="i">, </template><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></template>
              </p>
            </details>
            <a v-if="c.tema" class="no-acervo" :href="`#/ler/${c.tema}/indice.md`"><Icone nome="acervo" /> Está no seu Acervo</a>
          </article>
        </template>

        <!-- Simulado -->
        <Simulado
          v-if="id === 'simulado' && partes.simulado.resultado?.perguntas?.length"
          :tema="tema"
          :banco="partes.simulado.resultado.perguntas"
          :simulados="dados.simulados"
          :flashcards="flashcards"
          @entregue="carregar"
          @flashcards="emit('flashcards', $event)"
        />

        <div v-if="partes[id].estado !== 'processando'" class="linha-acoes">
          <button v-if="!partes[id].resultado" class="botao principal" :disabled="!!pedindo" @click="preparar(id)">{{ PREPARAR[id] }}</button>
          <button v-else class="botao discreto" :disabled="!!pedindo" @click="preparar(id)">
            {{ id === 'simulado' ? 'Refazer as perguntas' : 'Refazer' }}<span v-if="partes[id].atualizado_em" class="meta"> · preparado em {{ quando(partes[id].atualizado_em) }}</span>
          </button>
        </div>
      </section>

      <section id="estudar-flashcards" class="parte">
        <header>
          <h2>Flashcards</h2>
          <p class="descricao">Os cartões vão para o app Flashcards; você revisa a proposta lá antes de entrarem no baralho.</p>
        </header>
        <p v-if="!flashcards" class="suave">O Flashcards não respondeu agora.</p>
        <div v-else class="opcoes-cartoes">
          <button class="botao" @click="emit('flashcards', { origem: { arquivo: `${tema}/indice.md`, capitulo: '' }, titulo: 'do Tema inteiro' })">
            <Icone nome="cartoes" /> Do Tema inteiro
          </button>
          <button class="botao" :disabled="!conceitos.length" @click="emit('flashcards', { origem: { estudo: tema, parte: 'anteriores' }, titulo: 'dos conceitos anteriores' })">
            <Icone nome="cartoes" /> Dos conceitos anteriores
          </button>
          <p class="suave dica">Depois de um simulado, “Flashcards do que errei” transforma as respostas erradas em cartões.</p>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.estudar {
  display: grid;
  gap: 8px;
}

.atalhos {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 4px;
}

.atalhos .chip {
  border: 0;
  cursor: pointer;
  font: inherit;
  font-size: 0.82rem;
}

.parte {
  display: grid;
  gap: 12px;
  padding: 22px 0;
  border-top: 1px solid var(--linha);
  scroll-margin-top: 72px;
}

.parte:first-of-type {
  border-top: 0;
}

.parte header {
  display: grid;
  gap: 4px;
}

.parte h2 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 650;
  letter-spacing: -0.012em;
}

.descricao {
  margin: 0;
  color: var(--texto-suave);
  font-size: 0.906rem;
}

.parte > p {
  margin: 0;
}

.chip.andando {
  justify-self: start;
}

.ouvir {
  justify-self: start;
}

.resumo :deep(h2) {
  margin-top: 1.2em;
  font-size: 1.0625rem;
}

.resumo :deep(h2:first-child) {
  margin-top: 0;
}

.conceito {
  display: grid;
  gap: 6px;
  padding: 12px 0;
  border-top: 1px solid var(--linha);
}

.conceito:first-of-type {
  border-top: 0;
  padding-top: 0;
}

.cabeca-conceito {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cabeca-conceito h3 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 650;
}

.chip.base {
  color: var(--marca);
  background: var(--marca-fundo);
}

.por-que {
  margin: 0;
  color: var(--texto-suave);
  font-size: 0.95rem;
}

.conceito summary {
  cursor: pointer;
  color: var(--marca);
  font-size: 0.92rem;
}

.explicacao {
  margin-top: 8px;
  font-size: 0.98rem;
}

.fontes {
  margin: 6px 0 0;
  font-size: 0.85rem;
  overflow-wrap: anywhere;
}

.no-acervo {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.9rem;
}

.no-acervo :deep(svg) {
  width: 18px;
  height: 18px;
}

.linha-acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.88rem;
  margin-left: -8px;
}

.meta {
  color: var(--texto-fraco);
}

.opcoes-cartoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dica {
  flex-basis: 100%;
  margin: 0;
  font-size: 0.85rem;
}
</style>
