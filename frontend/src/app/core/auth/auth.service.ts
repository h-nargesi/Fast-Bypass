import { Injectable, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { Observable, tap } from 'rxjs';
import { ApiClient } from '../api/api-client.service';
import { LoginResponse, MeProfile, UserRole } from '../models';
import * as storage from './token-storage';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly api = inject(ApiClient);
  private readonly router = inject(Router);

  readonly role = signal<UserRole | null>(storage.getRole());
  readonly loggedIn = signal(storage.isLoggedIn());

  login(username: string, password: string): Observable<LoginResponse> {
    return this.api.post<LoginResponse>('/auth/login', { username, password }).pipe(
      tap((res) => {
        storage.saveSession(res);
        this.role.set(res.role);
        this.loggedIn.set(true);
      }),
    );
  }

  logout(): void {
    if (storage.isLoggedIn()) {
      this.api.post('/auth/logout').subscribe({ error: () => undefined });
    }
    storage.clearSession();
    this.role.set(null);
    this.loggedIn.set(false);
    void this.router.navigate(['/login']);
  }

  refresh(): Observable<{ access_token: string; refresh_token: string }> {
    const refresh_token = storage.getRefreshToken();
    return this.api.post<{ access_token: string; refresh_token: string }>('/auth/refresh', {
      refresh_token,
    });
  }

  homeRoute(): string {
    return this.role() === 'admin' ? '/admin' : '/';
  }

  isAdmin(): boolean {
    return this.role() === 'admin';
  }

  isManager(): boolean {
    return this.role() === 'manager';
  }
}

@Injectable({ providedIn: 'root' })
export class ProfileService {
  private readonly api = inject(ApiClient);

  getMe(): Observable<MeProfile> {
    return this.api.get<MeProfile>('/me');
  }

  patchDisplayName(display_name: string): Observable<MeProfile> {
    return this.api.patch<MeProfile>('/me', { display_name });
  }

  changePassword(current_password: string, new_password: string): Observable<void> {
    return this.api.post<void>('/me/password', { current_password, new_password });
  }

  getQuota(): Observable<{ quota: number; used: number; available: number }> {
    return this.api.get('/me/quota');
  }
}
