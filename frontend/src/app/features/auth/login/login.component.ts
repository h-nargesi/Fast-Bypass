import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { AuthService } from '../../../core/auth/auth.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [FormsModule],
  template: `
    <div class="login-wrap">
      <form class="card login-card" (ngSubmit)="submit()">
        <h2>{{ msg.login }}</h2>
        <label>
          نام کاربری
          <input name="username" [(ngModel)]="username" autocomplete="username" required />
        </label>
        <label>
          رمز عبور
          <input
            name="password"
            type="password"
            [(ngModel)]="password"
            autocomplete="current-password"
            required
          />
        </label>
        @if (error()) {
          <p class="err">{{ error() }}</p>
        }
        <button type="submit" class="btn primary" [disabled]="loading()">{{ msg.login }}</button>
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
      max-width: 22rem;
    }
    .login-card h2 {
      margin: 0 0 1rem;
      text-align: center;
    }
    label {
      display: block;
      margin-bottom: 0.85rem;
      font-size: 0.9rem;
    }
    input {
      display: block;
      width: 100%;
      margin-top: 0.3rem;
      padding: 0.5rem 0.65rem;
      border: 1px solid #ccc;
      border-radius: 6px;
      font: inherit;
      box-sizing: border-box;
    }
    .err {
      color: #c62828;
      font-size: 0.88rem;
      margin: 0 0 0.75rem;
    }
    .btn.primary {
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
