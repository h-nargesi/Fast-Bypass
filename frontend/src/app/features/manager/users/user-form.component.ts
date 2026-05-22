import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ProfileService } from '../../../core/auth/auth.service';
import { ManagerProfile } from '../../../core/models';
import { VpnUserService } from '../../../core/services/vpn-user.service';
import { environment } from '../../../../environments/environment';
import { FormActionsComponent } from '../../../shared/ui/form-actions.component';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-user-form',
  standalone: true,
  imports: [FormsModule, FormActionsComponent, ...MATERIAL_FORM],
  template: `
    <h2 class="page-title">کاربر جدید</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <form class="card form-stack" (ngSubmit)="submit()">
      <mat-checkbox [(ngModel)]="routerEnabled" name="ren">فعال</mat-checkbox>

      <mat-form-field appearance="outline">
        <mat-label>نام کاربر VPN</mat-label>
        <span matTextSuffix class="ltr-input prefix-chip" style="display: inline-block;">{{ prefix() }}</span>
        <input
          matInput
          class="ltr-input"
          [(ngModel)]="localName"
          name="localName"
          required
          placeholder="reza01"
        />
        <mat-hint>نام نهایی در روتر: <code dir="ltr">{{ fullName() }}</code></mat-hint>
      </mat-form-field>

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

      <mat-checkbox [(ngModel)]="assignProfile" name="assign">
        انتساب پروفایل {{ defaultProfile }}
      </mat-checkbox>

      @if (assignProfile) {
        <mat-form-field appearance="outline">
          <mat-label>مبلغ پرداخت (اختیاری)</mat-label>
          <input matInput type="number" [(ngModel)]="amountPaid" name="amount" />
        </mat-form-field>
      }

      <app-form-actions submitLabel="ذخیره" [submitDisabled]="saving()" cancelLink="/users" />
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
