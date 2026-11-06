// Lê a saída do "git diff" em arquivos e linhas, para a tela de revisão (D16).
export function lerDiff(texto) {
  const arquivos = []
  let atual = null
  for (const linha of texto.split('\n')) {
    if (linha.startsWith('diff --git ')) {
      atual = { caminho: linha.split(' b/').pop(), novo: false, linhas: [], mais: 0, menos: 0 }
      arquivos.push(atual)
    } else if (!atual) {
      continue
    } else if (linha.startsWith('new file mode')) {
      atual.novo = true
    } else if (linha.startsWith('+++ ') || linha.startsWith('--- ') || linha.startsWith('index ') || linha.startsWith('similarity') || linha.startsWith('rename ')) {
      continue
    } else if (linha.startsWith('@@')) {
      atual.linhas.push({ tipo: 'trecho', texto: linha })
    } else if (linha.startsWith('+')) {
      atual.linhas.push({ tipo: 'mais', texto: linha.slice(1) })
      atual.mais++
    } else if (linha.startsWith('-')) {
      atual.linhas.push({ tipo: 'menos', texto: linha.slice(1) })
      atual.menos++
    } else if (linha.startsWith(' ')) {
      atual.linhas.push({ tipo: 'igual', texto: linha.slice(1) })
    }
  }
  return arquivos
}

// Como o arquivo fica depois da mudança (só os trechos que o diff mostra).
export function versaoNova(arquivo) {
  return arquivo.linhas
    .filter((l) => l.tipo === 'mais' || l.tipo === 'igual')
    .map((l) => l.texto)
    .join('\n')
}
