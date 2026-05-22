import { Injectable, signal } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ToastService {
  readonly message = signal<string | null>(null);

  show(text: string, ms = 2500): void {
    this.message.set(text);
    setTimeout(() => {
      if (this.message() === text) {
        this.message.set(null);
      }
    }, ms);
  }
}
