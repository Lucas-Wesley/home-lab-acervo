<script setup>
defineProps({ nome: { type: String, required: true } })
</script>

<template>
  <svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path v-if="nome === 'voltar'" d="M15 5l-7 7 7 7" />
    <path v-else-if="nome === 'avancar'" d="M9 5l7 7-7 7" />
    <path v-else-if="nome === 'lista'" d="M9 6.5h11M9 12h11M9 17.5h11M4.5 6.5h.01M4.5 12h.01M4.5 17.5h.01" />
    <path v-else-if="nome === 'menu'" d="M4 7h16M4 12h16M4 17h16" />
    <path v-else-if="nome === 'pontos'" d="M5.5 12h.01M12 12h.01M18.5 12h.01" stroke-width="2.6" />
    <path v-else-if="nome === 'nota'" d="M5 19l1-4L16.5 4.5a2.1 2.1 0 013 3L9 18l-4 1z" />
    <path v-else-if="nome === 'fechar'" d="M6 6l12 12M18 6L6 18" />
    <path v-else-if="nome === 'expandir'" d="M4 9V4h5M15 4h5v5M20 15v5h-5M9 20H4v-5" />
    <path v-else-if="nome === 'recolher'" d="M9 4v5H4M20 9h-5V4M15 20v-5h5M4 15h5v5" />
    <path v-else-if="nome === 'mais'" d="M12 5v14M5 12h14" />
    <path v-else-if="nome === 'feito'" d="M5 12.5l4.5 4.5L19 7.5" />
    <path v-else-if="nome === 'link'" d="M10 14a4 4 0 005.7 0l3-3a4 4 0 00-5.7-5.7l-1 1M14 10a4 4 0 00-5.7 0l-3 3a4 4 0 005.7 5.7l1-1" />
    <path v-else-if="nome === 'acervo'" d="M4 5.5A1.5 1.5 0 015.5 4H10v16H5.5A1.5 1.5 0 014 18.5zM10 4h4v16h-4zM14.5 5l4-1 2.5 15-4 1z" />
    <path v-else-if="nome === 'revisao'" d="M9 11l2.5 2.5L16 9M5 4h14a1 1 0 011 1v14a1 1 0 01-1 1H5a1 1 0 01-1-1V5a1 1 0 011-1z" />
    <path v-else-if="nome === 'nuvem'" d="M7 18a4.5 4.5 0 01-.6-9A6 6 0 0118 9.5a4 4 0 01-.5 8.5z" />
    <template v-else-if="nome === 'referencias'">
      <rect x="3" y="5" width="18" height="14" rx="3" />
      <path d="M10 9.5v5l4.5-2.5z" />
    </template>
    <template v-else-if="nome === 'ajustes'">
      <path d="M4 7.5h9M17 7.5h3M4 16.5h3M11 16.5h9" />
      <circle cx="15" cy="7.5" r="2" />
      <circle cx="9" cy="16.5" r="2" />
    </template>
    <template v-else-if="nome === 'fila'">
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 7.5V12l3 2" />
    </template>
    <template v-else-if="nome === 'copiar'">
      <rect x="9" y="9" width="11" height="11" rx="2" />
      <path d="M5.5 15H5a1.5 1.5 0 01-1.5-1.5v-9A1.5 1.5 0 015 3h9a1.5 1.5 0 011.5 1.5V5" />
    </template>
    <template v-else-if="nome === 'info'">
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 11v5.5M12 7.75h.01" />
    </template>
    <path v-else-if="nome === 'livro'" d="M12 6.5C10.5 5 8 4.5 4.5 4.5v13c3.5 0 6 .5 7.5 2 1.5-1.5 4-2 7.5-2v-13c-3.5 0-6 .5-7.5 2zM12 6.5v13" />
    <template v-else-if="nome === 'pesquisar'">
      <circle cx="11" cy="11" r="6.5" />
      <path d="M16 16l4.5 4.5" />
    </template>
    <template v-else-if="nome === 'baixar'">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7.5v8M8.5 12.5l3.5 3.5 3.5-3.5" />
    </template>
    <template v-else-if="nome === 'cartoes'">
      <rect x="8" y="4" width="12" height="14" rx="1.5" />
      <path d="M5 7.5v11A1.5 1.5 0 006.5 20H16" />
    </template>
    <template v-else-if="nome === 'baixado'">
      <circle cx="12" cy="12" r="9" fill="currentColor" stroke="none" />
      <path d="M8 12.5l2.7 2.7L16 9.5" stroke="var(--fundo)" />
    </template>
  </svg>
</template>
