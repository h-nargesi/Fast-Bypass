import { describe, expect, it, beforeEach, afterEach } from 'vitest';
import { readDocumentLocale } from './document-locale';

describe('Document locale (RTL + Persian)', () => {
  const originalLang = document.documentElement.lang;
  const originalDir = document.documentElement.getAttribute('dir');

  afterEach(() => {
    document.documentElement.lang = originalLang;
    if (originalDir) {
      document.documentElement.setAttribute('dir', originalDir);
    } else {
      document.documentElement.removeAttribute('dir');
    }
  });

  it('reads lang=fa and dir=rtl from index.html contract', () => {
    document.documentElement.lang = 'fa';
    document.documentElement.setAttribute('dir', 'rtl');
    const { lang, dir } = readDocumentLocale();
    expect(lang).toBe('fa');
    expect(dir).toBe('rtl');
  });
});
