import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { authInterceptor } from './auth.interceptor';
import * as storage from './token-storage';

describe('authInterceptor', () => {
  let http: HttpTestingController;
  let client: HttpClient;

  beforeEach(() => {
    sessionStorage.clear();
    storage.saveSession({
      access_token: 'old-access',
      refresh_token: 'refresh-token',
      role: 'admin',
    });
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpTestingController);
    client = TestBed.inject(HttpClient);
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
    vi.unstubAllGlobals();
  });

  it('attaches bearer token to protected requests', () => {
    client.get('/api/v1/me').subscribe();
    const req = http.expectOne('/api/v1/me');
    expect(req.request.headers.get('Authorization')).toBe('Bearer old-access');
    req.flush({ username: 'admin', role: 'admin' });
  });

  it('does not refresh or retry when current password is wrong', () => {
    let status = 0;
    let code = '';

    client.post('/api/v1/me/password', {}).subscribe({
      error: (err) => {
        status = err.status;
        code = err.error?.error?.code;
      },
    });

    http
      .expectOne('/api/v1/me/password')
      .flush(
        { error: { code: 'INVALID_CURRENT_PASSWORD', message: 'رمز فعلی نادرست است' } },
        { status: 401, statusText: 'Unauthorized' },
      );

    http.expectNone('/api/v1/auth/refresh');
    expect(status).toBe(401);
    expect(code).toBe('INVALID_CURRENT_PASSWORD');
    expect(storage.getRefreshToken()).toBe('refresh-token');
  });

  it('refreshes token and retries request after unauthorized response', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({ access_token: 'new-access', refresh_token: 'new-refresh' }),
    });
    vi.stubGlobal('fetch', fetchMock);

    let completed = false;
    client.post('/api/v1/me/password', {}).subscribe({
      next: () => {
        completed = true;
      },
    });

    http
      .expectOne('/api/v1/me/password')
      .flush(
        { error: { code: 'UNAUTHORIZED', message: 'توکن نامعتبر است' } },
        { status: 401, statusText: 'Unauthorized' },
      );

    await vi.waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/auth/refresh');
    await vi.waitFor(() => expect(storage.getAccessToken()).toBe('new-access'));

    const retry = http.expectOne('/api/v1/me/password');
    expect(retry.request.headers.get('Authorization')).toBe('Bearer new-access');
    retry.flush(null, { status: 204, statusText: 'No Content' });

    await vi.waitFor(() => expect(completed).toBe(true));
  });

  it('clears session when refresh fails after unauthorized response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));

    let failed = false;
    client.get('/api/v1/me').subscribe({
      error: () => {
        failed = true;
      },
    });

    http
      .expectOne('/api/v1/me')
      .flush(
        { error: { code: 'UNAUTHORIZED', message: 'توکن نامعتبر است' } },
        { status: 401, statusText: 'Unauthorized' },
      );

    await vi.waitFor(() => expect(failed).toBe(true));
    await vi.waitFor(() => expect(storage.getAccessToken()).toBeNull());
    expect(storage.getRefreshToken()).toBeNull();
  });
});
