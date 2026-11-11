<script setup>
// O símbolo de onde veio a Referência (D46): Instagram ou Spotify, nas cores deles.
defineProps({ nome: { type: String, required: true } })
</script>

<template>
  <svg v-if="nome === 'instagram'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
    <rect x="3" y="3" width="18" height="18" rx="5.5" />
    <circle cx="12" cy="12" r="4.2" />
    <circle cx="17.4" cy="6.6" r="1.2" fill="currentColor" stroke="none" />
  </svg>
  <svg v-else-if="nome === 'spotify'" viewBox="0 0 24 24" aria-hidden="true">
    <circle cx="12" cy="12" r="10.5" fill="#1ed760" />
    <path d="M6.6 9.2c3.8-1.1 8.1-.8 11.2 1M7.2 12.4c3.1-.8 6.5-.5 9.2 1M7.8 15.4c2.5-.6 5-.4 7.1.8" fill="none" stroke="#121212" stroke-linecap="round" stroke-width="1.7" />
  </svg>
</template>
