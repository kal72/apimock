import test from 'node:test';
import assert from 'node:assert/strict';
import { createTheme } from './theme.js';

function browser(saved, dark = false, blocked = false) {
  const listeners = {}, media = { matches: dark, addEventListener(_name, handler) { listeners.system = handler; } };
  const environment = {
    matchMedia: () => media,
    document: { documentElement: { dataset: {}, style: {} } },
    localStorage: { getItem() { if (blocked) throw Error('blocked'); return saved; }, setItem(_key, value) { if (blocked) throw Error('blocked'); saved = value; } },
    addEventListener(name, handler) { listeners[name] = handler; }
  };
  return { environment, media, listeners, saved: () => saved };
}

test('theme defaults to system and tracks device changes only in system mode', () => {
  const b = browser(null, true), theme = createTheme(b.environment);
  assert.equal(theme.preference, 'system');
  assert.equal(b.environment.document.documentElement.dataset.theme, 'night');
  b.media.matches = false; b.listeners.system();
  assert.equal(b.environment.document.documentElement.dataset.theme, 'light');
  theme.set('night'); b.listeners.system();
  assert.equal(b.environment.document.documentElement.dataset.theme, 'night');
  assert.equal(b.saved(), 'night');
  theme.set('invalid'); assert.equal(theme.preference, 'night');
});

test('explicit preferences survive reload and blocked storage remains usable', () => {
  const b = browser('light', true); createTheme(b.environment);
  assert.equal(b.environment.document.documentElement.dataset.theme, 'light');
  const blocked = browser(null, false, true), theme = createTheme(blocked.environment);
  theme.set('night'); assert.equal(blocked.environment.document.documentElement.dataset.theme, 'night');
});
