<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { comparavel, escolher } from './estudo.js'
import { juntar, ouvir, temVoz } from './voz.js'

// Simulado (D41): sorteia perguntas do banco do Tema, uma por vez; o Lucas responde digitando ou
// falando (o ditado do navegador), e a IA corrige na fila. Fica o histórico, com a nota de cada um.
const props = defineProps({
  tema: { type: String, required: true },
  banco: { type: Array, required: true }, // [{ pergunta, esperada, capitulo, nivel }]
  simulados: { type: Array, required: true }, // os últimos, o mais novo primeiro
  historico: { type: Object, default: () => ({}) }, // pergunta (comparável) → { vezes, ultima, quando } (D52)
  flashcards: { type: Boolean, default: false },
})
const emit = defineEmits(['entregue', 'flashcards'])

const AVALIACAO = {
  certa: ['Certa', 'ok'],
  parcial: ['Em parte', 'meio'],
  errada: ['Errada', 'erro'],
  em_branco: ['Em branco', 'erro'],
}

const QUANTAS = [3, 5, 10]
const quantas = ref(Math.min(5, props.banco.length))
const rodada = ref(null) // [{ pergunta, esperada, capitulo, resposta, por_voz }]
const atual = ref(0)
const aberto = ref(null) // o simulado corrigido (ou corrigindo) na tela
const erro = ref('')
const entregando = ref(false)
const ouvindo = ref(false)
let parar = null
let espera = null

const pergunta = computed(() => rodada.value?.[atual.value])
const respondidas = computed(() => rodada.value?.filter((x) => x.resposta.trim()).length ?? 0)

// O que o banco já deu: perguntas nunca sorteadas e as que se errou da última vez (D52).
const nuncaFeitas = computed(() => props.banco.filter((p) => !props.historico[comparavel(p.pergunta)]?.vezes).length)
const erradas = computed(() => props.banco.filter((p) => ['errada', 'parcial', 'em_branco'].includes(props.historico[comparavel(p.pergunta)]?.ultima)).length)

function comecar() {
  pararDeOuvir()
  aberto.value = null
  erro.value = ''
  rodada.value = escolher(props.banco, quantas.value, props.historico, (p) => comparavel(p.pergunta)).map((p) => ({ ...p, resposta: '', por_voz: false }))
  atual.value = 0
}

function ir(i) {
  pararDeOuvir()
  atual.value = i
}

function alternarVoz() {
  if (ouvindo.value) return pararDeOuvir()
  erro.value = ''
  const item = pergunta.value
  const antes = item.resposta
  try {
    parar = ouvir({
      aoMudar: (dito) => {
        item.resposta = juntar(antes, dito)
        item.por_voz = true
      },
      aoTerminar: (msg) => {
        ouvindo.value = false
        parar = null
        if (msg) erro.value = msg
      },
    })
    ouvindo.value = true
  } catch {
    erro.value = 'Não consegui ligar o ditado neste navegador.'
  }
}

function pararDeOuvir() {
  parar?.()
  parar = null
  ouvindo.value = false
}

async function entregar() {
  pararDeOuvir()
  entregando.value = true
  erro.value = ''
  try {
    const respostas = rodada.value.map(({ pergunta, esperada, capitulo, resposta, por_voz }) => ({ pergunta, esperada, capitulo, resposta, por_voz }))
    aberto.value = await api('/api/simulados', { metodo: 'POST', corpo: { tema: props.tema, respostas } })
    rodada.value = null
    emit('entregue')
    acompanhar()
  } catch (e) {
    erro.value = e.status === 0 ? 'O simulado só é corrigido na rede de casa.' : e.message
  } finally {
    entregando.value = false
  }
}

async function acompanhar() {
  clearTimeout(espera)
  if (aberto.value?.estado !== 'processando') return
  espera = setTimeout(async () => {
    try {
      aberto.value = await api(`/api/simulados/${aberto.value.id}`)
      if (aberto.value.estado !== 'processando') emit('entregue')
    } catch {
      // tenta de novo na próxima volta
    }
    acompanhar()
  }, 4000)
}

function abrir(s) {
  rodada.value = null
  aberto.value = s
  acompanhar()
}

function quando(iso) {
  return new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

onBeforeUnmount(() => {
  pararDeOuvir()
  clearTimeout(espera)
})
</script>

<template>
  <div class="simulado">
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <!-- Respondendo -->
    <template v-if="rodada">
      <div class="progresso">
        <span class="suave">Pergunta {{ atual + 1 }} de {{ rodada.length }}</span>
        <div class="pontos">
          <button v-for="(x, i) in rodada" :key="i" :class="{ atual: i === atual, feita: x.resposta.trim() }" :aria-label="`Pergunta ${i + 1}`" @click="ir(i)" />
        </div>
      </div>
      <p class="enunciado">{{ pergunta.pergunta }}</p>
      <div class="resposta">
        <textarea v-model="pergunta.resposta" class="campo" rows="5" placeholder="Responda com as suas palavras…" />
        <button v-if="temVoz" class="botao microfone" :class="{ ouvindo }" :aria-pressed="ouvindo" @click="alternarVoz">
          <Icone :nome="ouvindo ? 'parar' : 'microfone'" /> {{ ouvindo ? 'Parar' : 'Falar a resposta' }}
        </button>
        <p v-if="ouvindo" class="suave dica">Ouvindo… fale com calma; o texto aparece enquanto você fala.</p>
      </div>
      <div class="linha-acoes">
        <button v-if="atual > 0" class="botao" @click="ir(atual - 1)">Anterior</button>
        <button v-if="atual < rodada.length - 1" class="botao principal" @click="ir(atual + 1)">Próxima</button>
        <button v-else class="botao principal" :disabled="entregando || !respondidas" @click="entregar">
          {{ entregando ? 'Entregando…' : `Entregar (${respondidas} de ${rodada.length})` }}
        </button>
        <button class="botao discreto" @click="pararDeOuvir(); rodada = null">Desistir</button>
      </div>
    </template>

    <!-- A correção -->
    <template v-else-if="aberto">
      <p v-if="aberto.estado === 'processando'" class="chip andando">Corrigindo as suas respostas…</p>
      <p v-else-if="aberto.estado === 'falhou'" class="aviso erro">{{ aberto.erro }}</p>
      <template v-else-if="aberto.correcao">
        <div class="nota">
          <strong>{{ aberto.correcao.nota }}</strong><span class="suave">de 100</span>
        </div>
        <p>{{ aberto.correcao.comentario }}</p>
        <p v-if="aberto.correcao.revisar.length" class="revisar"><span class="suave">Para reler:</span> {{ aberto.correcao.revisar.join(' · ') }}</p>
        <article v-for="(x, i) in aberto.respostas" :key="i" class="item">
          <div class="selos">
            <span class="suave">{{ i + 1 }}.</span>
            <span v-if="aberto.correcao.itens.find((it) => it.n === i + 1)" class="chip" :class="AVALIACAO[aberto.correcao.itens.find((it) => it.n === i + 1).avaliacao]?.[1]">
              {{ AVALIACAO[aberto.correcao.itens.find((it) => it.n === i + 1).avaliacao]?.[0] }}
            </span>
            <span v-if="x.por_voz" class="suave meta">falada</span>
          </div>
          <p class="enunciado-item">{{ x.pergunta }}</p>
          <p class="sua"><span class="suave">Você:</span> {{ x.resposta || '—' }}</p>
          <template v-for="it in aberto.correcao.itens.filter((it) => it.n === i + 1)" :key="it.n">
            <p>{{ it.comentario }}</p>
            <p v-if="it.faltou" class="faltou"><span class="suave">Faltou:</span> {{ it.faltou }}</p>
          </template>
          <details>
            <summary class="suave">Resposta esperada</summary>
            <p>{{ x.esperada }}</p>
          </details>
        </article>
        <div class="linha-acoes">
          <button class="botao principal" @click="comecar">Novo simulado</button>
          <button
            v-if="flashcards && aberto.correcao.itens.some((it) => it.avaliacao !== 'certa')"
            class="botao"
            @click="emit('flashcards', { origem: { estudo: tema, parte: `simulado:${aberto.id}` }, titulo: 'do que você errou' })"
          >
            <Icone nome="cartoes" /> Flashcards do que errei
          </button>
        </div>
      </template>
      <button class="botao discreto voltar" @click="aberto = null">Voltar</button>
    </template>

    <!-- Começar -->
    <template v-else>
      <p class="suave">
        {{ banco.length }} perguntas no banco<template v-if="nuncaFeitas < banco.length">: {{ nuncaFeitas }} nunca saíram<template v-if="erradas">, {{ erradas }} para acertar</template></template>.
        Cada simulado traz primeiro as que nunca saíram e as que você errou. Responda escrevendo ou falando{{ temVoz ? '' : ' (pelo ditado do teclado)' }}: a IA corrige e diz o que faltou.
      </p>
      <div class="linha-acoes">
        <label class="quantas">
          <span class="suave">Perguntas</span>
          <select v-model="quantas" class="campo">
            <option v-for="n in QUANTAS.filter((n) => n <= banco.length)" :key="n" :value="n">{{ n }}</option>
            <option v-if="!QUANTAS.includes(banco.length)" :value="banco.length">{{ banco.length }} (todas)</option>
          </select>
        </label>
        <button class="botao principal" @click="comecar">Começar o simulado</button>
      </div>
      <div v-if="simulados.length" class="historico">
        <h4>Simulados feitos</h4>
        <ul>
          <li v-for="s in simulados" :key="s.id">
            <button class="linha" @click="abrir(s)">
              <span>{{ quando(s.criado_em) }} · {{ s.respostas.length }} perguntas</span>
              <span v-if="s.estado === 'pronto'" class="nota-pequena">{{ s.correcao.nota }}</span>
              <span v-else class="suave">{{ s.estado === 'processando' ? 'corrigindo…' : 'falhou' }}</span>
            </button>
          </li>
        </ul>
      </div>
    </template>
  </div>
</template>

<style scoped>
.simulado {
  display: grid;
  gap: 12px;
}

.simulado p {
  margin: 0;
}

.progresso {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.88rem;
}

.pontos {
  display: flex;
  gap: 6px;
}

.pontos button {
  width: 10px;
  height: 10px;
  padding: 0;
  border: 1.5px solid var(--linha-forte);
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
}

.pontos button.feita {
  border-color: var(--marca);
  background: var(--marca-fundo);
}

.pontos button.atual {
  border-color: var(--marca);
  background: var(--marca);
}

.enunciado {
  font-size: 1.0625rem;
  font-weight: 650;
  line-height: 1.45;
}

.resposta {
  display: grid;
  gap: 8px;
}

.resposta textarea {
  width: 100%;
  min-height: 120px;
  resize: vertical;
  font: inherit;
  line-height: 1.5;
}

.microfone {
  justify-self: start;
}

.microfone.ouvindo {
  border-color: var(--vermelho);
  background: var(--vermelho-fundo);
  color: var(--vermelho);
}

.dica {
  font-size: 0.85rem;
}

.linha-acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
}

.voltar {
  justify-self: start;
}

.quantas {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.9rem;
}

.quantas select {
  width: auto;
}

.nota {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.nota strong {
  font-size: 2.25rem;
  font-weight: 700;
  letter-spacing: -0.02em;
}

.revisar {
  font-size: 0.92rem;
}

.item {
  display: grid;
  gap: 6px;
  padding: 14px 0;
  border-top: 1px solid var(--linha);
  font-size: 0.95rem;
}

.selos {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chip.meio {
  color: var(--texto);
  background: var(--superficie-forte);
}

.meta {
  font-size: 0.75rem;
}

.enunciado-item {
  font-weight: 650;
}

.sua,
.faltou {
  padding: 8px 10px;
  border-radius: var(--raio-p);
  background: var(--superficie);
}

details summary {
  cursor: pointer;
  font-size: 0.88rem;
}

details p {
  margin-top: 6px !important;
}

.historico h4 {
  margin: 8px 0 4px;
  font-size: 1.0625rem;
  font-weight: 650;
}

.historico ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

.historico .linha {
  display: flex;
  width: 100%;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border: 0;
  border-top: 1px solid var(--linha);
  background: none;
  color: var(--texto);
  font: inherit;
  font-size: 0.92rem;
  text-align: left;
  cursor: pointer;
}

.nota-pequena {
  font-weight: 650;
}
</style>
