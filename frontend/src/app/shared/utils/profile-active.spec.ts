import { describe, expect, it } from 'vitest';
import { isProfileActive, primaryProfileState } from './profile-active';

describe('profile-active helpers', () => {
  it('detects active profile state', () => {
    expect(isProfileActive({ id: '1', profile: 'p', state: 'active', end_time: '' })).toBe(true);
    expect(isProfileActive({ id: '2', profile: 'p', state: 'waiting', end_time: '' })).toBe(false);
  });

  it('prefers active profile state in list', () => {
    expect(
      primaryProfileState([
        { id: '1', profile: 'a', state: 'used', end_time: '' },
        { id: '2', profile: 'b', state: 'active', end_time: '' },
      ]),
    ).toBe('active');
  });
});
