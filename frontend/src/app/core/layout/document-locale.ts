/** بررسی تنظیمات RTL و زبان فارسی در سند */
export function readDocumentLocale(): { lang: string; dir: string } {
  const el = document.documentElement;
  return { lang: el.lang, dir: el.getAttribute('dir') ?? '' };
}
