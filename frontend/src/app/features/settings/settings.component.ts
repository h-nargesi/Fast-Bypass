import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, EMPTY } from 'rxjs';
import { MatButtonModule } from '@angular/material/button';
import { ApiClient } from '../../core/api/api-client.service';
import { AuthService, ProfileService } from '../../core/auth/auth.service';
import { AdminProfile, ManagerProfile, MeProfile } from '../../core/models';
import { ToastService } from '../../shared/services/toast.service';
import { MATERIAL_FORM } from '../../shared/ui/material-form';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [FormsModule, MatButtonModule, ...MATERIAL_FORM],
  template: `
    <h2 class="page-title">تنظیمات حساب</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (profile(); as p) {
      <section class="card">
        <h3>اطلاعات (فقط خواندنی)</h3>
        <dl class="meta">
          <dt>نام کاربری</dt><dd>{{ p.username }}</dd>
          @if (isManager(p)) {
            <dt>پیشوند (slug)</dt><dd>{{ p.slug }}</dd>
            <dt>سقف</dt><dd>{{ p.quota }} — مصرف: {{ p.used_quota }}</dd>
          }
        </dl>
      </section>
      @if (isManager(p)) {
        <section class="card">
          <h3>نام نمایشی</h3>
          <form class="form-stack" (ngSubmit)="saveName()">
            <mat-form-field appearance="outline">
              <mat-label>نام نمایشی</mat-label>
              <input matInput [(ngModel)]="displayName" name="dn" required />
            </mat-form-field>
            <button type="submit" mat-flat-button color="primary">ذخیره</button>
          </form>
        </section>
      }
      <section class="card">
        <h3>تغییر رمز</h3>
        <form class="form-stack" (ngSubmit)="changePassword()">
          <mat-form-field appearance="outline">
            <mat-label>رمز فعلی</mat-label>
            <input matInput type="password" class="ltr-input" [(ngModel)]="currentPw" name="cp" required />
          </mat-form-field>
          <mat-form-field appearance="outline">
            <mat-label>رمز جدید</mat-label>
            <input matInput type="password" class="ltr-input" [(ngModel)]="newPw" name="np" required />
          </mat-form-field>
          <button type="submit" mat-flat-button color="primary">تغییر رمز</button>
        </form>
      </section>
    }
  `,
  styles: `
    .meta {
      display: grid;
      grid-template-columns: 8rem 1fr;
      gap: 0.35rem 1rem;
      margin: 0;
    }
    dt {
      color: #666;
      margin: 0;
    }
    dd {
      margin: 0;
      font-weight: 600;
    }
  `,
})
export class SettingsComponent implements OnInit {
  private readonly profileSvc = inject(ProfileService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);

  readonly profile = signal<MeProfile | null>(null);
  readonly error = signal('');
  displayName = '';
  currentPw = '';
  newPw = '';

  ngOnInit(): void {
    this.profileSvc.getMe().subscribe((p) => {
      this.profile.set(p);
      if ('display_name' in p) {
        this.displayName = (p as ManagerProfile).display_name;
      }
    });
  }

  isManager(p: MeProfile): p is ManagerProfile {
    return 'slug' in p;
  }

  saveName(): void {
    if (!this.auth.isManager()) return;
    this.profileSvc.patchDisplayName(this.displayName).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return EMPTY;
      }),
    ).subscribe((p) => {
      if (p) {
        this.profile.set(p);
        this.toast.show('ذخیره شد');
      }
    });
  }

  changePassword(): void {
    this.error.set('');
    this.profileSvc.changePassword(this.currentPw, this.newPw).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return EMPTY;
      }),
    ).subscribe({
      next: () => {
        this.toast.show('رمز تغییر کرد');
        this.currentPw = '';
        this.newPw = '';
      },
    });
  }
}
