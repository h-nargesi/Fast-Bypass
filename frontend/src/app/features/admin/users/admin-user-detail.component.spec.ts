import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { ActivatedRoute, convertToParamMap, provideRouter } from '@angular/router';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AdminUserDetailComponent } from './admin-user-detail.component';

const bundle = {
  username: 'certu',
  password: 'Secret123',
  openvpn_key_password: 'key',
  l2tp_ipsec_secret: 'sec',
  l2tp_server: 'vpn.example.com',
  openvpn_download_url: 'http://example.com/ovpn',
};

const userDetail = {
  id: 1,
  mikrotik_name: 'certu',
  shared_users: 1,
  disabled: false,
  contact_info: null,
  notes: null,
  cert_title: 'user-cert-1',
  profiles: [],
  activations: [],
  connection_bundle: bundle,
  manager_id: null,
  manager_display_name: null,
  manager_username: null,
  manager_slug: null,
  owner_mismatch: false,
};

describe('AdminUserDetailComponent', () => {
  let fixture: ComponentFixture<AdminUserDetailComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AdminUserDetailComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideRouter([{ path: 'admin/users/:id', component: AdminUserDetailComponent }]),
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
    fixture = TestBed.createComponent(AdminUserDetailComponent);
    fixture.detectChanges();
    http.expectOne('/api/v1/admin/vpn-users/1').flush(userDetail);
    fixture.detectChanges();
  });

  afterEach(() => http.verify());

  it('opens confirm dialog when cert_title changes from a non-empty value', () => {
    fixture.componentInstance.editCertTitle = 'user-cert-2';
    fixture.componentInstance.saveEdit();
    expect(fixture.componentInstance.confirmCertRegenerate()).toBe(true);
    http.expectNone('/api/v1/admin/vpn-users/1');
  });

  it('patches new cert_title after dialog confirm (no extra API flag)', () => {
    fixture.componentInstance.editCertTitle = 'user-cert-2';
    fixture.componentInstance.onConfirmCertRegenerate();

    const req = http.expectOne('/api/v1/admin/vpn-users/1');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({
      shared_users: 1,
      contact_info: '',
      notes: '',
      disabled: false,
      cert_title: 'user-cert-2',
    });
    req.flush({ ...userDetail, cert_title: 'user-cert-2' });
  });

  it('patches cert_title without dialog when initially empty', () => {
    fixture.componentInstance.editCertTitleInitial = '';
    fixture.componentInstance.editCertTitle = 'fresh-cert';
    fixture.componentInstance.saveEdit();

    const req = http.expectOne('/api/v1/admin/vpn-users/1');
    expect(req.request.body).toMatchObject({ cert_title: 'fresh-cert' });
    req.flush({ ...userDetail, cert_title: 'fresh-cert' });
  });
});
