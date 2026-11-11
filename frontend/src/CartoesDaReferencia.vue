<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import EnviarFlashcards from './EnviarFlashcards.vue'
import { api } from './api.js'

// Flashcards de uma Referência (D28). O texto organizado vai para o app Flashcards, que monta a
// proposta na fila dele; aqui ela aparece junto com a Referência, para aprovar ou descartar sem sair
// do Acervo. Editar cartão por cartão (e trocar foto) continua no Flashcards.
const props = defineProps({ referencia: { type: Object, required: true } })

const info = ref({ disponivel: false, link: '', envios: [] })
const g = ref(null) // o pedido no Flashcards
const cartoes = ref([])
const nome = ref('')
const ocupado = ref(false)
const erro = ref('')
const feito = ref('')
const escolhendo = ref(false) // a janela de escolher o baralho
const tipo = ref('basico') // padrão (perguntas) ou leitura
let relogio

const ultimo = computed(() => info.value.envios[0])
const trabalhando = computed(() => ['na_fila', 'rodando'].includes(g.value?.estado))
const escolhidos = computed(() => cartoes.value.filter((c) => c.incluir))
const podeGerar = computed(() => ['organizado', 'validado', 'integrado'].includes(props.referencia.estado))
// Sem nada enviado, o cartão some: "Gerar flashcards" fica no menu da Referência.
const visivel = computed(() => info.value.disponivel && !!ultimo.value)
const disponivel = computed(() => info.value.disponivel && podeGerar.value)
const linkNoFlashcards = computed(() => (g.value ? `${info.value.link}/#/geracao/${g.value.id}` : ''))

async function carregar() {
  clearTimeout(relogio)
  try {
    info.value = await api(`/api/flashcards?alvo=${encodeURIComponent(`referencia:${props.referencia.id}`)}`)
    if (ultimo.value?.geracao_id) {
      const anterior = g.value?.estado
      g.value = await api(`/api/flashcards/${ultimo.value.geracao_id}`)
      if (g.value.estado === 'em_revisao' && anterior !== 'em_revisao') {
        nome.value = g.value.proposta.nome_baralho
        cartoes.value = g.value.proposta.cartoes.map((c) => ({ ...c, incluir: true }))
      }
    } else {
      g.value = null
    }
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? '' : e.message
  }
  if (trabalhando.value) relogio = setTimeout(carregar, 4000)
}

// Um pedido de cada vez (D44): com flashcards esperando decisão, o pedido novo não sai. Pelo botão
// (sem tipo), são os padrão.
function gerar(qual) {
  tipo.value = qual === 'leitura' ? 'leitura' : 'basico'
  feito.value = ''
  if (trabalhando.value || g.value?.estado === 'em_revisao') {
    erro.value = 'Os flashcards deste texto já foram pedidos: aprove ou descarte os que estão aqui antes de pedir de novo.'
    return
  }
  escolhendo.value = true
}

async function enviado() {
  escolhendo.value = false
  g.value = null
  await carregar()
}

async function decidir(acao, corpo) {
  if (acao === 'descartar' && !confirm('Descartar estes flashcards? Nenhum cartão será criado.')) return
  ocupado.value = true
  erro.value = ''
  try {
    const r = await api(`/api/flashcards/${g.value.id}/${acao}`, { metodo: 'POST', corpo, tempo: 150000 })
    if (acao === 'aprovar') {
      const partes = [r.criadas && `${r.criadas} ${r.criadas === 1 ? 'cartão criado' : 'cartões criados'}`, r.atualizadas && `${r.atualizadas} ${r.atualizadas === 1 ? 'atualizado' : 'atualizados'}`]
      feito.value = `${partes.filter(Boolean).join(', ')} no Flashcards.`
    }
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

function aprovar() {
  decidir('aprovar', { nome_baralho: nome.value, cartoes: escolhidos.value.map(({ incluir, ...c }) => c) })
}

defineExpose({ gerar, disponivel })

onMounted(carregar)
onBeforeUnmount(() => clearTimeout(relogio))
// A pesquisa que acabou de ficar pronta manda o texto sozinha: olhar de novo quando o estado muda.
watch(() => props.referencia.estado, carregar)
</script>

<template>
  <section v-if="visivel" class="cartao cartoes-ref" aria-label="Flashcards">
    <div class="cabeca">
      <strong>Flashcards</strong>
      <a v-if="g" class="suave" :href="linkNoFlashcards" target="_blank" rel="noopener">Abrir no Flashcards</a>
    </div>

    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <div v-if="feito" class="aviso ok">{{ feito }}</div>

    <template v-if="!ultimo.geracao_id">
      <p class="erro-texto">Não chegou ao Flashcards: {{ ultimo.erro }}</p>
      <div><button class="botao" :disabled="ocupado" @click="gerar">Tentar de novo</button></div>
    </template>

    <p v-else-if="!g" class="suave">Carregando…</p>

    <p v-else-if="trabalhando" class="suave">
      {{ g.estado === 'na_fila' ? 'Na fila do Flashcards: a IA faz um pedido por vez.' : 'A IA está lendo o texto e montando os cartões…' }}
      Pode sair desta tela.
    </p>

    <template v-else-if="g.estado === 'em_revisao'">
      <p v-if="g.proposta.observacoes" class="suave"><strong>Da IA:</strong> {{ g.proposta.observacoes }}</p>
      <label v-if="!g.baralho_id" class="grupo">
        <span class="suave">Baralho novo</span>
        <input v-model="nome" class="campo" aria-label="Nome do baralho" />
      </label>
      <p v-else class="suave">Vão para o baralho “{{ g.proposta.nome_baralho }}”, que já existe.</p>
      <ol class="lista-cartoes">
        <li v-for="(c, i) in cartoes" :key="i" :class="{ fora: !c.incluir }">
          <input v-model="c.incluir" type="checkbox" :aria-label="`Incluir o cartão ${i + 1}`" />
          <div>
            <div class="frente">{{ c.frente }}</div>
            <div class="verso suave">{{ c.verso }}</div>
            <span v-if="c.atualiza" class="chip">atualiza um cartão que já existe</span>
            <span v-if="c.imagem_url" class="chip">com imagem</span>
          </div>
        </li>
      </ol>
      <div class="linha-acoes">
        <button class="botao principal" :disabled="ocupado || !escolhidos.length || (!g.baralho_id && !nome.trim())" @click="aprovar">
          {{ ocupado ? 'Enviando…' : `Aprovar ${escolhidos.length} ${escolhidos.length === 1 ? 'cartão' : 'cartões'}` }}
        </button>
        <button class="botao perigo" :disabled="ocupado" @click="decidir('descartar')">Descartar</button>
      </div>
      <p class="suave dica">Para editar um cartão ou trocar a foto, abra no Flashcards.</p>
    </template>

    <template v-else-if="g.estado === 'concluida'">
      <p>Aprovados. <a :href="`${info.link}/#/baralho/${g.resultado}`" target="_blank" rel="noopener">Abrir o baralho</a></p>
      <div><button class="botao" :disabled="ocupado || !podeGerar" @click="gerar">Gerar de novo</button></div>
    </template>

    <template v-else>
      <p :class="g.estado === 'falhou' ? 'erro-texto' : 'suave'">{{ g.estado === 'falhou' ? `Falhou: ${g.erro}` : 'Descartados.' }}</p>
      <div class="linha-acoes">
        <button class="botao" :disabled="ocupado" @click="decidir('refazer')">Gerar de novo</button>
        <button v-if="podeGerar" class="botao" :disabled="ocupado" @click="gerar">Gerar em outro baralho</button>
      </div>
    </template>
  </section>

  <EnviarFlashcards
    v-if="escolhendo"
    :origem="{ referencia: referencia.id }"
    :tipo="tipo"
    :titulo="`${tipo === 'leitura' ? 'de leitura ' : ''}${referencia.tipo === 'pesquisa' ? 'desta pesquisa' : 'desta Referência'}`"
    @fechar="escolhendo = false"
    @enviado="enviado"
  />
</template>

<style scoped>
.cartoes-ref {
  display: grid;
  gap: 10px;
}

.cabeca {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.grupo {
  display: grid;
  gap: 4px;
}

.lista-cartoes {
  display: grid;
  margin: 0;
  padding: 0;
  list-style: none;
}

.lista-cartoes li {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 10px 0;
  border-top: 1px solid var(--linha);
}

.lista-cartoes li > div {
  min-width: 0;
  overflow-wrap: anywhere;
}

.lista-cartoes input {
  margin-top: 4px;
}

.fora {
  opacity: 0.45;
}

.frente {
  font-weight: 500;
}

.verso {
  white-space: pre-line;
}

.chip {
  margin-top: 4px;
  margin-right: 4px;
}

.dica {
  font-size: 0.85rem;
}
</style>
