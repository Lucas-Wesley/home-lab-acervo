<script setup>
import Icone from './Icone.vue'
import NovaPesquisa from './NovaPesquisa.vue'

// Pesquisar um tema (D27) numa página própria (D34), como Resumir capítulos: mais espaço para as
// fontes, a profundidade e as opções, cada uma com a explicação.
</script>

<template>
  <div class="pagina pesquisar">
    <a href="#/referencias" class="voltar"><Icone nome="voltar" /> Referências</a>
    <h1>Pesquisar um tema</h1>
    <p class="suave apresentacao">Escreva o tema ou a pergunta e escolha onde buscar: a IA monta as perguntas, o servidor junta o material das fontes e a IA escreve o texto citando a fonte de cada parte. O resultado vira uma Referência, com a sugestão de Tema.</p>
    <NovaPesquisa pagina />
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
