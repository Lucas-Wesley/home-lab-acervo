<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from './api.js'
import { listaDeDuvidas, renderizar, urlNoTempo } from './markdown.js'

// Aba Dúvidas (D37): cada dúvida do texto organizado, com o que se apurou. "Resolver" ouve de novo o
// áudio daquele ponto (Whisper large-v3 no Mac mini) e confere na internet; a decisão é do Lucas:
// Corrigir no texto, Pesquisar isto, ou deixar como está. A correção é pequena: o texto corrigido
// aparece aqui mesmo para aprovar ou descartar, sem ir a Revisões, e pedir de novo não cria outra
// (D44).
const props = defineProps({
  referencia: { type: Object, required: true },
  texto: { type: String, required: true },
})
const emit = defineEmits(['perguntar', 'aprofundar', 'mudou'])

const ESTADOS = {
  resolvida_audio: ['Resolvida pelo áudio', 'ok'],
  confirmada_internet: ['Confirmada na internet', 'ok'],
  incerta: ['Continua incerta', 'meio'],
  lacuna: ['Falta no material', 'meio'],
}

const resolvidas = ref([])
const erro = ref('')
const feito = ref('')
const pedindo = ref(false)
const decidindo = ref('') // id da dúvida cuja correção está sendo aprovada ou descartada
let espera = null

const temAudio = computed(() => ['video', 'podcast', 'reel'].includes(props.referencia.tipo))
const itens = computed(() => {
  const porTexto = Object.fromEntries(resolvidas.value.map((d) => [d.texto, d]))
  return listaDeDuvidas(props.texto).map((texto) => ({ texto, d: porTexto[texto] }))
})
const preparando = (d) => ['na_fila', 'rodando'].includes(d?.correcao?.estado)
const paraAprovar = (d) => d?.correcao?.estado === 'em_revisao'
const processando = computed(() => resolvidas.value.some((d) => d.estado === 'processando'))
const trabalhando = computed(() => processando.value || resolvidas.value.some(preparando))
const faltam = computed(() => itens.value.filter((i) => !i.d || i.d.estado === 'falhou').length)
const contagem = computed(() => {
  const n = (f) => itens.value.filter((i) => f(i.d)).length
  const certas = n((d) => ['resolvida_audio', 'confirmada_internet'].includes(d?.resultado?.estado))
  const aprovar = n(paraAprovar)
  return [
    certas && `${certas} ${certas === 1 ? 'resolvida' : 'resolvidas'}`,
    aprovar && `${aprovar} ${aprovar === 1 ? 'correção para aprovar' : 'correções para aprovar'}`,
  ].filter(Boolean)
})

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
  if (trabalhando.value) espera = setTimeout(carregar, 4000)
}

async function resolver(texto = '') {
  pedindo.value = true
  erro.value = ''
  feito.value = ''
  try {
    resolvidas.value = await api(`/api/referencias/${props.referencia.id}/duvidas`, { metodo: 'POST', corpo: texto ? { texto } : {} })
    carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    pedindo.value = false
  }
}

// Pedir duas vezes devolve a mesma proposta: o botão some assim que ela existe.
async function corrigir(item) {
  erro.value = ''
  feito.value = ''
  decidindo.value = item.d.id
  try {
    await api(`/api/duvidas/${item.d.id}/integrar`, { metodo: 'POST' })
    await carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    decidindo.value = ''
  }
}

async function decidir(item, decisao) {
  erro.value = ''
  feito.value = ''
  decidindo.value = item.d.id
  try {
    await api(`/api/revisoes/${item.d.correcao.tarefa_id}/${decisao}`, { metodo: 'POST' })
    if (decisao === 'aprovar') {
      feito.value = 'Corrigido no texto. A versão de antes fica em "Versão anterior".'
      emit('mudou')
    }
    await carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    decidindo.value = ''
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
    <header class="cabeca">
      <p class="descricao">
        {{ temAudio ? 'Resolver ouve de novo o áudio de cada trecho e confere na internet.' : 'Resolver confere cada dúvida na internet.' }}
        Você decide o que corrigir.
      </p>
      <div class="linha-cabeca">
        <button v-if="faltam || processando" class="botao principal" :disabled="pedindo || processando" @click="resolver()">
          {{ processando ? 'Resolvendo…' : faltam === itens.length ? 'Resolver as dúvidas' : `Resolver as ${faltam} que faltam` }}
        </button>
        <span v-for="c in contagem" :key="c" class="chip">{{ c }}</span>
      </div>
    </header>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div v-if="feito" class="aviso ok">{{ feito }}</div>

    <ol class="lista-duvidas">
      <li v-for="item in itens" :key="item.texto" class="duvida">
        <div v-if="item.d" class="selos">
          <span v-if="item.d.estado === 'processando'" class="chip andando">{{ temAudio && item.d.momento !== '' ? 'Ouvindo e conferindo…' : 'Conferindo…' }}</span>
          <span v-else-if="item.d.estado === 'falhou'" class="chip erro">Falhou</span>
          <span v-else-if="item.d.resultado" class="chip" :class="ESTADOS[item.d.resultado.estado]?.[1]">{{ ESTADOS[item.d.resultado.estado]?.[0] }}</span>
          <a v-if="item.d.momento && temAudio" class="meta" :href="urlNoTempo(referencia.url, item.d.momento)" target="_blank" rel="noopener">ouvir em {{ item.d.momento }}</a>
        </div>

        <p class="texto-da-duvida" v-html="linha(item.texto)" />

        <p v-if="item.d?.estado === 'falhou'" class="erro-texto">{{ item.d.erro }}</p>

        <div v-if="item.d?.resultado" class="apurado">
          <p v-if="item.d.resultado.no_audio" class="no-audio"><span class="rotulo-inline">No áudio</span> “{{ item.d.resultado.no_audio }}”</p>
          <p>{{ item.d.resultado.explicacao }}</p>
        </div>

        <!-- A correção: o que muda, depois preparando, depois o texto final para aprovar. -->
        <div v-if="paraAprovar(item.d)" class="correcao pronta">
          <p class="rotulo-inline">Como o texto fica</p>
          <template v-if="item.d.correcao.trechos?.length">
            <div v-for="(t, i) in item.d.correcao.trechos" :key="i" class="trecho">
              <div v-if="t.depois" class="texto-corrigido" v-html="renderizar(t.depois)" />
              <p v-else class="removido"><span class="rotulo-inline">Sai do texto</span> {{ t.antes }}</p>
            </div>
          </template>
          <p v-else class="suave">Só a dúvida sai da lista; o texto em si não muda.</p>
          <div class="linha-acoes">
            <button class="botao principal" :disabled="decidindo === item.d.id" @click="decidir(item, 'aprovar')">Aprovar</button>
            <button class="botao" :disabled="decidindo === item.d.id" @click="decidir(item, 'rejeitar')">Descartar</button>
            <a class="meta" :href="`#/revisao/${item.d.correcao.tarefa_id}`">ver a diferença</a>
          </div>
        </div>
        <div v-else-if="preparando(item.d)" class="correcao">
          <span class="chip andando">{{ item.d.correcao.estado === 'rodando' ? 'Preparando a correção…' : 'Correção na fila…' }}</span>
          <p class="suave">Onde está “{{ item.d.resultado?.trecho }}”, passa a ser “{{ item.d.resultado?.correto }}”. O texto corrigido aparece aqui para você aprovar.</p>
        </div>
        <div v-else-if="item.d?.resultado?.correto" class="correcao">
          <p><span class="rotulo-inline">Onde está</span> “{{ item.d.resultado.trecho }}”</p>
          <p><span class="rotulo-inline">Fica</span> “{{ item.d.resultado.correto }}”</p>
        </div>

        <div v-if="item.d?.estado !== 'processando'" class="acoes">
          <button
            v-if="item.d?.resultado?.correto && !item.d.correcao"
            class="botao"
            :disabled="decidindo === item.d.id"
            @click="corrigir(item)"
          >Corrigir no texto</button>
          <button v-if="['incerta', 'lacuna'].includes(item.d?.resultado?.estado)" class="botao" @click="pesquisar(item)">Pesquisar isto</button>
          <button
            v-if="!item.d?.correcao"
            class="botao discreto"
            :disabled="pedindo || processando"
            @click="resolver(item.texto)"
          >{{ item.d ? 'Resolver de novo' : 'Resolver' }}</button>
        </div>

        <div v-if="item.d?.resultado?.fontes.length || item.d?.audio" class="mais">
          <details v-if="item.d.resultado?.fontes.length">
            <summary>Fontes ({{ item.d.resultado.fontes.length }})</summary>
            <ol>
              <li v-for="f in item.d.resultado.fontes" :key="f.url"><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></li>
            </ol>
          </details>
          <details v-if="item.d.audio">
            <summary>O que o Whisper ouviu</summary>
            <pre class="ouvido">{{ item.d.audio }}</pre>
          </details>
        </div>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.duvidas {
  display: grid;
  gap: 12px;
}

.duvidas p {
  margin: 0;
}

.cabeca {
  display: grid;
  gap: 12px;
  padding-bottom: 4px;
}

.descricao {
  color: var(--texto-suave);
  font-size: 0.92rem;
  line-height: 1.5;
}

.linha-cabeca {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.linha-cabeca:empty {
  display: none;
}

.lista-duvidas {
  margin: 0;
  padding: 0;
  list-style: none;
  border-top: 1px solid var(--linha);
}

.duvida {
  display: grid;
  gap: 12px;
  padding: 22px 0;
  border-bottom: 1px solid var(--linha);
}

.selos {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.chip.meio {
  color: var(--texto);
  background: var(--superficie-forte);
}

.meta {
  font-size: 0.8rem;
  color: var(--texto-fraco);
}

a.meta:hover {
  color: var(--marca);
}

.texto-da-duvida {
  font-size: 1.02rem;
  font-weight: 500;
  line-height: 1.5;
}

.apurado {
  display: grid;
  gap: 6px;
  color: var(--texto-suave);
  font-size: 0.93rem;
  line-height: 1.55;
}

.no-audio {
  color: var(--texto);
}

.rotulo-inline {
  margin-right: 6px;
  color: var(--texto-suave);
  font-size: 0.8rem;
  font-weight: 600;
}

.correcao {
  display: grid;
  gap: 8px;
  justify-items: start;
  padding: 12px 14px;
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.93rem;
  line-height: 1.5;
}

.correcao.pronta {
  gap: 10px;
  border-left: 3px solid var(--marca);
}

.trecho {
  width: 100%;
}

.texto-corrigido {
  font-family: var(--leitura);
  font-size: 1rem;
  line-height: 1.65;
}

.texto-corrigido :deep(p) {
  margin: 0 0 8px;
}

.texto-corrigido :deep(p:last-child) {
  margin-bottom: 0;
}

.removido {
  color: var(--texto-suave);
  text-decoration: line-through;
}

.linha-acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-top: 2px;
}

.acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.acoes:empty {
  display: none;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
}

.erro-texto {
  color: var(--vermelho);
  font-size: 0.9rem;
}

.mais {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 18px;
}

.mais details {
  flex-basis: 100%;
}

.mais summary {
  display: inline;
  cursor: pointer;
  color: var(--texto-fraco);
  font-size: 0.82rem;
}

.mais summary:hover {
  color: var(--texto-suave);
}

.mais ol {
  margin: 8px 0 0;
  padding-left: 22px;
  font-size: 0.88rem;
  overflow-wrap: anywhere;
}

.ouvido {
  margin: 8px 0 0;
  padding: 10px 12px;
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.85rem;
  white-space: pre-wrap;
}
</style>
