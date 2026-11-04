const preferences = ['light', 'night'];

export function createTheme(environment = window) {
  const media = environment.matchMedia('(prefers-color-scheme: dark)');
  let saved;
  try { saved = environment.localStorage.getItem('mocklab-theme'); } catch {}
  const theme = {
    preference: preferences.includes(saved) ? saved : media.matches ? 'night' : 'light',
    get isNight(){return this.preference === 'night';},
    apply() {
      const night = this.isNight;
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
  environment.addEventListener('storage', event => {
    if (event.key !== 'mocklab-theme' && event.key !== null) return;
    theme.preference = preferences.includes(event.newValue) ? event.newValue : media.matches ? 'night' : 'light';
    theme.apply();
  });
  theme.apply();
  return theme;
}
