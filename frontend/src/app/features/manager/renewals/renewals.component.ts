import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { RenewalItem, RenewalsResponse } from '../../../core/models';
import { RenewalsService } from '../../../core/services/vpn-user.service';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-manager-renewals',
  standalone: true,
  imports: [FormsModule, ProfileStateChipComponent, JalaliDatePipe],
  template: `
    <h2 class="page-title">تمدیدهای من</h2>
    @if (data(); as d) {
      <div class="summary card">
        <span>جمع اتصال (تسویه‌نشده): <strong>{{ d.summary.unsettled_shared_users_sum }}</strong></span>
        @if (d.scope.manager_display_name) {
          <span>مدیر: {{ d.scope.manager_display_name }}</span>
        }
      </div>
      <div class="filters">
        <select [(ngModel)]="settled" (ngModelChange)="load()" name="settled">
          <option value="">تسویه‌نشده</option>
          <option value="settled">تسویه‌شده</option>
          <option value="all">همه</option>
        </select>
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
              </tr>
            }
          </tbody>
        </table>
      </div>
    }
  `,
})
export class ManagerRenewalsComponent implements OnInit {
  private readonly renewals = inject(RenewalsService);
  readonly msg = UI_MESSAGES;

  settled = '';
  readonly data = signal<RenewalsResponse | null>(null);
  readonly items = signal<RenewalItem[]>([]);
  readonly loading = signal(false);
  readonly error = signal('');

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.renewals
      .managerList(this.settled)
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
}
