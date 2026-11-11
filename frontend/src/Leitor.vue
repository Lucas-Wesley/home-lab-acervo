<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import CamadaDetalhes from './CamadaDetalhes.vue'
import EnviarFlashcards from './EnviarFlashcards.vue'
import Estudar from './Estudar.vue'
import Conversa from './Conversa.vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { capitulos, eLinkInterno, inserirCapitulo, relativo, renderizar, resolver, tituloDoTexto, trocarCapitulo } from './markdown.js'
import { marcar, usarDetalhes } from './detalhes.js'
import { baixarTema, Conflito, descartarPendente, estaBaixado, lerArquivo, salvar } from './offline.js'
import { guardarPosicao, podeVoltar, posicaoGuardada, trocarEndereco, voltandoNoHistorico } from './rolagem.js'

const props = defineProps({ caminho: { type: String, required: true } })

const [caminhoArquivo, ...resto] = props.caminho.split('?')
const opcao = resto.join('?')
// "?capitulo=Título": o link de um flashcard abre a página já no capítulo de onde ele saiu.
const capituloPedido = opcao.startsWith('capitulo=') ? opcao.slice('capitulo='.length) : ''
const arquivo = ref(null)
const erro = ref('')
const modo = ref(opcao === 'editar' ? 'editar' : 'ler')
const abaTema = ref(['estudar', 'conversar'].includes(opcao) ? opcao : 'ler') // D41, D48: Ler, Conversar ou Estudar
const texto = ref('')
const base = ref('')
const capituloEditado = ref(null) // editando só um capítulo da página
const previa = ref(false)
const salvando = ref(false)
const mensagem = ref('')
const conflito = ref(null) // { atual, minha }
const temas = ref([])
const capitulosDoTema = ref([]) // numa página de capítulo separado: os capítulos do Tema
const livrosDoTema = ref([]) // D38: capítulos dos livros da Biblioteca marcados com este Tema
const menu = ref(null) // capítulo com o menu aberto
const trechoDaConversa = ref(null) // D48: o trecho selecionado que vai para a conversa
const ocupado = ref(false)
const baixando = ref(false)
const editor = ref(null)
const marcaBaixados = ref(0) // o localStorage não é reativo; isto força recalcular
const introEl = ref(null)
const corpoEl = ref(null)
const detalhes = usarDetalhes(() => caminhoArquivo)
// Flashcards (D28): o texto vai para o app Flashcards, que monta a proposta; a revisão é lá.
const flashcards = ref({ disponivel: false, link: '', envios: [] })
const linkDosCartoes = ref('')
const envioAberto = ref(null) // { origem, titulo, aviso }: a janela de gerar flashcards

const CHAVE_ANCORA = 'acervo-ancora'

const eIndice = computed(() => caminhoArquivo.endsWith('/indice.md'))
const lista = computed(() => (arquivo.value ? capitulos(arquivo.value.conteudo) : []))
const capitulosDaPagina = computed(() => lista.value.filter((c) => !c.apendice))
const podeMexerNosCapitulos = computed(() => eIndice.value && !!arquivo.value?.tema && modo.value === 'ler')
// A página se divide em introdução (até o primeiro capítulo) e corpo, com o sumário entre os dois.
// Os trechos com "Mais informações" saem sublinhados nos dois.
const partes = computed(() => {
  if (!arquivo.value) return { intro: '', corpo: '', semLugar: [] }
  const opcoes = { fontes: arquivo.value.fontes }
  const conteudo = arquivo.value.conteudo
  const corte = lista.value[0]?.ini ?? conteudo.length
  const { htmls, semLugar } = marcar([renderizar(conteudo.slice(0, corte), opcoes), comAcoes(renderizar(conteudo.slice(corte), opcoes))], detalhes.lista.value)
  return { intro: htmls[0], corpo: htmls[1], semLugar }
})
// Lista do botão "Capítulos" no cabeçalho: os da página ou, num capítulo separado, os do Tema.
const listaDoCabecalho = computed(() => (eIndice.value ? lista.value : capitulosDoTema.value))
const capituloAtual = ref('') // id do capítulo que está na tela
const listaAberta = ref(false)
const previaHtml = computed(() => renderizar(texto.value, { fontes: arquivo.value?.fontes }))
const tituloAtual = computed(() => (arquivo.value ? tituloDoTexto(arquivo.value.conteudo) || caminhoArquivo.split('/').pop() : ''))
// Migalhas: os Temas que contêm este arquivo; no indice.md o último é o próprio Tema.
const migalhas = computed(() => (arquivo.value?.migalhas ?? []).filter((m) => !(eIndice.value && `${m.caminho}/indice.md` === caminhoArquivo)))
const subtemas = computed(() => (eIndice.value && arquivo.value?.tema ? temas.value.filter((t) => t.pai === arquivo.value.tema) : []))
const vizinhos = computed(() => {
  const caps = capitulosDoTema.value.filter((c) => !c.apendice)
  const i = caps.findIndex((c) => c.arquivo && resolver(`${arquivo.value.tema}/indice.md`, c.arquivo) === caminhoArquivo)
  if (eIndice.value || i < 0) return {}
  return { anterior: caps[i - 1], proximo: caps[i + 1] }
})
const temaBaixado = computed(() => marcaBaixados.value >= 0 && !!arquivo.value?.tema && estaBaixado(arquivo.value.tema))
const temAbas = computed(() => eIndice.value && !!arquivo.value?.tema && modo.value === 'ler')
// Conversar (D48) também nos capítulos separados: a conversa é a do Tema, e o trecho diz o arquivo.
const temConversa = computed(() => !!arquivo.value?.tema && modo.value === 'ler')
const arquivoNoTema = computed(() => (arquivo.value?.tema && caminhoArquivo.startsWith(`${arquivo.value.tema}/`) ? caminhoArquivo.slice(arquivo.value.tema.length + 1) : ''))
function conversarSobre(trecho) {
  trechoDaConversa.value = { trecho, arquivo: arquivoNoTema.value }
  trocarAba('conversar')
}
const temaRaiz = computed(() => arquivo.value?.migalhas?.[0])
const tituloDoTema = computed(() => arquivo.value?.migalhas?.find((m) => m.caminho === arquivo.value.tema)?.titulo || tituloAtual.value)

// Cada capítulo ganha um botão de menu ao lado do título (só na página do Tema).
function comAcoes(html) {
  if (!podeMexerNosCapitulos.value) return html
  const div = document.createElement('div')
  div.innerHTML = html
  for (const h of div.querySelectorAll('h2')) {
    const botao = document.createElement('button')
    botao.className = 'acao-capitulo'
    botao.type = 'button'
    botao.setAttribute('aria-label', `Opções do capítulo ${h.textContent}`)
    botao.dataset.titulo = h.textContent
    botao.textContent = '⋯'
    h.append(botao)
  }
  return div.innerHTML
}

async function carregar({ manterPosicao = false } = {}) {
  erro.value = ''
  const y = window.scrollY
  try {
    const a = await lerArquivo(caminhoArquivo)
    arquivo.value = a
    texto.value = a.conteudo
    base.value = a.hash
    if (a.pendente?.conflito) conflito.value = { atual: a.pendente.conflito, minha: a.pendente.conteudo }
    document.title = `${tituloAtual.value} · Acervo`
    if (!eIndice.value && a.tema) carregarCapitulosDoTema(a.tema)
    if (eIndice.value && a.tema) carregarLivrosDoTema(a.tema)
    await nextTick()
    const ancora = sessionStorage.getItem(CHAVE_ANCORA)
    sessionStorage.removeItem(CHAVE_ANCORA)
    if (manterPosicao) window.scrollTo(0, y)
    else if (voltandoNoHistorico()) window.scrollTo(0, posicaoGuardada(caminhoArquivo))
    else if (ancora && document.getElementById(ancora)) document.getElementById(ancora).scrollIntoView()
    else if (capituloPedido && lista.value.some((c) => c.titulo === capituloPedido)) document.getElementById(lista.value.find((c) => c.titulo === capituloPedido).id)?.scrollIntoView()
    else window.scrollTo(0, 0)
  } catch (e) {
    erro.value = e.status === 0 ? 'Sem conexão, e este Tema não foi baixado para ler fora da rede.' : e.message
  }
}

async function carregarCapitulosDoTema(tema) {
  try {
    capitulosDoTema.value = capitulos((await lerArquivo(`${tema}/indice.md`)).conteudo)
  } catch {
    capitulosDoTema.value = []
  }
}

async function carregarLivrosDoTema(tema) {
  try {
    livrosDoTema.value = await api(`/api/temas-livros?tema=${encodeURIComponent(tema)}`)
  } catch {
    livrosDoTema.value = [] // fora da rede de casa, a lista não aparece
  }
}

async function carregarTemas() {
  try {
    temas.value = await api('/api/temas')
  } catch {
    temas.value = []
  }
}

function aoClicarNoTexto(evento) {
  if (detalhes.aoClicar(evento)) return
  const botao = evento.target.closest('.acao-capitulo')
  if (botao) {
    menu.value = lista.value.find((c) => c.titulo === botao.dataset.titulo) ?? null
    return
  }
  const a = evento.target.closest('a')
  const href = a?.getAttribute('href')
  if (!href) return
  if (href.startsWith('#')) {
    evento.preventDefault()
    document.getElementById(href.slice(1))?.scrollIntoView({ behavior: 'smooth' })
    return
  }
  if (!eLinkInterno(href)) return
  evento.preventDefault()
  const [destino, ancora] = href.split('#')
  if (ancora) sessionStorage.setItem(CHAVE_ANCORA, ancora)
  abrirArquivo(resolver(caminhoArquivo, destino))
}

function abrirArquivo(caminho) {
  location.hash = `#/ler/${caminho}`
}

// Sumário: capítulo na página rola até ele; capítulo separado abre a página dele.
function irPara(capitulo) {
  if (capitulo.arquivo) abrirArquivo(resolver(caminhoArquivo, capitulo.arquivo))
  else document.getElementById(capitulo.id)?.scrollIntoView({ behavior: 'smooth' })
}

// Anterior/próximo numa página de capítulo: o vizinho pode estar na página do Tema ou separado.
function abrirCapituloDoTema(capitulo) {
  const indice = `${arquivo.value.tema}/indice.md`
  if (capitulo.arquivo) return abrirArquivo(resolver(indice, capitulo.arquivo))
  sessionStorage.setItem(CHAVE_ANCORA, capitulo.id)
  abrirArquivo(indice)
}

function escolherNaLista(capitulo) {
  listaAberta.value = false
  if (eIndice.value) irPara(capitulo)
  else abrirCapituloDoTema(capitulo)
}

function eAtual(capitulo) {
  if (eIndice.value) return capitulo.id === capituloAtual.value
  return !!capitulo.arquivo && resolver(`${arquivo.value.tema}/indice.md`, capitulo.arquivo) === caminhoArquivo
}

// Voltar leva ao texto de onde se veio; aberto direto (link salvo), sobe para o Tema que contém.
function voltar() {
  if (podeVoltar()) history.back()
  else location.hash = migalhas.value.length ? `#/ler/${migalhas.value.at(-1).caminho}/indice.md` : '#/'
}

// ---------- Flashcards ----------

async function carregarFlashcards() {
  try {
    flashcards.value = await api('/api/flashcards')
  } catch {
    // Fora da rede ou sem o Flashcards: o botão só não aparece.
  }
}

// O último envio deste texto (o Tema inteiro, ou um capítulo).
function ultimoEnvio(capitulo) {
  const alvo = `arquivo:${caminhoArquivo}${capitulo ? `#${capitulo}` : ''}`
  return flashcards.value.envios.find((e) => e.alvo === alvo && e.geracao_id)
}

const quando = (iso) => new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit' })

function gerarFlashcards(capitulo) {
  menu.value = null
  const antes = ultimoEnvio(capitulo)
  envioAberto.value = {
    origem: { arquivo: caminhoArquivo, capitulo: capitulo ?? '' },
    titulo: capitulo ? `do capítulo “${capitulo}”` : eIndice.value ? 'do Tema inteiro' : 'deste capítulo',
    aviso: antes ? `Já foi enviado em ${quando(antes.criado_em)}. No mesmo baralho, os cartões de mesma pergunta são atualizados, não repetidos.` : '',
  }
}

// "?estudar" no endereço: recarregar (ou o link de uma tarefa da fila) volta para a aba.
function trocarAba(aba) {
  abaTema.value = aba
  trocarEndereco(aba === 'ler' ? `#/ler/${caminhoArquivo}` : `#/ler/${caminhoArquivo}?${aba}`)
  window.scrollTo(0, 0)
}

// Flashcards pedidos na aba Estudar: do Tema, dos conceitos anteriores ou do que se errou.
function flashcardsDoEstudo({ origem, titulo, tipo }) {
  envioAberto.value = { origem, titulo, tipo, aviso: '' }
}

function flashcardsEnviados(e) {
  envioAberto.value = null
  mensagem.value = 'Enviado ao Flashcards. A IA está montando os cartões; a proposta fica em Pedidos à IA, para você aprovar.'
  linkDosCartoes.value = `${flashcards.value.link}/#/geracao/${e.geracao_id}`
  carregarFlashcards()
}

// ---------- Edição ----------

async function editar() {
  if (!temas.value.length) carregarTemas()
  capituloEditado.value = null
  texto.value = arquivo.value.conteudo
  modo.value = 'editar'
  previa.value = false
  await nextTick()
  editor.value?.focus()
}

async function editarCapitulo(capitulo) {
  menu.value = null
  if (!temas.value.length) carregarTemas()
  capituloEditado.value = capitulo
  texto.value = arquivo.value.conteudo.slice(capitulo.ini, capitulo.fim).trimEnd()
  modo.value = 'editar'
  previa.value = false
  window.scrollTo(0, 0)
  await nextTick()
  editor.value?.focus()
}

function cancelar() {
  const capitulo = capituloEditado.value
  texto.value = arquivo.value.conteudo
  capituloEditado.value = null
  modo.value = 'ler'
  if (opcao) trocarEndereco(`#/ler/${caminhoArquivo}`)
  if (capitulo) nextTick(() => document.getElementById(capitulo.id)?.scrollIntoView())
}

async function gravar() {
  salvando.value = true
  mensagem.value = ''
  const capitulo = capituloEditado.value
  const conteudo = capitulo ? trocarCapitulo(arquivo.value.conteudo, capitulo, texto.value) : texto.value
  try {
    const r = await salvar(caminhoArquivo, conteudo, base.value)
    base.value = r.hash
    arquivo.value = { ...arquivo.value, conteudo, hash: r.hash }
    texto.value = conteudo
    capituloEditado.value = null
    modo.value = 'ler'
    mensagem.value = r.pendente ? 'Sem conexão: a edição ficou salva no aparelho e sobe quando voltar à rede.' : 'Salvo.'
    window.dispatchEvent(new Event('acervo:pendentes'))
    if (opcao) trocarEndereco(`#/ler/${caminhoArquivo}`)
    if (!r.pendente) setTimeout(() => (mensagem.value = ''), 2500)
    if (capitulo) {
      const titulo = conteudo.slice(capitulo.ini).match(/^## (.+)$/m)?.[1]
      const id = capitulos(conteudo).find((c) => c.titulo === titulo?.trim())?.id
      nextTick(() => id && document.getElementById(id)?.scrollIntoView())
    }
  } catch (e) {
    if (e instanceof Conflito) conflito.value = { atual: e.atual, minha: e.minha }
    else mensagem.value = e.message
  } finally {
    salvando.value = false
  }
}

function aoTeclar(evento) {
  if ((evento.metaKey || evento.ctrlKey) && evento.key === 's') {
    evento.preventDefault()
    gravar()
  }
}

// Link para outro Tema: só link, o texto não é copiado (D20).
function inserirLink(evento) {
  const destino = temas.value.find((t) => t.caminho === evento.target.value)
  evento.target.value = ''
  if (!destino) return
  const link = `[${destino.titulo}](${relativo(caminhoArquivo, `${destino.caminho}/indice.md`)})`
  const el = editor.value
  const ini = el?.selectionStart ?? texto.value.length
  const fim = el?.selectionEnd ?? ini
  texto.value = texto.value.slice(0, ini) + link + texto.value.slice(fim)
  nextTick(() => {
    el?.focus()
    el?.setSelectionRange(ini + link.length, ini + link.length)
  })
}

async function manterMinha() {
  const { atual, minha } = conflito.value
  conflito.value = null
  capituloEditado.value = null
  texto.value = minha
  base.value = atual.hash
  descartarPendente(caminhoArquivo)
  await gravar()
}

function usarServidor() {
  const { atual } = conflito.value
  conflito.value = null
  capituloEditado.value = null
  descartarPendente(caminhoArquivo)
  window.dispatchEvent(new Event('acervo:pendentes'))
  arquivo.value = { ...arquivo.value, ...atual }
  texto.value = atual.conteudo
  base.value = atual.hash
  modo.value = 'ler'
}

// ---------- Capítulos (D23) ----------

async function mudarCapitulo(acao, capitulo) {
  menu.value = null
  ocupado.value = true
  mensagem.value = ''
  try {
    await api(`/api/capitulos/${acao}`, { metodo: 'POST', corpo: { arquivo: caminhoArquivo, titulo: capitulo.titulo, base: base.value } })
    await carregar({ manterPosicao: true })
    mensagem.value = acao === 'separar' ? `"${capitulo.titulo}" agora tem página própria.` : `"${capitulo.titulo}" voltou para a página.`
    setTimeout(() => (mensagem.value = ''), 3000)
  } catch (e) {
    mensagem.value = e.status === 0 ? 'Separar ou juntar capítulos só funciona na rede de casa.' : e.message
  } finally {
    ocupado.value = false
  }
}

async function novoCapitulo() {
  const titulo = prompt('Título do novo capítulo')?.trim()
  if (!titulo) return
  if (lista.value.some((c) => c.titulo === titulo)) {
    mensagem.value = 'Já existe um capítulo com esse título.'
    return
  }
  const conteudo = inserirCapitulo(arquivo.value.conteudo, titulo)
  try {
    const r = await salvar(caminhoArquivo, conteudo, base.value)
    arquivo.value = { ...arquivo.value, conteudo, hash: r.hash }
    base.value = r.hash
    editarCapitulo(capitulos(conteudo).find((c) => c.titulo === titulo))
  } catch (e) {
    mensagem.value = e instanceof Conflito ? 'O Tema mudou no servidor; recarregue a página.' : e.message
  }
}

async function novoSubtema() {
  const titulo = prompt('Título do subtema')?.trim()
  if (!titulo) return
  try {
    const { caminho } = await api('/api/temas', { metodo: 'POST', corpo: { titulo, pai: arquivo.value.tema } })
    location.hash = `#/ler/${caminho}/indice.md?editar`
  } catch (e) {
    mensagem.value = e.message
  }
}

async function baixar() {
  const raiz = temaRaiz.value
  if (!raiz || temaBaixado.value) return
  baixando.value = true
  try {
    await baixarTema(raiz)
    mensagem.value = `"${raiz.titulo}" baixado: dá para ler e editar fora da rede.`
  } catch (e) {
    mensagem.value = `Não foi possível baixar: ${e.message}`
  } finally {
    baixando.value = false
  }
  marcaBaixados.value++
}

function aoRolar() {
  guardarPosicao(caminhoArquivo, window.scrollY)
  marcarCapituloAtual()
}

// Capítulo na tela: o último título que já passou do topo. No fim da página, em que o último
// capítulo pode nunca chegar ao topo, vale o último título na metade de cima da tela.
function marcarCapituloAtual() {
  const noFim = window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 4
  const limite = noFim ? window.innerHeight / 2 : 120
  let atual = ''
  for (const c of lista.value) {
    const el = document.getElementById(c.id)
    if (el && el.getBoundingClientRect().top < limite) atual = c.id
  }
  capituloAtual.value = atual
}

function abrirLista() {
  marcarCapituloAtual()
  listaAberta.value = !listaAberta.value
}

onMounted(() => {
  carregar()
  carregarFlashcards()
  if (modo.value === 'editar' || eIndice.value) carregarTemas()
  window.addEventListener('scroll', aoRolar, { passive: true })
})
onBeforeUnmount(() => window.removeEventListener('scroll', aoRolar))
</script>

<template>
  <div class="leitor">
    <header class="barra">
      <button class="botao icone" aria-label="Voltar" @click="voltar"><Icone nome="voltar" /></button>
      <nav class="migalhas" aria-label="Onde estou">
        <a href="#/">Acervo</a>
        <template v-for="m in migalhas" :key="m.caminho">
          <span class="sep">›</span>
          <a :href="`#/ler/${m.caminho}/indice.md`">{{ m.titulo }}</a>
        </template>
      </nav>
      <div v-if="arquivo && modo === 'ler'" class="acoes">
        <button v-if="listaDoCabecalho.length" class="botao icone" aria-label="Capítulos" title="Capítulos" :aria-expanded="listaAberta" @click="abrirLista">
          <Icone nome="lista" />
        </button>
        <button v-if="temaRaiz" class="botao icone" :title="temaBaixado ? 'Baixado neste aparelho' : 'Baixar o Tema para ler fora da rede'" :disabled="baixando" @click="baixar">
          <Icone :nome="temaBaixado ? 'baixado' : 'baixar'" />
        </button>
        <button v-if="flashcards.disponivel && arquivo.tema" class="botao icone" :disabled="ocupado" :aria-label="eIndice ? 'Gerar flashcards do Tema inteiro' : 'Gerar flashcards deste capítulo'" :title="eIndice ? 'Gerar flashcards do Tema inteiro' : 'Gerar flashcards deste capítulo'" @click="gerarFlashcards()">
          <Icone nome="cartoes" />
        </button>
        <button class="botao" @click="editar"><Icone nome="nota" /> Editar</button>
      </div>
    </header>

    <div v-if="listaAberta" class="veu-lista" @click="listaAberta = false">
      <nav class="painel-capitulos" aria-label="Lista de capítulos" @click.stop>
        <strong>{{ eIndice ? tituloAtual : migalhas.at(-1)?.titulo }}</strong>
        <ol>
          <li v-for="c in listaDoCabecalho.filter((c) => !c.apendice)" :key="c.id">
            <button :class="{ atual: eAtual(c) }" @click="escolherNaLista(c)">
              {{ c.titulo }}<span v-if="c.arquivo" class="suave"> · página própria</span>
            </button>
          </li>
        </ol>
        <div v-if="listaDoCabecalho.some((c) => c.apendice)" class="apendices">
          <button v-for="c in listaDoCabecalho.filter((c) => c.apendice)" :key="c.id" :class="{ atual: eAtual(c) }" @click="escolherNaLista(c)">{{ c.titulo }}</button>
        </div>
      </nav>
    </div>

    <div class="pagina-leitura">
      <div v-if="erro" class="aviso erro">{{ erro }}</div>
      <div v-if="mensagem" class="aviso" :class="{ ok: mensagem === 'Salvo.' || mensagem.includes('baixado') || mensagem.includes('página') || !!linkDosCartoes }">
        {{ mensagem }}
        <a v-if="linkDosCartoes" :href="linkDosCartoes" target="_blank" rel="noopener">Abrir no Flashcards</a>
      </div>

      <nav v-if="temConversa && !conflito" class="abas" aria-label="Ler, conversar ou estudar">
        <button :class="{ ativa: abaTema === 'ler' }" @click="trocarAba('ler')">Ler</button>
        <button :class="{ ativa: abaTema === 'conversar' }" @click="trocarAba('conversar')">Conversar</button>
        <button v-if="temAbas" :class="{ ativa: abaTema === 'estudar' }" @click="trocarAba('estudar')">Estudar</button>
      </nav>

      <section v-if="conflito" class="conflito">
        <div class="aviso erro">Este arquivo mudou no servidor depois da sua edição. Escolha qual versão fica.</div>
        <div class="versoes">
          <div>
            <h3>Sua versão</h3>
            <pre>{{ conflito.minha }}</pre>
            <button class="botao principal" @click="manterMinha">Manter a minha</button>
          </div>
          <div>
            <h3>Versão do servidor</h3>
            <pre>{{ conflito.atual.conteudo }}</pre>
            <button class="botao" @click="usarServidor">Usar a do servidor</button>
          </div>
        </div>
      </section>

      <template v-else-if="arquivo && modo === 'editar'">
        <p v-if="capituloEditado" class="suave editando">Editando o capítulo <strong>{{ capituloEditado.titulo }}</strong>. O resto da página fica como está.</p>
        <div class="ferramentas">
          <button class="botao principal" :disabled="salvando" @click="gravar">{{ salvando ? 'Salvando…' : 'Salvar' }}</button>
          <button class="botao" @click="cancelar">Cancelar</button>
          <button class="botao" :class="{ ativo: previa }" @click="previa = !previa">{{ previa ? 'Voltar ao texto' : 'Pré-visualizar' }}</button>
          <select class="campo link-tema" aria-label="Inserir link para um Tema" @change="inserirLink">
            <option value="">Inserir link para Tema…</option>
            <option v-for="t in temas" :key="t.caminho" :value="t.caminho">{{ t.titulo }}</option>
          </select>
        </div>
        <article v-if="previa" class="texto" v-html="previaHtml" />
        <textarea v-else ref="editor" v-model="texto" class="editor" spellcheck="true" @keydown="aoTeclar" />
        <p class="suave dica">Markdown: <code>## Capítulo</code>, <code>### Seção</code>, <code>**negrito**</code>, <code>[texto](arquivo.md)</code>. Cmd/Ctrl+S salva.</p>
      </template>

      <template v-else-if="arquivo && temConversa && abaTema === 'conversar'">
        <h1 class="titulo-estudar">{{ tituloDoTema }}</h1>
        <Conversa
          :local="arquivo.tema"
          :titulo="tituloDoTema"
          :trecho="trechoDaConversa"
          @sem-trecho="trechoDaConversa = null"
          @mudou="carregar({ manterPosicao: true })"
        />
      </template>

      <template v-else-if="arquivo && temAbas && abaTema === 'estudar'">
        <h1 class="titulo-estudar">{{ tituloDoTema }}</h1>
        <Estudar :local="arquivo.tema" :titulo="tituloDoTema" :flashcards="flashcards.disponivel" @flashcards="flashcardsDoEstudo" />
        <EnviarFlashcards v-if="envioAberto" v-bind="envioAberto" @fechar="envioAberto = null" @enviado="flashcardsEnviados" />
      </template>

      <template v-else-if="arquivo">
        <article ref="introEl" class="texto" @click="aoClicarNoTexto" v-html="partes.intro" />

        <nav v-if="eIndice && capitulosDaPagina.length > 1" class="sumario" aria-label="Capítulos">
          <h3>Capítulos</h3>
          <ol>
            <li v-for="c in capitulosDaPagina" :key="c.id">
              <a :href="c.arquivo ? `#/ler/${resolver(caminhoArquivo, c.arquivo)}` : `#${c.id}`" @click.prevent="irPara(c)">{{ c.titulo }}</a>
              <span v-if="c.arquivo" class="chip">página própria</span>
            </li>
          </ol>
          <div v-if="lista.some((c) => c.apendice)" class="apendices">
            <a v-for="c in lista.filter((c) => c.apendice)" :key="c.id" :href="`#${c.id}`" @click.prevent="irPara(c)">{{ c.titulo }}</a>
          </div>
        </nav>

        <article ref="corpoEl" class="texto" :class="{ 'com-acoes': podeMexerNosCapitulos }" @click="aoClicarNoTexto" v-html="partes.corpo" />
        <CamadaDetalhes :detalhes="detalhes" :raizes="() => [introEl, corpoEl]" :titulo="tituloAtual" :sem-lugar="partes.semLugar" @conversar="conversarSobre" />
        <EnviarFlashcards v-if="envioAberto" v-bind="envioAberto" @fechar="envioAberto = null" @enviado="flashcardsEnviados" />

        <nav v-if="vizinhos.anterior || vizinhos.proximo" class="vizinhos">
          <a v-if="vizinhos.anterior" href="#" class="cartao" @click.prevent="abrirCapituloDoTema(vizinhos.anterior)">
            <span class="suave">← Capítulo anterior</span><span>{{ vizinhos.anterior.titulo }}</span>
          </a>
          <span v-else />
          <a v-if="vizinhos.proximo" href="#" class="cartao proximo" @click.prevent="abrirCapituloDoTema(vizinhos.proximo)">
            <span class="suave">Próximo capítulo →</span><span>{{ vizinhos.proximo.titulo }}</span>
          </a>
        </nav>

        <section v-if="subtemas.length" class="rodape-tema">
          <h3>Subtemas</h3>
          <ul>
            <li v-for="s in subtemas" :key="s.caminho"><a :href="`#/ler/${s.caminho}/indice.md`">{{ s.titulo }}</a></li>
          </ul>
        </section>

        <section v-if="arquivo.citado_em?.length" class="rodape-tema">
          <h3>Citado em</h3>
          <ul>
            <li v-for="c in arquivo.citado_em" :key="c.caminho"><a :href="`#/ler/${c.caminho}`">{{ c.titulo }}</a></li>
          </ul>
        </section>

        <section v-if="eIndice && arquivo.fontes?.length" class="rodape-tema">
          <h3>Referências deste Tema</h3>
          <ul>
            <li v-for="f in arquivo.fontes" :key="f.id">
              <a :href="`#/referencia/${f.id}`">{{ f.titulo }}</a>
              · <a :href="f.url" target="_blank" rel="noopener">vídeo</a>
            </li>
          </ul>
        </section>

        <section v-if="eIndice && livrosDoTema.length" class="rodape-tema livros-do-tema">
          <h3>Livros da sua biblioteca que falam disto</h3>
          <ul>
            <li v-for="l in livrosDoTema" :key="`${l.livro}-${l.capitulo}`">
              <a :href="l.link" target="_blank" rel="noopener">{{ l.titulo }}</a><span class="suave"><template v-if="l.autor"> · {{ l.autor }}</template> · “{{ l.capitulo }}”, p. {{ l.de }}–{{ l.ate }}</span>
              <span v-if="l.resumo" class="resumo-livro suave">{{ l.resumo }}</span>
            </li>
          </ul>
        </section>

        <div v-if="eIndice && arquivo.tema" class="linha-acoes extras">
          <button class="botao" @click="novoCapitulo"><Icone nome="mais" /> Novo capítulo</button>
          <button class="botao" @click="novoSubtema"><Icone nome="mais" /> Novo subtema</button>
        </div>
      </template>
    </div>

    <div v-if="menu" class="veu-menu" @click="menu = null">
      <div class="folha" role="dialog" :aria-label="`Capítulo ${menu.titulo}`" @click.stop>
        <strong>{{ menu.titulo }}</strong>
        <template v-if="menu.arquivo">
          <button class="botao" @click="irPara(menu); menu = null">Abrir a página do capítulo</button>
          <button class="botao" :disabled="ocupado" @click="mudarCapitulo('juntar', menu)">Trazer de volta para esta página</button>
        </template>
        <template v-else>
          <button class="botao" @click="editarCapitulo(menu)">Editar capítulo</button>
          <button v-if="!menu.apendice" class="botao" :disabled="ocupado" @click="mudarCapitulo('separar', menu)">Separar em página própria</button>
          <button v-if="flashcards.disponivel && !menu.apendice" class="botao" :disabled="ocupado" @click="gerarFlashcards(menu.titulo)">
            <Icone nome="cartoes" /> Gerar flashcards
            <span v-if="ultimoEnvio(menu.titulo)" class="suave">· enviado em {{ quando(ultimoEnvio(menu.titulo).criado_em) }}</span>
          </button>
        </template>
        <button class="botao" @click="menu = null">Cancelar</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.barra {
  position: sticky;
  top: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 56px;
  padding: max(8px, env(safe-area-inset-top)) 12px 8px;
  border-bottom: 1px solid var(--linha);
  background: var(--vidro);
  -webkit-backdrop-filter: var(--blur);
  backdrop-filter: var(--blur);
}

.migalhas {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  font-size: 0.88rem;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.migalhas a {
  color: var(--texto-suave);
  text-decoration: none;
}

.migalhas a:hover {
  color: var(--texto);
}

.sep {
  margin: 0 6px;
  color: var(--texto-suave);
}

.acoes {
  display: flex;
  gap: 4px;
}

.pagina-leitura {
  max-width: 720px;
  margin: 0 auto;
  padding: 28px 20px 96px;
}

.aviso {
  margin-bottom: 16px;
}

.ferramentas {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.ferramentas .ativo {
  background: var(--superficie-forte);
}

.link-tema {
  width: auto;
  min-height: 38px;
  flex: 1;
  min-width: 180px;
}

.editor {
  width: 100%;
  min-height: 65vh;
  padding: 16px;
  border: 1px solid var(--linha);
  border-radius: 12px;
  background: var(--superficie);
  font-family: ui-monospace, 'SF Mono', Menlo, monospace;
  font-size: 0.92rem;
  line-height: 1.6;
  resize: vertical;
}

.dica {
  font-size: 0.8rem;
}

.vizinhos {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  margin-top: 40px;
}

.vizinhos a {
  display: grid;
  gap: 2px;
  text-decoration: none;
  color: var(--texto);
  font-size: 0.92rem;
  text-transform: none;
}

.vizinhos a span:last-child::first-letter {
  text-transform: uppercase;
}

.vizinhos .proximo {
  text-align: right;
}

.abas {
  display: flex;
  gap: 4px;
  margin: 0 0 20px;
  border-bottom: 1px solid var(--linha);
}

.abas button {
  padding: 10px 12px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: none;
  color: var(--texto-suave);
  font: inherit;
  cursor: pointer;
}

.abas button.ativa {
  border-bottom-color: var(--destaque);
  color: var(--texto);
  font-weight: 600;
}

.titulo-estudar {
  margin: 0 0 12px;
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.022em;
}

.rodape-tema {
  margin-top: 36px;
  padding-top: 16px;
  border-top: 1px solid var(--linha);
}

.rodape-tema h3 {
  margin: 0 0 8px;
  font-size: 1.0625rem;
  font-weight: 650;
  color: var(--texto);
}

.resumo-livro {
  display: block;
  font-size: 0.906rem;
  line-height: 1.45;
}

.livros-do-tema li + li {
  margin-top: 6px;
}

.rodape-tema ul {
  margin: 0;
  padding-left: 18px;
  line-height: 1.7;
}

.extras {
  margin-top: 28px;
}

.editando {
  margin: 0 0 10px;
  font-size: 0.9rem;
}

.sumario {
  margin: 8px 0 32px;
  padding: 16px 0;
  border-top: 1px solid var(--linha);
  border-bottom: 1px solid var(--linha);
}

.sumario h3 {
  margin: 0 0 8px;
  font-family: var(--interface);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--texto-fraco);
}

.sumario ol {
  margin: 0;
  padding-left: 22px;
  font-family: var(--leitura);
  line-height: 1.75;
}

.sumario a {
  color: var(--texto);
  text-decoration: none;
}

.sumario a:hover {
  color: var(--azul);
}

.sumario .chip {
  margin-left: 6px;
  vertical-align: 1px;
}

.apendices {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--linha);
  font-size: 0.85rem;
}

.apendices a {
  color: var(--texto-suave);
}

.texto.com-acoes :deep(h2) {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.texto :deep(.acao-capitulo) {
  margin-left: auto;
  flex: none;
  width: 34px;
  height: 30px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--texto-suave);
  font-size: 1.1rem;
  line-height: 1;
  cursor: pointer;
}

.texto :deep(.acao-capitulo:hover) {
  background: var(--superficie);
  color: var(--texto);
}

.texto :deep(.capitulo-separado a) {
  display: inline-flex;
  padding: 10px 16px;
  border: 1px solid var(--linha);
  border-radius: 12px;
  background: var(--superficie);
  font-family: -apple-system, BlinkMacSystemFont, system-ui, sans-serif;
  font-size: 0.95rem;
  text-decoration: none;
}

.veu-menu {
  position: fixed;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 20%);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
}

.folha {
  display: grid;
  gap: 8px;
  width: min(480px, 100%);
  padding: 18px 16px max(18px, env(safe-area-inset-bottom));
  border-radius: var(--raio-g) var(--raio-g) 0 0;
  background: var(--fundo);
  box-shadow: var(--sombra);
}

.folha strong {
  padding: 0 4px 6px;
  font-family: var(--leitura);
  font-size: 1.05rem;
}

.folha .botao {
  justify-content: center;
  min-height: 46px;
}

.veu-lista {
  position: fixed;
  inset: 0;
  z-index: 25;
  background: rgb(0 0 0 / 6%);
}

.painel-capitulos {
  position: absolute;
  top: calc(max(8px, env(safe-area-inset-top)) + 52px);
  right: 12px;
  width: min(380px, calc(100% - 24px));
  max-height: calc(100vh - 90px);
  overflow-y: auto;
  padding: 14px 8px 10px;
  border: 1px solid var(--vidro-borda);
  border-radius: var(--raio-g);
  background: var(--vidro);
  -webkit-backdrop-filter: var(--blur);
  backdrop-filter: var(--blur);
  box-shadow: var(--sombra);
}

.painel-capitulos strong {
  display: block;
  padding: 0 10px 8px;
  font-size: 0.82rem;
  font-weight: 600;
  color: var(--texto-suave);
}

.painel-capitulos ol {
  margin: 0;
  padding: 0 0 0 30px;
  font-family: var(--leitura);
}

.painel-capitulos button {
  display: block;
  width: 100%;
  padding: 7px 10px;
  border: 0;
  border-radius: 8px;
  background: none;
  text-align: left;
  line-height: 1.35;
  cursor: pointer;
}

.painel-capitulos button:hover {
  background: var(--superficie);
}

.painel-capitulos button.atual {
  background: var(--superficie-forte);
  font-weight: 600;
}

.painel-capitulos .apendices {
  margin: 6px 0 0;
  padding: 6px 0 0 20px;
}

.painel-capitulos .apendices button {
  width: auto;
  color: var(--texto-suave);
  font-size: 0.88rem;
}

@media (min-width: 821px) {
  .veu-menu {
    align-items: center;
  }

  .folha {
    border-radius: 18px;
  }
}

.conflito .versoes {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.conflito pre {
  max-height: 55vh;
  overflow: auto;
  padding: 12px;
  border-radius: 10px;
  background: var(--superficie);
  font-size: 0.8rem;
  white-space: pre-wrap;
}

@media (max-width: 820px) {
  .conflito .versoes {
    grid-template-columns: 1fr;
  }
}
</style>
