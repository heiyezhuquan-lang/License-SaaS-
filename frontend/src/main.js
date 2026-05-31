import { createApp, h } from 'vue'
import naive from 'naive-ui'
import App from './App.vue'
import './style.css'
import './dashboard.css'
import './client-api.css'
import './login.css'

createApp({ render: () => h(App) }).use(naive).mount('#app')
