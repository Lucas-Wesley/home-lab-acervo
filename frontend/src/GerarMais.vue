<script setup>
import { ref } from 'vue'
import Icone from './Icone.vue'
import Microfone from './Microfone.vue'

// Gerar mais (D52): um lote novo de perguntas ou exercícios, que entra no banco sem repetir nada.
// O foco é opcional, escrito ou falado (o Whisper, como no foco dos flashcards, D51).
const props = defineProps({
  rotulo: { type: String, required: true }, // "Gerar mais perguntas"
  origem: { type: Object, default: null }, // para o Whisper acertar os nomes do texto
  ocupado: { type: Boolean, default: false },
})
const emit = defineEmits(['gerar'])

const aberto = ref(false)
const foco = ref('')
const erro = ref('')

function ouvido(texto) {
  foco.value = foco.value.trim() ? `${foco.value.trim()} ${texto}` : texto
}

function gerar() {
  emit('gerar', foco.value.trim())
  aberto.value = false
  foco.value = ''
}
</script>

<template>
  <div class="gerar-mais">
    <button v-if="!aberto" class="botao" :disabled="ocupado" @click="aberto = true"><Icone nome="mais" /> {{ props.rotulo }}</button>
    <div v-else class="formulario">
      <label class="suave" for="foco-do-lote">Foco do lote (opcional)</label>
      <div class="campo-com-voz">
        <textarea id="foco-do-lote" v-model="foco" class="campo" rows="3" maxlength="1000" placeholder="Ex.: só crase antes de pronomes; casos mais difíceis; exceções" />
        <Microfone :origem="origem" @texto="ouvido" @erro="erro = $event" />
      </div>
      <p v-if="erro" class="aviso erro">{{ erro }}</p>
      <div class="linha-acoes">
        <button class="botao principal" :disabled="ocupado" @click="gerar">Gerar</button>
        <button class="botao discreto" @click="aberto = false">Cancelar</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.formulario {
  display: grid;
  gap: 8px;
}

.campo-com-voz {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.campo-com-voz textarea {
  flex: 1;
  min-width: 0;
}

.linha-acoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.botao.discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
}

.botao :deep(svg) {
  width: 18px;
  height: 18px;
}
</style>
