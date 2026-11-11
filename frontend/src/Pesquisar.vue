<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import NovaPesquisa from './NovaPesquisa.vue'
import { api } from './api.js'
import { pedidoDoEndereco } from './pesquisa.js'

// Pesquisar um tema (D27) numa página própria (D34), como Resumir capítulos: mais espaço para as
// fontes, a profundidade e as opções, cada uma com a explicação. D49: pode chegar com o tema já
// escrito (de um conceito do Estudar ou da Conversa) e com o Tema que a pesquisa vai completar.
const pedido = pedidoDoEndereco(location.hash)
const temas = ref([])
const nomeDoTema = computed(() => temas.value.find((t) => t.caminho === pedido.para)?.titulo ?? pedido.para)
const voltar = computed(() => (pedido.para ? `#/ler/${pedido.para}/indice.md` : '#/referencias'))

onMounted(async () => {
  if (!pedido.para) return
  try {
    temas.value = await api('/api/temas')
  } catch {
    // Sem a lista, mostra a pasta.
  }
})
</script>

<template>
  <div class="pagina pesquisar">
    <a :href="voltar" class="voltar"><Icone nome="voltar" /> {{ pedido.para ? nomeDoTema : 'Referências' }}</a>
    <h1>{{ pedido.para ? 'Pesquisar para este Tema' : 'Pesquisar um tema' }}</h1>
    <p v-if="pedido.para" class="suave apresentacao">A pesquisa lê o Tema <strong>{{ nomeDoTema }}</strong> e vai atrás do que ele ainda não tem. Escreva o que quer saber, ou deixe o assunto do Tema. Pronta, ela sugere integrar nele, e você revisa como sempre.</p>
    <p v-else class="suave apresentacao">Escreva o tema ou a pergunta e escolha onde buscar: a IA monta as perguntas, o servidor junta o material das fontes e a IA escreve o texto citando a fonte de cada parte. O resultado vira uma Referência, com a sugestão de Tema.</p>
    <NovaPesquisa pagina :tema-inicial="pedido.tema || (pedido.para ? nomeDoTema : '')" :para-tema="pedido.para" :key="nomeDoTema" />
  </div>
</template>

<style scoped>
.voltar {
  display: inline-flex;
  gap: 2px;
  align-items: center;
  margin: 0 0 14px -4px;
  color: var(--texto-suave);
  font-size: 0.85rem;
  text-decoration: none;
}

.voltar:hover {
  color: var(--texto);
}

.voltar :deep(svg) {
  width: 16px;
  height: 16px;
}

.pesquisar h1 {
  margin-bottom: 8px;
}

.apresentacao {
  margin: 0 0 24px;
  font-size: 0.9rem;
  line-height: 1.5;
}
</style>
