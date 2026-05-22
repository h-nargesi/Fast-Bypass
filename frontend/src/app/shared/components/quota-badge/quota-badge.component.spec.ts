import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it } from 'vitest';
import { QuotaBadgeComponent } from './quota-badge.component';

describe('QuotaBadgeComponent', () => {
  let fixture: ComponentFixture<QuotaBadgeComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [QuotaBadgeComponent] }).compileComponents();
    fixture = TestBed.createComponent(QuotaBadgeComponent);
    fixture.componentRef.setInput('quota', 10);
    fixture.componentRef.setInput('used', 8);
    fixture.componentRef.setInput('available', 2);
    fixture.detectChanges();
  });

  it('shows quota numbers in Persian UI', () => {
    const text = fixture.nativeElement.textContent as string;
    expect(text).toMatch(/سقف کاربر/);
    expect(text).toContain('8');
    expect(text).toContain('10');
    expect(text).toMatch(/باقی‌مانده/);
  });

  it('adds warn class when available is zero', () => {
    fixture.componentRef.setInput('available', 0);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.quota')?.classList.contains('warn')).toBe(true);
  });
});
