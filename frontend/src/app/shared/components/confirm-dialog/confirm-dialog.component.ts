import { Component, input, output } from '@angular/core';

@Component({
  selector: 'app-confirm-dialog',
  standalone: true,
  template: `
    @if (open()) {
      <div class="backdrop" role="presentation" (click)="onCancel()"></div>
      <div
        class="dialog"
        role="alertdialog"
        [attr.aria-labelledby]="dialogId + '-title'"
        [attr.aria-describedby]="dialogId + '-body'"
      >
        <h2 [id]="dialogId + '-title'">{{ title() }}</h2>
        <p [id]="dialogId + '-body'">{{ message() }}</p>
        <div class="actions">
          <button type="button" class="btn secondary" (click)="onCancel()">{{ cancelLabel() }}</button>
          <button type="button" class="btn danger" (click)="onConfirm()">{{ confirmLabel() }}</button>
        </div>
      </div>
    }
  `,
  styles: `
    .backdrop {
      position: fixed;
      inset: 0;
      background: rgba(0, 0, 0, 0.45);
      z-index: 1000;
    }
    .dialog {
      position: fixed;
      top: 50%;
      right: 50%;
      transform: translate(50%, -50%);
      z-index: 1001;
      background: #fff;
      border-radius: 8px;
      padding: 1.25rem 1.5rem;
      min-width: 20rem;
      max-width: 28rem;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
      text-align: right;
    }
    h2 {
      margin: 0 0 0.5rem;
      font-size: 1.1rem;
    }
    p {
      margin: 0 0 1rem;
      color: #444;
    }
    .actions {
      display: flex;
      gap: 0.5rem;
      justify-content: flex-start;
    }
    .btn {
      padding: 0.4rem 1rem;
      border-radius: 6px;
      border: 1px solid #ccc;
      cursor: pointer;
      font: inherit;
    }
    .btn.danger {
      background: #c62828;
      color: #fff;
      border-color: #b71c1c;
    }
    .btn.secondary {
      background: #f5f5f5;
    }
  `,
})
export class ConfirmDialogComponent {
  readonly dialogId = 'confirm-' + Math.random().toString(36).slice(2, 9);

  open = input(false);
  title = input('تأیید');
  message = input('');
  confirmLabel = input('تأیید');
  cancelLabel = input('انصراف');

  confirmed = output<void>();
  cancelled = output<void>();

  onConfirm(): void {
    this.confirmed.emit();
  }

  onCancel(): void {
    this.cancelled.emit();
  }
}
