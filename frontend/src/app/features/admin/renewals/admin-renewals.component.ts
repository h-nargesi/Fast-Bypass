import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ManagerRow, RenewalItem, RenewalsResponse } from '../../../core/models';
import { AdminService, RenewalsService } from '../../../core/services/vpn-user.service';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-admin-renewals',
  standalone: true,
  imports: [FormsModule, ProfileStateChipComponent, JalaliDatePipe],
  template: `
    <h2 class="page-title">دفتر تمدیدها</h2>
    <div class="filters card">
      <label>
        مدیر
        <select [(ngModel)]="managerKey" (ngModelChange)="load()" name="mgr">
          <option value="orphan">بدون مدیر (پیش‌فرض)</option>
          @for (m of managers(); track m.id) {
            <option [value]="'m:' + m.id">{{ m.display_name }}</option>
          }
        </select>
      </label>
      <label>
        تسویه
        <select [(ngModel)]="settled" (ngModelChange)="load()" name="stl">
          <option value="">تسویه‌نشده</option>
          <option value="settled">تسویه‌شده</option>
          <option value="all">همه</option>
        </select>
      </label>
    </div>
    @if (data(); as d) {
      <div class="summary card">
        <span>جمع اتصال (تسویه‌نشده): <strong>{{ d.summary.unsettled_shared_users_sum }}</strong></span>
        <span>کل: {{ d.summary.all_shared_users_sum }}</span>
      </div>
    }
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (!loading() && items().length === 0) {
      <div class="card empty"><p>{{ msg.emptyRenewals }}</p></div>
    } @else {
      <div class="card table-wrap">
        <table>
          <thead>
            <tr>
              <th>تاریخ تمدید</th>
              <th>کاربر</th>
              <th>اتصال</th>
              <th>پروفایل</th>
              <th>وضعیت</th>
              <th>اعتبار</th>
              <th>تسویه</th>
              <th>عمل</th>
            </tr>
          </thead>
          <tbody>
            @for (r of items(); track r.id) {
              <tr>
                <td>{{ r.renewed_at | jalaliDate }}</td>
                <td>{{ r.mikrotik_name }}</td>
                <td>{{ r.shared_users }}</td>
                <td>{{ r.profile_name }}</td>
                <td><app-profile-state-chip [state]="r.profile_state" /></td>
                <td>{{ r.mikrotik_end_time | jalaliDate: 'datetime' }}</td>
                <td>{{ r.is_settled ? '✓' : '✗' }}</td>
                <td>
                  @if (!r.is_settled && data()?.can_settle) {
                    <button type="button" class="link" (click)="settle(r)">تسویه تا اینجا</button>
                  } @else {
                      —
                    }
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
    }
  `,
})
export class AdminRenewalsComponent implements OnInit {
  private readonly renewals = inject(RenewalsService);
  private readonly admin = inject(AdminService);
  readonly msg = UI_MESSAGES;

  readonly managers = signal<ManagerRow[]>([]);
  readonly data = signal<RenewalsResponse | null>(null);
  readonly items = signal<RenewalItem[]>([]);
  readonly loading = signal(false);
  readonly error = signal('');

  managerKey = 'orphan';
  settled = '';

  ngOnInit(): void {
    this.admin.listManagers().subscribe((r) => this.managers.set(r.items));
    this.load();
  }

  load(): void {
    this.loading.set(true);
    const opts: { manager_id?: number; settled?: string } = { settled: this.settled };
    if (this.managerKey.startsWith('m:')) {
      opts.manager_id = Number(this.managerKey.slice(2));
    }
    this.renewals
      .adminList(opts)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of(null);
        }),
      )
      .subscribe((res) => {
        this.loading.set(false);
        if (!res) return;
        this.data.set(res);
        this.items.set(res.items);
      });
  }

  settle(r: RenewalItem): void {
    const manager_id =
      this.managerKey.startsWith('m:') ? Number(this.managerKey.slice(2)) : undefined;
    this.renewals
      .settleThrough(r.id, manager_id)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of(null);
        }),
      )
      .subscribe((res) => {
        if (res) this.load();
      });
  }
}
