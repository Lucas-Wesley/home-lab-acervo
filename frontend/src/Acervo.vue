<script setup>
import { computed, onMounted, ref } from 'vue'
import Icone from './Icone.vue'
import { api } from './api.js'
import { baixarTema, carregarTemas, removerDownload, temasBaixados } from './offline.js'

const props = defineProps({ soBaixados: Boolean })

const temas = ref([])
const carregou = ref(false)
const semConexao = ref(false)
const erro = ref('')
const baixados = ref(temasBaixados())
const baixando = ref({})
const formulario = ref(null) // { titulo, pai }
const criando = ref(false)

const principais = computed(() => temas.value.filter((t) => !t.pai))
const filhos = computed(() => {
  const mapa = {}
  for (const t of temas.value) if (t.pai) (mapa[t.pai] ??= []).push(t)
  return mapa
})
const listaBaixados = computed(() =>
  Object.entries(baixados.value).map(([caminho, b]) => ({ caminho, ...b, tema: temas.value.find((t) => t.caminho === caminho) })),
)

async function carregar() {
  try {
    const r = await carregarTemas()
    temas.value = r.temas
    semConexao.value = r.semConexao
  } catch (e) {
    erro.value = e.message
  }
  carregou.value = true
}

function abrir(tema) {
  location.hash = `#/ler/${tema.caminho}/indice.md`
}

async function criar() {
  const { titulo, pai } = formulario.value
  if (!titulo.trim()) return
  criando.value = true
  erro.value = ''
  try {
    const { caminho } = await api('/api/temas', { metodo: 'POST', corpo: { titulo, pai } })
    location.hash = `#/ler/${caminho}/indice.md?editar`
  } catch (e) {
    erro.value = e.message
  } finally {
    criando.value = false
  }
}

async function alternarDownload(tema) {
  if (baixando.value[tema.caminho] !== undefined) return
  if (baixados.value[tema.caminho]) {
    await removerDownload(tema.caminho)
  } else {
    baixando.value = { ...baixando.value, [tema.caminho]: 0 }
    try {
      await baixarTema(tema, (f) => (baixando.value = { ...baixando.value, [tema.caminho]: f }))
    } catch (e) {
      erro.value = `Não foi possível baixar "${tema.titulo}": ${e.message}`
    }
    const { [tema.caminho]: _, ...resto } = baixando.value
    baixando.value = resto
  }
  baixados.value = temasBaixados()
}

onMounted(carregar)
</script>

<template>
  <div class="pagina">
    <template v-if="props.soBaixados">
      <h1>Baixados</h1>
      <p class="suave explicacao">Temas guardados neste aparelho para ler e editar fora da rede. Na rede, vale sempre a versão do servidor.</p>
      <div v-if="!listaBaixados.length" class="vazio">Nenhum Tema baixado. Abra um Tema e toque em <strong>Baixar</strong>.</div>
      <div v-for="b in listaBaixados" :key="b.caminho" class="tema baixado">
        <button class="abrir" @click="abrir(b)">
          <strong>{{ b.titulo }}</strong>
          <span class="suave">Baixado em {{ new Date(b.em).toLocaleString('pt-BR') }} · {{ b.arquivos.length }} arquivos</span>
        </button>
        <button class="botao perigo" @click="alternarDownload(b)">Remover</button>
      </div>
    </template>

    <template v-else>
      <div class="topo">
        <h1>Acervo</h1>
        <button class="botao" @click="formulario = formulario ? null : { titulo: '', pai: '' }"><Icone nome="mais" /> Novo Tema</button>
      </div>

      <div v-if="semConexao" class="aviso">Sem conexão: mostrando só os Temas baixados.</div>
      <div v-if="erro" class="aviso erro">{{ erro }}</div>

      <form v-if="formulario" class="cartao novo" @submit.prevent="criar">
        <label>
          <span>Título</span>
          <input v-model="formulario.titulo" class="campo" placeholder="Ex.: Segunda Guerra Mundial" autofocus />
        </label>
        <label>
          <span>Onde</span>
          <select v-model="formulario.pai" class="campo">
            <option value="">Tema principal</option>
            <option v-for="t in temas" :key="t.caminho" :value="t.caminho">Dentro de: {{ t.titulo }}</option>
          </select>
        </label>
        <div class="linha-acoes">
          <button class="botao principal" :disabled="criando || !formulario.titulo.trim()">Criar e escrever</button>
          <button type="button" class="botao" @click="formulario = null">Cancelar</button>
        </div>
      </form>

      <div v-if="carregou && !temas.length && !semConexao" class="vazio">
        O Acervo ainda está vazio.<br />
        Adicione um vídeo em <a href="#/referencias">Referências</a> ou crie um Tema à mão.
      </div>

      <div v-if="principais.length" class="temas lista">
        <article v-for="t in principais" :key="t.caminho" class="tema">
          <button class="abrir" @click="abrir(t)">
            <strong class="titulo">{{ t.titulo }}</strong>
            <span v-if="t.resumo" class="resumo">{{ t.resumo }}</span>
            <span class="suave meta">
              {{ t.capitulos }} {{ t.capitulos === 1 ? 'capítulo' : 'capítulos' }}<template v-if="filhos[t.caminho]"> · {{ filhos[t.caminho].length }} {{ filhos[t.caminho].length === 1 ? 'subtema' : 'subtemas' }}</template>
            </span>
          </button>
          <button
            class="botao icone"
            :aria-label="baixados[t.caminho] ? 'Remover do aparelho' : 'Baixar para ler fora da rede'"
            :title="baixados[t.caminho] ? 'Baixado (toque para remover)' : 'Baixar'"
            @click="alternarDownload(t)"
          >
            <span v-if="baixando[t.caminho] !== undefined" class="progresso">{{ Math.round(baixando[t.caminho] * 100) }}%</span>
            <Icone v-else :nome="baixados[t.caminho] ? 'baixado' : 'baixar'" />
          </button>
          <ul v-if="filhos[t.caminho]" class="subtemas">
            <li v-for="s in filhos[t.caminho]" :key="s.caminho">
              <a :href="`#/ler/${s.caminho}/indice.md`">{{ s.titulo }}</a>
              <span v-if="filhos[s.caminho]" class="suave"> · {{ filhos[s.caminho].length }} subtemas</span>
            </li>
          </ul>
        </article>
      </div>
    </template>
  </div>
</template>

<style scoped>
.topo {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 20px;
}

.topo h1 {
  margin: 0;
}

.explicacao {
  margin-top: -8px;
  line-height: 1.5;
}

.aviso {
  margin-bottom: 14px;
}

.novo {
  display: grid;
  gap: 12px;
  margin-bottom: 20px;
  background: var(--superficie);
  border-color: transparent;
}

.novo label {
  display: grid;
  gap: 4px;
  font-size: 0.8rem;
  color: var(--texto-suave);
}

.tema {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: start;
  gap: 2px 8px;
  padding: 18px 0;
}

.tema.baixado {
  align-items: center;
  border-bottom: 1px solid var(--linha);
}

.abrir {
  display: grid;
  gap: 4px;
  padding: 0;
  border: 0;
  background: none;
  text-align: left;
  cursor: pointer;
}

.abrir:hover .titulo,
.abrir:hover strong {
  color: var(--marca);
}

.titulo {
  font-size: 1.06rem;
  font-weight: 650;
  line-height: 1.3;
  letter-spacing: -0.012em;
  transition: color 0.15s ease;
}

.resumo {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  color: var(--texto-suave);
  font-size: 0.92rem;
  line-height: 1.5;
}

.meta {
  margin-top: 2px;
  color: var(--texto-fraco);
  font-size: 0.78rem;
}

.progresso {
  font-size: 0.72rem;
}

.subtemas {
  grid-column: 1 / -1;
  display: flex;
  flex-wrap: wrap;
  gap: 4px 14px;
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
  font-size: 0.85rem;
}

.subtemas li::before {
  content: '↳ ';
  color: var(--texto-fraco);
}

.subtemas a {
  color: var(--texto-suave);
  text-decoration: none;
}

.subtemas a:hover {
  color: var(--marca);
}
</style>
