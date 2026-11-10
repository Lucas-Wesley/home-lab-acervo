<script setup>
import { computed, ref, watch } from 'vue'
import Icone from './Icone.vue'
import Sugestoes from './Sugestoes.vue'
import { api } from './api.js'

// Ajustes em níveis (design system): a primeira tela lista as seções; cada uma abre numa página.
const props = defineProps({
  tela: { type: String, default: '' },
  secao: { type: String, default: '' },
})

const SECOES = [
  { id: 'instrucoes', nome: 'Instruções da IA', descricao: 'O que o Claude segue em cada etapa: organizar, traduzir, integrar, pesquisar, perguntar.' },
  { id: 'livros', nome: 'Livros marcados', descricao: 'Os livros da Biblioteca lidos por assunto, aos poucos, para aparecerem nos Temas.' },
  { id: 'sugestoes', nome: 'Sugestões de melhoria', descricao: 'Anote o que pode melhorar no Acervo; o Claude faz e responde aqui.' },
]
const atual = computed(() => SECOES.find((s) => s.id === props.secao))

const instrucoes = ref([])
const livros = ref(null)
const erro = ref('')
const ESTADO_DO_LIVRO = { marcando: 'Marcando', pronto: 'Pronto', falhou: 'Não deu' }

watch(
  () => props.secao,
  async (secao) => {
    if (secao !== 'livros') return
    try {
      livros.value = await api('/api/livros-marcados')
    } catch (e) {
      erro.value = e.status === 0 ? 'Sem conexão: só abre na rede de casa.' : e.message
    }
  },
  { immediate: true },
)

watch(
  () => props.secao,
  async (secao) => {
    if (secao !== 'instrucoes' || instrucoes.value.length) return
    try {
      instrucoes.value = await api('/api/instrucoes')
    } catch (e) {
      erro.value = e.status === 0 ? 'Sem conexão: as instruções só abrem na rede de casa.' : e.message
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="pagina">
    <template v-if="!atual">
      <h1>Ajustes</h1>
      <nav class="lista-de-ajustes" aria-label="Seções dos ajustes">
        <a v-for="s in SECOES" :key="s.id" :href="`#/ajustes/${s.id}`">
          <span>
            <span class="nome">{{ s.nome }}</span>
            <span class="descricao">{{ s.descricao }}</span>
          </span>
          <Icone nome="avancar" class="seta" />
        </a>
      </nav>
    </template>

    <template v-else>
      <a href="#/ajustes" class="voltar-ajustes"><Icone nome="voltar" /> Ajustes</a>
      <h1>{{ atual.nome }}</h1>

      <template v-if="atual.id === 'instrucoes'">
        <p class="descricao-da-pagina">O que o Claude segue em cada etapa. Toque numa para ler ou editar: a mudança vale a partir da próxima tarefa da fila.</p>
        <div v-if="erro" class="aviso erro">{{ erro }}</div>
        <nav v-if="instrucoes.length" class="lista-de-ajustes" aria-label="Instruções">
          <a v-for="i in instrucoes" :key="i.arquivo" :href="`#/ajustes/instrucoes/${i.arquivo}`">
            <span>
              <span class="nome">{{ i.nome }} <span v-if="i.editada" class="chip">Editada</span></span>
              <span class="descricao">{{ i.descricao }}</span>
            </span>
            <Icone nome="avancar" class="seta" />
          </a>
        </nav>
      </template>

      <template v-else-if="atual.id === 'livros'">
        <p class="descricao-da-pagina">
          Quando a fila está parada, o Acervo pega um livro da Biblioteca e um modelo menor lê os capítulos, marcando o assunto de cada um com as páginas.
          Cada Tema mostra os capítulos que falam dele, e o Perguntar usa como pista de fonte. Os livros novos entram sozinhos.
        </p>
        <div v-if="erro" class="aviso erro">{{ erro }}</div>
        <p v-if="livros && !livros.length" class="vazio">Nenhum livro marcado ainda. O primeiro entra na próxima vez que a fila ficar parada.</p>
        <ul v-if="livros?.length" class="lista-livros">
          <li v-for="l in livros" :key="l.livro">
            <span class="nome">{{ l.titulo }}<span v-if="l.autor" class="suave"> · {{ l.autor }}</span></span>
            <span class="suave">
              <span class="chip" :class="{ ok: l.estado === 'pronto', erro: l.estado === 'falhou' }">{{ ESTADO_DO_LIVRO[l.estado] || l.estado }}</span>
              {{ l.marcados }} {{ l.marcados === 1 ? 'capítulo' : 'capítulos' }}<template v-if="l.faltam"> · faltam {{ l.faltam }} {{ l.faltam === 1 ? 'lote' : 'lotes' }}</template>
              <template v-if="l.erro"> · {{ l.erro }}</template>
            </span>
          </li>
        </ul>
      </template>

      <template v-else-if="atual.id === 'sugestoes'">
        <p class="descricao-da-pagina">Anote o que pode melhorar no Acervo. Depois é só pedir ao Claude para ver as sugestões: ele faz as mudanças e responde aqui.</p>
        <Sugestoes :tela="tela" />
      </template>
    </template>
  </div>
</template>

<style scoped>
.aviso {
  margin-bottom: 14px;
}

.lista-livros {
  display: grid;
  margin: 0;
  padding: 0;
  list-style: none;
  border-top: 1px solid var(--linha);
}

.lista-livros li {
  display: grid;
  gap: 4px;
  padding: 14px 0;
  border-bottom: 1px solid var(--linha);
}

.lista-livros .nome {
  font-size: 1.0625rem;
  font-weight: 650;
}

.lista-livros .chip {
  margin-right: 6px;
}
</style>
