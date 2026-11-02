import Alpine from 'alpinejs';
import focus from '@alpinejs/focus';
import htmx from 'htmx.org';
import { createEditor } from './editor.js';
import './app.css';
import './conditions.css';

window.htmx = htmx;
Alpine.plugin(focus);
Alpine.data('mocklab', createEditor);
window.Alpine = Alpine;
Alpine.start();
