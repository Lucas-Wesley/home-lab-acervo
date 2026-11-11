<script setup>
import { computed, ref } from 'vue'
import Icone from './Icone.vue'
import { buscasDaPergunta } from './pesquisa.js'

// O esboço da pesquisa (D27): o Lucas ajusta as perguntas e as palavras de busca e aprova. D49: os
// capítulos dos livros e as páginas da internet que o esboço escolheu aparecem também, e dá para tirar.
const props = defineProps({
  pesquisa: { type: Object, required: true },
  ocupado: { type: Boolean, default: false },
})
const emit = defineEmits(['aprovar'])

const MAX_PERGUNTAS = 8
const perguntas = ref(props.pesquisa.esboco.map((p) => ({ ...p })))
const buscas = computed(() => buscasDaPergunta(props.pesquisa))
const validas = computed(() => perguntas.value.filter((p) => p.pergunta.trim()))
const capitulos = ref([...(props.pesquisa.capitulos ?? [])])
const paginas = ref([...(props.pesquisa.paginas ?? [])])

function site(url) {
  try {
    return new URL(url).hostname.replace(/^www\./, '')
  } catch {
    return url
  }
}

function aprovar() {
  emit('aprovar', { esboco: validas.value, capitulos: capitulos.value, paginas: paginas.value })
}

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

    <div v-if="capitulos.length" class="escolhidos">
      <strong class="titulo-lista">Capítulos que serão lidos inteiros</strong>
      <ul>
        <li v-for="(c, i) in capitulos" :key="`${c.livro}-${c.de}`">
          <span class="info"><span>“{{ c.capitulo }}”</span><span class="suave">{{ c.titulo }}{{ c.autor ? `, de ${c.autor}` : '' }} · p. {{ c.de }}–{{ c.ate }}</span></span>
          <button type="button" class="botao icone" :aria-label="`Tirar o capítulo ${c.capitulo}`" title="Tirar" @click="capitulos.splice(i, 1)"><Icone nome="fechar" /></button>
        </li>
      </ul>
    </div>

    <div v-if="paginas.length" class="escolhidos">
      <strong class="titulo-lista">Páginas da internet</strong>
      <ul>
        <li v-for="(p, i) in paginas" :key="p.url">
          <span class="info"><a :href="p.url" target="_blank" rel="noopener">{{ p.titulo || p.url }}</a><span class="suave">{{ site(p.url) }}</span></span>
          <button type="button" class="botao icone" :aria-label="`Tirar a página ${p.titulo || p.url}`" title="Tirar" @click="paginas.splice(i, 1)"><Icone nome="fechar" /></button>
        </li>
      </ul>
    </div>

    <div class="linha-acoes">
      <button type="button" class="botao principal" :disabled="ocupado || !validas.length" @click="aprovar">Pesquisar</button>
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

.escolhidos {
  display: grid;
  gap: 6px;
}

.titulo-lista {
  font-size: 0.85rem;
}

.escolhidos ul {
  display: grid;
  gap: 4px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.escolhidos li {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.escolhidos .info {
  display: grid;
  min-width: 0;
  font-size: 0.85rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.escolhidos .suave {
  font-size: 0.78rem;
}

.linha-acoes :deep(svg) {
  width: 16px;
  height: 16px;
}
</style>
