import { createApp } from 'vue'
import '@fontsource-variable/literata'
import App from './App.vue'
import { iniciarHistorico } from './rolagem.js'
import './style.css'

iniciarHistorico()

createApp(App).mount('#app')
