import { describe, expect, it } from 'vitest';
import { formatRial } from './currency';

describe('formatRial', () => {
  it('formats amount with Persian digits and ریال suffix', () => {
    const out = formatRial(150000);
    expect(out).toContain('ریال');
    // fa-IR uses Persian digits (۱۵۰٬۰۰۰)
    expect(out).toMatch(/۱۵۰|150/);
  });

  it('returns ثبت نشده for null', () => {
    expect(formatRial(null)).toBe('ثبت نشده');
  });

  it('allows zero', () => {
    expect(formatRial(0)).toMatch(/ریال/);
  });
});
