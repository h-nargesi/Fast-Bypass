import { toJalaali } from 'jalaali-js';

function tehranParts(d: Date): { gy: number; gm: number; gd: number; hh: string; mm: string } {
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat('en-US', {
      timeZone: 'Asia/Tehran',
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
      .formatToParts(d)
      .filter((p) => p.type !== 'literal')
      .map((p) => [p.type, p.value]),
  );
  return {
    gy: Number(parts['year']),
    gm: Number(parts['month']),
    gd: Number(parts['day']),
    hh: parts['hour']!.padStart(2, '0'),
    mm: parts['minute']!.padStart(2, '0'),
  };
}

/** تبدیل ISO8601 به تاریخ/ساعت شمسی (منطقه تهران) */
export function formatJalaliDateTime(iso: string | null | undefined): string {
  if (!iso) {
    return '—';
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return '—';
  }
  const { gy, gm, gd, hh, mm } = tehranParts(d);
  const { jy, jm, jd } = toJalaali(gy, gm, gd);
  return `${jy}/${String(jm).padStart(2, '0')}/${String(jd).padStart(2, '0')} ${hh}:${mm}`;
}

export function formatJalaliDate(iso: string | null | undefined): string {
  const full = formatJalaliDateTime(iso);
  return full === '—' ? full : full.split(' ')[0]!;
}
