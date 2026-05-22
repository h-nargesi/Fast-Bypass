import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it } from 'vitest';
import { ProfileStateChipComponent } from './profile-state-chip.component';

describe('ProfileStateChipComponent', () => {
  let fixture: ComponentFixture<ProfileStateChipComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ProfileStateChipComponent] }).compileComponents();
    fixture = TestBed.createComponent(ProfileStateChipComponent);
  });

  it('shows Persian label for active state', () => {
    fixture.componentRef.setInput('state', 'active');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('فعال');
  });

  it('shows dash when state is empty', () => {
    fixture.componentRef.setInput('state', '');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent?.trim()).toBe('—');
  });
});
