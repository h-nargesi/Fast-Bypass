import { VpnProfile } from '../../core/models';

export function isProfileActive(p: VpnProfile): boolean {
  const s = (p.state || '').toLowerCase();
  return s.includes('active');
}

export function primaryProfileState(profiles: VpnProfile[]): string {
  const active = profiles.find((p) => isProfileActive(p));
  return active?.state ?? profiles[0]?.state ?? '';
}
