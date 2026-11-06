// Um Flashcards de mentira para os testes de navegador: um baralho, e guarda o último texto recebido.
import http from 'node:http'

let ultimo = null

function responder(res, status, corpo) {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(corpo))
}

http
  .createServer((req, res) => {
    const url = new URL(req.url, 'http://localhost')
    if (url.pathname === '/api/baralhos') return responder(res, 200, [{ id: 'b1', nome: 'Tema de Teste' }, { id: 'b2', nome: 'Inglês' }])
    if (url.pathname === '/ultimo') return responder(res, 200, ultimo)
    if (req.method === 'POST' && url.pathname === '/api/textos') {
      let corpo = ''
      req.on('data', (parte) => (corpo += parte))
      req.on('end', () => {
        ultimo = JSON.parse(corpo)
        responder(res, 201, { id: 7, estado: 'na_fila' })
      })
      return
    }
    responder(res, 404, { erro: 'não encontrado' })
  })
  .listen(Number(process.env.PORT || 8097))
