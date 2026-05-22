import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { VpnUserDetail } from '../../../core/models';
import { PatchVpnBody, VpnUserService } from '../../../core/services/vpn-user.service';
import { ConnectionBundleComponent } from '../../../shared/components/connection-bundle/connection-bundle.component';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { formatRial } from '../../../core/format/currency';
import { environment } from '../../../../environments/environment';
import { MatButtonModule } from '@angular/material/button';
import { ToastService } from '../../../shared/services/toast.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-user-detail',
  standalone: true,
  imports: [
    FormsModule,
    RouterLink,
    MatButtonModule,
    ConnectionBundleComponent,
    ConfirmDialogComponent,
    ProfileStateChipComponent,
    JalaliDatePipe,
    ...MATERIAL_FORM,
  ],
  template: `
    <a routerLink="/users" class="back">← بازگشت</a>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (user(); as u) {
      <h2 class="page-title">{{ u.mikrotik_name }}</h2>
      @if (!u.id) {
        <p class="banner info">این کاربر هنوز در پنل ثبت نشده — با «ذخیره تغییرات» ثبت می‌شود.</p>
      }
      <app-connection-bundle
        [bundle]="u.connection_bundle"
        [downloadOvpn]="downloadFn"
      />
      <form class="card form-stack" (ngSubmit)="saveEdit()">
        <mat-checkbox [(ngModel)]="routerEnabled" name="ren">فعال</mat-checkbox>

        <mat-form-field appearance="outline">
          <mat-label>رمز VPN</mat-label>
          <input matInput type="password" class="ltr-input" [(ngModel)]="editPassword" name="pw" />
          <mat-hint>خالی بگذارید تا رمز تغییر نکند.</mat-hint>
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>اتصال همزمان</mat-label>
          <input matInput type="number" min="1" [(ngModel)]="editShared" name="su" />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>اطلاعات تماس</mat-label>
          <input matInput [(ngModel)]="editContactInfo" name="cn" />
        </mat-form-field>

        <mat-form-field appearance="outline">
          <mat-label>یادداشت</mat-label>
          <textarea matInput [(ngModel)]="editNotes" name="nt" rows="2"></textarea>
        </mat-form-field>

        <button type="submit" mat-flat-button color="primary" [disabled]="saving()">ذخیره تغییرات</button>
      </form>
      <section class="card">
        <h3>تمدید / انتساب پروفایل</h3>
        <form class="form-stack" (ngSubmit)="assign()">
          <p class="profile-name">پروفایل: <code dir="ltr">{{ defaultProfile }}</code></p>
          <mat-form-field appearance="outline">
            <mat-label>مبلغ (اختیاری)</mat-label>
            <input matInput type="number" [(ngModel)]="assignAmount" name="amt" />
          </mat-form-field>
          <mat-form-field appearance="outline">
            <mat-label>یادداشت</mat-label>
            <input matInput [(ngModel)]="assignNote" name="an" />
          </mat-form-field>
          <button type="submit" mat-flat-button color="primary">تمدید</button>
        </form>
      </section>
      <section class="card">
        <h3>پروفایل‌ها</h3>
        <table>
          <thead>
            <tr><th>پروفایل</th><th>وضعیت</th><th>اعتبار</th><th></th></tr>
          </thead>
          <tbody>
            @for (p of u.profiles; track p.id) {
              <tr>
                <td>{{ p.profile }}</td>
                <td><app-profile-state-chip [state]="p.state" /></td>
                <td>{{ p.end_time | jalaliDate: 'datetime' }}</td>
                <td>
                  @if (!isActive(p.state)) {
                    <button type="button" class="link danger" (click)="removeProfile(p.id)">حذف</button>
                  }
                </td>
              </tr>
            }
          </tbody>
        </table>
      </section>
      <section class="card">
        <h3>تاریخچه تمدید</h3>
        <table>
          <thead>
            <tr><th>تاریخ</th><th>پروفایل</th><th>مبلغ</th><th>تسویه</th></tr>
          </thead>
          <tbody>
            @for (a of u.activations; track a.id) {
              <tr>
                <td>{{ a.created_at | jalaliDate }}</td>
                <td>{{ a.profile_name }}</td>
                <td>{{ formatAmount(a.amount_paid) }}</td>
                <td>{{ a.is_settled ? '✓' : '✗' }}</td>
              </tr>
            }
          </tbody>
        </table>
      </section>
      <button type="button" class="btn danger" (click)="confirmDelete.set(true)">حذف کاربر</button>
    }
    <app-confirm-dialog
      [open]="confirmDelete()"
      [title]="msg.confirmDeleteTitle"
      [message]="msg.confirmDeleteBody"
      (confirmed)="deleteUser()"
      (cancelled)="confirmDelete.set(false)"
    />
  `,
  styles: `
    .profile-name {
      margin: 0 0 0.75rem;
      color: #444;
    }
    .profile-name code {
      background: #f5f5f5;
      padding: 0.15rem 0.4rem;
      border-radius: 4px;
    }
  `,
})
export class UserDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly vpn = inject(VpnUserService);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);

  readonly msg = UI_MESSAGES;
  readonly defaultProfile = environment.defaultProfile;
  readonly user = signal<VpnUserDetail | null>(null);
  readonly error = signal('');
  readonly saving = signal(false);
  readonly confirmDelete = signal(false);

  editPassword = '';
  editShared = 1;
  editContactInfo = '';
  editNotes = '';
  routerEnabled = true;
  assignAmount: number | null = null;
  assignNote = '';

  readonly formatAmount = formatRial;
  readonly downloadFn = () => this.downloadOvpn();

  ngOnInit(): void {
    this.route.paramMap.subscribe((p) => {
      const name = p.get('name');
      if (name) {
        this.loadByName(name);
        return;
      }
      const id = Number(p.get('id'));
      if (id) this.load(id);
    });
  }

  private applyUser(u: VpnUserDetail): void {
    this.user.set(u);
    this.editShared = u.shared_users;
    this.editContactInfo = u.contact_info ?? '';
    this.editNotes = u.notes ?? '';
    this.routerEnabled = !u.disabled;
  }

  private afterWrite(res: VpnUserDetail, toastMsg: string): void {
    this.applyUser(res);
    this.toast.show(toastMsg);
    if (res.id) {
      void this.router.navigate(['/users', res.id], { replaceUrl: true });
    }
  }

  load(id: number): void {
    this.vpn
      .get(id)
      .pipe(catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }))
      .subscribe((u) => {
        if (u) this.applyUser(u);
      });
  }

  loadByName(name: string): void {
    this.vpn
      .getByName(name)
      .pipe(catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }))
      .subscribe((u) => {
        if (u) this.applyUser(u);
      });
  }

  reload(): void {
    const u = this.user();
    if (!u) return;
    if (u.id) this.load(u.id);
    else this.loadByName(u.mikrotik_name);
  }

  saveEdit(): void {
    const u = this.user();
    if (!u) return;
    this.saving.set(true);
    const body: PatchVpnBody = {
      shared_users: this.editShared,
      contact_info: this.editContactInfo,
      notes: this.editNotes,
      disabled: !this.routerEnabled,
    };
    if (this.editPassword) body.password = this.editPassword;
    const req = u.id
      ? this.vpn.patch(u.id, body)
      : this.vpn.patchByName(u.mikrotik_name, body);
    req.pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        this.saving.set(false);
        return of(null);
      }),
    ).subscribe((res) => {
      this.saving.set(false);
      if (res) this.afterWrite(res, 'ذخیره شد');
    });
  }

  assign(): void {
    const u = this.user();
    if (!u) return;
    const body = {
      profile_name: this.defaultProfile,
      amount_paid: this.assignAmount ?? undefined,
      currency: this.assignAmount != null ? 'IRR' : undefined,
      note: this.assignNote || undefined,
    };
    const req = u.id
      ? this.vpn.assignProfile(u.id, body)
      : this.vpn.assignProfileByName(u.mikrotik_name, body);
    req.pipe(catchError((e) => {
      this.error.set(ApiClient.mapError(e));
      return of(null);
    })).subscribe((res) => {
      if (res) this.afterWrite(res, 'پروفایل انتساب شد');
    });
  }

  removeProfile(profileRowId: string): void {
    const u = this.user();
    if (!u) return;
    const done = () => this.reload();
    const req = u.id
      ? this.vpn.removeProfile(u.id, profileRowId)
      : this.vpn.removeProfileByName(u.mikrotik_name, profileRowId);
    req.subscribe({
      next: done,
      error: (e) => this.error.set(ApiClient.mapError(e)),
    });
  }

  deleteUser(): void {
    const u = this.user();
    if (!u) return;
    this.confirmDelete.set(false);
    const req = u.id ? this.vpn.delete(u.id) : this.vpn.deleteByName(u.mikrotik_name);
    req.subscribe({
      next: () => void this.router.navigate(['/users']),
      error: (e: unknown) => this.error.set(ApiClient.mapError(e)),
    });
  }

  downloadOvpn(): void {
    const u = this.user();
    if (!u) return;
    const req = u.id ? this.vpn.downloadOvpn(u.id) : this.vpn.downloadOvpnByName(u.mikrotik_name);
    req.subscribe({
      next: (blob) => {
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${u.mikrotik_name}.ovpn`;
        a.click();
        URL.revokeObjectURL(url);
      },
      error: (e) => this.error.set(ApiClient.mapError(e)),
    });
  }

  isActive(state: string): boolean {
    return (state || '').toLowerCase().includes('active');
  }
}
