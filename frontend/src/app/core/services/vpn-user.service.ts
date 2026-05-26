import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../api/api-client.service';
import { AdminStatsResponse, AdminVpnListResponse, ManagerRow, RenewalsResponse, VpnListResponse, VpnUserDetail } from '../models';

export interface CreateVpnBody {
  local_name: string;
  password?: string;
  shared_users: number;
  disabled?: boolean;
  contact_info?: string;
  notes?: string;
  assign_profile?: boolean;
  profile_name?: string;
  amount_paid?: number;
  currency?: string;
  /** Admin only — triggers MikroTik certificate + config-{username}.ovpn */
  cert_title?: string;
}

export interface PatchVpnBody {
  password?: string;
  shared_users?: number;
  disabled?: boolean;
  contact_info?: string;
  notes?: string;
}

/** Admin PATCH /admin/vpn-users — includes certificate fields */
export interface AdminPatchVpnBody extends PatchVpnBody {
  manager_id?: number;
  cert_title?: string | null;
}

function encName(name: string): string {
  return encodeURIComponent(name);
}

@Injectable({ providedIn: 'root' })
export class VpnUserService {
  private readonly api = inject(ApiClient);

  list(opts: { refresh?: boolean; q?: string; page?: number; page_size?: number } = {}): Observable<VpnListResponse> {
    return this.api.get('/vpn-users', {
      refresh: opts.refresh ? 'true' : undefined,
      q: opts.q || undefined,
      page: opts.page != null ? String(opts.page) : undefined,
      page_size: opts.page_size != null ? String(opts.page_size) : undefined,
    });
  }

  get(id: number): Observable<VpnUserDetail> {
    return this.api.get(`/vpn-users/${id}`);
  }

  getByName(name: string): Observable<VpnUserDetail> {
    return this.api.get(`/vpn-users/by-name/${encName(name)}`);
  }

  create(body: CreateVpnBody): Observable<VpnUserDetail> {
    return this.api.post('/vpn-users', body);
  }

  patch(id: number, body: PatchVpnBody): Observable<VpnUserDetail> {
    return this.api.patch(`/vpn-users/${id}`, body);
  }

  patchByName(name: string, body: PatchVpnBody): Observable<VpnUserDetail> {
    return this.api.patch(`/vpn-users/by-name/${encName(name)}`, body);
  }

  delete(id: number): Observable<void> {
    return this.api.delete(`/vpn-users/${id}`);
  }

  deleteByName(name: string): Observable<void> {
    return this.api.delete(`/vpn-users/by-name/${encName(name)}`);
  }

  assignProfile(
    id: number,
    body: { profile_name: string; amount_paid?: number; currency?: string; note?: string },
  ): Observable<VpnUserDetail> {
    return this.api.post(`/vpn-users/${id}/assign-profile`, body);
  }

  assignProfileByName(
    name: string,
    body: { profile_name: string; amount_paid?: number; currency?: string; note?: string },
  ): Observable<VpnUserDetail> {
    return this.api.post(`/vpn-users/by-name/${encName(name)}/assign-profile`, body);
  }

  removeProfile(id: number, profileRowId: string): Observable<void> {
    return this.api.delete(`/vpn-users/${id}/profiles/${profileRowId}`);
  }

  removeProfileByName(name: string, profileRowId: string): Observable<void> {
    return this.api.delete(`/vpn-users/by-name/${encName(name)}/profiles/${profileRowId}`);
  }

  downloadOvpn(id: number): Observable<Blob> {
    return this.api.download(`/vpn-users/${id}/ovpn`);
  }

  downloadOvpnByName(name: string): Observable<Blob> {
    return this.api.download(`/vpn-users/by-name/${encName(name)}/ovpn`);
  }
}

@Injectable({ providedIn: 'root' })
export class AdminVpnService {
  private readonly api = inject(ApiClient);

  list(opts: {
    refresh?: boolean;
    manager_id?: number;
    orphan?: boolean;
    q?: string;
    page?: number;
    page_size?: number;
  } = {}): Observable<AdminVpnListResponse> {
    return this.api.get('/admin/vpn-users', {
      refresh: opts.refresh ? 'true' : undefined,
      manager_id: opts.manager_id,
      orphan: opts.orphan ? 'true' : undefined,
      q: opts.q || undefined,
      page: opts.page != null ? String(opts.page) : undefined,
      page_size: opts.page_size != null ? String(opts.page_size) : undefined,
    });
  }

  get(id: number): Observable<VpnUserDetail> {
    return this.api.get(`/admin/vpn-users/${id}`);
  }

  getByName(name: string): Observable<VpnUserDetail> {
    return this.api.get(`/admin/vpn-users/by-name/${encName(name)}`);
  }

  create(body: CreateVpnBody & { manager_id?: number }): Observable<VpnUserDetail> {
    return this.api.post('/admin/vpn-users', body);
  }

  patch(id: number, body: AdminPatchVpnBody): Observable<VpnUserDetail> {
    return this.api.patch(`/admin/vpn-users/${id}`, body);
  }

  patchByName(name: string, body: AdminPatchVpnBody): Observable<VpnUserDetail> {
    return this.api.patch(`/admin/vpn-users/by-name/${encName(name)}`, body);
  }

  delete(id: number): Observable<void> {
    return this.api.delete(`/admin/vpn-users/${id}`);
  }

  deleteByName(name: string): Observable<void> {
    return this.api.delete(`/admin/vpn-users/by-name/${encName(name)}`);
  }

  assignProfile(
    id: number,
    body: { profile_name: string; amount_paid?: number; currency?: string; note?: string },
  ): Observable<VpnUserDetail> {
    return this.api.post(`/admin/vpn-users/${id}/assign-profile`, body);
  }

  assignProfileByName(
    name: string,
    body: { profile_name: string; amount_paid?: number; currency?: string; note?: string },
  ): Observable<VpnUserDetail> {
    return this.api.post(`/admin/vpn-users/by-name/${encName(name)}/assign-profile`, body);
  }

  removeProfile(id: number, profileRowId: string): Observable<void> {
    return this.api.delete(`/admin/vpn-users/${id}/profiles/${profileRowId}`);
  }

  removeProfileByName(name: string, profileRowId: string): Observable<void> {
    return this.api.delete(`/admin/vpn-users/by-name/${encName(name)}/profiles/${profileRowId}`);
  }

  downloadOvpn(id: number): Observable<Blob> {
    return this.api.download(`/admin/vpn-users/${id}/ovpn`);
  }

  downloadOvpnByName(name: string): Observable<Blob> {
    return this.api.download(`/admin/vpn-users/by-name/${encName(name)}/ovpn`);
  }
}

@Injectable({ providedIn: 'root' })
export class RenewalsService {
  private readonly api = inject(ApiClient);

  managerList(opts: { settled?: string; page?: number; page_size?: number; q?: string } = {}): Observable<RenewalsResponse> {
    return this.api.get('/renewals', {
      settled: opts.settled ?? '',
      page: opts.page != null ? String(opts.page) : undefined,
      page_size: opts.page_size != null ? String(opts.page_size) : undefined,
      q: opts.q || undefined,
    });
  }

  adminList(opts: {
    manager_id?: number;
    settled?: string;
    page?: number;
    page_size?: number;
    q?: string;
  } = {}): Observable<RenewalsResponse> {
    return this.api.get('/admin/renewals', {
      manager_id: opts.manager_id,
      settled: opts.settled,
      page: opts.page != null ? String(opts.page) : undefined,
      page_size: opts.page_size != null ? String(opts.page_size) : undefined,
      q: opts.q || undefined,
    });
  }

  settleThrough(through_activation_id: number, manager_id?: number): Observable<{ updated_count: number }> {
    return this.api.post('/admin/renewals/settle-through', {
      through_activation_id,
      manager_id,
    });
  }
}

@Injectable({ providedIn: 'root' })
export class AdminService {
  private readonly api = inject(ApiClient);

  listManagers(): Observable<{ items: ManagerRow[] }> {
    return this.api.get('/admin/managers');
  }

  getStats(refresh = false): Observable<AdminStatsResponse> {
    return this.api.get('/admin/stats', { refresh: refresh ? 'true' : undefined });
  }

  createManager(body: {
    username: string;
    password: string;
    display_name: string;
    slug: string;
    quota: number;
    cert_title?: string;
  }): Observable<{ id: number }> {
    return this.api.post('/admin/managers', body);
  }

  patchManager(
    id: number,
    body: Partial<{
      username: string;
      display_name: string;
      quota: number;
      is_active: boolean;
      password: string;
      cert_title: string | null;
    }>,
  ): Observable<ManagerRow> {
    return this.api.patch(`/admin/managers/${id}`, body);
  }
}
