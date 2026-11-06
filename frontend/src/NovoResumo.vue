<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from './api.js'

// Resumo de capítulos (D30): o Lucas escolhe um livro da Biblioteca e um ou mais capítulos (ou as
// páginas, num PDF sem sumário). O texto vem da Biblioteca; a IA resume e sugere o Tema.
const emit = defineEmits(['fechar'])

const MAX_CARACTERES = 600000
const livros = ref([])
const livroId = ref('')
const sumario = ref(null)
const escolhidos = ref([])
const de = ref(1)
const ate = ref(1)
const foco = ref('')
const carregando = ref(false)
const enviando = ref(false)
const erro = ref('')

const porPaginas = computed(() => sumario.value?.indexed && !sumario.value.chapters.length)
const caracteres = computed(() => {
  const s = sumario.value
  if (!s) return 0
  if (porPaginas.value) return Math.max(0, ate.value - de.value + 1) * 1500
  return s.chapters.filter((c) => escolhidos.value.includes(c.n)).reduce((soma, c) => soma + c.chars, 0)
})
const grandeDemais = computed(() => caracteres.value > MAX_CARACTERES)
const podeEnviar = computed(() => {
  if (!sumario.value?.indexed || grandeDemais.value) return false
  if (porPaginas.value) return de.value >= 1 && ate.value >= de.value && ate.value <= sumario.value.pages
  return escolhidos.value.length > 0
})

// "≈ 12 páginas": a régua das páginas da Biblioteca, 1.500 caracteres.
function tamanho(chars) {
  const paginas = Math.max(1, Math.round(chars / 1500))
  return `≈ ${paginas} ${paginas === 1 ? 'página' : 'páginas'}`
}

async function carregarLivros() {
  try {
    livros.value = await api('/api/livros')
  } catch (e) {
    erro.value = e.message
  }
}

async function escolherLivro() {
  sumario.value = null
  escolhidos.value = []
  erro.value = ''
  if (!livroId.value) return
  carregando.value = true
  try {
    sumario.value = await api(`/api/livros/${encodeURIComponent(livroId.value)}`)
    de.value = 1
    ate.value = Math.min(sumario.value.pages || 1, 20)
  } catch (e) {
    erro.value = e.message
  } finally {
    carregando.value = false
  }
}

async function resumir() {
  enviando.value = true
  erro.value = ''
  try {
    const corpo = porPaginas.value
      ? { livro: livroId.value, de: de.value, ate: ate.value, foco: foco.value }
      : { livro: livroId.value, capitulos: escolhidos.value, foco: foco.value }
    const ref_ = await api('/api/livros', { metodo: 'POST', corpo })
    location.hash = `#/referencia/${ref_.id}`
  } catch (e) {
    erro.value = e.message
  } finally {
    enviando.value = false
  }
}

onMounted(carregarLivros)
</script>

<template>
  <form class="cartao novo-resumo" @submit.prevent="resumir">
    <label class="grupo">
      <span class="titulo-grupo">Livro</span>
      <select v-model="livroId" class="campo" @change="escolherLivro">
        <option value="">Escolha um livro da Biblioteca</option>
        <option v-for="l in livros" :key="l.id" :value="l.id">{{ l.title }}{{ l.author ? ` — ${l.author}` : '' }}</option>
      </select>
    </label>

    <p v-if="carregando" class="suave dica">Lendo o sumário…</p>
    <div v-else-if="sumario && !sumario.indexed" class="aviso">O texto deste livro ainda não está disponível: {{ sumario.error }}</div>

    <fieldset v-else-if="sumario && !porPaginas" class="grupo">
      <legend class="titulo-grupo">Capítulos</legend>
      <ul class="capitulos">
        <li v-for="c in sumario.chapters" :key="c.n" :style="{ paddingLeft: `${(c.level - 1) * 1.2}rem` }">
          <label class="marcar">
            <input v-model="escolhidos" type="checkbox" :value="c.n" />
            <span>{{ c.title }} <span class="suave">· p. {{ c.first_page }}{{ c.last_page > c.first_page ? `–${c.last_page}` : '' }} · {{ tamanho(c.chars) }}</span></span>
          </label>
        </li>
      </ul>
    </fieldset>

    <fieldset v-else-if="porPaginas" class="grupo">
      <legend class="titulo-grupo">Páginas</legend>
      <p class="suave dica">Este livro não tem sumário no arquivo: escolha as páginas do capítulo (o livro tem {{ sumario.pages }}).</p>
      <div class="paginas">
        <label>De <input v-model.number="de" class="campo" type="number" min="1" :max="sumario.pages" /></label>
        <label>até <input v-model.number="ate" class="campo" type="number" :min="de" :max="sumario.pages" /></label>
      </div>
    </fieldset>

    <template v-if="sumario?.indexed">
      <label class="grupo">
        <span class="titulo-grupo">O que mais te interessa nestes capítulos? <span class="suave">(opcional)</span></span>
        <textarea v-model="foco" class="campo" rows="2" maxlength="500" placeholder="Ex.: as técnicas de fermentação" />
      </label>
      <p class="suave dica">
        <template v-if="caracteres">Escolhido: {{ tamanho(caracteres) }}. </template>
        <span v-if="grandeDemais" class="erro-texto">É texto demais para um resumo só: escolha menos capítulos.</span>
        <template v-else>A IA resume o que o livro ensina, sem repetições nem enchimento, com as páginas de cada parte, e sugere o Tema.</template>
      </p>
    </template>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div class="linha-acoes">
      <button class="botao principal" :disabled="enviando || !podeEnviar">{{ enviando ? 'Enviando…' : 'Resumir' }}</button>
      <button type="button" class="botao" @click="emit('fechar')">Cancelar</button>
    </div>
  </form>
</template>

<style scoped>
.novo-resumo {
  display: grid;
  gap: 16px;
  margin-bottom: 16px;
}

.grupo {
  display: grid;
  gap: 6px;
  min-width: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.titulo-grupo {
  padding: 0;
  font-size: 0.85rem;
  font-weight: 600;
}

textarea.campo {
  resize: vertical;
  font: inherit;
  line-height: 1.45;
}

.dica {
  margin: 0;
  font-size: 0.8rem;
  line-height: 1.4;
}

.capitulos {
  display: grid;
  gap: 2px;
  max-height: 22rem;
  overflow-y: auto;
  margin: 0;
  padding: 0;
  list-style: none;
}

.marcar {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 4px 0;
  font-size: 0.875rem;
  line-height: 1.4;
}

.marcar input {
  flex: none;
  margin-top: 3px;
  accent-color: var(--azul);
}

.paginas {
  display: flex;
  gap: 12px;
  align-items: center;
  font-size: 0.875rem;
}

.paginas label {
  display: flex;
  gap: 6px;
  align-items: center;
}

.paginas .campo {
  width: 6rem;
}

.erro-texto {
  color: var(--perigo);
}
</style>
