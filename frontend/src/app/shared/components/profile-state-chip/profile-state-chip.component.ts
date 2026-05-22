import { Component, input } from '@angular/core';

@Component({
  selector: 'app-profile-state-chip',
  standalone: true,
  template: `
    @if (state()) {
      <span class="chip" [class]="chipClass()">{{ label() }}</span>
    } @else {
      <span class="chip muted">—</span>
    }
  `,
  styles: `
    .chip {
      display: inline-block;
      padding: 0.15rem 0.55rem;
      border-radius: 999px;
      font-size: 0.78rem;
      font-weight: 600;
    }
    .active {
      background: #e8f5e9;
      color: #2e7d32;
    }
    .waiting {
      background: #fff3e0;
      color: #ef6c00;
    }
    .expired {
      background: #ffebee;
      color: #c62828;
    }
    .other {
      background: #eceff1;
      color: #455a64;
    }
    .muted {
      background: #f0f0f0;
      color: #888;
    }
  `,
})
export class ProfileStateChipComponent {
  state = input('');

  label(): string {
    const s = (this.state() || '').toLowerCase();
    if (s.includes('active') || s === 'فعال') return 'فعال';
    if (s.includes('waiting') || s.includes('reserve')) return 'رزرو';
    if (s.includes('expir') || s.includes('used')) return 'منقضی';
    return this.state() || '—';
  }

  chipClass(): string {
    const s = (this.state() || '').toLowerCase();
    if (s.includes('active')) return 'active';
    if (s.includes('waiting') || s.includes('reserve')) return 'waiting';
    if (s.includes('expir') || s.includes('used')) return 'expired';
    return 'other';
  }
}
