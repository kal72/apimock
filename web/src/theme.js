const preferences = ['light', 'night', 'system'];

export function createTheme(environment = window) {
  const media = environment.matchMedia('(prefers-color-scheme: dark)');
  let saved;
  try { saved = environment.localStorage.getItem('mocklab-theme'); } catch {}
  const theme = {
    preference: preferences.includes(saved) ? saved : 'system',
    apply() {
      const night = this.preference === 'night' || this.preference === 'system' && media.matches;
      environment.document.documentElement.dataset.theme = night ? 'night' : 'light';
      environment.document.documentElement.style.colorScheme = night ? 'dark' : 'light';
    },
    set(value) {
      if (!preferences.includes(value)) return;
      this.preference = value;
      try { environment.localStorage.setItem('mocklab-theme', value); } catch {}
      this.apply();
    }
  };
  media.addEventListener('change', () => theme.apply());
  environment.addEventListener('storage', event => {
    if (event.key !== 'mocklab-theme' && event.key !== null) return;
    theme.preference = preferences.includes(event.newValue) ? event.newValue : 'system';
    theme.apply();
  });
  theme.apply();
  return theme;
}
