import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { AdminService, RenewalsService, VpnUserService } from './vpn-user.service';

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
    vpn.list(true).subscribe((res) => expect(res.items.length).toBe(1));
    const req = http.expectOne(
      (req) => req.url === '/api/v1/vpn-users' && req.params.get('refresh') === 'true',
    );
    req.flush({ items: [{ mikrotik_name: 'a', shared_users: 1, profiles: [] }] });
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
});
