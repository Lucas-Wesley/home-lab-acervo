<script setup>
import { computed, ref, watch } from 'vue'
import EscolhaDeBaralho from './EscolhaDeBaralho.vue'
import { api } from './api.js'
import { FONTES, PROFUNDIDADES } from './pesquisa.js'

// Pesquisador (D27): o Lucas escreve o tema, escolhe as fontes e a profundidade. Com uma
// Referência, é para aprofundar a pesquisa dela (D29): as escolhas começam pelas da última rodada.
// Com "pagina", é a página Pesquisar um tema (D34): as opções em linhas, com a explicação de cada
// uma, e o botão numa barra fixa no pé.
const props = defineProps({ referencia: { type: Object, default: null }, pagina: { type: Boolean, default: false } })
const emit = defineEmits(['fechar', 'enviado'])

// As últimas escolhas voltam na próxima pesquisa.
const CHAVE = 'acervo-pesquisa-escolhas'
function lerEscolhas() {
  try {
    return JSON.parse(localStorage.getItem(CHAVE)) ?? {}
  } catch {
    return {}
  }
}
const aprofundando = !!props.referencia
const salvas = aprofundando ? props.referencia.pesquisa : lerEscolhas()

const tema = ref('')
const foco = ref('')
const fontes = ref(salvas.fontes ?? ['livros', 'wikipedia'])
const profundidade = ref(salvas.profundidade ?? 'rapida')
const revisarEsboco = ref(salvas.revisar_esboco ?? false)
// Sempre começa desmarcado: gerar cartões é uma decisão de cada pesquisa.
const flashcards = ref(false)
const baralho = ref({})
const enviando = ref(false)
const erro = ref('')

const semFontes = computed(() => fontes.value.length === 0)
const explicacao = computed(() => PROFUNDIDADES.find((p) => p.chave === profundidade.value)?.explicacao)
const resumoDaEscolha = computed(() => {
  const nomes = FONTES.filter((f) => fontes.value.includes(f.chave)).map((f) => f.curto)
  const prof = profundidade.value === 'detalhada' ? 'Detalhada' : 'Rápida'
  return `${prof} · ${nomes.length ? nomes.join(', ') : 'só a IA, sem fontes'}`
})

watch([fontes, profundidade, revisarEsboco], () => {
  if (aprofundando) return
  try {
    localStorage.setItem(CHAVE, JSON.stringify({ fontes: fontes.value, profundidade: profundidade.value, revisar_esboco: revisarEsboco.value }))
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
    if (aprofundando) {
      await api(`/api/referencias/${encodeURIComponent(props.referencia.id)}/aprofundar`, {
        metodo: 'POST',
        corpo: { fontes: fontes.value, profundidade: profundidade.value, revisar_esboco: revisarEsboco.value && !semFontes.value, foco: foco.value },
      })
      emit('enviado')
      return
    }
    const ref_ = await api('/api/pesquisas', {
      metodo: 'POST',
      corpo: { tema: tema.value, fontes: fontes.value, profundidade: profundidade.value, revisar_esboco: revisarEsboco.value && !semFontes.value, flashcards: flashcards.value, baralho: flashcards.value ? baralho.value : undefined },
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
  <form class="nova-pesquisa" :class="{ cartao: !pagina, 'em-pagina': pagina }" @submit.prevent="pesquisar">
    <template v-if="aprofundando">
      <div>
        <strong>Aprofundar a pesquisa</strong>
        <p class="suave dica">Uma nova rodada, a partir do texto atual: o esboço vai atrás do que ficou em aberto, o material já coletado não é buscado de novo, e o texto novo diz o que há de novo. O texto de agora fica na aba Versão anterior.</p>
      </div>
      <label class="grupo">
        <span class="titulo-grupo">O que mais quer saber? <span class="suave">(opcional)</span></span>
        <textarea v-model="foco" class="campo" rows="2" maxlength="500" placeholder="Ex.: as variedades e como escolher no mercado" />
      </label>
    </template>
    <label v-else class="grupo">
      <span class="titulo-grupo">Tema ou pergunta</span>
      <textarea v-model="tema" class="campo" :rows="pagina ? 3 : 2" maxlength="500" placeholder="Ex.: qual é a origem da laranja?" required />
    </label>

    <fieldset class="grupo">
      <legend class="titulo-grupo">Fontes</legend>
      <ul v-if="pagina" class="lista linhas">
        <li v-for="f in FONTES" :key="f.chave">
          <label class="linha" :class="{ marcada: fontes.includes(f.chave) }">
            <input type="checkbox" :checked="fontes.includes(f.chave)" @change="alternar(f.chave)" />
            <span class="info">
              <span class="nome">{{ f.nome }}</span>
              <span class="meta">{{ f.explicacao }}</span>
            </span>
          </label>
        </li>
      </ul>
      <div v-else class="opcoes">
        <button v-for="f in FONTES" :key="f.chave" type="button" class="opcao" :aria-pressed="fontes.includes(f.chave)" @click="alternar(f.chave)">{{ f.nome }}</button>
      </div>
      <p class="suave dica">{{ semFontes ? 'Nenhuma fonte: só a resposta da IA, com o que ela já sabe, sem pesquisar.' : 'O texto cita a fonte de cada parte. Os vídeos servem só de fonte e não viram Referência.' }}</p>
    </fieldset>

    <fieldset class="grupo">
      <legend class="titulo-grupo">Profundidade</legend>
      <ul v-if="pagina" class="lista linhas">
        <li v-for="p in PROFUNDIDADES" :key="p.chave">
          <label class="linha" :class="{ marcada: profundidade === p.chave }">
            <input v-model="profundidade" type="radio" name="profundidade" :value="p.chave" />
            <span class="info">
              <span class="nome">{{ p.nome }}</span>
              <span class="meta">{{ p.explicacao }}</span>
            </span>
          </label>
        </li>
      </ul>
      <template v-else>
        <div class="opcoes">
          <button v-for="p in PROFUNDIDADES" :key="p.chave" type="button" class="opcao" :aria-pressed="profundidade === p.chave" @click="profundidade = p.chave">{{ p.nome }}</button>
        </div>
        <p class="suave dica">{{ explicacao }}</p>
      </template>
    </fieldset>

    <fieldset class="grupo">
      <legend v-if="pagina" class="titulo-grupo">Opções</legend>
      <label v-if="!semFontes" class="marcar">
        <input v-model="revisarEsboco" type="checkbox" />
        <span>Revisar o esboço antes de buscar <span class="suave">— a pesquisa para nas perguntas, e você ajusta antes de ela buscar.</span></span>
      </label>

      <label v-if="!aprofundando" class="marcar">
        <input v-model="flashcards" type="checkbox" />
        <span>Gerar flashcards do resultado <span class="suave">— pronta a pesquisa, a IA propõe os cartões, e você aprova junto com o texto.</span></span>
      </label>
      <EscolhaDeBaralho v-if="flashcards" v-model="baralho" class="baralho" />
    </fieldset>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div v-if="pagina" class="barra-pesquisa">
      <span class="escolha">{{ resumoDaEscolha }}</span>
      <button class="botao principal" :disabled="enviando || tema.trim().length < 3">{{ enviando ? 'Enviando…' : 'Pesquisar' }}</button>
    </div>
    <div v-else class="linha-acoes">
      <button class="botao principal" :disabled="enviando || (!aprofundando && tema.trim().length < 3)">{{ enviando ? 'Enviando…' : aprofundando ? 'Aprofundar' : 'Pesquisar' }}</button>
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

.baralho {
  margin-left: 24px;
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

.grupo .marcar + .marcar {
  margin-top: 10px;
}

/* Página Pesquisar um tema: cada opção numa linha, com a explicação. */
.em-pagina.nova-pesquisa {
  gap: 24px;
}

.linhas {
  margin: 0;
  padding: 0;
  list-style: none;
}

.linha {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 11px 8px;
  border-radius: var(--raio-p);
  cursor: pointer;
  transition: background 0.15s ease;
}

.linha:hover {
  background: var(--superficie);
}

.linha.marcada {
  background: var(--azul-fundo);
}

.linha input {
  flex: none;
  width: 16px;
  height: 16px;
  margin: 2px 0 0;
  accent-color: var(--azul);
}

.linha .info {
  display: grid;
  gap: 2px;
}

.linha .nome {
  font-size: 0.9rem;
  font-weight: 500;
}

.linha .meta {
  color: var(--texto-fraco);
  font-size: 0.8rem;
  line-height: 1.4;
}

.em-pagina .dica {
  margin-top: 4px;
}

/* Barra de vidro que acompanha a rolagem, como a de Resumir capítulos. */
.barra-pesquisa {
  position: sticky;
  bottom: 0;
  z-index: 3;
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin: 0 calc(-1 * var(--margem));
  padding: 12px var(--margem) max(12px, env(safe-area-inset-bottom));
  border-top: 1px solid var(--linha);
  background: var(--vidro);
  -webkit-backdrop-filter: var(--blur);
  backdrop-filter: var(--blur);
}

.escolha {
  color: var(--texto-suave);
  font-size: 0.85rem;
}

@media (max-width: 820px) {
  .barra-pesquisa {
    bottom: calc(55px + max(6px, env(safe-area-inset-bottom)));
    padding-bottom: 12px;
  }
}

@media (pointer: coarse) {
  .opcao {
    min-height: 36px;
  }
}
</style>
