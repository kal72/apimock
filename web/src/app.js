import Alpine from 'alpinejs';
import focus from '@alpinejs/focus';
import htmx from 'htmx.org';
import { createEditor } from './editor.js';
import './app.css';
import './conditions.css';
import './cloudline.css';
import './moonlight.css';
import { createTheme } from './theme.js';

window.htmx = htmx;
Alpine.plugin(focus);
Alpine.store('theme', createTheme());
Alpine.data('mocklab', createEditor);
window.Alpine = Alpine;
Alpine.start();
