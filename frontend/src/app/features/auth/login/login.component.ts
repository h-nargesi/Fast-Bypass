import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { catchError, of } from 'rxjs';
import { MatButtonModule } from '@angular/material/button';
import { ApiClient } from '../../../core/api/api-client.service';
import { AuthService } from '../../../core/auth/auth.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [FormsModule, MatButtonModule, ...MATERIAL_FORM],
  template: `
    <div class="login-wrap">
      <form class="card login-card form-stack" (ngSubmit)="submit()">
        <h2>{{ msg.login }}</h2>
        <mat-form-field appearance="outline">
          <mat-label>نام کاربری</mat-label>
          <input matInput name="username" [(ngModel)]="username" autocomplete="username" class="ltr-input" required />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>رمز عبور</mat-label>
          <input
            matInput
            name="password"
            type="password"
            class="ltr-input"
            [(ngModel)]="password"
            autocomplete="current-password"
            required
          />
        </mat-form-field>
        @if (error()) {
          <p class="banner err">{{ error() }}</p>
        }
        <button type="submit" mat-flat-button color="primary" class="submit-btn" [disabled]="loading()">
          {{ msg.login }}
        </button>
      </form>
    </div>
  `,
  styles: `
    .login-wrap {
      display: flex;
      justify-content: center;
      padding: 2rem 1rem;
    }
    .login-card {
      width: 100%;
      max-width: 24rem;
    }
    .login-card h2 {
      margin: 0 0 0.5rem;
      text-align: center;
    }
    .submit-btn {
      width: 100%;
      margin-top: 0.25rem;
    }
  `,
})
export class LoginComponent {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);

  readonly msg = UI_MESSAGES;
  username = '';
  password = '';
  readonly error = signal('');
  readonly loading = signal(false);

  submit(): void {
    this.error.set('');
    this.loading.set(true);
    this.auth
      .login(this.username.trim(), this.password)
      .pipe(
        catchError((e) => {
          this.error.set(typeof e === 'string' ? e : ApiClient.mapError(e));
          return of(null);
        }),
      )
      .subscribe((res) => {
        this.loading.set(false);
        if (res) {
          void this.router.navigate([this.auth.homeRoute()]);
        }
      });
  }
}
