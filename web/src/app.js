import Alpine from 'alpinejs';
import focus from '@alpinejs/focus';
import htmx from 'htmx.org';
import { createLogin } from './identity.js';
import { createEditor } from './editor.js';
import './app.css';
import './conditions.css';
import './cloudline.css';
import './moonlight.css';
import './identity.css';
import './toolbar.css';
import './generator-docs.css';
import { createTheme } from './theme.js';

window.htmx = htmx;
Alpine.plugin(focus);
Alpine.store('theme', createTheme());
Alpine.data('mocklab', createEditor);
Alpine.data('identityLogin',createLogin);
window.Alpine = Alpine;
Alpine.data('generatorDocs',()=>({
 query:'',copyStatus:'',
 matches(text){return text.toLowerCase().includes(this.query.trim().toLowerCase());},
 hasMatches(available){return [...this.$root.querySelectorAll('[data-generator]')].some(card=>card.dataset.available===String(available)&&this.matches(card.dataset.search));},
 async copyExample(text){try{if(navigator.clipboard?.writeText){await navigator.clipboard.writeText(text);}else{const input=document.createElement('textarea');input.value=text;document.body.append(input);input.select();if(!document.execCommand('copy'))throw Error();input.remove();}this.copyStatus='JSON copied.';}catch{this.copyStatus='Copy unavailable. Select the example text to copy it.';}}
}));
Alpine.start();
