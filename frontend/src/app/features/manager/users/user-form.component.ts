import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ProfileService } from '../../../core/auth/auth.service';
import { ManagerProfile } from '../../../core/models';
import { VpnUserService } from '../../../core/services/vpn-user.service';
import { environment } from '../../../../environments/environment';

@Component({
  selector: 'app-user-form',
  standalone: true,
  imports: [FormsModule, RouterLink],
  template: `
    <h2 class="page-title">کاربر جدید</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <form class="card form" (ngSubmit)="submit()">
      <fieldset>
        <legend>نام کاربر VPN</legend>
        <div class="prefix-row">
          <span class="prefix">{{ prefix() }}</span>
          <input [(ngModel)]="localName" name="localName" required placeholder="reza01" />
        </div>
        <p class="hint">نام نهایی در روتر: <code dir="ltr">{{ fullName() }}</code></p>
      </fieldset>
      <label>
        رمز VPN
        <input type="password" [(ngModel)]="password" name="password" />
      </label>
      <p class="hint">خالی بگذارید تا سرور رمز تصادفی بسازد.</p>
      <label>اتصال همزمان <input type="number" min="1" [(ngModel)]="sharedUsers" name="shared" /></label>
      <label class="check">
        <input type="checkbox" [(ngModel)]="routerEnabled" name="ren" />
        فعال در روتر (User Manager)
      </label>
      <label>اطلاعات تماس <input [(ngModel)]="contactInfo" name="cinfo" /></label>
      <label>یادداشت <textarea [(ngModel)]="notes" name="notes" rows="2"></textarea></label>
      <label class="check">
        <input type="checkbox" [(ngModel)]="assignProfile" name="assign" />
        انتساب پروفایل {{ defaultProfile }}
      </label>
      @if (assignProfile) {
        <label>مبلغ پرداخت (اختیاری) <input type="number" [(ngModel)]="amountPaid" name="amount" /></label>
      }
      <div class="actions">
        <button type="submit" class="btn primary" [disabled]="saving()">ذخیره</button>
        <a routerLink="/users" class="btn">انصراف</a>
      </div>
    </form>
  `,
  styles: `
    .prefix-row {
      display: flex;
      gap: 0.35rem;
      align-items: center;
    }
    .prefix {
      background: #eceff1;
      padding: 0.5rem 0.65rem;
      border-radius: 6px;
      font-family: ui-monospace, monospace;
      direction: ltr;
    }
    .prefix-row input {
      flex: 1;
    }
    code {
      background: #f5f5f5;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
    }
    .check {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
  `,
})
export class UserFormComponent implements OnInit {
  private readonly profileSvc = inject(ProfileService);
  private readonly vpn = inject(VpnUserService);
  private readonly router = inject(Router);

  readonly defaultProfile = environment.defaultProfile;
  readonly prefix = signal('');
  localName = '';
  password = '';
  sharedUsers = 1;
  routerEnabled = true;
  contactInfo = '';
  notes = '';
  assignProfile = true;
  amountPaid: number | null = null;
  readonly saving = signal(false);
  readonly error = signal('');

  ngOnInit(): void {
    this.profileSvc.getMe().subscribe((me) => {
      this.prefix.set((me as ManagerProfile).name_prefix ?? '');
    });
  }

  fullName(): string {
    return `${this.prefix()}${this.localName}`;
  }

  submit(): void {
    this.saving.set(true);
    this.error.set('');
    const body = {
      local_name: this.localName.trim(),
      password: this.password.trim() || undefined,
      shared_users: this.sharedUsers,
      disabled: !this.routerEnabled,
      contact_info: this.contactInfo || undefined,
      notes: this.notes || undefined,
      assign_profile: this.assignProfile,
      profile_name: this.assignProfile ? this.defaultProfile : undefined,
      amount_paid: this.amountPaid ?? undefined,
      currency: this.amountPaid != null ? 'IRR' : undefined,
    };
    this.vpn
      .create(body)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          this.saving.set(false);
          return of(null);
        }),
      )
      .subscribe((res) => {
        if (res?.id) {
          void this.router.navigate(['/users', res.id]);
        }
      });
  }
}
