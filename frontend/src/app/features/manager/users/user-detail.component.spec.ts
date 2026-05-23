import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { UserDetailComponent } from './user-detail.component';

const bundle = {
  username: 'ali-u1',
  password: 'Secret123',
  openvpn_key_password: 'key',
  l2tp_ipsec_secret: 'sec',
  l2tp_server: 'vpn.example.com',
  openvpn_download_url: 'http://example.com/ovpn',
};

describe('UserDetailComponent', () => {
  let fixture: ComponentFixture<UserDetailComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UserDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([]),
        {
          provide: ActivatedRoute,
          useValue: {
            paramMap: {
              subscribe: (fn: (m: ReturnType<typeof convertToParamMap>) => void) => {
                fn(convertToParamMap({ id: '1' }));
              },
            },
          },
        },
      ],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(UserDetailComponent);
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  it('shows shared_users in renewal history table', () => {
    const req = http.expectOne('/api/v1/vpn-users/1');
    req.flush({
      id: 1,
      mikrotik_name: 'ali-u1',
      shared_users: 2,
      disabled: false,
      contact_info: null,
      notes: null,
      profiles: [],
      activations: [
        {
          id: 10,
          profile_name: 'profile-open-2M-30d',
          shared_users: 3,
          currency: 'IRR',
          is_settled: false,
          created_at: '2026-05-01T10:00:00Z',
        },
      ],
      connection_bundle: bundle,
      manager_id: 1,
      manager_display_name: null,
      manager_username: null,
      manager_slug: null,
      owner_mismatch: false,
    });
    fixture.detectChanges();

    const el: HTMLElement = fixture.nativeElement;
    expect(el.textContent).toContain('تاریخچه تمدید');
    const headers = Array.from(el.querySelectorAll('h3 + table thead th, .card table thead th'))
      .map((h) => h.textContent?.trim())
      .filter((t) => t === 'اتصال' || t === 'پروفایل');
    expect(headers).toContain('اتصال');

    const historySection = Array.from(el.querySelectorAll('.card')).find((c) =>
      c.textContent?.includes('تاریخچه تمدید'),
    );
    expect(historySection?.textContent).toContain('3');
  });
});
