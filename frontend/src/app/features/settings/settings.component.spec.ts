import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { authInterceptor } from '../../core/auth/auth.interceptor';
import * as storage from '../../core/auth/token-storage';
import { ToastService } from '../../shared/services/toast.service';
import { SettingsComponent } from './settings.component';

describe('SettingsComponent', () => {
  let fixture: ComponentFixture<SettingsComponent>;
  let http: HttpTestingController;
  let toast: ToastService;

  beforeEach(async () => {
    sessionStorage.clear();
    storage.saveSession({ access_token: 'access', refresh_token: 'refresh', role: 'admin' });
    await TestBed.configureTestingModule({
      imports: [SettingsComponent],
      providers: [
        provideHttpClient(withInterceptors([authInterceptor])),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    toast = TestBed.inject(ToastService);
    fixture = TestBed.createComponent(SettingsComponent);
    fixture.detectChanges();
    http.expectOne('/api/v1/me').flush({ username: 'admin', role: 'admin' });
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('shows success toast and clears fields when password change succeeds', () => {
    const show = vi.spyOn(toast, 'show');
    fixture.componentInstance.currentPw = 'AdminPass1';
    fixture.componentInstance.newPw = 'NewAdmin123';
    fixture.componentInstance.changePassword();

    const req = http.expectOne('/api/v1/me/password');
    expect(req.request.body).toEqual({
      current_password: 'AdminPass1',
      new_password: 'NewAdmin123',
    });
    req.flush(null, { status: 204, statusText: 'No Content' });

    expect(show).toHaveBeenCalledOnce();
    expect(show).toHaveBeenCalledWith('رمز تغییر کرد');
    expect(fixture.componentInstance.error()).toBe('');
    expect(fixture.componentInstance.currentPw).toBe('');
    expect(fixture.componentInstance.newPw).toBe('');
  });

  it('shows error and no success toast when current password is wrong', () => {
    const show = vi.spyOn(toast, 'show');
    fixture.componentInstance.currentPw = 'AdminPass1';
    fixture.componentInstance.newPw = 'NewAdmin123';
    fixture.componentInstance.changePassword();

    http
      .expectOne('/api/v1/me/password')
      .flush(
        { error: { code: 'INVALID_CURRENT_PASSWORD', message: 'رمز فعلی نادرست است' } },
        { status: 401, statusText: 'Unauthorized' },
      );

    expect(show).not.toHaveBeenCalled();
    expect(fixture.componentInstance.error()).toMatch(/رمز فعلی/);
    expect(fixture.componentInstance.currentPw).toBe('AdminPass1');
    expect(fixture.componentInstance.newPw).toBe('NewAdmin123');
  });
});
