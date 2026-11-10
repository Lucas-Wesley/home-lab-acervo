<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { api } from './api.js'
import Icone from './Icone.vue'

// Resumo de capítulos (D30, D32): uma página própria. Sem livro, a lista dos livros da Biblioteca
// (os que estão sendo lidos primeiro); com livro, o sumário para marcar os capítulos (ou as páginas,
// num PDF sem sumário). O texto vem da Biblioteca; a IA resume e sugere o Tema.
const props = defineProps({ livro: { type: String, default: '' } })

const MAX_CARACTERES = 600000
const LENDO_AGORA = 4

const livros = ref([])
const carregouLivros = ref(false)
const busca = ref('')
const sumario = ref(null)
const escolhidos = ref([])
const de = ref(1)
const ate = ref(1)
const foco = ref('')
const carregando = ref(false)
const enviando = ref(false)
const erro = ref('')

const capa = (id) => `/api/livros/${encodeURIComponent(id)}/capa`
const semAcento = (t) => (t || '').normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

const encontrados = computed(() => {
  const termo = semAcento(busca.value.trim())
  if (!termo) return livros.value
  return livros.value.filter((l) => semAcento(`${l.title} ${l.author}`).includes(termo))
})
// Os livros lidos por último, para não ter de procurar o que está aberto na Biblioteca.
const lendoAgora = computed(() => {
  if (busca.value.trim()) return []
  return livros.value
    .filter((l) => l.read_at && l.percent > 0 && l.percent < 1)
    .sort((a, b) => b.read_at.localeCompare(a.read_at))
    .slice(0, LENDO_AGORA)
})
const livroAtual = computed(() => livros.value.find((l) => l.id === props.livro) || sumario.value)

const porPaginas = computed(() => sumario.value?.indexed && !sumario.value.chapters.length)
const caracteres = computed(() => {
  const s = sumario.value
  if (!s) return 0
  if (porPaginas.value) return Math.max(0, ate.value - de.value + 1) * 1500
  return s.chapters.filter((c) => escolhidos.value.includes(c.n)).reduce((soma, c) => soma + c.chars, 0)
})
const grandeDemais = computed(() => caracteres.value > MAX_CARACTERES)
const podeEnviar = computed(() => {
  if (!sumario.value?.indexed || grandeDemais.value) return false
  if (porPaginas.value) return de.value >= 1 && ate.value >= de.value && ate.value <= sumario.value.pages
  return escolhidos.value.length > 0
})
const resumoDaEscolha = computed(() => {
  if (porPaginas.value) return `p. ${de.value} a ${ate.value} · ${tamanho(caracteres.value)}`
  const n = escolhidos.value.length
  if (!n) return 'Marque um ou mais capítulos'
  return `${n} ${n === 1 ? 'capítulo' : 'capítulos'} · ${tamanho(caracteres.value)}`
})

// "≈ 12 páginas": a régua das páginas da Biblioteca, 1.500 caracteres.
function tamanho(chars) {
  const paginas = Math.max(1, Math.round(chars / 1500))
  return `≈ ${paginas} ${paginas === 1 ? 'página' : 'páginas'}`
}

const formato = (l) => (l?.format || '').toUpperCase()
const autorEFormato = (l) => [l.author, l.percent > 0 ? `${Math.round(l.percent * 100)}% lido` : ''].filter(Boolean).join(' · ')

function marcar(c) {
  const i = escolhidos.value.indexOf(c.n)
  if (i >= 0) escolhidos.value.splice(i, 1)
  else escolhidos.value.push(c.n)
}

const todosMarcados = computed(() => sumario.value?.chapters.length > 0 && escolhidos.value.length === sumario.value.chapters.length)
function marcarTodos() {
  escolhidos.value = todosMarcados.value ? [] : sumario.value.chapters.map((c) => c.n)
}

function semCapa(evento) {
  evento.target.closest('.capa')?.classList.add('vazia')
}

async function carregarLivros() {
  try {
    livros.value = await api('/api/livros')
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão: os livros só abrem na rede de casa.' : e.message
  }
  carregouLivros.value = true
}

async function carregarSumario() {
  sumario.value = null
  escolhidos.value = []
  foco.value = ''
  erro.value = ''
  if (!props.livro) return
  carregando.value = true
  try {
    sumario.value = await api(`/api/livros/${encodeURIComponent(props.livro)}`)
    de.value = 1
    ate.value = Math.min(sumario.value.pages || 1, 20)
  } catch (e) {
    erro.value = e.message
  } finally {
    carregando.value = false
  }
}

async function resumir() {
  enviando.value = true
  erro.value = ''
  try {
    const corpo = porPaginas.value
      ? { livro: props.livro, de: de.value, ate: ate.value, foco: foco.value }
      : { livro: props.livro, capitulos: escolhidos.value, foco: foco.value }
    const ref_ = await api('/api/livros', { metodo: 'POST', corpo })
    location.hash = `#/referencia/${ref_.id}`
  } catch (e) {
    erro.value = e.message
  } finally {
    enviando.value = false
  }
}

watch(() => props.livro, () => {
  carregarSumario()
  window.scrollTo(0, 0)
})
onMounted(() => {
  carregarLivros()
  carregarSumario()
})
</script>

<template>
  <div class="pagina resumir">
    <!-- Lista dos livros -->
    <template v-if="!livro">
      <a href="#/referencias" class="voltar"><Icone nome="voltar" /> Referências</a>
      <h1>Resumir capítulos</h1>
      <p class="suave apresentacao">Escolha um livro da Biblioteca e marque os capítulos: a IA resume o que eles ensinam, com as páginas de cada parte, e sugere o Tema.</p>

      <input v-if="livros.length > 6" v-model="busca" class="campo busca" type="search" placeholder="Buscar por título ou autor" aria-label="Buscar livro" />
      <div v-if="erro" class="aviso erro">{{ erro }}</div>

      <template v-if="lendoAgora.length">
        <div class="rotulo">Lendo agora</div>
        <div class="lista">
          <a v-for="l in lendoAgora" :key="l.id" :href="`#/resumir/${encodeURIComponent(l.id)}`" class="livro">
            <span class="capa" :class="{ vazia: !l.has_cover }">
              <img v-if="l.has_cover" :src="capa(l.id)" alt="" loading="lazy" @error="semCapa" />
              <Icone nome="livro" />
            </span>
            <span class="info">
              <strong>{{ l.title }}</strong>
              <span class="meta">{{ autorEFormato(l) }}</span>
            </span>
            <span class="formato">{{ formato(l) }}</span>
            <Icone nome="avancar" class="seta" />
          </a>
        </div>
      </template>

      <div v-if="lendoAgora.length && encontrados.length" class="rotulo">Todos os livros</div>
      <div v-if="encontrados.length" class="lista" :class="{ 'sem-rotulo': !lendoAgora.length }">
        <a v-for="l in encontrados" :key="l.id" :href="`#/resumir/${encodeURIComponent(l.id)}`" class="livro">
          <span class="capa" :class="{ vazia: !l.has_cover }">
            <img v-if="l.has_cover" :src="capa(l.id)" alt="" loading="lazy" @error="semCapa" />
            <Icone nome="livro" />
          </span>
          <span class="info">
            <strong>{{ l.title }}</strong>
            <span class="meta">{{ autorEFormato(l) }}</span>
          </span>
          <span class="formato">{{ formato(l) }}</span>
          <Icone nome="avancar" class="seta" />
        </a>
      </div>
      <div v-else-if="carregouLivros && !erro" class="vazio">
        {{ busca.trim() ? 'Nenhum livro com esse título ou autor.' : 'Nenhum livro na Biblioteca ainda.' }}
      </div>
    </template>

    <!-- Capítulos de um livro -->
    <template v-else>
      <a href="#/resumir" class="voltar"><Icone nome="voltar" /> Livros</a>

      <header v-if="livroAtual" class="livro-escolhido">
        <span class="capa grande" :class="{ vazia: livroAtual.has_cover === false }">
          <img :src="capa(livro)" alt="" @error="semCapa" />
          <Icone nome="livro" />
        </span>
        <div class="info">
          <h1>{{ livroAtual.title }}</h1>
          <span class="suave">{{ livroAtual.author }}</span>
          <span class="meta">
            <span class="formato">{{ formato(livroAtual) }}</span>
            <template v-if="sumario?.pages"> · {{ sumario.pages }} páginas</template>
            <template v-if="sumario?.chapters?.length"> · {{ sumario.chapters.length }} capítulos</template>
          </span>
        </div>
      </header>

      <p v-if="carregando" class="suave">Lendo o sumário…</p>
      <div v-if="erro" class="aviso erro">{{ erro }}</div>
      <div v-if="sumario && !sumario.indexed" class="aviso">O texto deste livro ainda não está disponível: {{ sumario.error }}</div>

      <template v-else-if="sumario && !porPaginas">
        <div class="cabecalho-capitulos">
          <div class="rotulo">Capítulos</div>
          <button type="button" class="link-acao" @click="marcarTodos">{{ todosMarcados ? 'Desmarcar todos' : 'Marcar todos' }}</button>
        </div>
        <ul class="lista capitulos">
          <li v-for="c in sumario.chapters" :key="c.n">
            <label class="capitulo" :class="{ marcado: escolhidos.includes(c.n) }" :style="{ paddingLeft: `${(c.level - 1) * 20}px` }">
              <input type="checkbox" :checked="escolhidos.includes(c.n)" @change="marcar(c)" />
              <span class="info">
                <span class="titulo-capitulo" :class="{ secundario: c.level > 1 }">{{ c.title }}</span>
                <span class="meta">p. {{ c.first_page }}{{ c.last_page > c.first_page ? `–${c.last_page}` : '' }} · {{ tamanho(c.chars) }}</span>
              </span>
            </label>
          </li>
        </ul>
      </template>

      <template v-else-if="porPaginas">
        <div class="rotulo">Páginas</div>
        <p class="suave dica">Este livro não tem sumário no arquivo: escolha as páginas do capítulo (o livro tem {{ sumario.pages }}).</p>
        <div class="paginas">
          <label>De <input v-model.number="de" class="campo" type="number" min="1" :max="sumario.pages" /></label>
          <label>até <input v-model.number="ate" class="campo" type="number" :min="de" :max="sumario.pages" /></label>
        </div>
      </template>

      <template v-if="sumario?.indexed">
        <label class="foco">
          <span class="rotulo">O que mais te interessa? <span class="opcional">(opcional)</span></span>
          <textarea v-model="foco" class="campo" rows="2" maxlength="500" placeholder="Ex.: as técnicas de fermentação" />
        </label>

        <form class="barra-resumir" @submit.prevent="resumir">
          <span class="escolha" :class="{ 'erro-texto': grandeDemais }">
            {{ grandeDemais ? 'Texto demais para um resumo só: marque menos capítulos.' : resumoDaEscolha }}
          </span>
          <button class="botao principal" :disabled="enviando || !podeEnviar">{{ enviando ? 'Enviando…' : 'Resumir' }}</button>
        </form>
      </template>
    </template>
  </div>
</template>

<style scoped>
.voltar {
  display: inline-flex;
  gap: 2px;
  align-items: center;
  margin: 0 0 14px -4px;
  color: var(--texto-suave);
  font-size: 0.85rem;
  text-decoration: none;
}

.voltar:hover {
  color: var(--texto);
}

.voltar :deep(svg) {
  width: 16px;
  height: 16px;
}

.resumir h1 {
  margin-bottom: 8px;
}

.apresentacao {
  margin: 0 0 20px;
  font-size: 0.9rem;
  line-height: 1.5;
}

.busca {
  width: 100%;
  margin-bottom: 4px;
}

.lista.sem-rotulo {
  margin-top: 16px;
}

/* Linha de livro: capa pequena, título e autor, formato e a seta. */
.livro {
  display: flex;
  gap: 14px;
  align-items: center;
  padding: 10px 0;
  color: inherit;
  text-decoration: none;
}

.livro:hover strong {
  color: var(--azul);
}

.capa {
  position: relative;
  flex: none;
  display: grid;
  place-items: center;
  width: 36px;
  height: 52px;
  overflow: hidden;
  border-radius: 3px;
  background: var(--superficie-forte);
  color: var(--texto-fraco);
}

.capa img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.capa :deep(svg) {
  width: 18px;
  height: 18px;
}

.capa.vazia img {
  display: none;
}

.capa.grande {
  width: 64px;
  height: 92px;
  border-radius: 4px;
}

.capa.grande :deep(svg) {
  width: 26px;
  height: 26px;
}

.info {
  display: grid;
  gap: 3px;
  min-width: 0;
  flex: 1;
}

.livro strong {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  font-size: 0.9rem;
  font-weight: 600;
  line-height: 1.35;
  transition: color 0.15s ease;
}

.meta {
  color: var(--texto-fraco);
  font-size: 0.78rem;
}

.formato {
  flex: none;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--superficie-forte);
  color: var(--texto-suave);
  font-size: 0.68rem;
  font-weight: 600;
  letter-spacing: 0.04em;
}

.seta {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--texto-fraco);
}

/* Livro escolhido */
.livro-escolhido {
  display: flex;
  gap: 16px;
  align-items: flex-end;
  margin-bottom: 8px;
}

.livro-escolhido h1 {
  margin: 0;
  font-family: var(--leitura);
  font-size: 1.45rem;
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.01em;
}

.livro-escolhido .info {
  gap: 4px;
}

.livro-escolhido .suave {
  font-size: 0.88rem;
}

.cabecalho-capitulos {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.link-acao {
  padding: 0;
  border: 0;
  background: none;
  color: var(--azul);
  font-size: 0.8rem;
  cursor: pointer;
}

.capitulos {
  margin-bottom: 8px;
}

.capitulo {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 11px 8px;
  border-radius: var(--raio-p);
  cursor: pointer;
  transition: background 0.15s ease;
}

.capitulo:hover {
  background: var(--superficie);
}

.capitulo.marcado {
  background: var(--azul-fundo);
}

.capitulo input {
  flex: none;
  width: 16px;
  height: 16px;
  margin: 2px 0 0;
  accent-color: var(--azul);
}

.titulo-capitulo {
  font-size: 0.9rem;
  line-height: 1.4;
}

.titulo-capitulo.secundario {
  color: var(--texto-suave);
}

.paginas {
  display: flex;
  gap: 12px;
  align-items: center;
  margin-bottom: 8px;
  font-size: 0.875rem;
}

.paginas label {
  display: flex;
  gap: 6px;
  align-items: center;
}

.paginas .campo {
  width: 6rem;
}

.dica {
  margin: 0 0 12px;
  font-size: 0.82rem;
  line-height: 1.45;
}

.foco {
  display: grid;
}

/* Rótulo de campo, não título de seção. */
.foco .rotulo {
  margin: 20px 0 8px;
  color: var(--texto-suave);
  font-size: 0.82rem;
  font-weight: 600;
  letter-spacing: 0;
}

.opcional {
  font-weight: 400;
  letter-spacing: 0;
  text-transform: none;
}

.foco textarea {
  resize: vertical;
  font: inherit;
  line-height: 1.45;
}

/* Barra de vidro que acompanha a rolagem: o que foi marcado e o botão. */
.barra-resumir {
  position: sticky;
  bottom: 0;
  z-index: 3;
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin: 20px calc(-1 * var(--margem)) 0;
  padding: 12px var(--margem) max(12px, env(safe-area-inset-bottom));
  border-top: 1px solid var(--linha);
  background: var(--vidro);
  -webkit-backdrop-filter: var(--blur);
  backdrop-filter: var(--blur);
}

.escolha {
  font-size: 0.85rem;
  color: var(--texto-suave);
}

.erro-texto {
  color: var(--vermelho);
}

@media (max-width: 820px) {
  .barra-resumir {
    /* logo acima das abas de vidro (55 px + o respiro do pé da tela) */
    bottom: calc(55px + max(6px, env(safe-area-inset-bottom)));
    padding-bottom: 12px;
  }
}
</style>
