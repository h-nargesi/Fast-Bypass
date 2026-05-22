import { HttpClient, HttpErrorResponse, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, throwError } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiErrorBody } from '../models';
import { apiErrorMessage } from '../i18n/messages';

@Injectable({ providedIn: 'root' })
export class ApiClient {
  private readonly http = inject(HttpClient);
  private readonly base = environment.apiBaseUrl;

  get<T>(path: string, params?: Record<string, string | number | boolean | undefined>): Observable<T> {
    return this.http.get<T>(`${this.base}${path}`, { params: this.toParams(params) });
  }

  post<T>(path: string, body?: unknown): Observable<T> {
    return this.http.post<T>(`${this.base}${path}`, body ?? {});
  }

  patch<T>(path: string, body: unknown): Observable<T> {
    return this.http.patch<T>(`${this.base}${path}`, body);
  }

  delete(path: string): Observable<void> {
    return this.http.delete<void>(`${this.base}${path}`);
  }

  download(path: string): Observable<Blob> {
    return this.http.get(`${this.base}${path}`, { responseType: 'blob' });
  }

  static mapError(err: unknown): string {
    if (err instanceof HttpErrorResponse) {
      const body = err.error as ApiErrorBody | undefined;
      if (body?.error?.code) {
        return apiErrorMessage(body.error.code, body.error.message);
      }
      if (err.status === 0) {
        return 'ارتباط با سرور برقرار نشد';
      }
    }
    return 'خطای ناشناخته';
  }

  static throwMapped(err: unknown) {
    return throwError(() => ApiClient.mapError(err));
  }

  private toParams(
    params?: Record<string, string | number | boolean | undefined>,
  ): HttpParams | undefined {
    if (!params) return undefined;
    let hp = new HttpParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== '') {
        hp = hp.set(k, String(v));
      }
    }
    return hp;
  }
}
