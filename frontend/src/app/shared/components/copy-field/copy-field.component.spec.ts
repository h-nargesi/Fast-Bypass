import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { CopyFieldComponent } from './copy-field.component';
import { ToastService } from '../../services/toast.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';

describe('CopyFieldComponent', () => {
  let fixture: ComponentFixture<CopyFieldComponent>;
  let toast: ToastService;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [CopyFieldComponent] }).compileComponents();
    toast = TestBed.inject(ToastService);
    fixture = TestBed.createComponent(CopyFieldComponent);
    fixture.componentRef.setInput('label', 'نام کاربری');
    fixture.componentRef.setInput('value', 'ali-test');
    vi.stubGlobal(
      'navigator',
      { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } },
    );
  });

  it('masks secret values by default', () => {
    fixture.componentRef.setInput('secret', true);
    fixture.componentRef.setInput('value', 'Secret123');
    fixture.detectChanges();
    expect(fixture.nativeElement.textContent).toContain('••••');
  });

  it('copies value and shows Persian toast', async () => {
    const show = vi.spyOn(toast, 'show');
    fixture.detectChanges();
    const copyBtn = [...fixture.nativeElement.querySelectorAll('button')].find((b: HTMLButtonElement) =>
      b.textContent?.includes('کپی'),
    ) as HTMLButtonElement;
    copyBtn.click();
    await Promise.resolve();
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('ali-test');
    expect(show).toHaveBeenCalledWith(UI_MESSAGES.copyDone);
  });
});
