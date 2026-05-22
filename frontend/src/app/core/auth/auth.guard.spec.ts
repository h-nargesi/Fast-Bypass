import { TestBed } from '@angular/core/testing';
import { provideRouter, Router, UrlTree } from '@angular/router';
import { describe, expect, it, beforeEach } from 'vitest';
import { adminGuard, authGuard, guestGuard, managerGuard } from './auth.guard';
import { AuthService } from './auth.service';
import * as storage from './token-storage';

function runGuard(fn: typeof authGuard): boolean | UrlTree {
  return TestBed.runInInjectionContext(() => fn({} as never, {} as never)) as boolean | UrlTree;
}

describe('auth guards', () => {
  beforeEach(() => {
    sessionStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideRouter([]), AuthService],
    });
  });

  it('authGuard redirects guests to /login', () => {
    const result = runGuard(authGuard);
    expect(result instanceof UrlTree).toBe(true);
    expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toBe('/login');
  });

  it('authGuard allows logged-in users', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    TestBed.inject(AuthService).loggedIn.set(true);
    expect(runGuard(authGuard)).toBe(true);
  });

  it('guestGuard redirects logged-in users to home', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    const auth = TestBed.inject(AuthService);
    auth.loggedIn.set(true);
    auth.role.set('admin');
    const result = runGuard(guestGuard);
    expect(TestBed.inject(Router).serializeUrl(result as UrlTree)).toBe('/admin');
  });

  it('adminGuard blocks managers', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    const auth = TestBed.inject(AuthService);
    auth.loggedIn.set(true);
    auth.role.set('manager');
    const result = runGuard(adminGuard);
    expect(result instanceof UrlTree).toBe(true);
  });

  it('managerGuard blocks admins', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    const auth = TestBed.inject(AuthService);
    auth.loggedIn.set(true);
    auth.role.set('admin');
    const result = runGuard(managerGuard);
    expect(result instanceof UrlTree).toBe(true);
  });
});
