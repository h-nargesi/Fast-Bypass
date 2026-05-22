import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { DashboardComponent } from './dashboard.component';
import * as storage from '../../../core/auth/token-storage';

describe('DashboardComponent (مدیر)', () => {
  let fixture: ComponentFixture<DashboardComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    sessionStorage.clear();
    storage.saveSession({ access_token: 't', refresh_token: 'r', role: 'manager' });
    await TestBed.configureTestingModule({
      imports: [DashboardComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(DashboardComponent);
    fixture.detectChanges();
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('loads profile, quota, and users on init', () => {
    const me = http.expectOne('/api/v1/me');
    const quota = http.expectOne('/api/v1/me/quota');
    const users = http.expectOne((req) => req.url === '/api/v1/vpn-users');
    me.flush({
      username: 'ali',
      display_name: 'علی',
      slug: 'ali',
      name_prefix: 'ali-',
      quota: 10,
      used_quota: 2,
    });
    quota.flush({ quota: 10, used: 2, available: 8 });
    users.flush({
      items: [
        {
          mikrotik_name: 'ali-a',
          shared_users: 1,
          profiles: [{ id: '1', profile: 'p', state: 'active', end_time: '' }],
        },
      ],
    });
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toMatch(/داشبورد/);
    expect(fixture.nativeElement.textContent).toContain('علی');
    expect(fixture.nativeElement.textContent).toMatch(/کاربران با پروفایل فعال/);
    expect(fixture.nativeElement.textContent).toContain('1');
  });
});
