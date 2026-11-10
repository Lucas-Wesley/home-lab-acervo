<script setup>
import { computed, onMounted, ref } from 'vue'
import { api, NOMES_TAREFA } from './api.js'
import { lerDiff, versaoNova } from './diff.js'
import { renderizar } from './markdown.js'

const props = defineProps({ id: { type: String, default: '' } })

const lista = ref([])
const revisao = ref(null)
const erro = ref('')
const carregou = ref(false)
const decidindo = ref(false)
const modoArquivo = ref({}) // caminho → 'texto' | 'diff'

// O índice primeiro e os tópicos na ordem em que o índice os lista: é a ordem de leitura.
const arquivos = computed(() => {
  if (!revisao.value) return []
  const lista = lerDiff(revisao.value.diff)
  const indice = lista.find((a) => a.caminho.endsWith('/indice.md') && a.caminho.split('/').length === destino.value.split('/').length + 1)
  if (!indice) return lista
  const ordem = [...versaoNova(indice).matchAll(/\]\(([^)\s#]+\.md)/g)].map((m) => `${destino.value}/${m[1]}`)
  const posicao = (a) => (a === indice ? -1 : ordem.indexOf(a.caminho) === -1 ? ordem.length : ordem.indexOf(a.caminho))
  return [...lista].sort((a, b) => posicao(a) - posicao(b))
})
const fontes = computed(() => {
  const r = revisao.value?.referencia
  return r ? [{ id: r.id, titulo: r.titulo, url: r.url }] : []
})
const destino = computed(() => {
  const p = revisao.value?.tarefa.params ?? {}
  return p.pasta || p.tema || ''
})

async function carregar() {
  try {
    if (props.id) revisao.value = await api(`/api/revisoes/${encodeURIComponent(props.id)}`)
    else lista.value = await api('/api/revisoes')
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: as revisões só abrem na rede de casa.' : e.message
  }
  carregou.value = true
}

function modo(arquivo) {
  return modoArquivo.value[arquivo.caminho] ?? (arquivo.novo ? 'texto' : 'diff')
}

async function decidir(decisao) {
  decidindo.value = true
  erro.value = ''
  try {
    const { tema } = await api(`/api/revisoes/${props.id}/${decisao}`, { metodo: 'POST' })
    const refId = revisao.value.tarefa.referencia_id
    location.hash = decisao === 'aprovar' && tema ? `#/ler/${tema}/indice.md` : refId ? `#/referencia/${refId}` : tema ? `#/ler/${tema}/indice.md` : '#/revisoes'
  } catch (e) {
    erro.value = e.message
  } finally {
    decidindo.value = false
  }
}

onMounted(carregar)
</script>

<template>
  <div class="pagina larga">
    <template v-if="!props.id">
      <h1>Revisões</h1>
      <p class="suave explicacao">Propostas da IA para o Acervo. Nada entra antes de você aprovar.</p>
      <div v-if="erro" class="aviso erro">{{ erro }}</div>
      <div v-if="carregou && !lista.length" class="vazio">Nenhuma proposta aguardando revisão.</div>
      <a v-for="t in lista" :key="t.id" :href="`#/revisao/${t.id}`" class="item">
        <strong>{{ NOMES_TAREFA[t.tipo] }}: {{ t.params.titulo || t.params.tema }}</strong>
        <span class="suave">{{ t.referencia }} · {{ new Date(t.terminado_em).toLocaleString('pt-BR') }}</span>
      </a>
    </template>

    <template v-else>
      <a href="#/revisoes" class="voltar suave">← Revisões</a>
      <div v-if="erro" class="aviso erro">{{ erro }}</div>
      <template v-if="revisao">
        <h1>{{ { criar_tema: 'Novo Tema', integrar_texto: 'Integrar ao texto' }[revisao.tarefa.tipo] || 'Integração' }}: {{ revisao.tarefa.params.titulo || revisao.referencia?.titulo || destino }}</h1>
        <p v-if="revisao.tarefa.tipo === 'integrar_texto'" class="suave">
          {{ revisao.tarefa.params.origem === 'detalhe' ? 'De um “Mais informações”' : 'Da resposta a uma pergunta' }},
          <template v-if="destino">no Tema <code>{{ destino }}</code>.</template>
          <template v-else>no texto da Referência <a :href="`#/referencia/${revisao.referencia?.id}`">{{ revisao.referencia?.titulo }}</a>; aprovando, o texto de agora fica como Versão anterior.</template>
        </p>
        <p v-else class="suave">
          A partir de <a :href="`#/referencia/${revisao.referencia.id}`">{{ revisao.referencia.titulo }}</a>, na pasta <code>{{ destino }}</code>.
          {{ arquivos.length }} {{ arquivos.length === 1 ? 'arquivo' : 'arquivos' }}.
        </p>

        <div class="linha-acoes decisao">
          <button class="botao principal" :disabled="decidindo" @click="decidir('aprovar')">{{ revisao.tarefa.tipo === 'integrar_texto' ? 'Aprovar a mudança' : 'Aprovar e incluir no Acervo' }}</button>
          <button class="botao perigo" :disabled="decidindo" @click="decidir('rejeitar')">Rejeitar</button>
        </div>

        <section v-for="a in arquivos" :key="a.caminho" class="arquivo">
          <header>
            <code>{{ destino ? a.caminho.replace(destino + '/', '') : a.caminho.split('/').pop() }}</code>
            <span class="chip" :class="{ ok: a.novo }">{{ a.novo ? 'novo' : `+${a.mais} −${a.menos}` }}</span>
            <button class="botao" @click="modoArquivo = { ...modoArquivo, [a.caminho]: modo(a) === 'diff' ? 'texto' : 'diff' }">
              {{ modo(a) === 'diff' ? 'Ver como texto' : 'Ver diferenças' }}
            </button>
          </header>
          <article v-if="modo(a) === 'texto'" class="texto" v-html="renderizar(versaoNova(a), { fontes })" />
          <pre v-else class="diff"><template v-for="(l, i) in a.linhas" :key="i"><span :class="l.tipo">{{ l.tipo === 'mais' ? '+ ' : l.tipo === 'menos' ? '− ' : l.tipo === 'trecho' ? '' : '  ' }}{{ l.texto }}
</span></template></pre>
        </section>
      </template>
    </template>
  </div>
</template>

<style scoped>
.larga {
  max-width: 900px;
}

.explicacao {
  margin-top: -8px;
}

.voltar {
  display: inline-block;
  margin-bottom: 8px;
  text-decoration: none;
  font-size: 0.9rem;
}

.item {
  display: grid;
  gap: 4px;
  padding: 14px 0;
  border-bottom: 1px solid var(--linha);
  color: inherit;
  text-decoration: none;
}

.decisao {
  position: sticky;
  top: 0;
  z-index: 3;
  margin: 0 calc(-1 * var(--margem)) 16px;
  padding: 10px var(--margem);
  background: var(--vidro);
  -webkit-backdrop-filter: var(--blur);
  backdrop-filter: var(--blur);
}

.arquivo {
  margin-bottom: 22px;
  border: 1px solid var(--linha);
  border-radius: 14px;
  overflow: hidden;
}

.arquivo header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--linha);
  background: var(--superficie);
}

.arquivo header code {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.arquivo .texto {
  padding: 8px 20px 4px;
}

.diff {
  margin: 0;
  padding: 10px 0;
  overflow-x: auto;
  font-size: 0.8rem;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.diff span {
  display: block;
  padding: 0 14px;
}

.diff .mais {
  background: var(--mais);
}

.diff .menos {
  background: var(--menos);
  text-decoration: line-through;
  text-decoration-color: color-mix(in srgb, currentColor 40%, transparent);
}

.diff .trecho {
  margin-top: 6px;
  color: var(--texto-suave);
}

</style>
