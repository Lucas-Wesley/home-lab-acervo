<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { api, NOMES_ESTADO_TAREFA, NOMES_TAREFA, nomeDoEstado, situacao } from './api.js'
import CamadaDetalhes from './CamadaDetalhes.vue'
import CartoesDaReferencia from './CartoesDaReferencia.vue'
import EsbocoPesquisa from './EsbocoPesquisa.vue'
import Icone from './Icone.vue'
import NovaPesquisa from './NovaPesquisa.vue'
import Duvidas from './Duvidas.vue'
import Perguntas from './Perguntas.vue'
import { descreverLivro, descreverPesquisa } from './pesquisa.js'
import { marcar, usarDetalhes } from './detalhes.js'
import { duvidas, renderizar, tituloDoTexto } from './markdown.js'

const props = defineProps({ id: { type: String, required: true } })

const ref_ = ref(null)
const erro = ref('')
const aba = ref('')
const temas = ref([])
const acao = ref(null) // 'criar' | 'integrar' | 'aprofundar'
const focoDoAprofundar = ref('') // "Pesquisar isto" numa dúvida de pesquisa (D37)
const perguntasVersao = ref(0)

// "Pesquisar isto" (D37): a dúvida vira uma pergunta (D35), e a aba Perguntar abre com ela.
async function perguntarSobre(texto) {
  try {
    await api('/api/perguntas', { metodo: 'POST', corpo: { local: `referencias/${props.id}`, titulo: ref_.value.titulo, pergunta: texto } })
    perguntasVersao.value++
    aba.value = 'perguntar'
  } catch (e) {
    erro.value = e.message
  }
}
function aprofundarSobre(texto) {
  focoDoAprofundar.value = texto
  acao.value = 'aprofundar'
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
const novoTema = ref({ titulo: '', pai: '' })
const temaAlvo = ref('')
const ocupado = ref(false)
const menuAberto = ref(false)
const cartoes = ref(null) // CartoesDaReferencia: o "Gerar flashcards" do menu abre a janela dele

const ehPesquisa = computed(() => ref_.value?.tipo === 'pesquisa')
const ehLivro = computed(() => ref_.value?.tipo === 'livro')
const traduzida = computed(() => !!ref_.value?.textos.traducao)
const textoPrincipal = computed(() => ref_.value?.textos.traducao || ref_.value?.textos.organizado || '')
const abas = computed(() => {
  const t = ref_.value?.textos ?? {}
  const lista = []
  if (t.traducao) lista.push(['traducao', 'Português'])
  if (t.organizado) lista.push(['organizado', t.traducao ? 'Original' : 'Organizado'])
  if (duvidas(textoPrincipal.value)) lista.push(['duvidas', 'Dúvidas'])
  if (textoPrincipal.value) lista.push(['perguntar', 'Perguntar'], ['estudar', 'Estudar'])
  if (t.anterior) lista.push(['anterior', 'Versão anterior'])
  return lista
})
// A transcrição e o histórico abrem pelo menu ⋯, para as abas ficarem com o que se usa na leitura.
const nomeDoCru = computed(() => (ehPesquisa.value ? 'Material coletado' : ehLivro.value ? 'Texto do livro' : ref_.value?.tipo === 'artigo' ? 'Texto da página' : 'Transcrição'))
const doMenu = computed(() => ({ cru: nomeDoCru.value, historico: 'Histórico' })[aba.value])
const semTexto = (a) => ['historico', 'duvidas', 'perguntar', 'estudar'].includes(a)
const abaAtual = computed(() => aba.value || abas.value[0]?.[0])
const htmlDaAba = computed(() => {
  const r = ref_.value
  if (!r) return ''
  const opcoes = { url: r.url }
  if (abaAtual.value === 'cru') return renderizar(r.textos.cru.replace(/^\[(\d[\d:]*)\]/gm, '<sub>Trechos: [$1]</sub>\n\n'), opcoes)
  return renderizar(r.textos[abaAtual.value] ?? '', opcoes)
})
// Cada aba é um texto, com as suas marcações de "Mais informações".
const textoEl = ref(null)
const detalhes = usarDetalhes(() => (ref_.value && !semTexto(abaAtual.value) ? `referencias/${props.id}/${abaAtual.value}` : ''))
const marcado = computed(() => marcar([htmlDaAba.value], detalhes.lista.value))
const pronta = computed(() => ['organizado', 'validado', 'integrado'].includes(ref_.value?.estado))
const andando = computed(() => {
  const t = ref_.value?.tarefa
  return t && (t.estado === 'rodando' || t.estado === 'na_fila')
})
const sugestao = computed(() => ref_.value?.sugestao)
const temaPorCaminho = (c) => temas.value.find((t) => t.caminho === c)

// Menu "⋯": as ações ficam escondidas para o texto vir primeiro.
function escolher(fn) {
  menuAberto.value = false
  fn()
}

function fecharMenu(e) {
  if (e.type === 'keydown' ? e.key === 'Escape' : !e.target.closest?.('.menu-acoes')) menuAberto.value = false
}

async function excluir() {
  if (!confirm('Excluir esta Referência? O texto e o histórico dela somem; o que já entrou nos Temas continua no Acervo.')) return
  ocupado.value = true
  erro.value = ''
  try {
    await api(`/api/referencias/${encodeURIComponent(props.id)}`, { metodo: 'DELETE' })
    location.hash = '#/referencias'
  } catch (e) {
    erro.value = e.message
  } finally {
    ocupado.value = false
  }
}

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
  document.addEventListener('click', fecharMenu)
  document.addEventListener('keydown', fecharMenu)
  carregar()
  carregarTemas()
  relogio = setInterval(() => {
    if (document.visibilityState === 'visible' && (andando.value || !ref_.value)) carregar()
  }, 3000)
})
onBeforeUnmount(() => {
  clearInterval(relogio)
  document.removeEventListener('click', fecharMenu)
  document.removeEventListener('keydown', fecharMenu)
})
</script>

<template>
  <div class="pagina">
    <a href="#/referencias" class="voltar suave">← Referências</a>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <template v-if="ref_">
      <header class="cabeca">
        <div class="titulo-linha">
          <h1>{{ ref_.titulo || ref_.url }}</h1>
          <div class="menu-acoes">
            <button class="botao icone" aria-label="Mais ações" title="Mais ações" aria-haspopup="menu" :aria-expanded="menuAberto" @click="menuAberto = !menuAberto">
              <Icone nome="pontos" />
            </button>
            <div v-if="menuAberto" class="painel-menu" role="menu">
              <template v-if="pronta">
                <button v-if="cartoes?.disponivel" role="menuitem" @click="escolher(cartoes.gerar)">Gerar flashcards</button>
                <button role="menuitem" :disabled="ocupado || andando" @click="escolher(() => executar('sugerir'))">{{ sugestao ? 'Sugerir Tema de novo' : 'Sugerir Tema' }}</button>
                <button role="menuitem" :disabled="ocupado || andando" @click="escolher(abrirCriar)">Criar Tema</button>
                <button role="menuitem" :disabled="ocupado || andando || !temas.length" @click="escolher(abrirIntegrar)">Integrar em Tema</button>
                <button v-if="ehPesquisa" role="menuitem" :disabled="ocupado || andando" @click="escolher(() => (acao = 'aprofundar'))">Aprofundar</button>
                <hr />
              </template>
              <button v-if="ref_.textos.cru" role="menuitem" @click="escolher(() => (aba = 'cru'))">{{ nomeDoCru }}</button>
              <button role="menuitem" @click="escolher(() => (aba = 'historico'))">Histórico</button>
              <hr />
              <button role="menuitem" class="perigo" :disabled="ocupado || andando" @click="escolher(excluir)">Excluir</button>
            </div>
          </div>
        </div>
        <div v-if="ehPesquisa" class="suave meta">
          {{ descreverPesquisa(ref_.pesquisa) }}<template v-if="ref_.pesquisa.tema !== ref_.titulo"> · pedido: “{{ ref_.pesquisa.tema }}”</template>
          <template v-if="ref_.pesquisa.foco"> · a mais: “{{ ref_.pesquisa.foco }}”</template>
          <template v-if="ref_.pesquisa.relacionados?.length"> · comparada com: {{ ref_.pesquisa.relacionados.map((t) => temaPorCaminho(t)?.titulo ?? t).join(', ') }}</template>
        </div>
        <div v-else-if="ehLivro" class="suave meta">
          {{ descreverLivro(ref_.livro) }}<template v-if="ref_.livro.foco"> · foco: “{{ ref_.livro.foco }}”</template>
          · <a :href="ref_.url" target="_blank" rel="noopener">abrir na Biblioteca</a>
        </div>
        <div v-else class="suave meta">
          {{ [ref_.canal, ref_.duracao, ref_.idioma && `idioma: ${ref_.idioma}`].filter(Boolean).join(' · ') }}
          · <a :href="ref_.url" target="_blank" rel="noopener">{{ { artigo: 'abrir a página', reel: 'abrir no Instagram', podcast: ref_.url.includes('spotify') ? 'ouvir no Spotify' : 'ouvir no Apple Podcasts' }[ref_.tipo] || 'abrir no YouTube' }}</a>
        </div>
        <div class="linha-acoes estado">
          <span class="chip" :class="{ erro: ref_.estado === 'erro', ok: ref_.estado === 'integrado' }">{{ nomeDoEstado(ref_) }}</span>
          <span v-if="situacao(ref_)" class="chip andando">{{ situacao(ref_) }}</span>
          <a v-if="ref_.tarefa?.estado === 'em_revisao'" class="botao" :href="`#/revisao/${ref_.tarefa.id}`">Revisar proposta</a>
        </div>
      </header>

      <div v-if="ref_.erro" class="aviso erro">{{ ref_.erro }}</div>
      <div v-if="andando" class="aviso">
        {{ ref_.tarefa.estado === 'rodando' ? 'Trabalhando nisso agora. Etapas com o Claude levam alguns minutos; pode sair da tela.' : 'Na fila: roda assim que as tarefas anteriores terminarem.' }}
      </div>

      <EsbocoPesquisa
        v-if="ehPesquisa && ref_.estado === 'esboco' && !andando"
        :key="ref_.atualizado_em"
        :pesquisa="ref_.pesquisa"
        :ocupado="ocupado"
        @aprovar="(esboco) => executar('pesquisar', { esboco })"
      />

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

      <CartoesDaReferencia ref="cartoes" :referencia="ref_" />

      <div v-if="ref_.estado === 'erro' || (!pronta && !andando && ref_.estado !== 'descartado' && ref_.estado !== 'esboco')" class="linha-acoes barra-acoes">
        <button class="botao principal" :disabled="ocupado || andando" @click="executar('reprocessar')">Tentar de novo</button>
      </div>

      <NovaPesquisa v-if="acao === 'aprofundar'" :referencia="ref_" :foco-inicial="focoDoAprofundar" @fechar="acao = null; focoDoAprofundar = ''" @enviado="acao = null; focoDoAprofundar = ''; carregar()" />

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

      <p v-if="doMenu" class="aba-do-menu">
        <strong>{{ doMenu }}</strong>
        <button class="botao discreto" @click="aba = null">Voltar ao texto</button>
      </p>

      <Duvidas v-if="abaAtual === 'duvidas'" :referencia="ref_" :texto="textoPrincipal" @perguntar="perguntarSobre" @aprofundar="aprofundarSobre" />
      <Perguntas v-else-if="abaAtual === 'perguntar'" :key="perguntasVersao" embutido :local="`referencias/${ref_.id}`" :titulo="ref_.titulo" />
      <p v-else-if="abaAtual === 'estudar'" class="suave nota">Em breve.</p>
      <ul v-else-if="abaAtual === 'historico'" class="historico">
        <li v-for="t in ref_.tarefas" :key="t.id">
          <span>{{ NOMES_TAREFA[t.tipo] || t.tipo }}</span>
          <span class="chip" :class="{ erro: t.estado === 'falhou', ok: t.estado === 'concluida' }">{{ NOMES_ESTADO_TAREFA[t.estado] || t.estado }}</span>
          <span class="suave">{{ new Date(t.criado_em).toLocaleString('pt-BR') }}<template v-if="duracao(t)"> · {{ duracao(t) }}</template></span>
          <span v-if="t.erro" :class="t.estado === 'falhou' ? 'erro-texto' : 'suave'">{{ t.erro }}</span>
        </li>
      </ul>
      <p v-else-if="abaAtual === 'organizado' && traduzida" class="suave nota">Versão original, só para consulta. No Acervo entra a versão em português.</p>
      <p v-else-if="abaAtual === 'anterior'" class="suave nota">O texto antes da última mudança (aprofundar a pesquisa ou integrar ao texto), só para consulta.</p>
      <template v-if="!semTexto(abaAtual)">
        <article ref="textoEl" class="texto" @click="detalhes.aoClicar" v-html="marcado.htmls[0]" />
        <CamadaDetalhes :detalhes="detalhes" :raizes="() => [textoEl]" :titulo="ref_.titulo" :sem-lugar="marcado.semLugar" />
      </template>
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

.titulo-linha {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  justify-content: space-between;
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

.aba-do-menu {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 0 0 12px;
}

.aba-do-menu .discreto {
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.88rem;
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
