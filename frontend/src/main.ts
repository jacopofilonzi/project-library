import { mount } from 'svelte'
import '@fontsource/geist-sans/400.css'
import '@fontsource/geist-sans/500.css'
import '@fontsource/geist-sans/600.css'
import '@fontsource/geist-sans/700.css'
import '@fontsource/geist-mono/400.css'
import '@fontsource/geist-mono/500.css'
import './app.css'
import App from './App.svelte'
import Spotlight from './components/Spotlight.svelte'
import { isSpotlight } from './lib/state.svelte'

// la stessa build serve due finestre: la principale e la ricerca flottante (?view=spotlight)
mount(isSpotlight ? Spotlight : App, { target: document.getElementById('app')! })
