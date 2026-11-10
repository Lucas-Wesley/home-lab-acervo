<script setup>
import { onMounted, ref } from 'vue'
import { api } from './api.js'
import Icone from './Icone.vue'

// Compartilhar → Acervo, no Android (D34). O share_target abre "/compartilhar?text=…&url=…" no lugar
// do app inteiro: esta tela manda o link (vídeo do YouTube ou qualquer página) e se fecha sozinha,
// voltando ao app de onde se compartilhou. O navegador só deixa fechar uma janela aberta para isso;
// se o app já estava aberto, ela fica, com o resultado e o "voltar" do celular.
const estado = ref('enviando') // enviando | feito | erro
const mensagem = ref('Enviando ao Acervo…')
const abrir = ref('/#/referencias')
const ficou = ref(false)

onMounted(async () => {
  const p = new URLSearchParams(location.search)
  const enviado = [p.get('url'), p.get('text'), p.get('title')].filter(Boolean).join(' ')
  history.replaceState(null, '', '/compartilhar')
  if (!enviado) {
    estado.value = 'erro'
    mensagem.value = 'Nada para enviar: compartilhe um link.'
    return
  }
  try {
    const r = await api('/api/referencias', { metodo: 'POST', corpo: { url: enviado } })
    estado.value = 'feito'
    mensagem.value = { artigo: 'Página enviada para a fila.', podcast: 'Episódio enviado para a fila.', reel: 'Reel enviado para a fila.' }[r.tipo] || 'Vídeo enviado para a fila.'
    abrir.value = `/#/referencia/${r.id}`
  } catch (e) {
    if (e.status === 409 && e.dados?.id) {
      estado.value = 'feito'
      mensagem.value = e.message
      abrir.value = `/#/referencia/${e.dados.id}`
    } else {
      estado.value = 'erro'
      mensagem.value = e.message
      return // com erro, a tela fica para ser lida
    }
  }
  setTimeout(() => {
    window.close()
    // Se ainda está aqui, o navegador não deixou fechar.
    setTimeout(() => (ficou.value = true), 300)
  }, 1200)
})
</script>

<template>
  <main class="compartilhar">
    <div class="caixa vidro" :class="estado" role="status">
      <Icone :nome="estado === 'erro' ? 'info' : estado === 'feito' ? 'feito' : 'referencias'" />
      <span>{{ mensagem }}</span>
    </div>
    <p v-if="ficou || estado === 'erro'" class="suave dica">
      <template v-if="ficou">Pode voltar ao app de onde compartilhou. </template>
      <a :href="abrir">Abrir no Acervo</a>
    </p>
  </main>
</template>

<style scoped>
.compartilhar {
  display: grid;
  min-height: 100vh;
  place-content: center;
  justify-items: center;
  gap: 14px;
  padding: 24px 16px;
  text-align: center;
}

.caixa {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 18px;
  border: 1px solid var(--vidro-borda);
  border-radius: 999px;
  box-shadow: var(--sombra);
  font-size: 0.95rem;
}

.caixa :deep(svg) {
  flex: none;
  width: 20px;
  height: 20px;
}

.caixa.enviando :deep(svg) {
  animation: pulsar 1s ease-in-out infinite;
}

.caixa.feito :deep(svg) {
  color: var(--azul);
}

.caixa.erro {
  color: var(--vermelho);
}

.dica {
  margin: 0;
  font-size: 0.85rem;
}

@keyframes pulsar {
  50% {
    opacity: 0.35;
  }
}
</style>
