<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from './api.js'
import { listaDeDuvidas, renderizar, urlNoTempo } from './markdown.js'

// Aba Dúvidas (D37): cada dúvida do texto organizado, com o que se apurou. "Resolver" ouve de novo o
// áudio daquele ponto (Whisper large-v3 no Mac mini) e confere na internet; a decisão é do Lucas:
// Corrigir no texto (proposta revisada), Pesquisar isto, ou deixar como está.
const props = defineProps({
  referencia: { type: Object, required: true },
  texto: { type: String, required: true },
})
const emit = defineEmits(['perguntar', 'aprofundar'])

const ESTADOS = {
  resolvida_audio: ['Resolvida pelo áudio', 'ok'],
  confirmada_internet: ['Confirmada na internet', 'ok'],
  incerta: ['Continua incerta', 'meio'],
  lacuna: ['Falta no material', 'meio'],
}

const resolvidas = ref([])
const erro = ref('')
const pedindo = ref(false)
const recados = ref({}) // texto da dúvida → recado depois de uma ação
let espera = null

const temAudio = computed(() => ['video', 'podcast', 'reel'].includes(props.referencia.tipo))
const itens = computed(() => {
  const porTexto = Object.fromEntries(resolvidas.value.map((d) => [d.texto, d]))
  return listaDeDuvidas(props.texto).map((texto) => ({ texto, d: porTexto[texto] }))
})
const processando = computed(() => resolvidas.value.some((d) => d.estado === 'processando'))
const faltam = computed(() => itens.value.filter((i) => !i.d || i.d.estado === 'falhou').length)

function mensagem(e) {
  return e.status === 0 ? 'Resolver as dúvidas só funciona na rede de casa.' : e.message
}

async function carregar() {
  try {
    resolvidas.value = await api(`/api/referencias/${props.referencia.id}/duvidas`)
  } catch (e) {
    erro.value = mensagem(e)
  }
  clearTimeout(espera)
  if (processando.value) espera = setTimeout(carregar, 4000)
}

async function resolver(texto = '') {
  pedindo.value = true
  erro.value = ''
  try {
    resolvidas.value = await api(`/api/referencias/${props.referencia.id}/duvidas`, { metodo: 'POST', corpo: texto ? { texto } : {} })
    carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    pedindo.value = false
  }
}

async function corrigir(item) {
  try {
    await api(`/api/duvidas/${item.d.id}/integrar`, { metodo: 'POST' })
    recados.value = { ...recados.value, [item.texto]: 'A correção está na fila. Quando ficar pronta, aparece em Revisões para você aprovar.' }
  } catch (e) {
    erro.value = mensagem(e)
  }
}

// "Pesquisar isto": numa pesquisa, aprofundar com a dúvida como foco; nos outros, perguntar (D35).
function pesquisar(item) {
  const limpo = item.texto.replace(/\*\*|\[\?\]/g, '').trim()
  if (props.referencia.tipo === 'pesquisa') emit('aprofundar', limpo)
  else emit('perguntar', limpo.slice(0, 500))
}

const linha = (md) => renderizar(md).replace(/^<p>|<\/p>\s*$/g, '')

onMounted(carregar)
onBeforeUnmount(() => clearTimeout(espera))
</script>

<template>
  <section class="duvidas">
    <div class="cabeca">
      <p class="suave">
        {{ temAudio ? 'Resolver ouve de novo o áudio de cada trecho (Whisper no Mac mini) e confere na internet.' : 'Resolver confere cada dúvida na internet.' }}
        Você decide o que corrigir.
      </p>
      <button class="botao principal" :disabled="pedindo || processando || !faltam" @click="resolver()">
        {{ processando ? 'Resolvendo…' : faltam === itens.length ? 'Resolver as dúvidas' : faltam ? `Resolver as ${faltam} que faltam` : 'Todas resolvidas' }}
      </button>
    </div>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <article v-for="item in itens" :key="item.texto" class="duvida">
      <p class="texto-da-duvida" v-html="linha(item.texto)" />

      <p v-if="item.d?.estado === 'processando'" class="chip andando">{{ temAudio && item.d.momento !== '' ? 'Ouvindo e conferindo…' : 'Conferindo…' }}</p>
      <p v-else-if="item.d?.estado === 'falhou'" class="aviso erro">{{ item.d.erro }}</p>
      <template v-else-if="item.d?.resultado">
        <div class="selos">
          <span class="chip" :class="ESTADOS[item.d.resultado.estado]?.[1]">{{ ESTADOS[item.d.resultado.estado]?.[0] }}</span>
          <a v-if="item.d.momento && temAudio" class="suave momento" :href="urlNoTempo(referencia.url, item.d.momento)" target="_blank" rel="noopener">ouvir em {{ item.d.momento }}</a>
        </div>
        <p v-if="item.d.resultado.no_audio"><span class="suave">No áudio:</span> {{ item.d.resultado.no_audio }}</p>
        <p>{{ item.d.resultado.explicacao }}</p>
        <p v-if="item.d.resultado.correto" class="correcao">
          <span class="suave">Onde está</span> “{{ item.d.resultado.trecho }}”<span class="suave">, fica</span> “{{ item.d.resultado.correto }}”
        </p>
        <details v-if="item.d.resultado.fontes.length" class="fontes">
          <summary class="suave">Fontes ({{ item.d.resultado.fontes.length }})</summary>
          <ol>
            <li v-for="f in item.d.resultado.fontes" :key="f.url"><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></li>
          </ol>
        </details>
        <details v-if="item.d.audio" class="fontes">
          <summary class="suave">O que o Whisper ouviu</summary>
          <pre class="ouvido">{{ item.d.audio }}</pre>
        </details>
      </template>

      <p v-if="recados[item.texto]" class="aviso">{{ recados[item.texto] }} <a href="#/revisoes">Revisões</a></p>
      <div v-if="item.d?.estado !== 'processando'" class="acoes">
        <button v-if="item.d?.resultado?.correto && !recados[item.texto]" class="botao discreto marca" @click="corrigir(item)">Corrigir no texto</button>
        <button v-if="['incerta', 'lacuna'].includes(item.d?.resultado?.estado)" class="botao discreto marca" @click="pesquisar(item)">Pesquisar isto</button>
        <button class="botao discreto" :disabled="pedindo || processando" @click="resolver(item.texto)">{{ item.d ? 'Resolver de novo' : 'Resolver' }}</button>
      </div>
    </article>
  </section>
</template>

<style scoped>
.duvidas {
  display: grid;
  gap: 4px;
}

.cabeca {
  display: grid;
  gap: 10px;
  justify-items: start;
  margin-bottom: 8px;
}

.cabeca p,
.duvida p {
  margin: 0;
}

.duvida {
  display: grid;
  gap: 8px;
  padding: 14px 0;
  border-top: 1px solid var(--linha);
  font-size: 0.95rem;
}

.texto-da-duvida {
  font-size: 1rem;
  line-height: 1.5;
}

.selos {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.chip.meio {
  color: var(--texto);
  background: var(--superficie-forte);
}

.chip.andando {
  justify-self: start;
}

.momento {
  font-size: 0.88rem;
}

.correcao {
  padding: 8px 10px;
  border-radius: var(--raio);
  background: var(--superficie);
}

.fontes summary {
  cursor: pointer;
  font-size: 0.88rem;
}

.fontes ol {
  margin: 6px 0 0;
  padding-left: 22px;
  font-size: 0.88rem;
  overflow-wrap: anywhere;
}

.ouvido {
  margin: 6px 0 0;
  padding: 8px 10px;
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.85rem;
  white-space: pre-wrap;
}

.acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-left: -8px;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.88rem;
}

.botao.discreto.marca {
  color: var(--marca);
}
</style>
