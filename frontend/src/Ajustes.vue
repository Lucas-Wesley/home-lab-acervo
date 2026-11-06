<script setup>
import { onMounted, ref } from 'vue'
import Sugestoes from './Sugestoes.vue'
import { api } from './api.js'

defineProps({ tela: { type: String, default: '' } })

const instrucoes = ref([])
const erro = ref('')

onMounted(async () => {
  try {
    instrucoes.value = await api('/api/instrucoes')
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: as instruções só abrem na rede de casa.' : e.message
  }
})
</script>

<template>
  <div class="pagina">
    <h1>Ajustes</h1>

    <p class="rotulo">Instruções da IA</p>
    <p class="suave explicacao">O que o Claude segue em cada etapa. Toque para ler ou editar: a mudança vale a partir da próxima tarefa da fila.</p>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div v-if="instrucoes.length" class="lista">
      <a v-for="i in instrucoes" :key="i.arquivo" :href="`#/ajustes/instrucoes/${i.arquivo}`" class="instrucao">
        <span class="nome">{{ i.nome }} <span v-if="i.editada" class="chip">Editada</span></span>
        <span class="descricao">{{ i.descricao }}</span>
      </a>
    </div>

    <p class="rotulo">Sugestões de melhoria</p>
    <p class="suave explicacao">Anote o que pode melhorar no Acervo. Depois é só pedir ao Claude para ver as sugestões: ele faz as mudanças e responde aqui.</p>
    <Sugestoes :tela="tela" />
  </div>
</template>

<style scoped>
.rotulo:first-of-type {
  margin-top: 8px;
}

.explicacao {
  margin: 0 0 14px;
  font-size: 0.88rem;
  line-height: 1.5;
}

.aviso {
  margin-bottom: 14px;
}

.instrucao {
  display: grid;
  gap: 3px;
  padding: 14px 0;
  color: inherit;
  text-decoration: none;
}

.nome {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  transition: color 0.15s ease;
}

.instrucao:hover .nome {
  color: var(--azul);
}

.descricao {
  color: var(--texto-suave);
  font-size: 0.88rem;
  line-height: 1.45;
}
</style>
