import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AdminService, AdminVpnService, RenewalsService, VpnUserService } from './vpn-user.service';

describe('VpnUserService', () => {
  let http: HttpTestingController;
  let vpn: VpnUserService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting()],
    });
    http = TestBed.inject(HttpTestingController);
    vpn = TestBed.inject(VpnUserService);
  });

  afterEach(() => http.verify());

  it('lists vpn users with refresh query', () => {
    vpn.list({ refresh: true }).subscribe((res) => expect(res.items.length).toBe(1));
    const req = http.expectOne(
      (req) => req.url === '/api/v1/vpn-users' && req.params.get('refresh') === 'true',
    );
    req.flush({ items: [{ mikrotik_name: 'a', shared_users: 1, profiles: [] }], page: 1, page_size: 20, total: 1 });
  });

  it('list response maps disabled flag', () => {
    vpn.list().subscribe((res) => {
      expect(res.items[0].disabled).toBe(true);
      expect(res.items[1].disabled).toBe(false);
    });
    const req = http.expectOne('/api/v1/vpn-users');
    req.flush({
      items: [
        { mikrotik_name: 'x-off', shared_users: 1, disabled: true, profiles: [] },
        { mikrotik_name: 'x-on', shared_users: 1, disabled: false, profiles: [] },
      ],
      page: 1, page_size: 20, total: 2,
    });
  });

  it('patches vpn user with contact_info and notes', () => {
    vpn
      .patch(3, { contact_info: 'tg @u', notes: 'n' })
      .subscribe((u) => expect(u.contact_info).toBe('tg @u'));
    const req = http.expectOne('/api/v1/vpn-users/3');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({ contact_info: 'tg @u', notes: 'n' });
    req.flush({
      id: 3,
      mikrotik_name: 'ali-u',
      shared_users: 1,
      disabled: false,
      contact_info: 'tg @u',
      notes: 'n',
      profiles: [],
      activations: [],
      connection_bundle: {
        username: 'ali-u',
        password: 'x',
        openvpn_key_password: '',
        l2tp_ipsec_secret: '',
        l2tp_server: '',
        openvpn_download_url: '',
      },
      manager_id: 1,
      manager_display_name: null,
      manager_username: null,
      manager_slug: null,
      owner_mismatch: false,
    });
  });
});

describe('AdminVpnService', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), AdminVpnService],
    });
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('creates orphan user without manager_id', () => {
    const svc = TestBed.inject(AdminVpnService);
    svc
      .create({
        local_name: 'orphan1',
        password: 'Secret123',
        shared_users: 1,
        contact_info: 'info',
      })
      .subscribe((u) => expect(u.mikrotik_name).toBe('orphan1'));
    const req = http.expectOne('/api/v1/admin/vpn-users');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual({
      local_name: 'orphan1',
      password: 'Secret123',
      shared_users: 1,
      contact_info: 'info',
    });
    expect(req.request.body).not.toHaveProperty('manager_id');
    req.flush({
      id: 9,
      mikrotik_name: 'orphan1',
      shared_users: 1,
      disabled: false,
      contact_info: 'info',
      notes: null,
      profiles: [],
      activations: [],
      connection_bundle: {
        username: 'orphan1',
        password: 'Secret123',
        openvpn_key_password: '',
        l2tp_ipsec_secret: '',
        l2tp_server: '',
        openvpn_download_url: '',
      },
      manager_id: null,
      manager_display_name: null,
      manager_username: null,
      manager_slug: null,
      owner_mismatch: false,
    });
  });

  it('creates user for manager with manager_id', () => {
    const svc = TestBed.inject(AdminVpnService);
    svc
      .create({
        manager_id: 2,
        local_name: 'u1',
        password: 'Secret123',
        shared_users: 1,
      })
      .subscribe();
    const req = http.expectOne('/api/v1/admin/vpn-users');
    expect(req.request.body).toMatchObject({ manager_id: 2, local_name: 'u1' });
    req.flush({
      id: 10,
      mikrotik_name: 'bob-u1',
      shared_users: 1,
      disabled: false,
      contact_info: null,
      notes: null,
      profiles: [],
      activations: [],
      connection_bundle: {
        username: 'bob-u1',
        password: 'Secret123',
        openvpn_key_password: '',
        l2tp_ipsec_secret: '',
        l2tp_server: '',
        openvpn_download_url: '',
      },
      manager_id: 2,
      manager_display_name: null,
      manager_username: null,
      manager_slug: null,
      owner_mismatch: false,
    });
  });
});

describe('RenewalsService', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), RenewalsService],
    });
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('admin settle-through posts activation id', () => {
    const svc = TestBed.inject(RenewalsService);
    svc.settleThrough(42, 3).subscribe((r) => expect(r.updated_count).toBe(2));
    const req = http.expectOne('/api/v1/admin/renewals/settle-through');
    expect(req.request.body).toEqual({ through_activation_id: 42, manager_id: 3 });
    req.flush({ updated_count: 2 });
  });
});

describe('AdminService', () => {
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [provideHttpClient(), provideHttpClientTesting(), AdminService],
    });
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('creates manager via POST /admin/managers', () => {
    const svc = TestBed.inject(AdminService);
    svc
      .createManager({
        username: 'bob',
        password: 'Pass1234',
        display_name: 'باب',
        slug: 'bob',
        quota: 5,
      })
      .subscribe((r) => expect(r.id).toBe(9));
    const req = http.expectOne('/api/v1/admin/managers');
    expect(req.request.method).toBe('POST');
    req.flush({ id: 9 });
  });

  it('patches manager username and password via PATCH /admin/managers/:id', () => {
    const svc = TestBed.inject(AdminService);
    svc
      .patchManager(4, { username: 'ali2', password: 'ResetPass1', quota: 12, is_active: true })
      .subscribe((m) => expect(m.username).toBe('ali2'));
    const req = http.expectOne('/api/v1/admin/managers/4');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({
      username: 'ali2',
      password: 'ResetPass1',
      quota: 12,
      is_active: true,
    });
    req.flush({
      id: 4,
      username: 'ali2',
      display_name: 'علی',
      slug: 'ali',
      quota: 12,
      used_quota: 0,
      is_active: true,
    });
  });

  it('loads admin stats via GET /admin/stats', () => {
    const svc = TestBed.inject(AdminService);
    svc.getStats(true).subscribe((s) => {
      expect(s.manager_count).toBe(1);
      expect(s.totals.connections).toBe(5);
      expect(s.by_manager[0].connections).toBe(5);
    });
    const req = http.expectOne(
      (r) => r.url === '/api/v1/admin/stats' && r.params.get('refresh') === 'true',
    );
    req.flush({
      manager_count: 1,
      totals: { vpn_users: 2, connections: 5 },
      orphan: { vpn_users: 0, connections: 0 },
      by_manager: [
        {
          manager_id: 1,
          display_name: 'علی',
          username: 'ali',
          quota: 10,
          vpn_users: 2,
          connections: 5,
        },
      ],
    });
  });
});
