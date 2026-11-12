<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { api, nomeDoEstado, situacao } from './api.js'
import Icone from './Icone.vue'
import Marca from './Marca.vue'
import { descreverLivro, descreverPesquisa } from './pesquisa.js'

const referencias = ref([])
const carregou = ref(false)
const link = ref('')
const enviando = ref(false)
const erro = ref('')
const filtro = ref('todas')
const novo = ref('') // 'video', 'reel', 'podcast' ou 'artigo': o campo do link aberto pelo menu ⋯
const menuAberto = ref(false)
const campo = ref(null)

// D34: as entradas novas ficam no menu ⋯, para a lista vir primeiro.
const ENTRADAS = {
  video: { placeholder: 'Cole o link de um vídeo do YouTube', rotulo: 'Link do vídeo' },
  reel: { placeholder: 'Cole o link de um Reel do Instagram', rotulo: 'Link do Reel' },
  podcast: { placeholder: 'Cole o link de um episódio do Spotify ou do Apple Podcasts', rotulo: 'Link do episódio' },
  artigo: { placeholder: 'Cole o link de um artigo, blog ou site', rotulo: 'Link da página' },
}

async function abrir(tipo) {
  menuAberto.value = false
  novo.value = tipo
  erro.value = ''
  await nextTick()
  campo.value?.focus()
}

function fecharMenu(e) {
  if (e.type === 'keydown' ? e.key === 'Escape' : !e.target.closest?.('.menu-acoes')) menuAberto.value = false
}

const FILTROS = {
  todas: () => true,
  andamento: (r) => r.tarefa && r.tarefa.estado !== 'em_revisao',
  prontas: (r) => r.estado === 'organizado' || r.estado === 'validado',
  integradas: (r) => r.estado === 'integrado',
  erro: (r) => r.estado === 'erro' || r.erro,
}
const visiveis = computed(() => referencias.value.filter((r) => r.estado !== 'descartado' || filtro.value === 'descartadas').filter(FILTROS[filtro.value] ?? ((r) => r.estado === 'descartado')))

// Documentário em episódios (D50): os episódios da mesma série viram um item só, no lugar do mais
// recente, com os episódios em ordem embaixo.
const itens = computed(() => {
  const lista = []
  const grupos = {}
  for (const r of visiveis.value) {
    const d = r.tipo === 'documentario' ? r.documentario : null
    if (!d?.episodio) {
      lista.push(r)
      continue
    }
    if (!grupos[d.chave]) {
      grupos[d.chave] = { grupo: true, chave: d.chave, nome: d.nome, ano: d.ano, refs: [] }
      lista.push(grupos[d.chave])
    }
    grupos[d.chave].refs.push(r)
  }
  for (const g of Object.values(grupos)) {
    g.refs.sort((a, b) => a.documentario.temporada - b.documentario.temporada || a.documentario.episodio - b.documentario.episodio)
    g.capa = g.refs.find((r) => r.capa)?.id || ''
    g.andando = g.refs.filter((r) => r.tarefa && r.tarefa.estado !== 'em_revisao').length
  }
  return lista
})

let relogio
async function carregar() {
  try {
    referencias.value = await api('/api/referencias')
    erro.value = ''
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: as Referências só abrem na rede de casa.' : e.message
  }
  carregou.value = true
}

async function adicionar() {
  if (!link.value.trim()) return
  enviando.value = true
  erro.value = ''
  try {
    await api('/api/referencias', { metodo: 'POST', corpo: { url: link.value, tipo: novo.value } })
    link.value = ''
    novo.value = ''
    await carregar()
  } catch (e) {
    if (e.status === 409 && e.dados?.id) location.hash = `#/referencia/${e.dados.id}`
    else erro.value = e.message
  } finally {
    enviando.value = false
  }
}

// marca: o Reel é do Instagram; o podcast, do Spotify quando veio de lá (D46).
function marca(r) {
  if (r.tipo === 'reel') return 'instagram'
  if (r.tipo === 'podcast' && r.url.includes('spotify.com')) return 'spotify'
  return ''
}

function classeDoEstado(r) {
  if (r.estado === 'erro') return 'erro'
  if (r.estado === 'integrado') return 'ok'
  return ''
}

onMounted(() => {
  carregar()
  relogio = setInterval(() => document.visibilityState === 'visible' && carregar(), 3000)
  document.addEventListener('click', fecharMenu)
  document.addEventListener('keydown', fecharMenu)
})
onBeforeUnmount(() => {
  clearInterval(relogio)
  document.removeEventListener('click', fecharMenu)
  document.removeEventListener('keydown', fecharMenu)
})
</script>

<template>
  <div class="pagina">
    <div class="cabecalho">
      <h1>Referências</h1>
      <div class="menu-acoes">
        <button class="botao icone" aria-label="Mais opções" title="Mais opções" aria-haspopup="menu" :aria-expanded="menuAberto" @click="menuAberto = !menuAberto">
          <Icone nome="pontos" />
        </button>
        <div v-if="menuAberto" class="painel-menu" role="menu">
          <button role="menuitem" @click="abrir('video')"><Icone nome="referencias" /> Link de vídeo</button>
          <button role="menuitem" @click="abrir('reel')"><Icone nome="reel" /> Reel do Instagram</button>
          <button role="menuitem" @click="abrir('podcast')"><Icone nome="podcast" /> Episódio de podcast</button>
          <button role="menuitem" @click="abrir('artigo')"><Icone nome="link" /> Link de artigo ou site</button>
          <hr />
          <a role="menuitem" href="#/pesquisar"><Icone nome="pesquisar" /> Pesquisar um tema</a>
          <a role="menuitem" href="#/resumir"><Icone nome="livro" /> Resumir capítulos</a>
        </div>
      </div>
    </div>

    <form v-if="novo" class="adicionar" @submit.prevent="adicionar">
      <input ref="campo" v-model="link" class="campo" type="url" inputmode="url" :placeholder="ENTRADAS[novo].placeholder" :aria-label="ENTRADAS[novo].rotulo" />
      <button class="botao principal" :disabled="enviando || !link.trim()">{{ enviando ? 'Enviando…' : 'Adicionar' }}</button>
      <button type="button" class="botao icone" aria-label="Cancelar" title="Cancelar" @click="novo = ''; link = ''; erro = ''"><Icone nome="fechar" /></button>
    </form>
    <div v-if="erro" class="aviso erro">{{ erro }}</div>

    <div class="filtros">
      <button v-for="(nome, chave) in { todas: 'Todas', andamento: 'Em andamento', prontas: 'Prontas', integradas: 'Integradas', erro: 'Com erro', descartadas: 'Descartadas' }" :key="chave" class="chip-filtro" :class="{ ativo: filtro === chave }" @click="filtro = chave">{{ nome }}</button>
    </div>

    <div v-if="carregou && !visiveis.length" class="vazio">
      {{ referencias.length ? 'Nada neste filtro.' : 'Nenhuma Referência ainda. No ⋯ acima, mande o link de um vídeo, de um Reel, de um podcast ou de um artigo, pesquise um tema ou resuma capítulos de um livro: tudo é organizado sozinho, na fila.' }}
    </div>

    <div v-if="visiveis.length" class="lista">
    <template v-for="r in itens" :key="r.grupo ? `g:${r.chave}` : r.id">
    <div v-if="r.grupo" class="grupo-doc">
      <div class="referencia cabeca-grupo">
        <span v-if="r.capa" class="miniatura com-capa" aria-hidden="true"><img :src="`/api/referencias/${r.capa}/capa`" alt="" loading="lazy" /></span>
        <span v-else class="miniatura pesquisa" aria-hidden="true"><Icone nome="documentario" /></span>
        <div class="info">
          <strong>{{ r.nome }}</strong>
          <span class="suave meta">{{ ['Documentário', r.ano, `${r.refs.length} ${r.refs.length === 1 ? 'episódio' : 'episódios'}`].filter(Boolean).join(' · ') }}</span>
          <span v-if="r.andando" class="linha-estado"><span class="chip andando">{{ r.andando === 1 ? '1 episódio em andamento' : `${r.andando} episódios em andamento` }}</span></span>
        </div>
      </div>
      <a v-for="ep in r.refs" :key="ep.id" :href="`#/referencia/${ep.id}`" class="episodio-doc">
        <span class="num">T{{ ep.documentario.temporada }} E{{ ep.documentario.episodio }}</span>
        <span class="titulo-ep">{{ ep.documentario.titulo_episodio || `Episódio ${ep.documentario.episodio}` }}</span>
        <span class="chip" :class="classeDoEstado(ep)">{{ situacao(ep) || nomeDoEstado(ep) }}</span>
        <span v-if="ep.erro" class="erro-texto">{{ ep.erro }}</span>
      </a>
    </div>
    <a v-else :href="`#/referencia/${r.id}`" class="referencia">
      <span v-if="r.tipo === 'pesquisa'" class="miniatura pesquisa" aria-hidden="true"><Icone nome="pesquisar" /></span>
      <span v-else-if="r.tipo === 'livro'" class="miniatura pesquisa" aria-hidden="true"><Icone nome="livro" /></span>
      <span v-else-if="r.tipo === 'artigo'" class="miniatura pesquisa" aria-hidden="true"><Icone nome="link" /></span>
      <span v-else-if="r.capa" class="miniatura com-capa" aria-hidden="true">
        <img :src="`/api/referencias/${r.id}/capa`" alt="" loading="lazy" />
        <span v-if="marca(r)" class="selo" :class="marca(r)"><Marca :nome="marca(r)" /></span>
      </span>
      <span v-else-if="marca(r)" class="miniatura marca" :class="marca(r)" aria-hidden="true"><Marca :nome="marca(r)" /></span>
      <span v-else-if="r.tipo === 'podcast'" class="miniatura pesquisa" aria-hidden="true"><Icone nome="podcast" /></span>
      <span v-else-if="r.tipo === 'documentario'" class="miniatura pesquisa" aria-hidden="true"><Icone nome="documentario" /></span>
      <img v-else :src="`https://i.ytimg.com/vi/${r.video_id}/mqdefault.jpg`" alt="" class="miniatura" loading="lazy" />
      <div class="info">
        <strong>{{ r.titulo || r.url }}</strong>
        <span class="suave meta">{{ r.tipo === 'pesquisa' ? descreverPesquisa(r.pesquisa) : r.tipo === 'livro' ? descreverLivro(r.livro) : [r.canal, r.duracao].filter(Boolean).join(' · ') }}</span>
        <span class="linha-estado">
          <span class="chip" :class="classeDoEstado(r)">{{ nomeDoEstado(r) }}</span>
          <span v-if="situacao(r)" class="chip andando">{{ situacao(r) }}</span>
        </span>
        <span v-if="r.erro" class="erro-texto">{{ r.erro }}</span>
      </div>
    </a>
    </template>
    </div>
  </div>
</template>

<style scoped>
.cabecalho {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  justify-content: space-between;
}

.adicionar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.adicionar .campo {
  flex: 1;
  min-width: 0;
}

.aviso {
  margin-bottom: 12px;
}

.filtros {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  margin: 8px 0 16px;
  padding-bottom: 2px;
  scrollbar-width: none;
}

.chip-filtro {
  padding: 4px 11px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.82rem;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.chip-filtro:hover {
  color: var(--texto);
}

.chip-filtro.ativo {
  background: var(--superficie-forte);
  color: var(--texto);
  font-weight: 600;
}

.referencia {
  display: flex;
  gap: 14px;
  padding: 14px 0;
  color: inherit;
  text-decoration: none;
}

.referencia:hover strong {
  color: var(--azul);
}

.miniatura {
  flex: none;
  width: 96px;
  height: 54px;
  border-radius: 6px;
  object-fit: cover;
  background: var(--superficie-forte);
}

.miniatura.pesquisa {
  display: grid;
  place-items: center;
  color: var(--azul);
  background: var(--azul-fundo);
}

.miniatura.pesquisa :deep(svg) {
  width: 22px;
  height: 22px;
}

/* Documentário em episódios (D50): o cabeçalho e os episódios em linhas finas. */
.grupo-doc {
  padding-bottom: 8px;
}

.cabeca-grupo {
  padding-bottom: 6px;
}

.episodio-doc {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  align-items: baseline;
  margin-left: 110px;
  padding: 6px 0;
  border-top: 1px solid var(--linha);
  color: inherit;
  font-size: 0.9rem;
  text-decoration: none;
}

.episodio-doc:hover .titulo-ep {
  color: var(--azul);
}

.episodio-doc .num {
  color: var(--texto-suave);
  font-variant-numeric: tabular-nums;
}

.episodio-doc .titulo-ep {
  flex: 1;
  min-width: 0;
}

.episodio-doc .erro-texto {
  flex-basis: 100%;
}

@media (max-width: 520px) {
  .episodio-doc {
    margin-left: 0;
  }
}

/* Reel e podcast (D46): a capa, com o selo de onde veio; sem capa, o símbolo nas cores da marca. */
.miniatura.com-capa {
  position: relative;
  overflow: hidden;
}

.com-capa img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.selo {
  position: absolute;
  left: 4px;
  bottom: 4px;
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border-radius: 5px;
}

.selo svg {
  width: 13px;
  height: 13px;
}

.selo.spotify {
  width: auto;
  height: auto;
  border-radius: 50%;
  background: #121212;
  padding: 1px;
}

.selo.spotify svg {
  width: 16px;
  height: 16px;
}

.miniatura.marca {
  display: grid;
  place-items: center;
}

.miniatura.marca svg {
  width: 26px;
  height: 26px;
}

.instagram {
  color: #fff;
  background: radial-gradient(circle at 30% 107%, #fdf497 0%, #fdf497 5%, #fd5949 45%, #d6249f 60%, #285aeb 90%);
}

.miniatura.spotify {
  background: #121212;
}

.info {
  display: grid;
  gap: 4px;
  min-width: 0;
  align-content: start;
}

.info strong {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-weight: 600;
  line-height: 1.35;
  transition: color 0.15s ease;
}

.meta {
  color: var(--texto-fraco);
  font-size: 0.78rem;
}

.linha-estado {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.erro-texto {
  color: var(--vermelho);
  font-size: 0.82rem;
  line-height: 1.4;
}

@media (max-width: 520px) {
  .miniatura {
    width: 80px;
    height: 45px;
  }
}
</style>
