import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { UserListComponent } from './user-list.component';

describe('UserListComponent', () => {
  let fixture: ComponentFixture<UserListComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UserListComponent],
      providers: [provideHttpClient(), provideHttpClientTesting(), provideRouter([])],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(UserListComponent);
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  it('shows active column and distinguishes disabled rows', () => {
    const req = http.expectOne((r) => r.url === '/api/v1/vpn-users');
    req.flush({
      items: [
        {
          mikrotik_name: 'ali-active',
          shared_users: 2,
          disabled: false,
          profiles: [{ id: '1', profile: 'p', state: 'active', end_time: '' }],
        },
        {
          mikrotik_name: 'ali-off',
          shared_users: 1,
          disabled: true,
          profiles: [],
        },
      ],
    });
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('فعال');
    expect(el.textContent).toContain('غیرفعال');

    const headers = Array.from(el.querySelectorAll('th')).map((h) => h.textContent?.trim());
    expect(headers).toContain('فعال');

    const rows = el.querySelectorAll('tbody tr');
    expect(rows.length).toBe(2);
    expect(rows[0].classList.contains('row-disabled')).toBe(false);
    expect(rows[1].classList.contains('row-disabled')).toBe(true);

    const badges = el.querySelectorAll('.router-status');
    expect(badges.length).toBe(2);
    expect(badges[0].classList.contains('off')).toBe(false);
    expect(badges[1].classList.contains('off')).toBe(true);
  });
});
