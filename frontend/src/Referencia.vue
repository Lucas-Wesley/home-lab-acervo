<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api, NOMES_ESTADO, NOMES_ESTADO_TAREFA, NOMES_TAREFA, situacao } from './api.js'
import { duvidas, renderizar, tituloDoTexto } from './markdown.js'

const props = defineProps({ id: { type: String, required: true } })

const ref_ = ref(null)
const erro = ref('')
const aba = ref('')
const temas = ref([])
const acao = ref(null) // 'criar' | 'integrar'
const novoTema = ref({ titulo: '', pai: '' })
const temaAlvo = ref('')
const ocupado = ref(false)

const traduzida = computed(() => !!ref_.value?.textos.traducao)
const textoPrincipal = computed(() => ref_.value?.textos.traducao || ref_.value?.textos.organizado || '')
const abas = computed(() => {
  const t = ref_.value?.textos ?? {}
  const lista = []
  if (t.traducao) lista.push(['traducao', 'Português'])
  if (t.organizado) lista.push(['organizado', t.traducao ? 'Original' : 'Organizado'])
  if (duvidas(textoPrincipal.value)) lista.push(['duvidas', 'Dúvidas'])
  if (t.cru) lista.push(['cru', 'Transcrição'])
  lista.push(['historico', 'Histórico'])
  return lista
})
const abaAtual = computed(() => aba.value || abas.value[0]?.[0])
const htmlDaAba = computed(() => {
  const r = ref_.value
  if (!r) return ''
  const opcoes = { url: r.url }
  if (abaAtual.value === 'duvidas') return renderizar(duvidas(textoPrincipal.value), opcoes)
  if (abaAtual.value === 'cru') return renderizar(r.textos.cru.replace(/^\[(\d[\d:]*)\]/gm, '<sub>Trechos: [$1]</sub>\n\n'), opcoes)
  return renderizar(r.textos[abaAtual.value] ?? '', opcoes)
})
const pronta = computed(() => ['organizado', 'validado', 'integrado'].includes(ref_.value?.estado))
const andando = computed(() => {
  const t = ref_.value?.tarefa
  return t && (t.estado === 'rodando' || t.estado === 'na_fila')
})
const sugestao = computed(() => ref_.value?.sugestao)
const temaPorCaminho = (c) => temas.value.find((t) => t.caminho === c)

let relogio
async function carregar() {
  try {
    ref_.value = await api(`/api/referencias/${encodeURIComponent(props.id)}`)
    erro.value = ''
    document.title = `${ref_.value.titulo || 'Referência'} · Acervo`
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: as Referências só abrem na rede de casa.' : e.message
  }
}

async function carregarTemas() {
  try {
    temas.value = await api('/api/temas')
  } catch {
    temas.value = []
  }
}

async function executar(nome, corpo) {
  ocupado.value = true
  erro.value = ''
  try {
    ref_.value = { ...ref_.value, ...(await api(`/api/referencias/${encodeURIComponent(props.id)}/${nome}`, { metodo: 'POST', corpo })) }
    acao.value = null
    await carregar()
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

function abrirCriar() {
  const s = sugestao.value
  novoTema.value = {
    titulo: s && s.acao !== 'integrar' ? s.titulo : tituloDoTexto(textoPrincipal.value) || ref_.value.titulo,
    pai: s?.acao === 'subtema' ? s.tema : '',
  }
  acao.value = 'criar'
}

function abrirIntegrar() {
  temaAlvo.value = sugestao.value?.acao === 'integrar' ? sugestao.value.tema : temas.value[0]?.caminho ?? ''
  acao.value = 'integrar'
}

function usarSugestao() {
  if (sugestao.value.acao === 'integrar') abrirIntegrar()
  else abrirCriar()
}

function descreverSugestao(s) {
  if (s.acao === 'novo') return `criar um Tema principal novo, "${s.titulo}"`
  if (s.acao === 'subtema') return `criar "${s.titulo}" como subtema de "${temaPorCaminho(s.tema)?.titulo ?? s.tema}"`
  return `integrar ao Tema "${temaPorCaminho(s.tema)?.titulo ?? s.tema}"`
}

function duracao(t) {
  if (!t.iniciado_em || !t.terminado_em) return ''
  const s = Math.round((new Date(t.terminado_em) - new Date(t.iniciado_em)) / 1000)
  return s < 60 ? `${s} s` : `${Math.floor(s / 60)} min ${s % 60} s`
}

onMounted(() => {
  carregar()
  carregarTemas()
  relogio = setInterval(() => {
    if (document.visibilityState === 'visible' && (andando.value || !ref_.value)) carregar()
  }, 3000)
})
onBeforeUnmount(() => clearInterval(relogio))
</script>

<template>
  <div class="pagina">
    <a href="#/referencias" class="voltar suave">← Referências</a>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <template v-if="ref_">
      <header class="cabeca">
        <h1>{{ ref_.titulo || ref_.url }}</h1>
        <div class="suave meta">
          {{ [ref_.canal, ref_.duracao, ref_.idioma && `idioma: ${ref_.idioma}`].filter(Boolean).join(' · ') }}
          · <a :href="ref_.url" target="_blank" rel="noopener">abrir no YouTube</a>
        </div>
        <div class="linha-acoes estado">
          <span class="chip" :class="{ erro: ref_.estado === 'erro', ok: ref_.estado === 'integrado' }">{{ NOMES_ESTADO[ref_.estado] || ref_.estado }}</span>
          <span v-if="situacao(ref_)" class="chip andando">{{ situacao(ref_) }}</span>
          <a v-if="ref_.tarefa?.estado === 'em_revisao'" class="botao" :href="`#/revisao/${ref_.tarefa.id}`">Revisar proposta</a>
        </div>
      </header>

      <div v-if="ref_.erro" class="aviso erro">{{ ref_.erro }}</div>
      <div v-if="andando" class="aviso">
        {{ ref_.tarefa.estado === 'rodando' ? 'Trabalhando nisso agora. Etapas com o Claude levam alguns minutos; pode sair da tela.' : 'Na fila: roda assim que as tarefas anteriores terminarem.' }}
      </div>

      <section v-if="ref_.temas.length" class="cartao temas-ligados">
        <span class="suave">No Acervo:</span>
        <a v-for="t in ref_.temas" :key="t" :href="`#/ler/${t}/indice.md`">{{ temaPorCaminho(t)?.titulo ?? t }}</a>
      </section>

      <section v-if="sugestao && pronta" class="cartao sugestao">
        <div><strong>Sugestão da IA:</strong> {{ descreverSugestao(sugestao) }}.</div>
        <div class="suave">{{ sugestao.motivo }}</div>
        <ul v-if="sugestao.outros?.length" class="suave">
          <li v-for="o in sugestao.outros" :key="o.tema">Também: {{ temaPorCaminho(o.tema)?.titulo ?? o.tema }} — {{ o.motivo }}</li>
        </ul>
        <div><button class="botao" :disabled="andando" @click="usarSugestao">Usar sugestão</button></div>
      </section>

      <div class="linha-acoes barra-acoes">
        <template v-if="ref_.estado === 'erro' || (!pronta && !andando && ref_.estado !== 'descartado')">
          <button class="botao principal" :disabled="ocupado || andando" @click="executar('reprocessar')">Tentar de novo</button>
        </template>
        <template v-if="pronta">
          <button class="botao" :disabled="ocupado || andando" @click="executar('sugerir')">{{ sugestao ? 'Sugerir de novo' : 'Sugerir Tema' }}</button>
          <button class="botao" :disabled="ocupado || andando" @click="abrirCriar">Criar Tema</button>
          <button class="botao" :disabled="ocupado || andando || !temas.length" @click="abrirIntegrar">Integrar em Tema</button>
          <button v-if="ref_.estado === 'organizado'" class="botao" :disabled="ocupado" @click="executar('validar')">Marcar como validada</button>
        </template>
        <button v-if="ref_.estado !== 'descartado'" class="botao perigo" :disabled="ocupado" @click="executar('descartar')">Descartar</button>
      </div>

      <form v-if="acao === 'criar'" class="cartao formulario" @submit.prevent="executar('criar-tema', novoTema)">
        <strong>Criar um Tema com esta Referência</strong>
        <label><span>Título</span><input v-model="novoTema.titulo" class="campo" /></label>
        <label>
          <span>Onde</span>
          <select v-model="novoTema.pai" class="campo">
            <option value="">Tema principal</option>
            <option v-for="t in temas" :key="t.caminho" :value="t.caminho">Subtema de: {{ t.titulo }}</option>
          </select>
        </label>
        <p class="suave">A IA monta o índice e os tópicos; nada entra no Acervo antes de você revisar.</p>
        <div class="linha-acoes">
          <button class="botao principal" :disabled="ocupado || !novoTema.titulo.trim()">Pedir à IA</button>
          <button type="button" class="botao" @click="acao = null">Cancelar</button>
        </div>
      </form>

      <form v-if="acao === 'integrar'" class="cartao formulario" @submit.prevent="executar('integrar', { tema: temaAlvo })">
        <strong>Integrar a um Tema existente</strong>
        <label>
          <span>Tema</span>
          <select v-model="temaAlvo" class="campo">
            <option v-for="t in temas" :key="t.caminho" :value="t.caminho">{{ t.titulo }}</option>
          </select>
        </label>
        <p class="suave">A IA procura o que falta no Tema e propõe onde encaixar. Você revisa a diferença antes de entrar.<template v-if="traduzida"> Entra a versão em português.</template></p>
        <div class="linha-acoes">
          <button class="botao principal" :disabled="ocupado || !temaAlvo">Pedir à IA</button>
          <button type="button" class="botao" @click="acao = null">Cancelar</button>
        </div>
      </form>

      <nav v-if="abas.length" class="abas">
        <button v-for="[chave, nome] in abas" :key="chave" :class="{ ativa: abaAtual === chave }" @click="aba = chave">{{ nome }}</button>
      </nav>

      <ul v-if="abaAtual === 'historico'" class="historico">
        <li v-for="t in ref_.tarefas" :key="t.id">
          <span>{{ NOMES_TAREFA[t.tipo] || t.tipo }}</span>
          <span class="chip" :class="{ erro: t.estado === 'falhou', ok: t.estado === 'concluida' }">{{ NOMES_ESTADO_TAREFA[t.estado] || t.estado }}</span>
          <span class="suave">{{ new Date(t.criado_em).toLocaleString('pt-BR') }}<template v-if="duracao(t)"> · {{ duracao(t) }}</template></span>
          <span v-if="t.erro" :class="t.estado === 'falhou' ? 'erro-texto' : 'suave'">{{ t.erro }}</span>
        </li>
      </ul>
      <p v-else-if="abaAtual === 'organizado' && traduzida" class="suave nota">Versão original, só para consulta. No Acervo entra a versão em português.</p>
      <article v-if="abaAtual !== 'historico'" class="texto" v-html="htmlDaAba" />
    </template>
  </div>
</template>

<style scoped>
.voltar {
  display: inline-block;
  margin-bottom: 8px;
  text-decoration: none;
  font-size: 0.9rem;
}

.cabeca h1 {
  margin-bottom: 6px;
  font-family: var(--leitura);
  line-height: 1.25;
}

.meta {
  font-size: 0.88rem;
}

.estado {
  margin: 12px 0;
}

.aviso,
.cartao {
  margin-bottom: 12px;
}

.temas-ligados {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: baseline;
}

.sugestao {
  display: grid;
  gap: 8px;
  line-height: 1.45;
}

.sugestao ul {
  margin: 0;
  padding-left: 18px;
}

.barra-acoes {
  margin: 4px 0 14px;
}

.formulario {
  display: grid;
  gap: 10px;
}

.formulario label {
  display: grid;
  gap: 4px;
  font-size: 0.85rem;
  color: var(--texto-suave);
}

.formulario p {
  margin: 0;
  font-size: 0.85rem;
}

.abas {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  margin: 18px 0 20px;
  border-bottom: 1px solid var(--linha);
}

.abas button {
  padding: 10px 12px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--texto-suave);
  cursor: pointer;
  white-space: nowrap;
}

.abas button.ativa {
  border-bottom-color: var(--destaque);
  color: var(--texto);
  font-weight: 600;
}

.nota {
  font-size: 0.85rem;
}

.historico {
  margin: 0;
  padding: 0;
  list-style: none;
}

.historico li {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid var(--linha);
  font-size: 0.9rem;
}

.erro-texto {
  width: 100%;
  color: var(--perigo);
}
</style>
