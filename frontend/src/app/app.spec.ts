import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { By } from '@angular/platform-browser';
import { provideRouter, Router } from '@angular/router';
import { beforeEach, describe, expect, it } from 'vitest';
import { App } from './app';
import { routes } from './app.routes';
import { UI_MESSAGES } from './core/i18n/messages';
import * as storage from './core/auth/token-storage';
import { LoginComponent } from './features/auth/login/login.component';

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

  function flushAdminDashboard(): void {
    http.expectOne('/api/v1/admin/stats').flush({
      manager_count: 0,
      totals: { vpn_users: 0, connections: 0 },
      orphan: { vpn_users: 0, connections: 0 },
      by_manager: [],
    });
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

  it('redirects logged-in admin away from /login and shows admin nav', async () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    fixture.componentInstance.auth.loggedIn.set(true);
    fixture.componentInstance.auth.role.set('admin');
    await router.navigateByUrl('/login');
    fixture.detectChanges();
    flushAdminDashboard();
    fixture.detectChanges();

    expect(router.url).toBe('/admin');
    expect(fixture.componentInstance.showNav()).toBe(true);
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/مدیران/);
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
    flushAdminDashboard();
    fixture.detectChanges();
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/مدیران/);
    expect(nav?.textContent).toMatch(/کاربران VPN/);
  });

  it('shows admin nav after login redirect from /login to /admin', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();

    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    fixture.componentInstance.auth.loggedIn.set(true);
    fixture.componentInstance.auth.role.set('admin');
    fixture.detectChanges();
    expect(fixture.componentInstance.showNav()).toBe(false);

    await router.navigateByUrl('/admin');
    fixture.detectChanges();
    flushAdminDashboard();
    fixture.detectChanges();

    expect(fixture.componentInstance.showNav()).toBe(true);
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/مدیران/);
    expect(nav?.textContent).toMatch(/کاربران VPN/);
    expect(nav?.textContent).not.toMatch(/داشبورد/);
  });

  it('shows manager nav after login redirect from /login to /', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('nav')).toBeNull();

    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    fixture.componentInstance.auth.loggedIn.set(true);
    fixture.componentInstance.auth.role.set('manager');
    fixture.detectChanges();
    expect(fixture.componentInstance.showNav()).toBe(false);

    await router.navigateByUrl('/');
    fixture.detectChanges();
    flushManagerDashboard();
    fixture.detectChanges();

    expect(fixture.componentInstance.showNav()).toBe(true);
    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/داشبورد/);
    expect(nav?.textContent).toMatch(/کاربران/);
    expect(nav?.textContent).not.toMatch(/کاربران VPN/);
    expect(nav?.textContent).not.toMatch(/مدیران/);
  });

  it('shows admin nav after login form submit redirects to /admin', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();

    const loginDe = fixture.debugElement.query(By.directive(LoginComponent));
    expect(loginDe).toBeTruthy();
    const login = loginDe.componentInstance as LoginComponent;
    login.username = 'admin';
    login.password = 'secret';
    login.submit();
    fixture.detectChanges();

    const req = http.expectOne('/api/v1/auth/login');
    expect(req.request.body).toEqual({ username: 'admin', password: 'secret' });
    req.flush({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    await fixture.whenStable();
    fixture.detectChanges();

    expect(router.url).toBe('/admin');
    http.expectOne('/api/v1/admin/stats').flush({
      manager_count: 0,
      totals: { vpn_users: 0, connections: 0 },
      orphan: { vpn_users: 0, connections: 0 },
      by_manager: [],
    });
    fixture.detectChanges();

    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/مدیران/);
    expect(nav?.textContent).toMatch(/کاربران VPN/);
  });

  it('shows manager nav after login form submit redirects to /', async () => {
    await router.navigateByUrl('/login');
    fixture.detectChanges();

    const login = fixture.debugElement.query(By.directive(LoginComponent))
      .componentInstance as LoginComponent;
    login.username = 'ali';
    login.password = 'pass';
    login.submit();
    fixture.detectChanges();

    http
      .expectOne('/api/v1/auth/login')
      .flush({ access_token: 'a', refresh_token: 'r', role: 'manager' });
    await fixture.whenStable();
    fixture.detectChanges();

    expect(router.url).toBe('/');
    flushManagerDashboard();
    fixture.detectChanges();

    const nav = fixture.nativeElement.querySelector('nav');
    expect(nav?.textContent).toMatch(/داشبورد/);
    expect(nav?.textContent).toMatch(/کاربران/);
  });
});
