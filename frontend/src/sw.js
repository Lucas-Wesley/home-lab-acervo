import { clientsClaim } from 'workbox-core'
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'

self.skipWaiting()
clientsClaim()

precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()
registerRoute(new NavigationRoute(createHandlerBoundToURL('index.html'), { denylist: [/^\/api\//] }))

// Os Temas baixados ficam neste cache (offline.js). Na rede, vale sempre a versão do servidor, e
// ela atualiza a cópia baixada; fora da rede, responde a cópia (D21).
const TEMAS = 'acervo-temas'
const ARQUIVO = /^\/api\/arquivos\//

registerRoute(
  ({ url, request }) => url.origin === location.origin && request.method === 'GET' && ARQUIVO.test(url.pathname),
  async ({ url, request }) => {
    const cache = await caches.open(TEMAS)
    try {
      const resposta = await fetch(request, { signal: AbortSignal.timeout(5000) })
      if (resposta.ok && (await cache.match(url.pathname))) await cache.put(url.pathname, resposta.clone())
      return resposta
    } catch (erro) {
      const copia = await cache.match(url.pathname)
      if (copia) return copia
      throw erro
    }
  },
)
