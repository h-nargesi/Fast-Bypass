import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ConfirmDialogComponent } from './confirm-dialog.component';
import { UI_MESSAGES } from '../../../core/i18n/messages';

describe('ConfirmDialogComponent (modal تأیید)', () => {
  let fixture: ComponentFixture<ConfirmDialogComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfirmDialogComponent],
    }).compileComponents();
    fixture = TestBed.createComponent(ConfirmDialogComponent);
  });

  it('is hidden when open is false', () => {
    fixture.componentRef.setInput('open', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('[role="alertdialog"]')).toBeNull();
  });

  it('shows Persian title and message when open', () => {
    fixture.componentRef.setInput('open', true);
    fixture.componentRef.setInput('title', UI_MESSAGES.confirmDeleteTitle);
    fixture.componentRef.setInput('message', UI_MESSAGES.confirmDeleteBody);
    fixture.detectChanges();

    const dialog = fixture.nativeElement.querySelector('[role="alertdialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog.textContent).toContain('تأیید حذف');
    expect(dialog.textContent).toContain('حذف این کاربر');
  });

  it('emits confirmed when confirm button clicked', () => {
    fixture.componentRef.setInput('open', true);
    fixture.detectChanges();
    const spy = vi.fn();
    fixture.componentInstance.confirmed.subscribe(spy);

    const buttons = fixture.nativeElement.querySelectorAll('button');
    const confirmBtn = [...buttons].find((b: HTMLButtonElement) =>
      b.textContent?.includes('تأیید') && !b.classList.contains('secondary'),
    ) as HTMLButtonElement;
    confirmBtn.click();
    expect(spy).toHaveBeenCalledOnce();
  });

  it('emits cancelled on cancel button or backdrop', () => {
    fixture.componentRef.setInput('open', true);
    fixture.detectChanges();
    const spy = vi.fn();
    fixture.componentInstance.cancelled.subscribe(spy);

    const cancelBtn = fixture.nativeElement.querySelector('.btn.secondary') as HTMLButtonElement;
    cancelBtn.click();
    expect(spy).toHaveBeenCalledOnce();
  });

  it('dialog uses RTL-friendly text alignment', () => {
    fixture.componentRef.setInput('open', true);
    fixture.detectChanges();
    const dialog = fixture.nativeElement.querySelector('.dialog') as HTMLElement;
    expect(getComputedStyle(dialog).textAlign).toBe('right');
  });
});
