import { expect, it } from 'vitest'
import { lerDiff, versaoNova } from './diff.js'

const DIFF = `diff --git a/temas/a/indice.md b/temas/a/indice.md
index 1..2 100644
--- a/temas/a/indice.md
+++ b/temas/a/indice.md
@@ -1,3 +1,4 @@
 # A
-velho
+novo
+mais um
 fim
diff --git a/temas/a/novo.md b/temas/a/novo.md
new file mode 100644
--- /dev/null
+++ b/temas/a/novo.md
@@ -0,0 +1,2 @@
+# Novo
+Texto
`

it('lê arquivos, linhas e contagens', () => {
  const [indice, novo] = lerDiff(DIFF)
  expect(indice.caminho).toBe('temas/a/indice.md')
  expect(indice.novo).toBe(false)
  expect([indice.mais, indice.menos]).toEqual([2, 1])
  expect(novo.novo).toBe(true)
  expect(versaoNova(novo)).toBe('# Novo\nTexto')
  expect(versaoNova(indice)).toBe('# A\nnovo\nmais um\nfim')
})
