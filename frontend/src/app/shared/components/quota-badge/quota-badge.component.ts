import { Component, input } from '@angular/core';

@Component({
  selector: 'app-quota-badge',
  standalone: true,
  template: `
    <div class="quota" [class.warn]="available() <= 0">
      <span class="label">سقف اتصال</span>
      <span class="nums">{{ used() }} / {{ quota() }}</span>
      <span class="avail">باقی‌مانده: {{ available() }}</span>
    </div>
  `,
  styles: `
    .quota {
      background: #fff;
      border: 1px solid #e0e0e0;
      border-radius: 8px;
      padding: 0.85rem 1rem;
      display: flex;
      flex-wrap: wrap;
      gap: 0.35rem 1rem;
      align-items: baseline;
    }
    .quota.warn {
      border-color: #ffb74d;
      background: #fff8e1;
    }
    .label {
      font-weight: 600;
      color: #555;
    }
    .nums {
      font-size: 1.15rem;
      font-weight: 700;
    }
    .avail {
      color: #666;
      font-size: 0.88rem;
    }
  `,
})
export class QuotaBadgeComponent {
  quota = input(0);
  used = input(0);
  available = input(0);
}
