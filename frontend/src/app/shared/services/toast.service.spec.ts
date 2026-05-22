import { TestBed } from '@angular/core/testing';
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { ToastService } from './toast.service';

describe('ToastService', () => {
  let service: ToastService;

  beforeEach(() => {
    vi.useFakeTimers();
    TestBed.configureTestingModule({});
    service = TestBed.inject(ToastService);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('sets and clears message', () => {
    service.show('کپی شد');
    expect(service.message()).toBe('کپی شد');
    vi.advanceTimersByTime(2600);
    expect(service.message()).toBeNull();
  });
});
