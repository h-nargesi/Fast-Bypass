import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ManagerRow } from '../../../core/models';
import { AdminService, AdminVpnService } from '../../../core/services/vpn-user.service';
import { environment } from '../../../../environments/environment';

@Component({
  selector: 'app-admin-user-form',
  standalone: true,
  imports: [FormsModule, RouterLink],
  template: `
    <a routerLink="/admin/users" class="back">← بازگشت</a>
    <h2 class="page-title">کاربر VPN جدید</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <form class="card form" (ngSubmit)="submit()">
      <label>
        مدیر
        <select [(ngModel)]="managerId" name="mgr">
          <option [ngValue]="null">بدون مدیر</option>
          @for (m of managers(); track m.id) {
            <option [ngValue]="m.id">{{ m.display_name }} ({{ m.slug }})</option>
          }
        </select>
      </label>
      @if (selectedManager(); as mgr) {
        <fieldset>
          <legend>نام کاربر VPN</legend>
          <div class="prefix-row">
            <span class="prefix">{{ namePrefix(mgr) }}</span>
            <input [(ngModel)]="localName" name="localName" required placeholder="reza01" />
          </div>
          <p class="hint">نام نهایی در روتر: <code dir="ltr">{{ fullName(mgr) }}</code></p>
        </fieldset>
      } @else {
        <label>
          نام کاربر در روتر
          <input [(ngModel)]="localName" name="localNameFull" required placeholder="reza01" dir="ltr" />
        </label>
        <p class="hint">کاربر بدون مدیر — comment روتر خالی می‌ماند.</p>
      }
      <label>رمز VPN <input type="password" [(ngModel)]="password" name="password" required /></label>
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
        <button type="submit" class="btn primary" [disabled]="saving() || !localName.trim()">ذخیره</button>
        <a routerLink="/admin/users" class="btn">انصراف</a>
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
export class AdminUserFormComponent implements OnInit {
  private readonly admin = inject(AdminService);
  private readonly vpn = inject(AdminVpnService);
  private readonly router = inject(Router);

  readonly defaultProfile = environment.defaultProfile;
  readonly managers = signal<ManagerRow[]>([]);
  readonly saving = signal(false);
  readonly error = signal('');

  managerId: number | null = null;
  localName = '';
  password = '';
  sharedUsers = 1;
  routerEnabled = true;
  contactInfo = '';
  notes = '';
  assignProfile = true;
  amountPaid: number | null = null;

  ngOnInit(): void {
    this.admin.listManagers().subscribe((r) => this.managers.set(r.items));
  }

  selectedManager(): ManagerRow | undefined {
    if (this.managerId == null) return undefined;
    return this.managers().find((m) => m.id === this.managerId);
  }

  namePrefix(mgr: ManagerRow): string {
    return `${mgr.slug}-`;
  }

  fullName(mgr: ManagerRow): string {
    return `${this.namePrefix(mgr)}${this.localName}`;
  }

  submit(): void {
    this.saving.set(true);
    this.error.set('');
    const body: Parameters<AdminVpnService['create']>[0] = {
      local_name: this.localName.trim(),
      password: this.password,
      shared_users: this.sharedUsers,
      disabled: !this.routerEnabled,
      contact_info: this.contactInfo || undefined,
      notes: this.notes || undefined,
      assign_profile: this.assignProfile,
      profile_name: this.assignProfile ? this.defaultProfile : undefined,
      amount_paid: this.amountPaid ?? undefined,
      currency: this.amountPaid != null ? 'IRR' : undefined,
    };
    if (this.managerId != null) {
      body.manager_id = this.managerId;
    }
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
        this.saving.set(false);
        if (res?.id) {
          void this.router.navigate(['/admin/users', res.id]);
        }
      });
  }
}
