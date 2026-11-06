<script setup>
import { computed, ref, watch } from 'vue'
import { api } from './api.js'
import { FONTES, PROFUNDIDADES } from './pesquisa.js'

// Pesquisador (D27): o Lucas escreve o tema, escolhe as fontes e a profundidade.
const emit = defineEmits(['fechar'])

// As últimas escolhas voltam na próxima pesquisa.
const CHAVE = 'acervo-pesquisa-escolhas'
function lerEscolhas() {
  try {
    return JSON.parse(localStorage.getItem(CHAVE)) ?? {}
  } catch {
    return {}
  }
}
const salvas = lerEscolhas()

const tema = ref('')
const fontes = ref(salvas.fontes ?? ['livros', 'wikipedia'])
const profundidade = ref(salvas.profundidade ?? 'rapida')
const revisarEsboco = ref(salvas.revisar_esboco ?? false)
const flashcards = ref(salvas.flashcards ?? false)
const enviando = ref(false)
const erro = ref('')

const semFontes = computed(() => fontes.value.length === 0)
const explicacao = computed(() => PROFUNDIDADES.find((p) => p.chave === profundidade.value)?.explicacao)

watch([fontes, profundidade, revisarEsboco, flashcards], () => {
  try {
    localStorage.setItem(CHAVE, JSON.stringify({ fontes: fontes.value, profundidade: profundidade.value, revisar_esboco: revisarEsboco.value, flashcards: flashcards.value }))
  } catch {
    // Sem espaço no aparelho, só não lembra as escolhas.
  }
}, { deep: true })

function alternar(chave) {
  fontes.value = fontes.value.includes(chave) ? fontes.value.filter((f) => f !== chave) : [...fontes.value, chave]
}

async function pesquisar() {
  enviando.value = true
  erro.value = ''
  try {
    const ref_ = await api('/api/pesquisas', {
      metodo: 'POST',
      corpo: { tema: tema.value, fontes: fontes.value, profundidade: profundidade.value, revisar_esboco: revisarEsboco.value && !semFontes.value, flashcards: flashcards.value },
    })
    location.hash = `#/referencia/${ref_.id}`
  } catch (e) {
    erro.value = e.message
  } finally {
    enviando.value = false
  }
}
</script>

<template>
  <form class="cartao nova-pesquisa" @submit.prevent="pesquisar">
    <label class="grupo">
      <span class="titulo-grupo">Tema ou pergunta</span>
      <textarea v-model="tema" class="campo" rows="2" maxlength="500" placeholder="Ex.: qual é a origem da laranja?" required />
    </label>

    <fieldset class="grupo">
      <legend class="titulo-grupo">Fontes</legend>
      <div class="opcoes">
        <button v-for="f in FONTES" :key="f.chave" type="button" class="opcao" :aria-pressed="fontes.includes(f.chave)" @click="alternar(f.chave)">{{ f.nome }}</button>
      </div>
      <p class="suave dica">{{ semFontes ? 'Nenhuma fonte: só a resposta da IA, com o que ela já sabe, sem pesquisar.' : 'O texto cita a fonte de cada parte. Os vídeos servem só de fonte e não viram Referência.' }}</p>
    </fieldset>

    <fieldset class="grupo">
      <legend class="titulo-grupo">Profundidade</legend>
      <div class="opcoes">
        <button v-for="p in PROFUNDIDADES" :key="p.chave" type="button" class="opcao" :aria-pressed="profundidade === p.chave" @click="profundidade = p.chave">{{ p.nome }}</button>
      </div>
      <p class="suave dica">{{ explicacao }}</p>
    </fieldset>

    <label v-if="!semFontes" class="marcar">
      <input v-model="revisarEsboco" type="checkbox" />
      <span>Revisar o esboço antes de buscar <span class="suave">— a pesquisa para nas perguntas, e você ajusta antes de ela buscar.</span></span>
    </label>

    <label class="marcar">
      <input v-model="flashcards" type="checkbox" />
      <span>Gerar flashcards do resultado <span class="suave">— pronta a pesquisa, a IA propõe os cartões, e você aprova junto com o texto.</span></span>
    </label>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div class="linha-acoes">
      <button class="botao principal" :disabled="enviando || tema.trim().length < 3">{{ enviando ? 'Enviando…' : 'Pesquisar' }}</button>
      <button type="button" class="botao" @click="emit('fechar')">Cancelar</button>
    </div>
  </form>
</template>

<style scoped>
.nova-pesquisa {
  display: grid;
  gap: 16px;
  margin-bottom: 16px;
}

.grupo {
  display: grid;
  gap: 6px;
  min-width: 0;
  margin: 0;
  padding: 0;
  border: 0;
}

.titulo-grupo {
  padding: 0;
  font-size: 0.85rem;
  font-weight: 600;
}

textarea.campo {
  resize: vertical;
  font: inherit;
  line-height: 1.45;
}

.opcoes {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.opcao {
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid var(--linha-forte);
  border-radius: 999px;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.opcao:hover {
  color: var(--texto);
}

.opcao[aria-pressed='true'] {
  border-color: var(--azul);
  background: var(--azul-fundo);
  color: var(--azul);
}

.dica {
  margin: 0;
  font-size: 0.8rem;
  line-height: 1.4;
}

.marcar {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  font-size: 0.875rem;
  line-height: 1.4;
}

.marcar input {
  flex: none;
  margin-top: 3px;
  accent-color: var(--azul);
}

@media (pointer: coarse) {
  .opcao {
    min-height: 36px;
  }
}
</style>
