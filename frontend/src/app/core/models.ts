export type UserRole = 'admin' | 'manager';

export interface ApiErrorBody {
  error: { code: string; message: string };
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  role: UserRole;
  manager_id?: number;
  slug?: string;
  name_prefix?: string;
}

export interface AdminProfile {
  username: string;
  role: 'admin';
}

export interface ManagerProfile {
  username: string;
  display_name: string;
  slug: string;
  name_prefix: string;
  quota: number;
  used_quota: number;
}

export type MeProfile = AdminProfile | ManagerProfile;

export interface QuotaInfo {
  quota: number;
  used: number;
  available: number;
}

export interface VpnProfile {
  id: string;
  profile: string;
  state: string;
  end_time: string;
}

export interface VpnListItem {
  id?: number;
  mikrotik_name: string;
  shared_users: number;
  disabled?: boolean;
  profiles: VpnProfile[];
}

export interface Activation {
  id: number;
  profile_name: string;
  shared_users: number;
  currency: string;
  is_settled: boolean;
  created_at: string;
  amount_paid?: number | null;
  mikrotik_end_time?: string | null;
  note?: string | null;
}

export interface ConnectionBundle {
  username: string;
  password: string | null;
  openvpn_key_password: string;
  l2tp_ipsec_secret: string;
  l2tp_server: string;
  openvpn_download_url: string;
}

export interface VpnUserDetail {
  id?: number | null;
  mikrotik_name: string;
  shared_users: number;
  disabled: boolean;
  contact_info: string | null;
  notes: string | null;
  profiles: VpnProfile[];
  activations: Activation[];
  connection_bundle: ConnectionBundle;
  manager_id: number | null;
  manager_display_name: string | null;
  manager_username: string | null;
  manager_slug: string | null;
  owner_mismatch: boolean;
  mikrotik_comment?: string;
}

export interface AdminVpnListItem extends VpnListItem {
  mikrotik_comment: string;
  manager_id: number | null;
  manager_display_name: string | null;
  manager_username: string | null;
  manager_slug: string | null;
  owner_mismatch: boolean;
}

export interface RenewalItem {
  id: number;
  renewed_at: string;
  mikrotik_name: string;
  shared_users: number;
  profile_name: string;
  profile_state: string;
  mikrotik_end_time: string | null;
  is_settled: boolean;
  amount_paid: number | null;
  currency: string;
}

export interface RenewalsResponse {
  scope: {
    orphan?: boolean;
    manager_id?: number | null;
    manager_display_name?: string;
  };
  can_settle: boolean;
  summary: {
    unsettled_shared_users_sum: number;
    all_shared_users_sum: number;
  };
  items: RenewalItem[];
  page: number;
  page_size: number;
  total: number;
}

export interface ManagerRow {
  id: number;
  username: string;
  display_name: string;
  slug: string;
  quota: number;
  used_quota: number;
  is_active: boolean;
}
