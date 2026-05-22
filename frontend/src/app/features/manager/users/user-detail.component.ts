import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { VpnUserDetail } from '../../../core/models';
import { VpnUserService } from '../../../core/services/vpn-user.service';
import { ConnectionBundleComponent } from '../../../shared/components/connection-bundle/connection-bundle.component';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { formatRial } from '../../../core/format/currency';
import { environment } from '../../../../environments/environment';
import { ToastService } from '../../../shared/services/toast.service';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-user-detail',
  standalone: true,
  imports: [
    FormsModule,
    RouterLink,
    ConnectionBundleComponent,
    ConfirmDialogComponent,
    ProfileStateChipComponent,
    JalaliDatePipe,
  ],
  template: `
    <a routerLink="/users" class="back">← بازگشت</a>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (user(); as u) {
      <h2 class="page-title">{{ u.mikrotik_name }}</h2>
      <app-connection-bundle
        [bundle]="u.connection_bundle"
        [downloadOvpn]="downloadFn"
      />
      <form class="edit-group" (ngSubmit)="saveEdit()">
        <section class="card">
          <h3>رمز VPN</h3>
          <label>رمز جدید <input type="password" [(ngModel)]="editPassword" name="pw" /></label>
        </section>
        <section class="card">
          <h3>اتصال همزمان</h3>
          <label>حداکثر اتصال <input type="number" min="1" [(ngModel)]="editShared" name="su" /></label>
        </section>
        <section class="card">
          <h3>اطلاعات تماس</h3>
          <label>تماس <input [(ngModel)]="editContactInfo" name="cn" /></label>
        </section>
        <section class="card">
          <h3>یادداشت</h3>
          <label>یادداشت <textarea [(ngModel)]="editNotes" name="nt" rows="2"></textarea></label>
        </section>
        <section class="card">
          <h3>وضعیت روتر</h3>
          <label class="check">
            <input type="checkbox" [(ngModel)]="routerEnabled" name="ren" />
            فعال در روتر (User Manager)
          </label>
        </section>
        <div class="edit-actions">
          <button type="submit" class="btn primary" [disabled]="saving()">ذخیره تغییرات</button>
        </div>
      </form>
      <section class="card">
        <h3>تمدید / انتساب پروفایل</h3>
        <form (ngSubmit)="assign()">
          <p class="profile-name">پروفایل: <code dir="ltr">{{ defaultProfile }}</code></p>
          <label>مبلغ (اختیاری) <input type="number" [(ngModel)]="assignAmount" name="amt" /></label>
          <label>یادداشت <input [(ngModel)]="assignNote" name="an" /></label>
          <button type="submit" class="btn primary">تمدید</button>
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
    .edit-group {
      display: flex;
      flex-direction: column;
      gap: 0.75rem;
    }
    .edit-group .card h3 {
      margin: 0 0 0.5rem;
      font-size: 0.95rem;
    }
    .edit-actions {
      display: flex;
      justify-content: flex-end;
    }
    .check {
      display: flex;
      align-items: center;
      gap: 0.5rem;
    }
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
      const id = Number(p.get('id'));
      if (id) this.load(id);
    });
  }

  load(id: number): void {
    this.vpn
      .get(id)
      .pipe(catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }))
      .subscribe((u) => {
        if (!u) return;
        this.user.set(u);
        this.editShared = u.shared_users;
        this.editContactInfo = u.contact_info ?? '';
        this.editNotes = u.notes ?? '';
        this.routerEnabled = !u.disabled;
      });
  }

  saveEdit(): void {
    const u = this.user();
    if (!u) return;
    this.saving.set(true);
    const body: Record<string, unknown> = {
      shared_users: this.editShared,
      contact_info: this.editContactInfo,
      notes: this.editNotes,
      disabled: !this.routerEnabled,
    };
    if (this.editPassword) body['password'] = this.editPassword;
    this.vpn.patch(u.id, body).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        this.saving.set(false);
        return of(null);
      }),
    ).subscribe((res) => {
      this.saving.set(false);
      if (res) {
        this.user.set(res);
        this.toast.show('ذخیره شد');
      }
    });
  }

  assign(): void {
    const u = this.user();
    if (!u) return;
    this.vpn.assignProfile(u.id, {
      profile_name: this.defaultProfile,
      amount_paid: this.assignAmount ?? undefined,
      currency: this.assignAmount != null ? 'IRR' : undefined,
      note: this.assignNote || undefined,
    }).pipe(catchError((e) => {
      this.error.set(ApiClient.mapError(e));
      return of(null);
    })).subscribe((res) => {
      if (res) {
        this.user.set(res);
        this.toast.show('پروفایل انتساب شد');
      }
    });
  }

  removeProfile(profileRowId: string): void {
    const u = this.user();
    if (!u) return;
    this.vpn.removeProfile(u.id, profileRowId).subscribe({
      next: () => this.load(u.id),
      error: (e) => this.error.set(ApiClient.mapError(e)),
    });
  }

  deleteUser(): void {
    const u = this.user();
    if (!u) return;
    this.confirmDelete.set(false);
    this.vpn.delete(u.id).subscribe({
      next: () => void this.router.navigate(['/users']),
      error: (e: unknown) => this.error.set(ApiClient.mapError(e)),
    });
  }

  downloadOvpn(): void {
    const u = this.user();
    if (!u) return;
    this.vpn.downloadOvpn(u.id).subscribe({
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
