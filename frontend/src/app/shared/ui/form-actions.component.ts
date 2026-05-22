import { Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';

@Component({
  selector: 'app-form-actions',
  standalone: true,
  imports: [MatButtonModule, RouterLink],
  template: `
    <div class="form-actions">
      <button type="submit" mat-flat-button color="primary" [disabled]="submitDisabled()">
        {{ submitLabel() }}
      </button>
      @if (cancelLink()) {
        <a mat-stroked-button [routerLink]="cancelLink()">{{ cancelLabel() }}</a>
      }
    </div>
  `,
  styles: `
    .form-actions {
      display: flex;
      flex-wrap: wrap;
      gap: 0.75rem;
      margin-top: 0.5rem;
      padding-top: 0.25rem;
    }
  `,
})
export class FormActionsComponent {
  readonly submitLabel = input('ذخیره');
  readonly submitDisabled = input(false);
  readonly cancelLink = input<string | null>(null);
  readonly cancelLabel = input('انصراف');
}
