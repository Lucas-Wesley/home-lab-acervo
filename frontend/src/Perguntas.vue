<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { renderizar } from './markdown.js'

// Janela de "Perguntar" (D35): a pergunta sobre o texto, a resposta verificada na internet e as
// divergências com o texto. Fica a lista das perguntas já feitas, a mais nova em cima.
const props = defineProps({
  local: { type: String, required: true }, // temas/<pasta> ou referencias/<id>
  titulo: { type: String, default: '' },
  embutido: { type: Boolean, default: false }, // numa aba da página, em vez de janela
  // Desde o Conversar (D48): só as perguntas verificadas de antes, sem o campo de perguntar.
  antigas: { type: Boolean, default: false },
})
const emit = defineEmits(['fechar'])

const COBERTURA = {
  responde: ['Seu texto responde', 'ok'],
  em_parte: ['Seu texto responde em parte', 'meio'],
  toca: ['Seu texto só toca no assunto', 'meio'],
  nao_responde: ['Seu texto não responde', 'erro'],
}
const SELO = {
  confirmado: 'Confirmado em mais de uma fonte',
  uma_fonte: 'Uma fonte só',
  divergem: 'As fontes divergem',
  nao_encontrado: 'Não encontrado nas fontes',
}
const TIPO = { erro: 'Erro', desatualizado: 'Desatualizado', omissao: 'Falta no texto', melhoria: 'Pode melhorar' }

const perguntas = ref([])
const texto = ref('')
const enviando = ref(false)
const erro = ref('')
const aberta = ref(null) // id da resposta verificada aberta quando o texto já responde
let espera = null

const processando = computed(() => perguntas.value.some((p) => p.estado === 'processando' || ['na_fila', 'rodando'].includes(p.integracao?.estado)))

async function carregar() {
  try {
    const lista = await api(`/api/perguntas?local=${encodeURIComponent(props.local)}`)
    perguntas.value = props.antigas ? lista.filter((p) => p.modo !== 'conversa') : lista
  } catch (e) {
    erro.value = e.status === 0 ? 'Perguntar só funciona na rede de casa.' : e.message
  }
  clearTimeout(espera)
  if (processando.value) espera = setTimeout(carregar, 4000)
}

async function perguntar() {
  if (!texto.value.trim()) return
  enviando.value = true
  erro.value = ''
  try {
    await api('/api/perguntas', { metodo: 'POST', corpo: { local: props.local, titulo: props.titulo, pergunta: texto.value } })
    texto.value = ''
    await carregar()
  } catch (e) {
    erro.value = e.status === 0 ? 'Perguntar só funciona na rede de casa.' : e.message
  } finally {
    enviando.value = false
  }
}

async function refazer(p) {
  try {
    await api(`/api/perguntas/${p.id}/refazer`, { metodo: 'POST' })
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

async function apagar(p) {
  if (!confirm('Apagar esta pergunta?')) return
  try {
    await api(`/api/perguntas/${p.id}`, { metodo: 'DELETE' })
    await carregar()
  } catch (e) {
    erro.value = e.message
  }
}

// Integrar ao texto (D36): o Lucas escolhe o que fazer com cada divergência e se a resposta entra.
// As opções dependem do tipo: o que está errado se substitui ou se mantém ao lado; o que falta se
// acrescenta. A escolha já vem marcada com o mais comum, mas é sempre dele.
const OPCOES = {
  erro: [['substituir', 'Substituir'], ['manter', 'Manter as duas versões'], ['ignorar', 'Ignorar']],
  desatualizado: [['substituir', 'Substituir'], ['manter', 'Manter as duas versões'], ['ignorar', 'Ignorar']],
  omissao: [['acrescentar', 'Acrescentar'], ['ignorar', 'Ignorar']],
  melhoria: [['acrescentar', 'Acrescentar'], ['ignorar', 'Ignorar']],
}
const integrando = ref(null) // { id, resposta, escolhas: [acao por divergência] }
const enviandoIntegracao = ref(false)

const podeIntegrar = (p) => p.resultado && (p.resultado.cobertura !== 'responde' || p.resultado.divergencias.length)

function abrirIntegracao(p) {
  integrando.value = {
    id: p.id,
    resposta: p.resultado.cobertura !== 'responde',
    escolhas: p.resultado.divergencias.map((d) => (OPCOES[d.tipo] || OPCOES.melhoria)[0][0]),
  }
}

const algoParaIntegrar = computed(() => integrando.value && (integrando.value.resposta || integrando.value.escolhas.some((a) => a !== 'ignorar')))

async function integrar() {
  const i = integrando.value
  enviandoIntegracao.value = true
  erro.value = ''
  try {
    await api(`/api/perguntas/${i.id}/integrar`, {
      metodo: 'POST',
      corpo: { resposta: i.resposta, escolhas: i.escolhas.map((acao, divergencia) => ({ divergencia, acao })) },
    })
    integrando.value = null
    await carregar()
  } catch (e) {
    erro.value = e.status === 0 ? 'Integrar só funciona na rede de casa.' : e.message
  } finally {
    enviandoIntegracao.value = false
  }
}

// Com o texto respondendo por inteiro e sem divergência, a resposta da IA fica recolhida.
const textoBasta = (r) => r.cobertura === 'responde' && !r.divergencias.length

function html(md) {
  return renderizar(md)
}

function quando(iso) {
  return new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function aoTeclar(ev) {
  if (ev.key === 'Escape' && !props.embutido) emit('fechar')
}
onMounted(() => {
  document.addEventListener('keydown', aoTeclar)
  carregar()
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', aoTeclar)
  clearTimeout(espera)
})
</script>

<template>
  <Teleport to="body" :disabled="embutido">
    <div :class="embutido ? 'embutido' : 'veu'" @click="embutido || emit('fechar')">
      <div class="janela" :role="embutido ? null : 'dialog'" :aria-modal="embutido ? null : 'true'" :aria-labelledby="embutido ? null : 'titulo-perguntas'" @click.stop>
        <header v-if="!embutido">
          <h2 id="titulo-perguntas">Perguntar<span v-if="titulo" class="suave"> · {{ titulo }}</span></h2>
          <button type="button" class="botao icone" aria-label="Fechar" @click="emit('fechar')"><Icone nome="fechar" /></button>
        </header>
        <form v-if="!antigas" class="nova" @submit.prevent="perguntar">
          <textarea v-model="texto" class="campo" rows="2" maxlength="500" placeholder="O que você quer saber sobre este texto?" @keydown.enter.exact.prevent="perguntar" />
          <button class="botao principal" :disabled="enviando || !texto.trim()">{{ enviando ? 'Enviando…' : 'Perguntar' }}</button>
        </form>
        <p v-if="!antigas" class="suave">A IA vê quanto o seu texto responde e verifica a resposta na internet, em mais de uma fonte. Leva alguns minutos; pode fechar e voltar depois.</p>
        <div v-if="erro" class="aviso erro">{{ erro }}</div>

        <article v-for="p in perguntas" :key="p.id" class="pergunta">
          <div class="cabeca">
            <strong>{{ p.pergunta }}</strong>
            <span class="suave quando">{{ quando(p.criado_em) }}</span>
          </div>

          <p v-if="p.estado === 'processando'" class="chip andando">Verificando…</p>
          <template v-else-if="p.estado === 'falhou'">
            <p class="aviso erro">{{ p.erro }}</p>
          </template>

          <template v-else-if="p.resultado">
            <div class="selos">
              <span class="chip" :class="COBERTURA[p.resultado.cobertura]?.[1]">{{ COBERTURA[p.resultado.cobertura]?.[0] }}</span>
              <span class="chip" :class="{ ok: p.resultado.selo === 'confirmado' }">{{ SELO[p.resultado.selo] }}</span>
            </div>
            <p v-if="p.resultado.no_texto" class="no-texto"><span class="suave">No seu texto:</span> {{ p.resultado.no_texto }}</p>

            <template v-if="textoBasta(p.resultado)">
              <button class="botao discreto" @click="aberta = aberta === p.id ? null : p.id">
                {{ aberta === p.id ? 'Esconder a resposta verificada' : 'Ver a resposta verificada' }}
              </button>
              <div v-if="aberta === p.id" class="texto resposta" v-html="html(p.resultado.resposta)" />
            </template>
            <div v-else class="texto resposta" v-html="html(p.resultado.resposta)" />

            <section v-if="p.resultado.divergencias.length" class="divergencias">
              <h3>O que difere do seu texto</h3>
              <div v-for="(d, i) in p.resultado.divergencias" :key="i" class="divergencia" :class="d.tipo">
                <div class="linha">
                  <span class="tipo">{{ TIPO[d.tipo] || d.tipo }}</span>
                  <span v-if="d.capitulo" class="suave">· {{ d.capitulo }}</span>
                </div>
                <p v-if="d.no_texto"><span class="suave">O texto diz:</span> “{{ d.no_texto }}”</p>
                <p><span class="suave">As fontes dizem:</span> {{ d.correto }}</p>
                <p class="suave pequeno">{{ SELO[d.selo] }}<template v-if="d.fontes?.length"> · {{ d.fontes.map((n) => `[${n}]`).join(' ') }}</template></p>
              </div>
            </section>

            <details v-if="p.resultado.fontes.length" class="fontes" :open="!textoBasta(p.resultado)">
              <summary class="suave">Fontes ({{ p.resultado.fontes.length }})</summary>
              <ol>
                <li v-for="f in p.resultado.fontes" :key="f.url"><a :href="f.url" target="_blank" rel="noopener">{{ f.titulo || f.url }}</a></li>
              </ol>
            </details>
          </template>

          <section v-if="integrando?.id === p.id" class="integrar">
            <h3>Integrar ao texto</h3>
            <p class="suave">A IA encaixa no capítulo e no ponto certos, nunca colado no fim, e você revisa a mudança antes de entrar.</p>
            <label class="marcar"><input v-model="integrando.resposta" type="checkbox" /> Levar a resposta verificada para o texto</label>
            <div v-for="(d, i) in p.resultado.divergencias" :key="i" class="escolha">
              <span class="suave">{{ TIPO[d.tipo] || d.tipo }}<template v-if="d.capitulo"> · {{ d.capitulo }}</template>: {{ d.correto }}</span>
              <div class="opcoes" role="radiogroup" :aria-label="`O que fazer com: ${d.correto}`">
                <button
                  v-for="[valor, nome] in OPCOES[d.tipo] || OPCOES.melhoria"
                  :key="valor"
                  type="button"
                  role="radio"
                  class="opcao"
                  :aria-checked="integrando.escolhas[i] === valor"
                  :class="{ ativa: integrando.escolhas[i] === valor }"
                  @click="integrando.escolhas[i] = valor"
                >{{ nome }}</button>
              </div>
            </div>
            <div class="linha-acoes">
              <button class="botao principal" :disabled="enviandoIntegracao || !algoParaIntegrar" @click="integrar">{{ enviandoIntegracao ? 'Enviando…' : 'Preparar a proposta' }}</button>
              <button class="botao" @click="integrando = null">Cancelar</button>
            </div>
          </section>
          <!-- Uma proposta por pergunta (D44): enquanto ela espera, nada de pedir outra nem verificar de novo. -->
          <p v-else-if="p.integracao?.estado === 'em_revisao'" class="aviso">A proposta está pronta. <a :href="`#/revisao/${p.integracao.tarefa_id}`">Revisar e aprovar</a></p>
          <p v-else-if="p.integracao" class="aviso">Preparando a proposta. Quando ficar pronta, o link para revisar aparece aqui.</p>

          <div v-if="p.estado !== 'processando'" class="acoes">
            <button v-if="podeIntegrar(p) && !p.integracao && integrando?.id !== p.id" class="botao discreto marca" @click="abrirIntegracao(p)">Integrar ao texto</button>
            <button v-if="!p.integracao" class="botao discreto" @click="refazer(p)">Verificar de novo</button>
            <button class="botao discreto perigo" @click="apagar(p)">Apagar</button>
          </div>
        </article>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.veu {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  background: rgb(0 0 0 / 22%);
  -webkit-backdrop-filter: blur(4px);
  backdrop-filter: blur(4px);
}

.janela {
  display: grid;
  align-content: start;
  gap: 12px;
  width: min(640px, 100%);
  max-height: 90vh;
  overflow-y: auto;
  padding: 16px 20px max(16px, env(safe-area-inset-bottom));
  border-radius: var(--raio-g) var(--raio-g) 0 0;
  background: var(--fundo);
  box-shadow: var(--sombra);
}

.embutido .janela {
  width: 100%;
  max-height: none;
  overflow: visible;
  padding: 0;
  border-radius: 0;
  background: none;
  box-shadow: none;
}

@media (min-width: 640px) {
  .veu {
    align-items: center;
  }

  .janela {
    border-radius: var(--raio-g);
  }
}

header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-right: -8px;
}

h2 {
  margin: 4px 0 0;
  font-size: 1.1rem;
  font-weight: 600;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

p {
  margin: 0;
}

.nova {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.nova textarea {
  flex: 1;
  resize: vertical;
  min-height: 44px;
  font: inherit;
}

.pergunta {
  display: grid;
  gap: 10px;
  padding-top: 14px;
  border-top: 1px solid var(--linha);
}

.cabeca {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.cabeca strong {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.quando {
  flex-shrink: 0;
  font-size: 0.8rem;
}

.selos {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.chip.meio {
  color: var(--texto);
  background: var(--superficie-forte);
}

.chip.andando {
  justify-self: start;
}

.no-texto {
  font-size: 0.92rem;
}

.resposta {
  font-size: 0.98rem;
}

.resposta :deep(p:first-child) {
  margin-top: 0;
}

.resposta :deep(p:last-child) {
  margin-bottom: 0;
}

.divergencias {
  display: grid;
  gap: 8px;
}

.divergencias h3 {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 600;
}

.divergencia {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border-radius: var(--raio);
  background: var(--superficie);
  font-size: 0.92rem;
}

.divergencia.erro,
.divergencia.desatualizado {
  background: var(--vermelho-fundo);
}

.divergencia .linha {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.divergencia .tipo {
  font-weight: 600;
}

.pequeno {
  font-size: 0.8rem;
}

.fontes summary {
  cursor: pointer;
  font-size: 0.88rem;
}

.fontes ol {
  margin: 6px 0 0;
  padding-left: 22px;
  font-size: 0.88rem;
}

.fontes a {
  overflow-wrap: anywhere;
}

.acoes {
  display: flex;
  gap: 4px;
  margin-left: -8px;
}

.botao.discreto {
  justify-self: start;
  border-color: transparent;
  background: transparent;
  color: var(--texto-suave);
  font-size: 0.88rem;
}

.botao.discreto.perigo {
  color: var(--perigo);
}

.botao.discreto.marca {
  color: var(--marca);
}

.integrar {
  display: grid;
  gap: 10px;
  padding: 12px 14px;
  border-radius: var(--raio);
  background: var(--superficie);
}

.integrar h3 {
  margin: 0;
  font-size: 1.0625rem;
  font-weight: 650;
}

.integrar > p {
  font-size: 0.906rem;
}

.marcar {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 0.95rem;
}

.escolha {
  display: grid;
  gap: 6px;
  font-size: 0.92rem;
}

.opcoes {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.opcao {
  padding: 6px 12px;
  border: 1px solid var(--linha-forte);
  border-radius: 999px;
  background: var(--fundo);
  color: var(--texto);
  font: inherit;
  font-size: 0.88rem;
  cursor: pointer;
}

.opcao.ativa {
  border-color: var(--marca);
  background: var(--marca-fundo);
  color: var(--marca);
  font-weight: 600;
}
</style>
