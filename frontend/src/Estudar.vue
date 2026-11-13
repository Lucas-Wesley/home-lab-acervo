<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Exercicios from './Exercicios.vue'
import GerarMais from './GerarMais.vue'
import Icone from './Icone.vue'
import Info from './Info.vue'
import Simulado from './Simulado.vue'
import { api } from './api.js'
import { ler, paraFala, temLeitura, trechos } from './leitura.js'
import { renderizar } from './markdown.js'
import { enderecoDePesquisa } from './pesquisa.js'

// Aba Estudar de um Tema ou de uma Referência (D41): resumo inteligente, conceitos anteriores,
// exercícios (D52), simulado e flashcards. Cada parte é preparada pela IA na fila, a partir do Tema inteiro (com os
// subtemas) ou do texto da Referência, e fica guardada; quando o texto muda, a parte avisa.
// A aba abre só com os cartões; o escolhido abre embaixo, e os exercícios e o simulado, em tela
// própria, porque responder pede foco e a tela inteira. Os dois crescem com "gerar mais" (D52).
const props = defineProps({
  local: { type: String, required: true }, // temas/<pasta> ou referencias/<id>
  titulo: { type: String, required: true },
  flashcards: { type: Boolean, default: false }, // o Flashcards está no ar
})
const emit = defineEmits(['flashcards'])

const ehReferencia = computed(() => props.local.startsWith('referencias/'))
const doTexto = computed(() => (ehReferencia.value ? 'o texto' : 'o Tema'))

const SECOES = computed(() => [
  { id: 'resumo', nome: 'Resumo inteligente', icone: 'resumo', descricao: `${ehReferencia.value ? 'O texto' : 'O Tema'} inteiro em poucos minutos: a ideia central, o essencial, como as partes se ligam e o que costuma confundir.` },
  { id: 'anteriores', nome: 'Conceitos anteriores', icone: 'camadas', descricao: `O que ajuda a entender ${doTexto.value} e não está explicado nele: a base que ele supõe e os assuntos ao redor.` },
  { id: 'exercicios', nome: 'Exercícios', icone: 'exercicios', descricao: `Prática com resposta certa: completar a frase ou escolher a opção, num caso novo. A correção é na hora, com o porquê. Os que você nunca fez e os que errou vêm primeiro; "Gerar mais" traz um lote novo, com o foco que você quiser.` },
  { id: 'simulado', nome: 'Simulado', icone: 'simulado', descricao: `Perguntas de teoria ${ehReferencia.value ? 'do texto' : 'do Tema'}, sobretudo o porquê, para ver se você entendeu. Responda escrevendo ou falando; a IA corrige. As que nunca saíram e as que você errou vêm primeiro; "Gerar mais" acrescenta perguntas novas ao banco.` },
  { id: 'flashcards', nome: 'Flashcards', icone: 'cartoes', descricao: `Padrão: perguntas de resposta curta. Leitura: em geral um cartão só, com um resumo ${ehReferencia.value ? 'do texto' : 'do Tema'} para reler; mais de um só se ${ehReferencia.value ? 'ele' : 'o Tema'} muda de assunto. A proposta vai para o app Flashcards, e você aprova antes de os cartões entrarem no baralho.` },
])
const PREPARAR = { resumo: 'Preparar o resumo', anteriores: 'Achar os conceitos anteriores', exercicios: 'Preparar os exercícios', simulado: 'Preparar as perguntas' }
const TELA_PROPRIA = ['exercicios', 'simulado']

const dados = ref(null) // { partes: { resumo, anteriores, simulado }, simulados }
const erro = ref('')
const pedindo = ref('')
const aberta = ref('') // a parte aberta; vazio = só os cartões
let espera = null

const partes = computed(() => dados.value?.partes ?? {})
const secao = computed(() => SECOES.value.find((s) => s.id === aberta.value))
const parte = computed(() => partes.value[aberta.value])
const processando = computed(() => Object.values(partes.value).some((p) => p.estado === 'processando') || dados.value?.simulados?.some((s) => s.estado === 'processando'))
const conceitos = computed(() => partes.value.anteriores?.resultado?.conceitos ?? [])
const banco = computed(() => partes.value.simulado?.resultado?.perguntas ?? [])
const exercicios = computed(() => partes.value.exercicios?.resultado?.exercicios ?? [])
const historico = computed(() => dados.value?.historico ?? { perguntas: {}, exercicios: {} })
const origemDoTexto = computed(() => (ehReferencia.value ? { referencia: props.local.slice('referencias/'.length) } : { arquivo: `${props.local}/indice.md`, capitulo: '' }))
const textoResumo = computed(() => partes.value.resumo?.resultado?.texto ?? '')

function mensagem(e) {
  return e.status === 0 ? 'Estudar precisa da rede de casa.' : e.message
}

// O que o cartão diz sobre a parte: pronta (e quanto tem), preparando, com erro ou por fazer.
function situacao(id) {
  if (id === 'flashcards') return props.flashcards ? { texto: 'Padrão ou leitura' } : { texto: 'O Flashcards não respondeu', classe: 'com-erro' }
  const p = partes.value[id]
  if (!p) return { texto: '' }
  if (p.estado === 'processando') return { texto: p.resultado ? 'Refazendo…' : 'Preparando…', classe: 'na-fila' }
  if (p.estado === 'falhou' && !p.resultado) return { texto: 'Falhou: tente de novo', classe: 'com-erro' }
  if (!p.resultado) return { texto: 'Ainda não preparado', classe: 'por-fazer' }
  let texto = ''
  if (id === 'resumo') texto = `${Math.max(1, Math.round(textoResumo.value.split(/\s+/).length / 200))} min de leitura`
  if (id === 'anteriores') texto = `${conceitos.value.length} ${conceitos.value.length === 1 ? 'conceito' : 'conceitos'}`
  if (id === 'simulado') {
    const nota = dados.value.simulados.find((s) => s.correcao)?.correcao.nota
    texto = `${banco.value.length} perguntas${nota !== undefined ? ` · última nota ${nota}` : ''}`
  }
  if (id === 'exercicios') {
    const feitos = exercicios.value.filter((x) => historico.value.exercicios[x.id]?.vezes)
    const certos = feitos.filter((x) => historico.value.exercicios[x.id].ultima === 'certo').length
    texto = `${exercicios.value.length} exercícios${feitos.length ? ` · ${certos} de ${feitos.length} certos` : ''}`
  }
  return p.desatualizado ? { texto: `${texto} · o texto mudou`, classe: 'mudou' } : { texto }
}

async function carregar() {
  try {
    dados.value = await api(`/api/estudos?tema=${encodeURIComponent(props.local)}`)
    erro.value = ''
  } catch (e) {
    erro.value = mensagem(e)
  }
  clearTimeout(espera)
  if (processando.value) espera = setTimeout(carregar, 4000)
}

async function preparar(id, extra = {}) {
  // Refazer o simulado ou os exercícios troca o banco inteiro: confirmar.
  const quantos = id === 'simulado' ? banco.value.length : id === 'exercicios' ? exercicios.value.length : 0
  if (!extra.mais && quantos && !confirm(`Trocar ${id === 'simulado' ? `as ${quantos} perguntas` : `os ${quantos} exercícios`} por um banco novo? Para acrescentar sem perder nada, use "Gerar mais".`)) return
  pedindo.value = id
  erro.value = ''
  try {
    await api('/api/estudos', { metodo: 'POST', corpo: { tema: props.local, parte: id, ...extra } })
    await carregar()
  } catch (e) {
    erro.value = mensagem(e)
  } finally {
    pedindo.value = ''
  }
}

const raiz = ref(null)
function abrir(id) {
  const propria = TELA_PROPRIA.includes(id)
  aberta.value = aberta.value === id && !propria ? '' : id
  if (propria) raiz.value?.scrollIntoView({ block: 'start' })
}

// ---------- Ouvir o resumo ----------

const leitura = ref({ estado: 'parado', trecho: 0, total: 0 }) // parado, lendo ou pausado
const erroLeitura = ref('')
const trechosDoResumo = computed(() => trechos(paraFala(textoResumo.value)))
let pararLeitura = null

function ouvir(inicio = 0) {
  erroLeitura.value = ''
  const lista = trechosDoResumo.value
  leitura.value = { estado: 'lendo', trecho: inicio, total: lista.length }
  pararLeitura = ler(lista, {
    inicio,
    aoMudar: (i) => (leitura.value.trecho = i),
    aoTerminar: (msg) => {
      pararLeitura = null
      leitura.value = { estado: 'parado', trecho: 0, total: lista.length }
      erroLeitura.value = msg
    },
  })
}

function pausar() {
  const onde = pararLeitura?.() ?? 0
  pararLeitura = null
  leitura.value = { ...leitura.value, estado: 'pausado', trecho: onde }
}

function parar() {
  pararLeitura?.()
  pararLeitura = null
  leitura.value = { estado: 'parado', trecho: 0, total: 0 }
}

// Sair do resumo (ou refazê-lo) para a leitura.
watch([aberta, textoResumo], parar)

// Flashcards do Tema (ou do texto da Referência) inteiro: padrão (perguntas) ou de leitura.
function gerarCartoes(tipo) {
  emit('flashcards', { origem: origemDoTexto.value, titulo: tipo === 'leitura' ? 'de leitura' : '', tipo })
}

// Flashcards de um conceito só (D49): a explicação dele vira os cartões.
function cartoesDoConceito(c) {
  emit('flashcards', { origem: { estudo: props.local, parte: `anteriores:${c.nome}` }, titulo: `de “${c.nome}”`, tipo: 'basico' })
}

function quando(iso) {
  return iso ? new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' }) : ''
}

onMounted(carregar)
onBeforeUnmount(() => {
  clearTimeout(espera)
  parar()
})
</script>

<template>
  <div ref="raiz" class="estudar">
    <div v-if="erro" class="aviso erro">{{ erro }}</div>
    <p v-if="!dados && !erro" class="suave">Carregando…</p>

    <template v-if="dados">
      <!-- Os exercícios e o simulado, em tela própria -->
      <section v-if="TELA_PROPRIA.includes(aberta)" class="parte tela-simulado">
        <button class="botao discreto voltar-estudar" @click="aberta = ''"><Icone nome="voltar" /> Estudar</button>
        <header>
          <h2>{{ secao.nome }} <Info>{{ secao.descricao }}</Info></h2>
        </header>
        <p v-if="parte.estado === 'processando'" class="chip andando">{{ parte.resultado ? 'Atualizando o banco…' : 'Preparando…' }} (na fila)</p>
        <p v-else-if="parte.estado === 'falhou'" class="aviso erro">{{ parte.erro }}</p>
        <p v-if="parte.desatualizado && parte.estado !== 'processando'" class="aviso">{{ ehReferencia ? 'O texto' : 'O Tema' }} mudou desde que {{ aberta === 'simulado' ? 'as perguntas foram preparadas' : 'os exercícios foram preparados' }}. Refaça para valer o texto de agora.</p>
        <Simulado
          v-if="aberta === 'simulado' && banco.length"
          :tema="local"
          :banco="banco"
          :simulados="dados.simulados"
          :historico="historico.perguntas"
          :flashcards="flashcards"
          @entregue="carregar"
          @flashcards="emit('flashcards', $event)"
        />
        <Exercicios v-if="aberta === 'exercicios' && exercicios.length" :tema="local" :banco="exercicios" :historico="historico.exercicios" @respondido="carregar" />
        <div v-if="parte.estado !== 'processando'" class="linha-acoes acoes-do-banco">
          <button v-if="!parte.resultado" class="botao principal" :disabled="!!pedindo" @click="preparar(aberta)">{{ PREPARAR[aberta] }}</button>
          <template v-else>
            <GerarMais
              :rotulo="aberta === 'simulado' ? 'Gerar mais perguntas' : 'Gerar mais exercícios'"
              :origem="origemDoTexto"
              :ocupado="!!pedindo"
              @gerar="(foco) => preparar(aberta, { mais: true, foco })"
            />
            <button class="botao discreto refazer" :disabled="!!pedindo" @click="preparar(aberta)">
              Refazer do zero<span v-if="parte.atualizado_em" class="meta"> · atualizado em {{ quando(parte.atualizado_em) }}</span>
            </button>
          </template>
        </div>
      </section>

      <template v-else>
        <div class="cartoes" role="group" aria-label="Partes do estudo">
          <button
            v-for="s in SECOES"
            :key="s.id"
            class="cartao-estudo"
            :class="{ aberta: aberta === s.id }"
            :aria-expanded="TELA_PROPRIA.includes(s.id) ? undefined : aberta === s.id"
            @click="abrir(s.id)"
          >
            <span class="icone-cartao"><Icone :nome="s.icone" /></span>
            <span class="nome">{{ s.nome }}</span>
            <span class="situacao" :class="situacao(s.id).classe">{{ situacao(s.id).texto }}</span>
          </button>
        </div>

        <section v-if="secao" :key="aberta" class="parte painel">
          <header>
            <h2>{{ secao.nome }} <Info>{{ secao.descricao }}</Info></h2>
          </header>

          <template v-if="parte">
            <p v-if="parte.estado === 'processando'" class="chip andando">{{ parte.resultado ? 'Refazendo…' : 'Preparando…' }} (na fila)</p>
            <p v-else-if="parte.estado === 'falhou'" class="aviso erro">{{ parte.erro }}</p>
            <p v-if="parte.desatualizado && parte.estado !== 'processando'" class="aviso">{{ ehReferencia ? 'O texto' : 'O Tema' }} mudou desde que isto foi preparado. Refaça para valer o texto de agora.</p>
          </template>

          <!-- Resumo -->
          <template v-if="aberta === 'resumo' && textoResumo">
            <div v-if="temLeitura" class="leitor-voz">
              <template v-if="leitura.estado === 'parado'">
                <button class="botao" @click="ouvir()"><Icone nome="ouvir" /> Ouvir o resumo</button>
              </template>
              <template v-else>
                <button v-if="leitura.estado === 'lendo'" class="botao" @click="pausar"><Icone nome="pausar" /> Pausar</button>
                <button v-else class="botao" @click="ouvir(leitura.trecho)"><Icone nome="ouvir" /> Continuar</button>
                <button class="botao" aria-label="Parar a leitura" @click="parar"><Icone nome="parar" /> Parar</button>
                <div class="progresso-voz" role="progressbar" :aria-valuenow="leitura.trecho + 1" aria-valuemin="1" :aria-valuemax="leitura.total" aria-label="Leitura do resumo">
                  <span :style="{ width: `${((leitura.trecho + 1) / Math.max(1, leitura.total)) * 100}%` }" />
                </div>
              </template>
            </div>
            <p v-if="erroLeitura" class="aviso erro">{{ erroLeitura }}</p>
            <article class="texto resumo" v-html="renderizar(textoResumo)" />
          </template>

          <!-- Conceitos anteriores -->
          <template v-if="aberta === 'anteriores' && conceitos.length">
            <article v-for="c in conceitos" :key="c.nome" class="conceito">
              <div class="cabeca-conceito">
                <h3>{{ c.nome }}</h3>
                <span class="chip" :class="{ base: c.tipo === 'base' }">{{ c.tipo === 'base' ? 'Base' : 'Ao redor' }}</span>
              </div>
              <p class="por-que">{{ c.por_que }}</p>
              <details>
                <summary>Entender o conceito</summary>
                <div class="texto explicacao" v-html="renderizar(c.explicacao)" />
                <p v-if="c.fontes.length" class="fontes suave">
                  Fontes:
                  <template v-for="(f, i) in c.fontes" :key="f.url"><template v-if="i">, </template><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></template>
                </p>
                <button v-if="flashcards" type="button" class="gerar-do-conceito" @click="cartoesDoConceito(c)">Gerar flashcards</button>
              </details>
              <a v-if="c.tema" class="no-acervo" :href="`#/ler/${c.tema}/indice.md`"><Icone nome="acervo" /> Está no seu Acervo</a>
              <a v-else class="no-acervo" :href="enderecoDePesquisa(c.nome)"><Icone nome="pesquisar" /> Pesquisar este conceito</a>
            </article>
          </template>

          <!-- Flashcards -->
          <template v-if="aberta === 'flashcards'">
            <p v-if="!flashcards" class="suave">O Flashcards não respondeu agora.</p>
            <div v-else class="opcoes-cartoes">
              <button class="botao" @click="gerarCartoes('basico')"><Icone nome="cartoes" /> Padrão</button>
              <button class="botao" @click="gerarCartoes('leitura')"><Icone nome="resumo" /> Leitura</button>
            </div>
          </template>

          <div v-if="parte && parte.estado !== 'processando'" class="linha-acoes">
            <button v-if="!parte.resultado" class="botao principal" :disabled="!!pedindo" @click="preparar(aberta)">{{ PREPARAR[aberta] }}</button>
            <button v-else class="botao discreto refazer" :disabled="!!pedindo" @click="preparar(aberta)">
              Refazer<span v-if="parte.atualizado_em" class="meta"> · preparado em {{ quando(parte.atualizado_em) }}</span>
            </button>
          </div>
        </section>
      </template>
    </template>
  </div>
</template>

<style scoped>
.estudar {
  display: grid;
  gap: 8px;
  scroll-margin-top: 72px;
}

/* Os quatro cartões */
.cartoes {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.cartao-estudo {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  aspect-ratio: 1;
  padding: 14px;
  border: 1px solid var(--linha);
  border-radius: var(--raio);
  background: var(--superficie);
  color: var(--texto);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
}

.cartao-estudo:hover {
  border-color: var(--linha-forte);
  transform: translateY(-1px);
}

.cartao-estudo:focus-visible {
  outline: 2px solid var(--marca);
  outline-offset: 2px;
}

.cartao-estudo.aberta {
  border-color: var(--marca);
  background: var(--marca-fundo);
}

.icone-cartao {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  margin-bottom: auto;
  border-radius: var(--raio-p);
  background: var(--fundo);
  color: var(--marca);
}

.icone-cartao :deep(svg) {
  width: 19px;
  height: 19px;
}

.nome {
  font-size: 0.95rem;
  font-weight: 650;
  line-height: 1.25;
  letter-spacing: -0.005em;
}

.situacao {
  color: var(--texto-suave);
  font-size: 0.78rem;
  line-height: 1.3;
}

.situacao.por-fazer {
  color: var(--texto-fraco);
}

.situacao.na-fila {
  color: var(--azul);
}

.situacao.com-erro,
.situacao.mudou {
  color: var(--vermelho);
}

@media (max-width: 760px) {
  .cartoes {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cartao-estudo {
    aspect-ratio: 1.3;
  }

  /* Cinco cartões em duas colunas: o último ocupa a linha inteira. */
  .cartao-estudo:last-child {
    grid-column: 1 / -1;
    aspect-ratio: auto;
    min-height: 96px;
  }
}

.acoes-do-banco {
  align-items: flex-start;
  margin-top: 8px;
  padding-top: 14px;
  border-top: 1px solid var(--linha);
}

/* A parte aberta */
.parte {
  display: grid;
  gap: 12px;
}

.painel {
  padding-top: 22px;
}

.parte header {
  display: grid;
  gap: 4px;
}

.parte h2 {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 0;
  font-size: 1.25rem;
  font-weight: 650;
  letter-spacing: -0.012em;
}


.parte > p {
  margin: 0;
}

.chip.andando {
  justify-self: start;
}

.voltar-estudar {
  justify-self: start;
  margin-left: -8px;
}

.voltar-estudar :deep(svg) {
  width: 18px;
  height: 18px;
}

.leitor-voz {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.progresso-voz {
  flex: 1 1 120px;
  max-width: 240px;
  height: 4px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--superficie-forte);
}

.progresso-voz span {
  display: block;
  height: 100%;
  background: var(--marca);
  transition: width 0.3s ease;
}

.resumo :deep(h2) {
  margin-top: 1.2em;
  font-size: 1.0625rem;
}

.resumo :deep(h2:first-child) {
  margin-top: 0;
}

.conceito {
  display: grid;
  gap: 6px;
  padding: 12px 0;
  border-top: 1px solid var(--linha);
}

.conceito:first-of-type {
  border-top: 0;
  padding-top: 0;
}

.cabeca-conceito {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cabeca-conceito h3 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 650;
}

.chip.base {
  color: var(--marca);
  background: var(--marca-fundo);
}

.por-que {
  margin: 0;
  color: var(--texto-suave);
  font-size: 0.95rem;
}

.conceito summary {
  cursor: pointer;
  color: var(--marca);
  font-size: 0.92rem;
}

.explicacao {
  margin-top: 8px;
  font-size: 0.98rem;
}

.fontes {
  margin: 6px 0 0;
  font-size: 0.85rem;
  overflow-wrap: anywhere;
}

.no-acervo {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.9rem;
}

.no-acervo :deep(svg) {
  width: 18px;
  height: 18px;
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
  font-size: 0.88rem;
}

.refazer {
  margin-left: -8px;
}

.meta {
  color: var(--texto-fraco);
}

.opcoes-cartoes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.gerar-do-conceito {
  display: block;
  margin-top: 8px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--marca);
  font: inherit;
  font-size: 0.92rem;
  cursor: pointer;
}
</style>
