import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { provideRouter, ActivatedRoute } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AdminUserListComponent } from './admin-user-list.component';

describe('AdminUserListComponent', () => {
  let fixture: ComponentFixture<AdminUserListComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminUserListComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: { queryParams: { subscribe: (fn: (q: object) => void) => fn({}) } },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(AdminUserListComponent);
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  it('loads managers and shows disabled user styling', () => {
    http.expectOne('/api/v1/admin/managers').flush({ items: [] });
    const list = http.expectOne((r) => r.url === '/api/v1/admin/vpn-users');
    list.flush({
      items: [
        {
          mikrotik_name: 'ali-ok',
          shared_users: 1,
          disabled: false,
          mikrotik_comment: 'panel:ali',
          manager_id: 1,
          manager_display_name: 'علی',
          manager_username: 'ali',
          manager_slug: 'ali',
          owner_mismatch: false,
          profiles: [],
        },
        {
          mikrotik_name: 'ali-off',
          shared_users: 1,
          disabled: true,
          mikrotik_comment: 'panel:ali',
          manager_id: 1,
          manager_display_name: 'علی',
          manager_username: 'ali',
          manager_slug: 'ali',
          owner_mismatch: true,
          profiles: [],
        },
      ],
    });
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(Array.from(el.querySelectorAll('th')).map((h) => h.textContent?.trim())).toContain('فعال');

    const rows = el.querySelectorAll('tbody tr');
    expect(rows[0].classList.contains('row-disabled')).toBe(false);
    expect(rows[0].classList.contains('warn-row')).toBe(false);

    expect(rows[1].classList.contains('row-disabled')).toBe(true);
    expect(rows[1].classList.contains('warn-row')).toBe(false);

    const offBadge = rows[1].querySelector('.router-status.off');
    expect(offBadge?.textContent?.trim()).toBe('غیرفعال');
  });
});
