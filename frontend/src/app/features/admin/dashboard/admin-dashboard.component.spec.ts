import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AdminDashboardComponent } from './admin-dashboard.component';
import * as storage from '../../../core/auth/token-storage';

describe('AdminDashboardComponent', () => {
  let fixture: ComponentFixture<AdminDashboardComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    sessionStorage.clear();
    storage.saveSession({ access_token: 't', refresh_token: 'r', role: 'admin' });
    await TestBed.configureTestingModule({
      imports: [AdminDashboardComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AdminDashboardComponent);
    fixture.detectChanges();
  });

  afterEach(() => {
    http.verify();
    sessionStorage.clear();
  });

  it('loads admin stats on init', () => {
    const req = http.expectOne('/api/v1/admin/stats');
    req.flush({
      manager_count: 2,
      totals: { vpn_users: 4, connections: 9 },
      orphan: { vpn_users: 1, connections: 2 },
      by_manager: [
        {
          manager_id: 1,
          display_name: 'علی',
          username: 'ali',
          quota: 10,
          vpn_users: 2,
          connections: 5,
        },
        {
          manager_id: 2,
          display_name: 'باب',
          username: 'bob',
          quota: 8,
          vpn_users: 1,
          connections: 2,
        },
      ],
    });
    fixture.detectChanges();
    const text = fixture.nativeElement.textContent;
    expect(text).toMatch(/داشبورد ادمین/);
    expect(text).toContain('کاربران فعال');
    expect(text).toContain('9');
    expect(text).toContain('به تفکیک مدیر');
    expect(text).toContain('علی');
    expect(text).toContain('5 / 10');
  });
});
