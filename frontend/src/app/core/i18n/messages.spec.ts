import { describe, expect, it } from 'vitest';
import { API_ERROR_MESSAGES, UI_MESSAGES, apiErrorMessage } from './messages';

describe('UI messages (Persian)', () => {
  it('app title is Persian', () => {
    expect(UI_MESSAGES.appTitle).toMatch(/[\u0600-\u06FF]/);
    expect(UI_MESSAGES.appTitle).toContain('پنل');
  });

  it('confirm delete strings are Persian', () => {
    expect(UI_MESSAGES.confirmDeleteTitle).toBe('تأیید حذف');
    expect(UI_MESSAGES.confirmDeleteBody).toMatch(/حذف/);
  });

  it('empty state messages are Persian', () => {
    expect(UI_MESSAGES.emptyUsers).toMatch(/کاربر/);
    expect(UI_MESSAGES.emptyRenewals).toMatch(/تمدید/);
  });

  it('currency suffix is ریال', () => {
    expect(UI_MESSAGES.currencySuffix).toBe('ریال');
  });
});

describe('apiErrorMessage', () => {
  it('maps known API codes to Persian', () => {
    expect(apiErrorMessage('QUOTA_EXCEEDED')).toBe(API_ERROR_MESSAGES['QUOTA_EXCEEDED']);
    expect(apiErrorMessage('MANAGER_DISABLED')).toMatch(/غیرفعال/);
  });

  it('uses fallback for unknown codes', () => {
    expect(apiErrorMessage('UNKNOWN_CODE', 'پیام سفارشی')).toBe('پیام سفارشی');
  });
});
