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

  it('submits credentials to /auth/login', async () => {
    const navigate = vi.spyOn(router, 'navigate').mockResolvedValue(true);
    fixture.componentInstance.username = 'admin';
    fixture.componentInstance.password = 'admin';
    fixture.componentInstance.submit();

    const req = http.expectOne('/api/v1/auth/login');
    req.flush({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    await Promise.resolve();

    expect(storage.getRole()).toBe('admin');
    expect(navigate).toHaveBeenCalledWith(['/admin']);
  });
});
