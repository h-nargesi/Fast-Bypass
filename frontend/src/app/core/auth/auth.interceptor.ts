import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { catchError, from, switchMap, throwError } from 'rxjs';
import { ApiErrorBody } from '../models';
import * as storage from './token-storage';

function apiErrorCode(err: HttpErrorResponse): string | undefined {
  const body = err.error as ApiErrorBody | undefined;
  return body?.error?.code;
}

let refreshInFlight: Promise<boolean> | null = null;

export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const token = storage.getAccessToken();
  const authReq =
    token && !req.url.includes('/auth/login') && !req.url.includes('/auth/refresh')
      ? req.clone({ setHeaders: { Authorization: `Bearer ${token}` } })
      : req;

  return next(authReq).pipe(
    catchError((err: HttpErrorResponse) => {
      if (err.status !== 401 || req.url.includes('/auth/')) {
        return throwError(() => err);
      }
      if (apiErrorCode(err) === 'INVALID_CURRENT_PASSWORD') {
        return throwError(() => err);
      }
      return from(refreshTokens()).pipe(
        switchMap((ok) => {
          if (!ok) {
            storage.clearSession();
            return throwError(() => err);
          }
          const retry = req.clone({
            setHeaders: { Authorization: `Bearer ${storage.getAccessToken()}` },
          });
          return next(retry);
        }),
      );
    }),
  );
};

function refreshTokens(): Promise<boolean> {
  if (refreshInFlight) {
    return refreshInFlight;
  }
  const refresh = storage.getRefreshToken();
  if (!refresh) {
    return Promise.resolve(false);
  }
  refreshInFlight = fetch('/api/v1/auth/refresh', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refresh }),
  })
    .then(async (r) => {
      if (!r.ok) return false;
      const data = (await r.json()) as { access_token: string; refresh_token: string };
      storage.saveTokens(data.access_token, data.refresh_token);
      return true;
    })
    .catch(() => false)
    .finally(() => {
      refreshInFlight = null;
    });
  return refreshInFlight;
}
