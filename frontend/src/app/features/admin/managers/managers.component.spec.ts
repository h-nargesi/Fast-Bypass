import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import {
  HttpTestingController,
  provideHttpClientTesting,
} from '@angular/common/http/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { ManagersComponent } from './managers.component';

const managerRow = {
  id: 2,
  username: 'ali',
  display_name: 'علی',
  slug: 'ali',
  quota: 10,
  used_quota: 3,
  is_active: true,
};

describe('ManagersComponent', () => {
  let fixture: ComponentFixture<ManagersComponent>;
  let http: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ManagersComponent],
      providers: [provideHttpClient(), provideHttpClientTesting()],
    }).compileComponents();
    http = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(ManagersComponent);
    fixture.detectChanges();
    http.expectOne('/api/v1/admin/managers').flush({ items: [managerRow] });
  });

  afterEach(() => http.verify());

  it('patches username and password when saving edit', () => {
    fixture.componentInstance.startEdit(managerRow);
    fixture.componentInstance.editUsername = 'ali2';
    fixture.componentInstance.editPassword = 'ResetPass1';
    fixture.componentInstance.editQuota = 15;
    fixture.componentInstance.editActive = false;
    fixture.componentInstance.saveEdit(managerRow);

    const req = http.expectOne('/api/v1/admin/managers/2');
    expect(req.request.method).toBe('PATCH');
    expect(req.request.body).toEqual({
      username: 'ali2',
      password: 'ResetPass1',
      quota: 15,
      is_active: false,
    });
    req.flush({ ...managerRow, username: 'ali2', quota: 15, is_active: false });
    http.expectOne('/api/v1/admin/managers').flush({ items: [{ ...managerRow, username: 'ali2', quota: 15, is_active: false }] });
    expect(fixture.componentInstance.editId()).toBeNull();
  });

  it('shows error when patch rejects invalid password', () => {
    fixture.componentInstance.startEdit(managerRow);
    fixture.componentInstance.editPassword = 'short';
    fixture.componentInstance.saveEdit(managerRow);

    const req = http.expectOne('/api/v1/admin/managers/2');
    req.flush(
      { error: { code: 'VALIDATION', message: 'رمز نامعتبر است (حداقل ۸ کاراکتر، حروف و عدد)' } },
      { status: 400, statusText: 'Bad Request' },
    );
    expect(fixture.componentInstance.error()).toContain('رمز نامعتبر');
    expect(fixture.componentInstance.editId()).toBe(2);
  });

  it('creates manager with cert_title when provided', () => {
    fixture.componentInstance.newUser = {
      username: 'm2',
      password: 'ManagerPass1',
      display_name: 'M2',
      slug: 'm2',
      quota: 5,
      cert_title: 'mgr-cert',
    };
    fixture.componentInstance.create();

    const req = http.expectOne('/api/v1/admin/managers');
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toMatchObject({
      username: 'm2',
      cert_title: 'mgr-cert',
    });
    req.flush({ ...managerRow, id: 3, username: 'm2', cert_title: 'mgr-cert' });
    http.expectOne('/api/v1/admin/managers').flush({ items: [] });
  });

  it('shows confirm before patching changed cert_title', () => {
    fixture.componentInstance.startEdit({ ...managerRow, cert_title: 'old-cert' });
    fixture.componentInstance.editCertTitle = 'new-cert';
    fixture.componentInstance.saveEdit({ ...managerRow, cert_title: 'old-cert' });

    expect(fixture.componentInstance.confirmCertRegenerate()).toBe(true);
    http.expectNone('/api/v1/admin/managers/2');
  });

  it('patches cert_title after confirm', () => {
    fixture.componentInstance.startEdit({ ...managerRow, cert_title: 'old-cert' });
    fixture.componentInstance.editCertTitle = 'new-cert';
    fixture.componentInstance.saveEdit({ ...managerRow, cert_title: 'old-cert' });
    fixture.componentInstance.onConfirmCertRegenerate();

    const req = http.expectOne('/api/v1/admin/managers/2');
    expect(req.request.body).toMatchObject({ cert_title: 'new-cert' });
    req.flush({ ...managerRow, cert_title: 'new-cert' });
    http.expectOne('/api/v1/admin/managers').flush({ items: [{ ...managerRow, cert_title: 'new-cert' }] });
    expect(fixture.componentInstance.editId()).toBeNull();
  });

  it('omits password from patch body when edit password is empty', () => {
    fixture.componentInstance.startEdit(managerRow);
    fixture.componentInstance.editUsername = 'ali';
    fixture.componentInstance.editPassword = '';
    fixture.componentInstance.saveEdit(managerRow);

    const req = http.expectOne('/api/v1/admin/managers/2');
    expect(req.request.body).toEqual({
      username: 'ali',
      quota: 10,
      is_active: true,
    });
    req.flush(managerRow);
    http.expectOne('/api/v1/admin/managers').flush({ items: [managerRow] });
  });
});
