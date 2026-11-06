<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import EscolhaDeBaralho from './EscolhaDeBaralho.vue'
import Icone from './Icone.vue'
import { api } from './api.js'

// Janela de "Gerar flashcards" (D28): de onde vem o texto, para qual baralho vai, e o envio.
const props = defineProps({
  origem: { type: Object, required: true }, // { arquivo, capitulo } ou { referencia }
  titulo: { type: String, required: true }, // "do Tema inteiro", "do capítulo X"…
  aviso: { type: String, default: '' },
})
const emit = defineEmits(['fechar', 'enviado'])

const escolha = ref({})
const enviando = ref(false)
const erro = ref('')

async function enviar() {
  enviando.value = true
  erro.value = ''
  try {
    const e = await api('/api/flashcards', { metodo: 'POST', corpo: { ...props.origem, ...escolha.value }, tempo: 30000 })
    emit('enviado', e)
  } catch (e) {
    erro.value = e.status === 0 ? 'Gerar flashcards só funciona na rede de casa.' : e.message
  } finally {
    enviando.value = false
  }
}

function aoTeclar(ev) {
  if (ev.key === 'Escape') emit('fechar')
}
onMounted(() => document.addEventListener('keydown', aoTeclar))
onBeforeUnmount(() => document.removeEventListener('keydown', aoTeclar))
</script>

<template>
  <Teleport to="body">
    <div class="veu" @click="emit('fechar')">
      <form class="janela" role="dialog" aria-modal="true" aria-labelledby="titulo-flashcards" @click.stop @submit.prevent="enviar">
        <header>
          <h2 id="titulo-flashcards">Gerar flashcards {{ titulo }}</h2>
          <button type="button" class="botao icone" aria-label="Fechar" @click="emit('fechar')"><Icone nome="fechar" /></button>
        </header>
        <p class="suave">A IA monta os cartões no Flashcards, em segundo plano; você revisa e aprova antes de eles entrarem no baralho.</p>
        <p v-if="aviso" class="aviso">{{ aviso }}</p>
        <EscolhaDeBaralho v-model="escolha" :origem="origem" />
        <div v-if="erro" class="aviso erro">{{ erro }}</div>
        <div class="linha-acoes">
          <button class="botao principal" :disabled="enviando || (!escolha.baralho_id && !escolha.baralho)">{{ enviando ? 'Enviando…' : 'Gerar flashcards' }}</button>
          <button type="button" class="botao" @click="emit('fechar')">Cancelar</button>
        </div>
      </form>
    </div>
  </Teleport>
</template>

<style scoped>
.veu {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 22%);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
}

.janela {
  display: grid;
  gap: 12px;
  width: min(520px, 100%);
  max-height: 86vh;
  overflow-y: auto;
  padding: 16px 20px max(16px, env(safe-area-inset-bottom));
  border-radius: var(--raio-g) var(--raio-g) 0 0;
  background: var(--fundo);
  box-shadow: var(--sombra);
}

@media (min-width: 640px) {
  .veu {
    align-items: center;
  }

  .janela {
    border-radius: var(--raio-g);
  }
}

header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-right: -8px;
}

h2 {
  margin: 4px 0 0;
  font-size: 1.1rem;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

p {
  margin: 0;
}
</style>
