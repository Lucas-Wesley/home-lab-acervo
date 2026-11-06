// Uma Biblioteca de mentira para os testes de navegador: a API de livros e capítulos (R33 de lá).
import http from 'node:http'

const livro = { id: 'livro1', title: 'Pães de Fermentação Natural', author: 'Autora', format: 'epub' }
const capitulos = [
  { n: 1, title: 'O levain', level: 1, first_page: 1, last_page: 12, chars: 18000 },
  { n: 2, title: 'A farinha', level: 1, first_page: 13, last_page: 20, chars: 12000 },
  { n: 3, title: 'Tipos de trigo', level: 2, first_page: 15, last_page: 20, chars: 9000 },
]

function responder(res, status, corpo) {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(corpo))
}

http
  .createServer((req, res) => {
    const url = new URL(req.url, 'http://localhost')
    if (url.pathname === '/api/books') return responder(res, 200, [livro])
    if (url.pathname === '/api/books/livro1/chapters') return responder(res, 200, { ...livro, indexed: true, pages: 20, chapters: capitulos })
    if (url.pathname === '/api/books/livro1/text') {
      const ns = (url.searchParams.get('chapters') ?? '').split(',').map(Number)
      const parts = capitulos
        .filter((c) => ns.includes(c.n))
        .map((c) => ({
          title: c.title,
          first_page: c.first_page,
          last_page: c.last_page,
          passages: [{ page: c.first_page, text: `Texto do capítulo ${c.title}. `.repeat(20), anchor: `{"pagina":${c.first_page}}` }],
        }))
      return responder(res, 200, { ...livro, parts })
    }
    if (url.pathname === '/api/search') return responder(res, 200, { results: [] })
    responder(res, 404, { error: 'não encontrado' })
  })
  .listen(Number(process.env.PORT || 8093))
