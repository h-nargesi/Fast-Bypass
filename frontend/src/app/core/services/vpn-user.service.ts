import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiClient } from '../api/api-client.service';
import { AdminVpnListItem, VpnListItem, VpnUserDetail } from '../models';

export interface CreateVpnBody {
  local_name: string;
  password: string;
  shared_users: number;
  contact_phone?: string;
  contact_note?: string;
  notes?: string;
  assign_profile?: boolean;
  profile_name?: string;
  amount_paid?: number;
  currency?: string;
}

export interface PatchVpnBody {
  password?: string;
  shared_users?: number;
  contact_phone?: string;
  contact_note?: string;
  notes?: string;
}

export interface AssignProfileBody {
  profile_name: string;
  amount_paid?: number;
  currency?: string;
  note?: string;
}

@Injectable({ providedIn: 'root' })
export class VpnUserService {
  private readonly api = inject(ApiClient);

  list(refresh = false, activeOnly = false): Observable<{ items: VpnListItem[] }> {
    return this.api.get('/vpn-users', { refresh, active_only: activeOnly });
  }

  get(id: number): Observable<VpnUserDetail> {
    return this.api.get(`/vpn-users/${id}`);
  }

  create(body: CreateVpnBody): Observable<VpnUserDetail> {
    return this.api.post('/vpn-users', body);
  }

  patch(id: number, body: PatchVpnBody): Observable<VpnUserDetail> {
    return this.api.patch(`/vpn-users/${id}`, body);
  }

  remove(id: number): Observable<void> {
    return this.api.delete(`/vpn-users/${id}`);
  }

  assignProfile(id: number, body: AssignProfileBody): Observable<VpnUserDetail> {
    return this.api.post(`/vpn-users/${id}/assign-profile`, body);
  }

  removeProfile(id: number, profileRowId: string): Observable<void> {
    return this.api.delete(`/vpn-users/${id}/profiles/${profileRowId}`);
  }

  downloadOvpn(id: number): Observable<Blob> {
    return this.api.download(`/vpn-users/${id}/ovpn`);
  }
}

@Injectable({ providedIn: 'root' })
export class AdminVpnService {
  private readonly api = inject(ApiClient);

  list(opts: {
    refresh?: boolean;
    manager_id?: number;
    orphan?: boolean;
  }): Observable<{ items: AdminVpnListItem[] }> {
    return this.api.get('/admin/vpn-users', {
      refresh: opts.refresh,
      manager_id: opts.manager_id,
      orphan: opts.orphan,
    });
  }

  get(id: number): Observable<VpnUserDetail> {
    return this.api.get(`/admin/vpn-users/${id}`);
  }
}

@Injectable({ providedIn: 'root' })
export class RenewalsService {
  private readonly api = inject(ApiClient);

  managerList(settled = '', page = 1): Observable<import('../models').RenewalsResponse> {
    return this.api.get('/renewals', { settled, page, page_size: 50 });
  }

  adminList(opts: {
    manager_id?: number;
    settled?: string;
    q?: string;
    page?: number;
  }): Observable<import('../models').RenewalsResponse> {
    return this.api.get('/admin/renewals', {
      manager_id: opts.manager_id,
      settled: opts.settled ?? '',
      q: opts.q,
      page: opts.page ?? 1,
      page_size: 50,
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

  listManagers(): Observable<{ items: import('../models').ManagerRow[] }> {
    return this.api.get('/admin/managers');
  }

  createManager(body: {
    username: string;
    password: string;
    display_name: string;
    slug: string;
    quota: number;
  }): Observable<{ id: number }> {
    return this.api.post('/admin/managers', body);
  }

  patchManager(
    id: number,
    body: {
      display_name?: string;
      quota?: number;
      is_active?: boolean;
      password?: string;
    },
  ): Observable<import('../models').ManagerRow> {
    return this.api.patch(`/admin/managers/${id}`, body);
  }
}
