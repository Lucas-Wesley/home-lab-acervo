<script setup>
import { computed, nextTick, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { confere, escolher, partesDoEnunciado } from './estudo.js'

// Exercícios (D52): prática com resposta certa. O app confere na hora e mostra o porquê; cada
// resposta vai ao servidor, para o próximo sorteio trazer primeiro o que nunca saiu e o que se errou.
const props = defineProps({
  tema: { type: String, required: true },
  banco: { type: Array, required: true }, // [{ id, tipo, enunciado, opcoes, resposta, aceitas, explicacao, assunto, capitulo }]
  historico: { type: Object, default: () => ({}) }, // id → { vezes, acertos, ultima, quando }
})
const emit = defineEmits(['respondido'])

const QUANTOS = [5, 10, 20]
const quantos = ref(10)
const assunto = ref('') // vazio = todos
const rodada = ref(null) // [{ ...exercicio, dada, certo }]
const atual = ref(0)
const escrita = ref('')
const campo = ref(null)

const assuntos = computed(() => {
  const contagem = {}
  for (const x of props.banco) if (x.assunto) contagem[x.assunto] = (contagem[x.assunto] ?? 0) + 1
  return Object.entries(contagem).sort((a, b) => b[1] - a[1])
})
const feitos = computed(() => props.banco.filter((x) => props.historico[x.id]?.vezes))
const acertadosDaUltima = computed(() => feitos.value.filter((x) => props.historico[x.id].ultima === 'certo').length)
const errados = computed(() => props.banco.filter((x) => props.historico[x.id]?.ultima === 'errado'))
const item = computed(() => rodada.value?.[atual.value])
const respondido = computed(() => item.value && item.value.certo !== undefined)
const fim = computed(() => rodada.value && atual.value >= rodada.value.length)
const acertos = computed(() => rodada.value?.filter((x) => x.certo).length ?? 0)

function comecar(lista = null) {
  const base = lista ?? props.banco.filter((x) => !assunto.value || x.assunto === assunto.value)
  rodada.value = (lista ? lista : escolher(base, quantos.value, props.historico)).map((x) => ({ ...x, dada: '', certo: undefined }))
  atual.value = 0
  escrita.value = ''
  focar()
}

function focar() {
  nextTick(() => campo.value?.focus())
}

function responder(dada) {
  const x = item.value
  if (respondido.value) return
  x.dada = dada
  x.certo = x.tipo === 'escolha' ? dada === x.resposta : confere(dada, x)
  // Guardar não pode atrapalhar o treino: se falhar, segue sem o histórico deste.
  api('/api/exercicios/respostas', { metodo: 'POST', corpo: { tema: props.tema, exercicio: x.id, resposta: dada, certo: x.certo } })
    .then(() => emit('respondido'))
    .catch(() => {})
}

function proximo() {
  atual.value++
  escrita.value = ''
  focar()
}

function classeDaOpcao(o) {
  if (!respondido.value) return ''
  if (o === item.value.resposta) return 'certa'
  if (o === item.value.dada) return 'errada'
  return 'apagada'
}
</script>

<template>
  <div class="exercicios">
    <!-- Treinando -->
    <template v-if="rodada && !fim">
      <div class="progresso">
        <span class="suave">Exercício {{ atual + 1 }} de {{ rodada.length }}</span>
        <span v-if="item.assunto" class="chip">{{ item.assunto }}</span>
      </div>
      <p class="enunciado">
        <template v-for="(p, i) in partesDoEnunciado(item.enunciado)" :key="i">
          <span v-if="p.lacuna" class="lacuna" :class="{ certa: respondido && item.certo, errada: respondido && !item.certo }">{{ respondido ? item.dada || '…' : '  ' }}</span>
          <template v-else>{{ p.texto }}</template>
        </template>
      </p>

      <div v-if="item.tipo === 'escolha'" class="opcoes">
        <button v-for="o in item.opcoes" :key="o" class="botao opcao" :class="classeDaOpcao(o)" :disabled="respondido" @click="responder(o)">{{ o }}</button>
      </div>
      <form v-else class="completar" @submit.prevent="responder(escrita)">
        <input ref="campo" v-model="escrita" class="campo" :disabled="respondido" autocomplete="off" autocapitalize="off" spellcheck="false" placeholder="O que vai na lacuna" />
        <button v-if="!respondido" class="botao principal" :disabled="!escrita.trim()">Conferir</button>
      </form>

      <div v-if="respondido" class="retorno" :class="item.certo ? 'ok' : 'erro'" role="status">
        <strong><Icone :nome="item.certo ? 'feito' : 'fechar'" /> {{ item.certo ? 'Certo' : 'Não' }}</strong>
        <p v-if="!item.certo">A resposta é <b>{{ item.resposta }}</b>.</p>
        <p>{{ item.explicacao }}</p>
        <p v-if="item.capitulo" class="suave capitulo">{{ item.capitulo }}</p>
      </div>

      <div class="linha-acoes">
        <button v-if="respondido" class="botao principal" @click="proximo">{{ atual < rodada.length - 1 ? 'Próximo' : 'Ver o resultado' }}</button>
        <button class="botao discreto" @click="rodada = null">Parar</button>
      </div>
    </template>

    <!-- Resultado -->
    <template v-else-if="rodada && fim">
      <div class="nota">
        <strong>{{ acertos }}</strong><span class="suave">de {{ rodada.length }}</span>
      </div>
      <template v-if="rodada.some((x) => !x.certo)">
        <h4>Para rever</h4>
        <article v-for="x in rodada.filter((x) => !x.certo)" :key="x.id" class="item">
          <p class="enunciado-item">{{ x.enunciado.replace(/_{3,}/, `[${x.resposta}]`) }}</p>
          <p class="suave">Você: {{ x.dada || '—' }}</p>
          <p>{{ x.explicacao }}</p>
        </article>
      </template>
      <div class="linha-acoes">
        <button class="botao principal" @click="comecar()">Praticar de novo</button>
        <button v-if="rodada.some((x) => !x.certo)" class="botao" @click="comecar(rodada.filter((x) => !x.certo))">Refazer os que errei</button>
        <button class="botao discreto" @click="rodada = null">Voltar</button>
      </div>
    </template>

    <!-- Começar -->
    <template v-else>
      <p class="suave">
        {{ banco.length }} exercícios no banco<template v-if="feitos.length">; {{ feitos.length }} já feitos, {{ acertadosDaUltima }} certos da última vez</template>.
        Cada rodada traz primeiro os que você nunca fez e os que errou.
      </p>
      <div class="linha-acoes">
        <label class="seletor">
          <span class="suave">Quantos</span>
          <select v-model="quantos" class="campo">
            <option v-for="n in QUANTOS" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
        <label v-if="assuntos.length > 1" class="seletor assunto">
          <span class="suave">Assunto</span>
          <select v-model="assunto" class="campo">
            <option value="">Todos</option>
            <option v-for="[a, n] in assuntos" :key="a" :value="a">{{ a }} ({{ n }})</option>
          </select>
        </label>
      </div>
      <div class="linha-acoes">
        <button class="botao principal" @click="comecar()">Praticar</button>
        <button v-if="errados.length" class="botao" @click="comecar(errados.slice(0, 20))">Os que errei ({{ errados.length }})</button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.exercicios {
  display: grid;
  gap: 14px;
}

.exercicios > p,
.retorno p,
.item p {
  margin: 0;
}

.progresso {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.enunciado {
  margin: 0;
  font-size: 1.2rem;
  line-height: 1.6;
  white-space: pre-wrap;
}

.lacuna {
  display: inline-block;
  min-width: 3em;
  padding: 0 6px;
  border-bottom: 2px solid var(--marca);
  font-weight: 650;
  text-align: center;
}

.lacuna.certa {
  border-color: var(--marca);
  color: var(--marca);
}

.lacuna.errada {
  border-color: var(--vermelho);
  color: var(--vermelho);
}

.opcoes {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.opcao {
  min-width: 72px;
  min-height: 48px;
  font-size: 1.05rem;
}

.opcao:disabled {
  opacity: 1;
}

.opcao.certa {
  border-color: var(--marca);
  color: var(--marca);
}

.opcao.errada {
  border-color: var(--vermelho);
  color: var(--vermelho);
  text-decoration: line-through;
}

.opcao.apagada {
  opacity: 0.5;
}

.completar {
  display: flex;
  gap: 8px;
}

.completar .campo {
  flex: 1;
  min-width: 0;
  font-size: 1.05rem;
}

.retorno {
  display: grid;
  gap: 6px;
  padding: 12px 14px;
  border-left: 3px solid var(--marca);
  border-radius: var(--raio-p);
  background: var(--superficie);
}

.retorno.erro {
  border-left-color: var(--vermelho);
}

.retorno strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.retorno.ok strong {
  color: var(--marca);
}

.retorno.erro strong {
  color: var(--vermelho);
}

.retorno :deep(svg) {
  width: 18px;
  height: 18px;
}

.capitulo {
  font-size: 0.85rem;
}

.linha-acoes {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 8px;
}

.seletor {
  display: grid;
  gap: 4px;
  font-size: 0.85rem;
}

.seletor.assunto {
  flex: 1 1 200px;
  min-width: 0;
}

.seletor.assunto select {
  width: 100%;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
}

.nota {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.nota strong {
  font-size: 2.4rem;
  line-height: 1;
}

h4 {
  margin: 4px 0 0;
}

.item {
  display: grid;
  gap: 4px;
  padding: 10px 0;
  border-top: 1px solid var(--linha);
}

.enunciado-item {
  font-weight: 600;
  white-space: pre-wrap;
}
</style>
