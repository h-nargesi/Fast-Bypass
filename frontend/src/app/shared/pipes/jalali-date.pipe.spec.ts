import { describe, expect, it } from 'vitest';
import { JalaliDatePipe } from './jalali-date.pipe';

describe('JalaliDatePipe', () => {
  const pipe = new JalaliDatePipe();

  it('formats datetime by default', () => {
    expect(pipe.transform('2026-05-21T06:30:00Z')).toMatch(/1405\/02\/31/);
  });

  it('formats date only in date mode', () => {
    expect(pipe.transform('2026-05-21T06:30:00Z', 'date')).toBe('1405/02/31');
  });
});
