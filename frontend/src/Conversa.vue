<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Icone from './Icone.vue'
import Perguntas from './Perguntas.vue'
import { api } from './api.js'
import { renderizar } from './markdown.js'
import { juntar, ouvir, temVoz } from './voz.js'

// Conversar (D48): a conversa com a IA sobre o texto aberto (Tema ou Referência), no lugar do
// Perguntar (D35). A IA lembra as mensagens anteriores, procura no Acervo, pesquisa na internet só
// quando precisa e, quando o Lucas pede para mudar ou acrescentar, devolve cada mudança para ele
// aplicar ou descartar ali mesmo. A mensagem pode levar o trecho selecionado no texto.
const props = defineProps({
  local: { type: String, required: true }, // temas/<pasta> ou referencias/<id>
  titulo: { type: String, default: '' },
  trecho: { type: Object, default: null }, // { trecho, arquivo } vindo da seleção
})
const emit = defineEmits(['mudou', 'sem-trecho'])

const mensagens = ref([])
const antigas = ref(0) // perguntas verificadas de antes do Conversar
const texto = ref('')
const enviando = ref(false)
const erro = ref('')
const decidindo = ref('')
const campo = ref(null)
const ouvindo = ref(false)
let parar = null
let espera = null

const processando = computed(() => mensagens.value.some((m) => m.estado === 'processando'))

async function carregar() {
  try {
    const lista = await api(`/api/perguntas?local=${encodeURIComponent(props.local)}`)
    mensagens.value = lista.filter((m) => m.modo === 'conversa').reverse() // a mais antiga em cima, como numa conversa
    antigas.value = lista.length - mensagens.value.length
  } catch (e) {
    erro.value = e.status === 0 ? 'Conversar só funciona na rede de casa.' : e.message
  }
  clearTimeout(espera)
  if (processando.value) espera = setTimeout(carregar, 4000)
}

async function enviar() {
  const msg = texto.value.trim()
  if (!msg || enviando.value) return
  pararVoz()
  enviando.value = true
  erro.value = ''
  try {
    await api('/api/perguntas', {
      metodo: 'POST',
      corpo: { local: props.local, titulo: props.titulo, pergunta: msg, trecho: props.trecho?.trecho ?? '', arquivo: props.trecho?.arquivo ?? '' },
    })
    texto.value = ''
    emit('sem-trecho')
    await carregar()
    await nextTick()
    campo.value?.scrollIntoView({ block: 'end', behavior: 'smooth' })
  } catch (e) {
    erro.value = e.status === 0 ? 'Conversar só funciona na rede de casa.' : e.message
  } finally {
    enviando.value = false
  }
}

async function decidir(m, n, decisao) {
  decidindo.value = `${m.id}/${n}`
  erro.value = ''
  try {
    const nova = await api(`/api/perguntas/${m.id}/alteracoes/${n}/${decisao}`, { metodo: 'POST' })
    mensagens.value = mensagens.value.map((x) => (x.id === m.id ? nova : x))
    if (decisao === 'aplicar') emit('mudou')
  } catch (e) {
    erro.value = e.message
  } finally {
    decidindo.value = ''
  }
}

async function refazer(m) {
  try {
    await api(`/api/perguntas/${m.id}/refazer`, { metodo: 'POST' })
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

async function apagar(m) {
  if (!confirm('Apagar esta mensagem e a resposta? A IA deixa de lembrar dela.')) return
  try {
    await api(`/api/perguntas/${m.id}`, { metodo: 'DELETE' })
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

// Ditado pela voz, como no Simulado (D41): o que se fala entra no campo.
function alternarVoz() {
  if (ouvindo.value) return pararVoz()
  const antes = texto.value.trim()
  ouvindo.value = true
  parar = ouvir({
    aoMudar: (dito) => (texto.value = juntar(antes, dito)),
    aoTerminar: (e) => {
      ouvindo.value = false
      parar = null
      if (e) erro.value = e
    },
  })
}
function pararVoz() {
  parar?.()
  parar = null
  ouvindo.value = false
}

const ESTADO = { aplicada: 'Aplicada no texto', descartada: 'Descartada' }
const linkDoLugar = (c) => (c.startsWith('referencias/') ? `#/referencia/${c.slice(12)}` : `#/ler/${c.endsWith('.md') ? c : `${c}/indice.md`}`)
const quando = (iso) => new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })

// Um trecho novo vindo da seleção: o campo ganha o foco.
watch(
  () => props.trecho,
  async (t) => {
    if (!t) return
    await nextTick()
    campo.value?.focus()
  },
  { immediate: true },
)

onMounted(carregar)
onBeforeUnmount(() => {
  clearTimeout(espera)
  pararVoz()
})
</script>

<template>
  <section class="conversa">
    <p v-if="!mensagens.length && !processando" class="suave">
      Pergunte, peça para conferir, explicar, mudar ou acrescentar algo no texto, ou pergunte se o assunto já está no seu Acervo. A IA lembra
      da conversa e pesquisa na internet quando precisa. Para falar de uma parte, selecione o trecho no texto e toque em
      <strong>Conversar</strong>.
    </p>

    <details v-if="antigas" class="antigas">
      <summary class="suave">Perguntas verificadas de antes ({{ antigas }})</summary>
      <Perguntas embutido antigas :local="local" :titulo="titulo" />
    </details>

    <article v-for="m in mensagens" :key="m.id" class="mensagem">
      <div class="lucas">
        <blockquote v-if="m.trecho" class="citado">{{ m.trecho }}</blockquote>
        <p>{{ m.pergunta }}</p>
        <span class="suave quando">{{ quando(m.criado_em) }}</span>
      </div>

      <div class="ia">
        <p v-if="m.estado === 'processando'" class="chip andando">Pensando… pode sair da tela; o celular avisa quando responder.</p>
        <p v-else-if="m.estado === 'falhou'" class="aviso erro">{{ m.erro }}</p>
        <template v-else-if="m.conversa">
          <div v-if="m.conversa.resposta" class="texto resposta" v-html="renderizar(m.conversa.resposta)" />

          <div v-for="(alt, n) in m.conversa.alteracoes" :key="n" class="alteracao" :class="alt.estado">
            <p class="rotulo-inline">{{ alt.resumo || 'Mudança no texto' }}<span v-if="ESTADO[alt.estado]" class="chip" :class="{ ok: alt.estado === 'aplicada' }">{{ ESTADO[alt.estado] }}</span></p>
            <template v-if="alt.estado === 'proposta'">
              <div class="texto-novo" v-html="renderizar(alt.depois)" />
              <details class="como-era">
                <summary class="suave">Como está hoje</summary>
                <div class="texto-velho" v-html="renderizar(alt.antes)" />
              </details>
              <div class="linha-acoes">
                <button class="botao principal" :disabled="decidindo === `${m.id}/${n}`" @click="decidir(m, n, 'aplicar')">Aplicar</button>
                <button class="botao" :disabled="decidindo === `${m.id}/${n}`" @click="decidir(m, n, 'descartar')">Descartar</button>
              </div>
            </template>
          </div>

          <div v-if="m.conversa.no_acervo?.length" class="no-acervo">
            <span class="rotulo-inline">No seu Acervo</span>
            <a v-for="l in m.conversa.no_acervo" :key="l.caminho" class="botao" :href="linkDoLugar(l.caminho)">{{ l.titulo || l.caminho }}</a>
          </div>
          <details v-if="m.conversa.fontes?.length" class="fontes">
            <summary class="suave">Fontes ({{ m.conversa.fontes.length }})</summary>
            <ol>
              <li v-for="f in m.conversa.fontes" :key="f.url"><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></li>
            </ol>
          </details>
        </template>
        <div v-if="m.estado !== 'processando'" class="acoes">
          <button class="botao discreto" @click="refazer(m)">Responder de novo</button>
          <button class="botao discreto" @click="apagar(m)">Apagar</button>
        </div>
      </div>
    </article>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <form ref="campo" class="nova" @submit.prevent="enviar">
      <div v-if="trecho" class="trecho-escolhido">
        <blockquote class="citado">{{ trecho.trecho }}</blockquote>
        <button type="button" class="botao icone" aria-label="Tirar o trecho" title="Tirar o trecho" @click="emit('sem-trecho')"><Icone nome="fechar" /></button>
      </div>
      <textarea
        v-model="texto"
        class="campo"
        rows="3"
        maxlength="2000"
        aria-label="Mensagem"
        :placeholder="trecho ? 'O que você quer saber ou mudar neste trecho?' : 'Pergunte, peça uma explicação ou uma mudança no texto…'"
        @keydown.enter.exact.prevent="enviar"
      />
      <div class="linha-acoes">
        <button class="botao principal" :disabled="enviando || !texto.trim()">{{ enviando ? 'Enviando…' : 'Enviar' }}</button>
        <button v-if="temVoz" type="button" class="botao" :class="{ ouvindo }" :aria-pressed="ouvindo" @click="alternarVoz">
          <Icone :nome="ouvindo ? 'parar' : 'microfone'" /> {{ ouvindo ? 'Parar' : 'Falar' }}
        </button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.conversa {
  display: grid;
  gap: 16px;
}
.antigas summary {
  cursor: pointer;
}
.mensagem {
  display: grid;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid var(--linha);
}
.lucas {
  justify-self: end;
  max-width: min(92%, 560px);
  display: grid;
  gap: 6px;
  padding: 10px 12px;
  border-radius: var(--raio);
  background: var(--superficie);
}
.lucas p {
  margin: 0;
  white-space: pre-wrap;
}
.quando {
  font-size: 0.8rem;
  justify-self: end;
}
.citado {
  margin: 0;
  padding: 2px 0 2px 10px;
  border-left: 3px solid var(--marca);
  color: var(--texto-suave);
  font-family: var(--leitura);
  font-size: 0.92rem;
  max-height: 7.5em;
  overflow: hidden;
}
.ia {
  display: grid;
  gap: 10px;
}
.resposta :deep(p:first-child) {
  margin-top: 0;
}
.alteracao {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--linha-forte);
  border-radius: var(--raio);
}
.alteracao.descartada {
  opacity: 0.6;
}
.rotulo-inline {
  margin: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  font-weight: 600;
}
.texto-novo,
.texto-velho {
  font-family: var(--leitura);
  font-size: 1rem;
  line-height: 1.6;
}
.texto-novo {
  padding: 8px 10px;
  border-radius: var(--raio-p);
  background: var(--mais);
}
.texto-velho {
  padding: 8px 10px;
  border-radius: var(--raio-p);
  background: var(--menos);
}
.texto-novo :deep(p),
.texto-velho :deep(p) {
  margin: 0 0 0.6em;
}
.texto-novo :deep(p:last-child),
.texto-velho :deep(p:last-child) {
  margin-bottom: 0;
}
.como-era summary,
.fontes summary {
  cursor: pointer;
}
.no-acervo {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.fontes ol {
  margin: 6px 0 0;
  padding-left: 22px;
}
.acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.nova {
  display: grid;
  gap: 8px;
  position: sticky;
  bottom: 0;
  padding: 10px 0 calc(10px + env(safe-area-inset-bottom));
  background: var(--fundo);
  border-top: 1px solid var(--linha);
}
.nova textarea {
  resize: vertical;
}
.trecho-escolhido {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.trecho-escolhido .citado {
  flex: 1;
}
.ouvindo {
  color: var(--perigo);
}
</style>
