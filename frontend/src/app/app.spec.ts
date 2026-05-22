import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter, Router } from '@angular/router';
import { beforeEach, describe, expect, it } from 'vitest';
import { App } from './app';
import { routes } from './app.routes';
import { UI_MESSAGES } from './core/i18n/messages';
import * as storage from './core/auth/token-storage';

describe('App (Persian shell)', () => {
  let fixture: ComponentFixture<App>;
  let router: Router;
  let http: HttpTestingController;

  beforeEach(async () => {
    sessionStorage.clear();
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideRouter(routes),
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    router = TestBed.inject(Router);
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(App);
  });

  function flushManagerDashboard(): void {
    http.expectOne('/api/v1/me').flush({
      username: 'ali',
      display_name: 'علی',
      slug: 'ali',
      name_prefix: 'ali-',
      quota: 10,
      used_quota: 0,
    });
    http.expectOne('/api/v1/me/quota').flush({ quota: 10, used: 0, available: 10 });
    http.expectOne((req) => req.url === '/api/v1/vpn-users').flush({ items: [] });
  }

  it('renders Persian app title in header', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();
    const h1 = fixture.nativeElement.querySelector('h1');
    expect(h1?.textContent).toBe(UI_MESSAGES.appTitle);
    expect(h1?.textContent).toMatch(/[\u0600-\u06FF]/);
  });

  it('hides nav on login page', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();
  });

  it('shows manager nav when logged in as manager', async () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    fixture.componentInstance.auth.loggedIn.set(true);
    fixture.componentInstance.auth.role.set('manager');
    await router.navigateByUrl('/');
    fixture.detectChanges();
    flushManagerDashboard();
    fixture.detectChanges();
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/کاربران/);
    expect(nav?.textContent).toContain(UI_MESSAGES.logout);
  });

  it('shows admin nav when logged in as admin', async () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    fixture.componentInstance.auth.loggedIn.set(true);
    fixture.componentInstance.auth.role.set('admin');
    await router.navigateByUrl('/admin');
    fixture.detectChanges();
    http.expectOne('/api/v1/admin/managers').flush({ items: [] });
    http.expectOne((req) => req.url === '/api/v1/admin/vpn-users').flush({ items: [] });
    fixture.detectChanges();
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/مدیران/);
    expect(nav?.textContent).toMatch(/کاربران VPN/);
  });
});
