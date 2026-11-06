<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { renderizar } from './markdown.js'

const props = defineProps({ arquivo: { type: String, required: true } })

const instrucao = ref(null)
const texto = ref('')
const editando = ref(false)
const salvando = ref(false)
const erro = ref('')
const salvo = ref(false)

const html = computed(() => (instrucao.value ? renderizar(instrucao.value.conteudo) : ''))
const mudou = computed(() => instrucao.value && texto.value !== instrucao.value.conteudo)

async function carregar() {
  try {
    instrucao.value = await api(`/api/instrucoes/${encodeURIComponent(props.arquivo)}`)
    texto.value = instrucao.value.conteudo
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: as instruções só abrem na rede de casa.' : e.message
  }
}

async function guardar(metodo, corpo) {
  salvando.value = true
  erro.value = ''
  try {
    instrucao.value = await api(`/api/instrucoes/${encodeURIComponent(props.arquivo)}`, { metodo, corpo })
    texto.value = instrucao.value.conteudo
    editando.value = false
    salvo.value = true
    setTimeout(() => (salvo.value = false), 3000)
  } catch (e) {
    erro.value = e.message
  } finally {
    salvando.value = false
  }
}

function salvar() {
  guardar('PUT', { conteudo: texto.value })
}

function restaurar() {
  if (confirm('Descartar a sua versão e voltar ao texto original desta instrução?')) guardar('DELETE')
}

function cancelar() {
  if (mudou.value && !confirm('Descartar as mudanças?')) return
  texto.value = instrucao.value.conteudo
  editando.value = false
}

const data = (iso) => new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })

onMounted(carregar)
</script>

<template>
  <div class="pagina">
    <a href="#/ajustes" class="voltar"><Icone nome="voltar" /> Ajustes</a>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <template v-if="instrucao">
      <h1>{{ instrucao.nome }}</h1>
      <p class="suave descricao">{{ instrucao.descricao }}</p>
      <p class="estado">
        <span v-if="instrucao.editada" class="chip">Editada em {{ data(instrucao.editada_em) }}</span>
        <span v-else class="chip">Texto original</span>
        <span v-if="salvo" class="chip ok">Salvo</span>
      </p>

      <template v-if="editando">
        <textarea v-model="texto" class="editor" spellcheck="true" aria-label="Texto da instrução" />
        <div class="acoes">
          <button class="botao principal" :disabled="salvando || !mudou || !texto.trim()" @click="salvar">{{ salvando ? 'Salvando…' : 'Salvar' }}</button>
          <button class="botao" :disabled="salvando" @click="cancelar">Cancelar</button>
        </div>
      </template>
      <template v-else>
        <div class="acoes">
          <button class="botao principal" @click="editando = true"><Icone nome="nota" /> Editar</button>
          <button v-if="instrucao.editada" class="botao perigo" :disabled="salvando" @click="restaurar">Voltar ao padrão</button>
        </div>
        <article class="texto instrucao" v-html="html" />
      </template>
    </template>
  </div>
</template>

<style scoped>
.voltar {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin: 0 0 16px -6px;
  color: var(--texto-suave);
  font-size: 0.88rem;
  text-decoration: none;
}

.voltar:hover {
  color: var(--texto);
}

.voltar :deep(svg) {
  width: 18px;
  height: 18px;
}

.pagina h1 {
  margin-bottom: 6px;
}

.descricao {
  margin: 0 0 10px;
  line-height: 1.5;
}

.estado {
  display: flex;
  gap: 6px;
  margin: 0 0 18px;
}

.aviso {
  margin-bottom: 14px;
}

.acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
}

.acoes :deep(svg) {
  width: 16px;
  height: 16px;
}

.editor {
  width: 100%;
  min-height: 65vh;
  margin-bottom: 12px;
  padding: 14px;
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.92rem;
  line-height: 1.6;
  resize: vertical;
}

.editor:focus {
  border-color: var(--azul);
  outline: none;
}

/* A instrução é um documento de trabalho: um pouco menor que o texto do Acervo. */
.instrucao {
  font-size: 1rem;
  padding-top: 18px;
  border-top: 1px solid var(--linha);
}

.instrucao :deep(h1) {
  font-size: 1.4rem;
}

.instrucao :deep(h2) {
  font-size: 1.15rem;
  margin-top: 1.6em;
}

.instrucao :deep(pre) {
  overflow-x: auto;
  padding: 12px;
  border-radius: var(--raio-p);
  background: var(--superficie);
  font-size: 0.8rem;
  line-height: 1.5;
}

.instrucao :deep(pre code) {
  padding: 0;
  background: none;
}
</style>
