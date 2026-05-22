import { readFileSync } from 'fs';
import { join } from 'path';
import { describe, expect, it } from 'vitest';

describe('index.html locale (RTL + Persian)', () => {
  const html = readFileSync(join(process.cwd(), 'src', 'index.html'), 'utf-8');

  it('sets lang=fa on html element', () => {
    expect(html).toMatch(/<html[^>]*\blang=["']fa["']/i);
  });

  it('sets dir=rtl on html element', () => {
    expect(html).toMatch(/<html[^>]*\bdir=["']rtl["']/i);
  });
});
