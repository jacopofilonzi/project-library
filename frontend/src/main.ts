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

// the same build serves two windows: the main one and the floating search (?view=spotlight)
mount(isSpotlight ? Spotlight : App, { target: document.getElementById('app')! })
