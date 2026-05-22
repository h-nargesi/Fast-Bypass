import { Component, inject } from '@angular/core';
import { ToastService } from '../../services/toast.service';

@Component({
  selector: 'app-toast',
  standalone: true,
  template: `
    @if (toast.message()) {
      <div class="toast" role="status">{{ toast.message() }}</div>
    }
  `,
  styles: `
    .toast {
      position: fixed;
      bottom: 1.25rem;
      left: 50%;
      transform: translateX(-50%);
      background: #323232;
      color: #fff;
      padding: 0.6rem 1.2rem;
      border-radius: 8px;
      z-index: 2000;
      font-size: 0.9rem;
      box-shadow: 0 4px 16px rgba(0, 0, 0, 0.25);
    }
  `,
})
export class ToastComponent {
  readonly toast = inject(ToastService);
}
