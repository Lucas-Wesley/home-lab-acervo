<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api, NOMES_ESTADO_TAREFA, NOMES_TAREFA } from './api.js'

const tarefas = ref([])
const carregou = ref(false)
const erro = ref('')

const ativas = computed(() => tarefas.value.filter((t) => t.estado === 'rodando' || t.estado === 'na_fila').reverse())
const terminadas = computed(() => tarefas.value.filter((t) => t.estado !== 'rodando' && t.estado !== 'na_fila'))

let relogio
async function carregar() {
  try {
    tarefas.value = await api('/api/tarefas')
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: a fila só abre na rede de casa.' : e.message
  }
  carregou.value = true
}

async function cancelar(t) {
  try {
    await api(`/api/tarefas/${t.id}/cancelar`, { metodo: 'POST' })
  } catch (e) {
    erro.value = e.message
  }
  carregar()
}

function tempo(t) {
  const fim = t.terminado_em ? new Date(t.terminado_em) : new Date()
  if (!t.iniciado_em) return ''
  const s = Math.max(0, Math.round((fim - new Date(t.iniciado_em)) / 1000))
  return s < 60 ? `${s} s` : `${Math.floor(s / 60)} min ${s % 60} s`
}

function classe(t) {
  return { erro: t.estado === 'falhou', ok: t.estado === 'concluida', andando: t.estado === 'rodando' || t.estado === 'em_revisao' }
}

onMounted(() => {
  carregar()
  relogio = setInterval(() => document.visibilityState === 'visible' && carregar(), 2000)
})
onBeforeUnmount(() => clearInterval(relogio))
</script>

<template>
  <div class="pagina">
    <h1>Fila</h1>
    <p class="suave explicacao">Uma tarefa por vez, para não sobrecarregar o servidor nem o limite do Claude.</p>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <h2>Agora</h2>
    <div v-if="carregou && !ativas.length" class="vazio curto">Nada na fila.</div>
    <div v-for="(t, i) in ativas" :key="t.id" class="tarefa">
      <div class="info">
        <strong>{{ NOMES_TAREFA[t.tipo] || t.tipo }}</strong>
        <a :href="`#/referencia/${t.referencia_id}`">{{ t.referencia || t.referencia_id }}</a>
        <span class="linha-acoes">
          <span class="chip" :class="classe(t)">{{ t.estado === 'na_fila' ? `${i}ª na fila` : 'Rodando' }}</span>
          <span v-if="t.estado === 'rodando'" class="suave">{{ tempo(t) }}</span>
        </span>
      </div>
      <button class="botao perigo" @click="cancelar(t)">Cancelar</button>
    </div>

    <h2>Recentes</h2>
    <div v-for="t in terminadas" :key="t.id" class="tarefa">
      <div class="info">
        <strong>{{ NOMES_TAREFA[t.tipo] || t.tipo }}</strong>
        <a :href="`#/referencia/${t.referencia_id}`">{{ t.referencia || t.referencia_id }}</a>
        <span class="linha-acoes">
          <span class="chip" :class="classe(t)">{{ NOMES_ESTADO_TAREFA[t.estado] || t.estado }}</span>
          <span class="suave">{{ new Date(t.criado_em).toLocaleString('pt-BR') }}<template v-if="tempo(t)"> · {{ tempo(t) }}</template></span>
          <a v-if="t.estado === 'em_revisao'" :href="`#/revisao/${t.id}`">Revisar</a>
        </span>
        <span v-if="t.erro" :class="t.estado === 'falhou' ? 'erro-texto' : 'suave'">{{ t.erro }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.explicacao {
  margin-top: -8px;
}

h2 {
  margin: 28px 0 4px;
  font-size: 0.72rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--texto-fraco);
}

.curto {
  padding: 16px;
}

.tarefa {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--linha);
}

.info strong {
  font-size: 0.92rem;
  font-weight: 600;
}

.info {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.info a {
  color: var(--texto-suave);
  font-size: 0.85rem;
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.erro-texto {
  color: var(--perigo);
  font-size: 0.88rem;
}
</style>
