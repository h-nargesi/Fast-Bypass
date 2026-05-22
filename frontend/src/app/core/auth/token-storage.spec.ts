import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import * as storage from './token-storage';

describe('token-storage', () => {
  beforeEach(() => sessionStorage.clear());
  afterEach(() => sessionStorage.clear());

  it('saves and reads session from login response', () => {
    storage.saveSession({
      access_token: 'at',
      refresh_token: 'rt',
      role: 'manager',
      name_prefix: 'ali-',
    });
    expect(storage.getAccessToken()).toBe('at');
    expect(storage.getRefreshToken()).toBe('rt');
    expect(storage.getRole()).toBe('manager');
    expect(storage.getNamePrefix()).toBe('ali-');
    expect(storage.isLoggedIn()).toBe(true);
  });

  it('clears session on logout', () => {
    storage.saveSession({ access_token: 'a', refresh_token: 'r', role: 'admin' });
    storage.clearSession();
    expect(storage.isLoggedIn()).toBe(false);
    expect(storage.getRole()).toBeNull();
  });
});
