import { LoginResponse, UserRole } from '../models';

const ACCESS = 'panel_access';
const REFRESH = 'panel_refresh';
const ROLE = 'panel_role';
const NAME_PREFIX = 'panel_name_prefix';

export function saveSession(res: LoginResponse): void {
  sessionStorage.setItem(ACCESS, res.access_token);
  sessionStorage.setItem(REFRESH, res.refresh_token);
  sessionStorage.setItem(ROLE, res.role);
  if (res.name_prefix) {
    sessionStorage.setItem(NAME_PREFIX, res.name_prefix);
  } else {
    sessionStorage.removeItem(NAME_PREFIX);
  }
}

export function saveTokens(access: string, refresh: string): void {
  sessionStorage.setItem(ACCESS, access);
  sessionStorage.setItem(REFRESH, refresh);
}

export function getAccessToken(): string | null {
  return sessionStorage.getItem(ACCESS);
}

export function getRefreshToken(): string | null {
  return sessionStorage.getItem(REFRESH);
}

export function getRole(): UserRole | null {
  const r = sessionStorage.getItem(ROLE);
  return r === 'admin' || r === 'manager' ? r : null;
}

export function getNamePrefix(): string | null {
  return sessionStorage.getItem(NAME_PREFIX);
}

export function clearSession(): void {
  sessionStorage.removeItem(ACCESS);
  sessionStorage.removeItem(REFRESH);
  sessionStorage.removeItem(ROLE);
  sessionStorage.removeItem(NAME_PREFIX);
}

export function isLoggedIn(): boolean {
  return !!getAccessToken();
}
