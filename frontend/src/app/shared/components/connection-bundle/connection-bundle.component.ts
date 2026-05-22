import { Component, inject, input, signal } from '@angular/core';
import { ConnectionBundle } from '../../../core/models';
import { CopyFieldComponent } from '../copy-field/copy-field.component';
import { ToastService } from '../../services/toast.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-connection-bundle',
  standalone: true,
  imports: [CopyFieldComponent],
  template: `
    <section class="card bundle">
      <div class="head">
        <div>
          <h2>اطلاعات اتصال برای مشتری</h2>
          <p class="sub">این بخش را می‌توانید کپی یا برای مشتری ارسال کنید.</p>
        </div>
        <div class="head-actions">
          @if (allowOvpn()) {
            <button type="button" class="btn primary" (click)="onDownloadOvpn()">دانلود ovpn</button>
          }
          <button type="button" class="btn" (click)="copyAll()">کپی همه</button>
        </div>
      </div>
      <p class="warn">این اطلاعات حساس است؛ فقط از کانال امن برای مشتری ارسال کنید.</p>
      <div class="tabs">
        <button type="button" [class.active]="tab() === 'ovpn'" (click)="tab.set('ovpn')">OpenVPN</button>
        <button type="button" [class.active]="tab() === 'l2tp'" (click)="tab.set('l2tp')">L2TP</button>
      </div>
      @if (b(); as bundle) {
        <p class="section-title">مختص این کاربر</p>
        <app-copy-field label="نام کاربری" [value]="bundle.username" [ltr]="true" />
        <app-copy-field
          label="رمز عبور"
          [value]="bundle.password"
          [secret]="true"
          [ltr]="true"
        />
        @if (!bundle.password) {
          <p class="hint">برای نمایش رمز، یک‌بار صفحه را بروزرسانی کنید.</p>
        }
        @if (tab() === 'ovpn') {
          <p class="section-title">ثابت سرویس</p>
          <app-copy-field
            label="رمز کلید OpenVPN"
            [value]="bundle.openvpn_key_password"
            [secret]="true"
            [ltr]="true"
          />
          <app-copy-field label="لینک دانلود" [value]="bundle.openvpn_download_url" [ltr]="true" />
          @if (bundle.openvpn_download_url) {
            <a class="ext" [href]="bundle.openvpn_download_url" target="_blank" rel="noopener">باز کردن لینک</a>
          }
        } @else {
          <p class="section-title">ثابت سرویس</p>
          <app-copy-field
            label="رمز L2TP IPsec"
            [value]="bundle.l2tp_ipsec_secret"
            [secret]="true"
            [ltr]="true"
          />
          <app-copy-field label="سرور L2TP" [value]="bundle.l2tp_server" [ltr]="true" />
        }
        <details class="preview">
          <summary>پیش‌نمایش پیام</summary>
          <pre dir="ltr">{{ previewText() }}</pre>
          <button type="button" class="btn" (click)="copyPreview()">کپی پیش‌نمایش</button>
        </details>
      }
    </section>
  `,
  styles: `
    .bundle h2 {
      margin: 0;
      font-size: 1.05rem;
    }
    .sub {
      margin: 0.25rem 0 0;
      color: #666;
      font-size: 0.88rem;
    }
    .head {
      display: flex;
      justify-content: space-between;
      gap: 1rem;
      flex-wrap: wrap;
    }
    .head-actions {
      display: flex;
      gap: 0.5rem;
      flex-wrap: wrap;
    }
    .warn {
      background: #fff8e1;
      border: 1px solid #ffe082;
      border-radius: 6px;
      padding: 0.5rem 0.75rem;
      font-size: 0.85rem;
      margin: 0.75rem 0;
    }
    .tabs {
      display: flex;
      gap: 0.35rem;
      margin-bottom: 0.75rem;
    }
    .tabs button {
      border: 1px solid #ccc;
      background: #fafafa;
      padding: 0.35rem 0.85rem;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
    }
    .tabs button.active {
      background: #1565c0;
      color: #fff;
      border-color: #1565c0;
    }
    .section-title {
      font-size: 0.82rem;
      color: #888;
      margin: 0.5rem 0 0.25rem;
    }
    .hint {
      color: #ef6c00;
      font-size: 0.85rem;
    }
    .ext {
      font-size: 0.85rem;
      color: #1565c0;
    }
    .preview {
      margin-top: 15pt;
    }
    .preview pre {
      background: #f5f5f5;
      padding: 0.75rem;
      border-radius: 6px;
      white-space: pre-wrap;
      font-size: 0.82rem;
      direction: ltr;
      text-align: left;
      unicode-bidi: plaintext;
    }
    .btn {
      padding: 0.4rem 0.9rem;
      border-radius: 6px;
      border: 1px solid #ccc;
      background: #fff;
      cursor: pointer;
      font: inherit;
    }
    .btn.primary {
      background: #1565c0;
      color: #fff;
      border-color: #1565c0;
    }
  `,
})
export class ConnectionBundleComponent {
  private readonly toast = inject(ToastService);

  bundle = input<ConnectionBundle | null>(null);
  allowOvpn = input(true);
  downloadOvpn = input<(() => void) | null>(null);

  readonly tab = signal<'ovpn' | 'l2tp'>('ovpn');

  b = () => this.bundle();

  previewText(): string {
    const bundle = this.bundle();
    if (!bundle) return '';
    const pw = bundle.password ?? '***';
    return [
      `Username: ${bundle.username}`,
      `Password: ${pw}`,
      `OpenVPN Private Key Password: ${bundle.openvpn_key_password || '***'}`,
      `L2TP IPsec Secret: ${bundle.l2tp_ipsec_secret || '***'}`,
      `L2TP Server: ${bundle.l2tp_server || ''}`,
      `OpenVpn Download: ${bundle.openvpn_download_url || ''}`,
    ].join('\n');
  }

  copyAll(): void {
    void navigator.clipboard.writeText(this.previewText()).then(() => this.toast.show(UI_MESSAGES.copyDone));
  }

  copyPreview(): void {
    this.copyAll();
  }

  onDownloadOvpn(): void {
    this.downloadOvpn()?.();
  }
}
