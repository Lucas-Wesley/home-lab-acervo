<script setup>
import { onBeforeUnmount, ref, watch } from 'vue'
import Icone from './Icone.vue'

// O "i" ao lado de um título ou campo: a explicação só aparece para quem tocar. As telas ficam com o
// essencial à vista.
defineProps({ rotulo: { type: String, default: 'Mais informações' } })
const aberto = ref(false)
const raiz = ref(null)
const posicao = ref({})

// O balão fica preso à tela, perto do "i", sem sair pela borda.
function alternar() {
  aberto.value = !aberto.value
  if (!aberto.value) return
  const r = raiz.value.getBoundingClientRect()
  const largura = Math.min(280, window.innerWidth - 32)
  const esquerda = Math.max(16, Math.min(r.left + r.width / 2 - largura / 2, window.innerWidth - 16 - largura))
  posicao.value = { top: `${r.bottom + 6}px`, left: `${esquerda}px`, width: `${largura}px` }
}

function fora(ev) {
  if (!raiz.value?.contains(ev.target)) aberto.value = false
}
const fechar = () => (aberto.value = false)
function ouvir(liga) {
  const metodo = liga ? 'addEventListener' : 'removeEventListener'
  document[metodo]('pointerdown', fora)
  window[metodo]('scroll', fechar, true)
}
watch(aberto, ouvir)
onBeforeUnmount(() => ouvir(false))
</script>

<template>
  <span ref="raiz" class="info">
    <button type="button" class="botao-info" :aria-label="rotulo" :aria-expanded="aberto" @click.prevent.stop="alternar">
      <Icone nome="info" />
    </button>
    <span v-if="aberto" class="balao" role="note" :style="posicao"><slot /></span>
  </span>
</template>

<style scoped>
.info {
  position: relative;
  display: inline-flex;
  vertical-align: middle;
}

.botao-info {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--texto-fraco);
  cursor: pointer;
}

.botao-info:hover,
.botao-info[aria-expanded='true'] {
  color: var(--texto);
}

.botao-info svg {
  width: 17px;
  height: 17px;
}

.balao {
  position: fixed;
  z-index: 30;
  padding: 10px 12px;
  border: 1px solid var(--linha-forte);
  border-radius: var(--raio-p);
  background: var(--fundo);
  box-shadow: var(--sombra);
  color: var(--texto-suave);
  font-size: 0.84rem;
  font-weight: 400;
  letter-spacing: 0;
  line-height: 1.5;
  text-align: left;
  white-space: normal;
}
</style>
