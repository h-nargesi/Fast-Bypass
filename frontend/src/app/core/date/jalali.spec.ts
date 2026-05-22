import { describe, expect, it } from 'vitest';
import { formatJalaliDate, formatJalaliDateTime } from './jalali';

describe('Jalali date formatting (Asia/Tehran)', () => {
  it('formats known UTC instant to Jalali date-time', () => {
    // 2026-05-21 06:30 UTC = 10:00 Asia/Tehran → 1405/02/31 شمسی
    const out = formatJalaliDateTime('2026-05-21T06:30:00Z');
    expect(out).toMatch(/^1405\/02\/31 10:00$/);
  });

  it('formatJalaliDate returns date part only', () => {
    expect(formatJalaliDate('2026-05-21T06:30:00Z')).toBe('1405/02/31');
  });

  it('returns em dash for null/invalid', () => {
    expect(formatJalaliDate(null)).toBe('—');
    expect(formatJalaliDateTime('not-a-date')).toBe('—');
  });
});
