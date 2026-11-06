<script setup>
import { computed, ref } from 'vue'
import Icone from './Icone.vue'
import { buscasDaPergunta } from './pesquisa.js'

// O esboço da pesquisa (D27): o Lucas ajusta as perguntas e as palavras de busca e aprova.
const props = defineProps({
  pesquisa: { type: Object, required: true },
  ocupado: { type: Boolean, default: false },
})
const emit = defineEmits(['aprovar'])

const MAX_PERGUNTAS = 8
const perguntas = ref(props.pesquisa.esboco.map((p) => ({ ...p })))
const buscas = computed(() => buscasDaPergunta(props.pesquisa))
const validas = computed(() => perguntas.value.filter((p) => p.pergunta.trim()))

function acrescentar() {
  perguntas.value.push({ pergunta: '', livros: '', wikipedia: '', youtube: '' })
}

function tirar(i) {
  perguntas.value.splice(i, 1)
}
</script>

<template>
  <section class="cartao esboco">
    <div>
      <strong>Esboço da pesquisa</strong>
      <p class="suave">A pesquisa parou aqui para você revisar. Ajuste as perguntas e as palavras que cada fonte vai buscar; uma busca vazia usa a própria pergunta.</p>
    </div>

    <ol class="perguntas">
      <li v-for="(p, i) in perguntas" :key="i">
        <div class="linha-pergunta">
          <input v-model="p.pergunta" class="campo" :aria-label="`Pergunta ${i + 1}`" placeholder="Pergunta" />
          <button type="button" class="botao icone" :aria-label="`Tirar a pergunta ${i + 1}`" title="Tirar a pergunta" @click="tirar(i)"><Icone nome="fechar" /></button>
        </div>
        <label v-for="f in buscas" :key="f.chave" class="busca">
          <span>{{ f.curto }}</span>
          <input v-model="p[f.chave]" class="campo" :aria-label="`Busca em ${f.curto} da pergunta ${i + 1}`" />
        </label>
      </li>
    </ol>

    <div class="linha-acoes">
      <button type="button" class="botao principal" :disabled="ocupado || !validas.length" @click="emit('aprovar', validas)">Pesquisar</button>
      <button v-if="perguntas.length < MAX_PERGUNTAS" type="button" class="botao" @click="acrescentar"><Icone nome="mais" /> Pergunta</button>
    </div>
  </section>
</template>

<style scoped>
.esboco {
  display: grid;
  gap: 14px;
  margin-top: 14px;
}

.esboco p {
  margin: 4px 0 0;
  font-size: 0.85rem;
  line-height: 1.45;
}

.perguntas {
  display: grid;
  gap: 14px;
  margin: 0;
  padding-left: 1.4em;
}

.perguntas li {
  display: grid;
  gap: 6px;
}

.linha-pergunta {
  display: flex;
  gap: 4px;
  align-items: center;
}

.busca {
  display: grid;
  grid-template-columns: 5.5rem 1fr;
  gap: 8px;
  align-items: center;
  color: var(--texto-suave);
  font-size: 0.8rem;
}

.busca .campo {
  min-height: 32px;
  padding: 4px 10px;
  font-size: 0.85rem;
}

.linha-acoes :deep(svg) {
  width: 16px;
  height: 16px;
}
</style>
