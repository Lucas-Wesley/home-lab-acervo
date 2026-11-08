import { createApp } from 'vue'
import '@fontsource-variable/literata'
import App from './App.vue'
import Compartilhar from './Compartilhar.vue'
import { iniciarHistorico } from './rolagem.js'
import './style.css'

iniciarHistorico()

// O "Compartilhar" do Android abre só a tela curta que manda o link (D34); "/?text=…" é o endereço
// antigo, de quem instalou o app antes, até o Android atualizar o manifest.
const compartilhado = location.pathname === '/compartilhar' || (location.pathname === '/' && /[?&](url|text)=/.test(location.search))
createApp(compartilhado ? Compartilhar : App).mount('#app')
