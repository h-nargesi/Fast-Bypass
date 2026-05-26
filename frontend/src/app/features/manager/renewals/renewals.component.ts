import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, debounceTime, distinctUntilChanged, of, Subject } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { RenewalItem, RenewalsResponse } from '../../../core/models';
import { RenewalsService } from '../../../core/services/vpn-user.service';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { UI_MESSAGES } from '../../../core/i18n/messages';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-manager-renewals',
  standalone: true,
  imports: [FormsModule, ProfileStateChipComponent, JalaliDatePipe, ...MATERIAL_FORM],
  template: `
    <h2 class="page-title">تمدیدهای من</h2>
    @if (data(); as d) {
      <div class="summary card">
        <span>جمع اتصال (تسویه‌نشده): <strong>{{ d.summary.unsettled_shared_users_sum }}</strong></span>
        @if (d.scope.manager_display_name) {
          <span>مدیر: {{ d.scope.manager_display_name }}</span>
        }
      </div>
    }
    <div class="filters card">
      <mat-form-field appearance="outline">
        <mat-label>وضعیت تسویه</mat-label>
        <mat-select [(ngModel)]="settled" (ngModelChange)="resetAndLoad()" name="settled">
          <mat-option value="">تسویه‌نشده</mat-option>
          <mat-option value="settled">تسویه‌شده</mat-option>
          <mat-option value="all">همه</mat-option>
        </mat-select>
      </mat-form-field>
      <mat-form-field appearance="outline" class="search-field">
        <mat-label>جستجو در نام کاربر</mat-label>
        <input matInput [(ngModel)]="searchText" (ngModelChange)="onSearch($event)" name="q" autocomplete="off" />
        @if (searchText) {
          <button matSuffix mat-icon-button aria-label="پاک کردن" (click)="clearSearch()">✕</button>
        }
      </mat-form-field>
    </div>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (!loading() && items().length === 0) {
      <div class="card empty"><p>{{ searchText ? 'تمدیدی با این نام یافت نشد' : msg.emptyRenewals }}</p></div>
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
        @if (total() > pageSize) {
          <div class="pagination">
            <button (click)="goTo(page() - 1)" [disabled]="page() <= 1">&#8249; قبلی</button>
            <span class="page-info">صفحه {{ page() }} از {{ totalPages() }}</span>
            <button (click)="goTo(page() + 1)" [disabled]="page() >= totalPages()">بعدی &#8250;</button>
            <span class="muted">({{ total() }} تمدید)</span>
          </div>
        }
      </div>
    }
  `,
  styles: `
    .search-field { min-width: 18rem; }
  `,
})
export class ManagerRenewalsComponent implements OnInit {
  private readonly renewals = inject(RenewalsService);
  readonly msg = UI_MESSAGES;

  settled = '';
  searchText = '';
  readonly data = signal<RenewalsResponse | null>(null);
  readonly items = signal<RenewalItem[]>([]);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly total = signal(0);
  readonly page = signal(1);
  readonly pageSize = 20;

  private readonly search$ = new Subject<string>();

  ngOnInit(): void {
    this.search$.pipe(debounceTime(300), distinctUntilChanged()).subscribe(() => {
      this.page.set(1);
      this.load();
    });
    this.load();
  }

  onSearch(val: string): void {
    this.searchText = val;
    this.search$.next(val);
  }

  clearSearch(): void {
    this.searchText = '';
    this.page.set(1);
    this.load();
  }

  resetAndLoad(): void {
    this.page.set(1);
    this.load();
  }

  goTo(p: number): void {
    this.page.set(p);
    this.load();
  }

  totalPages(): number {
    return Math.max(1, Math.ceil(this.total() / this.pageSize));
  }

  load(): void {
    this.loading.set(true);
    this.renewals
      .managerList({ settled: this.settled, page: this.page(), page_size: this.pageSize, q: this.searchText || undefined })
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
        this.total.set(res.total);
      });
  }
}
