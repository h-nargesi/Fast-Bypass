import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter, Router } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { LoginComponent } from './login.component';
import { UI_MESSAGES } from '../../../core/i18n/messages';
import * as storage from '../../../core/auth/token-storage';

describe('LoginComponent', () => {
  let fixture: ComponentFixture<LoginComponent>;
  let http: HttpTestingController;
  let router: Router;

  beforeEach(async () => {
    sessionStorage.clear();
    await TestBed.configureTestingModule({
      imports: [LoginComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(LoginComponent);
    fixture.detectChanges();
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('renders Persian login form', () => {
    expect(fixture.nativeElement.textContent).toContain(UI_MESSAGES.login);
    expect(fixture.nativeElement.querySelector('input[name="username"]')).toBeTruthy();
    expect(fixture.nativeElement.querySelector('input[name="password"]')).toBeTruthy();
  });

  it('submits credentials to /auth/login and redirects admin to /admin', async () => {
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.componentInstance.username = 'admin';
    fixture.componentInstance.password = 'admin';
    fixture.componentInstance.submit();

    const req = http.expectOne('/api/v1/auth/login');
    expect(req.request.body).toEqual({ username: 'admin', password: 'admin' });
    req.flush({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    await Promise.resolve();

    expect(storage.getRole()).toBe('admin');
    expect(storage.isLoggedIn()).toBe(true);
    expect(navigate).toHaveBeenCalledWith(['/admin']);
  });

  it('redirects manager to home after successful login', async () => {
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.componentInstance.username = 'ali';
    fixture.componentInstance.password = 'pass';
    fixture.componentInstance.submit();

    http
      .expectOne('/api/v1/auth/login')
      .flush({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    await Promise.resolve();

    expect(storage.getRole()).toBe('manager');
    expect(navigate).toHaveBeenCalledWith(['/']);
  });

  it('does not navigate when login fails', async () => {
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.componentInstance.username = 'bad';
    fixture.componentInstance.password = 'wrong';
    fixture.componentInstance.submit();

    http.expectOne('/api/v1/auth/login').flush(
      { code: 'UNAUTHORIZED', message: 'نام کاربری یا رمز اشتباه است' },
      { status: 401, statusText: 'Unauthorized' },
    );
    await Promise.resolve();

    expect(navigate).not.toHaveBeenCalled();
    expect(storage.isLoggedIn()).toBe(false);
  });
});
