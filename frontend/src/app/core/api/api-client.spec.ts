import { HttpErrorResponse } from '@angular/common/http';
import { describe, expect, it } from 'vitest';
import { ApiClient } from './api-client.service';

describe('ApiClient.mapError', () => {
  it('maps known API error code to Persian', () => {
    const err = new HttpErrorResponse({
      status: 409,
      error: { error: { code: 'QUOTA_EXCEEDED', message: 'x' } },
    });
    expect(ApiClient.mapError(err)).toMatch(/سقف/);
  });

  it('uses server message as fallback for unknown code', () => {
    const err = new HttpErrorResponse({
      status: 400,
      error: { error: { code: 'CUSTOM', message: 'پیام سرور' } },
    });
    expect(ApiClient.mapError(err)).toBe('پیام سرور');
  });

  it('reports connection failure when status is 0', () => {
    expect(ApiClient.mapError(new HttpErrorResponse({ status: 0 }))).toMatch(/ارتباط/);
  });

  it('maps INVALID_CURRENT_PASSWORD to Persian', () => {
    const err = new HttpErrorResponse({
      status: 401,
      error: { error: { code: 'INVALID_CURRENT_PASSWORD', message: 'x' } },
    });
    expect(ApiClient.mapError(err)).toMatch(/رمز فعلی/);
  });
});
