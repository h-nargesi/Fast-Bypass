import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { catchError, finalize, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ManagerRow } from '../../../core/models';
import { AdminService, AdminVpnService } from '../../../core/services/vpn-user.service';
import { environment } from '../../../../environments/environment';
import { FormActionsComponent } from '../../../shared/ui/form-actions.component';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-admin-user-form',
  standalone: true,
  imports: [FormsModule, RouterLink, FormActionsComponent, ...MATERIAL_FORM],
  template: `
    <a routerLink="/admin/users" class="back">← بازگشت</a>
    <h2 class="page-title">کاربر VPN جدید</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <form class="card form-stack" (ngSubmit)="submit()">
      <mat-checkbox [(ngModel)]="routerEnabled" name="ren">فعال</mat-checkbox>

      <mat-form-field appearance="outline">
        <mat-label>مدیر</mat-label>
        <mat-select [(ngModel)]="managerId" name="mgr">
          <mat-option [value]="null">بدون مدیر</mat-option>
          @for (m of managers(); track m.id) {
            <mat-option [value]="m.id">{{ m.display_name }} ({{ m.slug }})</mat-option>
          }
        </mat-select>
      </mat-form-field>

      @if (selectedManager(); as mgr) {
        <mat-form-field appearance="outline">
          <mat-label>نام کاربر VPN</mat-label>
          <span matTextSuffix class="ltr-input prefix-chip" style="display: inline-block;">{{ namePrefix(mgr) }}</span>
          <input
            matInput
            class="ltr-input"
            [(ngModel)]="localName"
            name="localName"
            required
            placeholder="reza01"
          />
          <mat-hint>نام نهایی در روتر: <code dir="ltr">{{ fullName(mgr) }}</code></mat-hint>
        </mat-form-field>
      } @else {
        <mat-form-field appearance="outline">
          <mat-label>نام کاربر در روتر</mat-label>
          <input
            matInput
            class="ltr-input"
            [(ngModel)]="localName"
            name="localNameFull"
            required
            placeholder="reza01"
          />
          <mat-hint>کاربر بدون مدیر — comment روتر خالی می‌ماند.</mat-hint>
        </mat-form-field>
      }

      <mat-form-field appearance="outline">
        <mat-label>رمز VPN</mat-label>
        <input matInput type="password" class="ltr-input" [(ngModel)]="password" name="password" />
        <mat-hint>خالی بگذارید تا سرور رمز تصادفی بسازد.</mat-hint>
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>اتصال همزمان</mat-label>
        <input matInput type="number" min="1" [(ngModel)]="sharedUsers" name="shared" />
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>اطلاعات تماس</mat-label>
        <input matInput [(ngModel)]="contactInfo" name="cinfo" />
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>یادداشت</mat-label>
        <textarea matInput [(ngModel)]="notes" name="notes" rows="2"></textarea>
      </mat-form-field>

      <mat-form-field appearance="outline">
        <mat-label>عنوان گواهی (اختیاری)</mat-label>
        <input matInput class="ltr-input" [(ngModel)]="certTitle" name="certTitle" placeholder="my-cert" />
        <mat-hint>در صورت پر شدن، گواهی OpenVPN هنگام ایجاد کاربر ساخته می‌شود.</mat-hint>
      </mat-form-field>

      <mat-checkbox [(ngModel)]="assignProfile" name="assign">
        انتساب پروفایل {{ defaultProfile }}
      </mat-checkbox>

      @if (assignProfile) {
        <mat-form-field appearance="outline">
          <mat-label>مبلغ پرداخت (اختیاری)</mat-label>
          <input matInput type="number" [(ngModel)]="amountPaid" name="amount" />
        </mat-form-field>
      }

      <app-form-actions
        [submitLabel]="submitLabel()"
        [submitDisabled]="saving() || !localName.trim()"
        cancelLink="/admin/users"
      />
    </form>
  `,
  styles: `
    code {
      background: #f5f5f5;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
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

  readonly submitLabel = computed(() => {
    if (!this.saving()) {
      return 'ذخیره';
    }
    return this.certTitle.trim() ? 'در حال ذخیره و ساخت گواهی…' : 'در حال ذخیره…';
  });

  managerId: number | null = null;
  localName = '';
  password = '';
  sharedUsers = 1;
  routerEnabled = true;
  contactInfo = '';
  notes = '';
  certTitle = '';
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
    if (this.managerId != null) {
      body.manager_id = this.managerId;
    }
    const ct = this.certTitle.trim();
    if (ct) {
      body.cert_title = ct;
    }
    this.vpn
      .create(body)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of(null);
        }),
        finalize(() => this.saving.set(false)),
      )
      .subscribe((res) => {
        if (res?.id) {
          void this.router.navigate(['/admin/users', res.id]);
        }
      });
  }
}
