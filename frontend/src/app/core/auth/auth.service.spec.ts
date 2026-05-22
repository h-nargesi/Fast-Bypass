import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AuthService } from './auth.service';
import * as storage from './token-storage';

describe('AuthService', () => {
  let service: AuthService;
  let http: HttpTestingController;

  beforeEach(() => {
    sessionStorage.clear();
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    });
    service = TestBed.inject(AuthService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('login stores tokens and role', () => {
    service.login('admin', 'secret').subscribe((res) => {
      expect(res.role).toBe('admin');
    });
    const req = http.expectOne('/api/v1/auth/login');
    expect(req.request.body).toEqual({ username: 'admin', password: 'secret' });
    req.flush({
      access_token: 'a1',
      refresh_token: 'r1',
      role: 'admin',
    });
    expect(storage.getRole()).toBe('admin');
    expect(service.loggedIn()).toBe(true);
  });

  it('homeRoute returns /admin for admin and / for manager', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    service.role.set('admin');
    expect(service.homeRoute()).toBe('/admin');

    storage.saveSession({
      access_token: 'a',
      refresh_token: 'r',
      role: 'manager',
    });
    service.role.set('manager');
    expect(service.homeRoute()).toBe('/');
  });
});
