<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { api } from './api.js'

// Para qual baralho do Flashcards vão os cartões (D28): um que já existe (os de mesma pergunta são
// atualizados) ou um novo. Começa no baralho com o nome do Tema, se houver; senão, num novo com esse nome.
const props = defineProps({
  origem: { type: Object, default: null }, // { arquivo, capitulo } ou { referencia }: dá o nome sugerido
  sugestao: { type: String, default: '' }, // sem origem (uma pesquisa nova): o nome sugerido
})
const escolha = defineModel({ type: Object, default: () => ({}) }) // { baralho_id } ou { baralho }

const baralhos = ref([])
const carregando = ref(true)
const erro = ref('')
const destino = ref('') // id do baralho, ou '' para um novo
const nome = ref(props.sugestao)

const novo = computed(() => destino.value === '')

watch([destino, nome], () => {
  escolha.value = novo.value ? { baralho: nome.value.trim() } : { baralho_id: destino.value }
}, { immediate: true })

onMounted(async () => {
  try {
    const params = new URLSearchParams(Object.entries(props.origem ?? {}).filter(([, v]) => v))
    const r = await api(`/api/flashcards/baralhos?${params}`)
    baralhos.value = r.baralhos
    const sugestao = r.sugestao || props.sugestao
    nome.value = sugestao
    destino.value = r.baralhos.find((b) => b.nome.trim().toLowerCase() === sugestao.trim().toLowerCase())?.id ?? ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão com o Flashcards.' : e.message
  } finally {
    carregando.value = false
  }
})
</script>

<template>
  <div class="escolha-baralho">
    <label class="grupo">
      <span class="suave">Baralho</span>
      <select v-model="destino" class="campo" :disabled="carregando" aria-label="Baralho">
        <option value="">Baralho novo…</option>
        <option v-for="b in baralhos" :key="b.id" :value="b.id">{{ b.nome }}</option>
      </select>
    </label>
    <label v-if="novo" class="grupo">
      <span class="suave">Nome do baralho novo</span>
      <input v-model="nome" class="campo" maxlength="120" :placeholder="sugestao || 'O título da pesquisa'" aria-label="Nome do baralho novo" />
    </label>
    <p v-else class="suave dica">Os cartões entram nesse baralho; os de mesma pergunta atualizam os que já existem, sem perder o histórico.</p>
    <p v-if="erro" class="erro-texto">{{ erro }}</p>
  </div>
</template>

<style scoped>
.escolha-baralho,
.grupo {
  display: grid;
  gap: 6px;
}

.escolha-baralho {
  gap: 10px;
}

.dica {
  margin: 0;
  font-size: 0.85rem;
}
</style>
