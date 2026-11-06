import { api, caminhoApi, ErroApi } from './api.js'

// Temas baixados para ler e editar fora da rede (D21). Os arquivos ficam no Cache do navegador,
// com a mesma resposta da API, para o service worker devolver quando não houver rede.
const CACHE = 'acervo-temas'
const CHAVE_BAIXADOS = 'acervo-baixados'
const CHAVE_PENDENTES = 'acervo-pendentes'
const CHAVE_TEMAS = 'acervo-temas-lista'

function ler(chave, padrao) {
  try {
    return JSON.parse(localStorage.getItem(chave)) ?? padrao
  } catch {
    return padrao
  }
}

function gravar(chave, valor) {
  try {
    localStorage.setItem(chave, JSON.stringify(valor))
  } catch {
    // Sem armazenamento (janela privada): o app segue funcionando, só sem a cópia local.
  }
}

const urlDoArquivo = (caminho) => `/api/arquivos/${caminhoApi(caminho)}`

// ---------- Lista de Temas ----------

export async function carregarTemas() {
  try {
    const temas = await api('/api/temas', { tempo: 6000 })
    gravar(CHAVE_TEMAS, temas)
    return { temas, semConexao: false }
  } catch (erro) {
    if (erro.status !== 0) throw erro
    // Fora da rede, só os Temas baixados abrem.
    const baixados = temasBaixados()
    const temas = ler(CHAVE_TEMAS, []).filter((t) => Object.keys(baixados).some((b) => t.caminho === b || t.caminho.startsWith(b + '/')))
    return { temas, semConexao: true }
  }
}

// ---------- Baixar ----------

export function temasBaixados() {
  return ler(CHAVE_BAIXADOS, {})
}

export function estaBaixado(caminhoTema) {
  return Object.keys(temasBaixados()).some((b) => caminhoTema === b || caminhoTema.startsWith(b + '/'))
}

export async function baixarTema(tema, aoAvancar = () => {}) {
  const pacote = await api(`/api/pacotes/${caminhoApi(tema.caminho)}`)
  const caminhos = Object.keys(pacote.arquivos)
  const cache = await caches.open(CACHE)
  let feitos = 0
  for (const caminho of caminhos) {
    const resposta = await fetch(urlDoArquivo(caminho), { cache: 'no-store' })
    if (!resposta.ok) throw new Error(`não foi possível baixar ${caminho}`)
    await cache.put(urlDoArquivo(caminho), resposta)
    aoAvancar(++feitos / caminhos.length)
  }
  gravar(CHAVE_BAIXADOS, { ...temasBaixados(), [tema.caminho]: { titulo: tema.titulo, em: new Date().toISOString(), arquivos: caminhos } })
}

export async function removerDownload(caminhoTema) {
  const baixados = temasBaixados()
  const cache = await caches.open(CACHE)
  for (const caminho of baixados[caminhoTema]?.arquivos ?? []) await cache.delete(urlDoArquivo(caminho))
  delete baixados[caminhoTema]
  gravar(CHAVE_BAIXADOS, baixados)
}

// ---------- Ler e salvar ----------

export function pendentes() {
  return ler(CHAVE_PENDENTES, {})
}

function gravarPendente(caminho, valor) {
  const lista = pendentes()
  if (valor) lista[caminho] = valor
  else delete lista[caminho]
  gravar(CHAVE_PENDENTES, lista)
}

// Lê um arquivo; uma edição ainda não enviada aparece no lugar da versão do servidor.
export async function lerArquivo(caminho) {
  const arquivo = await api(urlDoArquivo(caminho), { tempo: 8000 })
  const pendente = pendentes()[caminho]
  if (pendente) return { ...arquivo, conteudo: pendente.conteudo, hash: pendente.base, pendente }
  return arquivo
}

async function atualizarCopia(caminho, mudar) {
  if (!('caches' in window)) return
  const cache = await caches.open(CACHE)
  const copia = await cache.match(urlDoArquivo(caminho))
  if (!copia) return
  const dados = mudar(await copia.json())
  await cache.put(urlDoArquivo(caminho), new Response(JSON.stringify(dados), { headers: { 'Content-Type': 'application/json' } }))
}

export class Conflito extends Error {
  constructor(atual, minha) {
    super('O arquivo mudou no servidor.')
    this.atual = atual
    this.minha = minha
  }
}

// salvar envia a edição; sem rede, guarda no aparelho para subir depois.
export async function salvar(caminho, conteudo, base) {
  try {
    const { hash } = await api(urlDoArquivo(caminho), { metodo: 'PUT', corpo: { conteudo, base } })
    gravarPendente(caminho, null)
    await atualizarCopia(caminho, (a) => ({ ...a, conteudo, hash }))
    return { hash, pendente: false }
  } catch (erro) {
    if (erro instanceof ErroApi && erro.status === 409) throw new Conflito(erro.dados.atual, conteudo)
    if (!(erro instanceof ErroApi) || erro.status !== 0) throw erro
    gravarPendente(caminho, { conteudo, base, em: new Date().toISOString() })
    return { hash: base, pendente: true }
  }
}

// sincronizar sobe as edições feitas fora da rede. Se o arquivo mudou no servidor nesse meio
// tempo, a edição fica marcada como conflito para o Lucas escolher (D21).
export async function sincronizar() {
  const resultado = { enviadas: 0, conflitos: [] }
  for (const [caminho, pendente] of Object.entries(pendentes())) {
    if (pendente.conflito) {
      resultado.conflitos.push(caminho)
      continue
    }
    try {
      await salvar(caminho, pendente.conteudo, pendente.base)
      resultado.enviadas++
    } catch (erro) {
      if (erro instanceof Conflito) {
        gravarPendente(caminho, { ...pendente, conflito: erro.atual })
        resultado.conflitos.push(caminho)
      } else break
    }
  }
  return resultado
}

export function descartarPendente(caminho) {
  gravarPendente(caminho, null)
}

export function conflitos() {
  return Object.entries(pendentes())
    .filter(([, p]) => p.conflito)
    .map(([caminho]) => caminho)
}
