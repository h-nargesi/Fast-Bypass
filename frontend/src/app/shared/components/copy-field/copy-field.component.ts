import { Component, inject, input, signal } from '@angular/core';
import { ToastService } from '../../services/toast.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-copy-field',
  standalone: true,
  template: `
    <div class="row">
      <span class="lbl">{{ label() }}</span>
      <span class="val" [class.ltr]="ltr()">
        @if (secret() && !revealed()) {
          ••••••••
        } @else {
          {{ value() || '—' }}
        }
      </span>
      <div class="acts">
        @if (secret()) {
          <button type="button" class="link" (click)="toggle()">{{ revealed() ? 'پنهان' : 'نمایش' }}</button>
        }
        @if (url()) {
          <a class="link" [href]="url()" target="_blank" rel="noopener">باز کردن لینک</a>
        }
        <button type="button" class="link" (click)="copy()">کپی</button>
      </div>
    </div>
  `,
  styles: `
    .row {
      display: grid;
      grid-template-columns: 8rem 1fr auto;
      gap: 0.5rem;
      align-items: center;
      padding: 0.35rem 0;
      border-bottom: 1px solid #f0f0f0;
    }
    .lbl {
      color: #555;
      font-size: 0.88rem;
    }
    .val {
      font-family: inherit;
      word-break: break-all;
    }
    .val.ltr {
      direction: ltr;
      text-align: left;
      font-family: ui-monospace, monospace;
      font-size: 0.9rem;
    }
    .acts {
      display: flex;
      gap: 0.35rem;
    }
    .link {
      background: none;
      border: none;
      color: #1565c0;
      cursor: pointer;
      font: inherit;
      font-size: 0.82rem;
      padding: 0;
    }
  `,
})
export class CopyFieldComponent {
  private readonly toast = inject(ToastService);

  label = input('');
  url = input<string | null>('');
  value = input<string | null>('');
  secret = input(false);
  ltr = input(false);

  readonly revealed = signal(false);

  toggle(): void {
    this.revealed.update((v) => !v);
  }

  copy(): void {
    const text = this.secret() && !this.revealed() ? '' : (this.value() ?? '');
    if (!text) return;
    void navigator.clipboard.writeText(text).then(() => this.toast.show(UI_MESSAGES.copyDone));
  }
}
