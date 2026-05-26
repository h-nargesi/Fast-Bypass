import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { catchError, debounceTime, distinctUntilChanged, of, Subject } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { AdminVpnListItem, ManagerRow } from '../../../core/models';
import { AdminService, AdminVpnService } from '../../../core/services/vpn-user.service';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { primaryProfileState } from '../../../shared/utils/profile-active';
import { UI_MESSAGES } from '../../../core/i18n/messages';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-admin-user-list',
  standalone: true,
  imports: [FormsModule, RouterLink, ProfileStateChipComponent, ...MATERIAL_FORM],
  template: `
    <div class="toolbar">
      <h2 class="page-title">همه کاربران VPN</h2>
      <div class="toolbar-actions">
        <a routerLink="/admin/users/new" class="btn primary">کاربر جدید</a>
        <button type="button" class="btn" (click)="refresh()">بروزرسانی</button>
      </div>
    </div>
    <div class="filters card">
      <mat-form-field appearance="outline">
        <mat-label>مدیر</mat-label>
        <mat-select [(ngModel)]="filterMode" (ngModelChange)="onFilterChange()" name="fm">
          <mat-option value="orphan">بدون مدیر</mat-option>
          <mat-option value="all">همه (بدون فیلتر مالک)</mat-option>
          @for (m of managers(); track m.id) {
            <mat-option [value]="'m:' + m.id">{{ m.display_name }}</mat-option>
          }
        </mat-select>
      </mat-form-field>
      <mat-form-field appearance="outline" class="search-field">
        <mat-label>جستجو در نام / comment</mat-label>
        <input matInput [(ngModel)]="searchText" (ngModelChange)="onSearch($event)" name="q" autocomplete="off" />
        @if (searchText) {
          <button matSuffix mat-icon-button aria-label="پاک کردن" (click)="clearSearch()">✕</button>
        }
      </mat-form-field>
    </div>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <div class="card table-wrap">
      <table>
        <thead>
          <tr>
            <th>نام</th>
            <th>مدیر</th>
            <th>فعال</th>
            <th>تعداد اتصالات همزمان</th>
            <th>پروفایل</th>
            <th>comment</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          @for (u of items(); track u.mikrotik_name) {
            <tr
              [class.warn-row]="u.owner_mismatch && !u.disabled"
              [class.row-disabled]="u.disabled"
            >
              <td>{{ u.mikrotik_name }}</td>
              <td>
                @if (u.manager_display_name) {
                  {{ u.manager_display_name }}
                  @if (u.owner_mismatch) { <span class="warn-tag">⚠</span> }
                } @else {
                  <span class="orphan">{{ msg.orphanLabel }}</span>
                }
              </td>
              <td>
                <span class="router-status" [class.off]="u.disabled">
                  {{ u.disabled ? 'غیرفعال' : 'فعال' }}
                </span>
              </td>
              <td>{{ u.shared_users }}</td>
              <td><app-profile-state-chip [state]="stateOf(u)" /></td>
              <td dir="ltr" class="mono">{{ u.mikrotik_comment || '—' }}</td>
              <td>
                @if (u.id) {
                  <a [routerLink]="['/admin/users', u.id]">ویرایش</a>
                } @else {
                  <a [routerLink]="['/admin/users/by-name', u.mikrotik_name]">ویرایش</a>
                }
              </td>
            </tr>
          }
        </tbody>
      </table>
      @if (total() > pageSize) {
        <div class="pagination">
          <button (click)="goTo(page() - 1)" [disabled]="page() <= 1">&#8249; قبلی</button>
          <span class="page-info">صفحه {{ page() }} از {{ totalPages() }}</span>
          <button (click)="goTo(page() + 1)" [disabled]="page() >= totalPages()">بعدی &#8250;</button>
          <span class="muted">({{ total() }} کاربر)</span>
        </div>
      }
    </div>
  `,
  styles: `
    .toolbar-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
    .orphan { color: #666; font-size: 0.85rem; }
    .warn-row { background: #fffde7; }
    tr.warn-row.row-disabled td { background: #eceff1; }
    .warn-tag { color: #f57f17; }
    .mono { font-family: ui-monospace, monospace; font-size: 0.82rem; }
    .search-field { min-width: 18rem; }
  `,
})
export class AdminUserListComponent implements OnInit {
  private readonly vpn = inject(AdminVpnService);
  private readonly admin = inject(AdminService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly msg = UI_MESSAGES;
  readonly items = signal<AdminVpnListItem[]>([]);
  readonly managers = signal<ManagerRow[]>([]);
  readonly error = signal('');
  readonly total = signal(0);
  readonly page = signal(1);
  readonly pageSize = 20;

  filterMode = 'all';
  searchText = '';
  private readonly search$ = new Subject<string>();

  ngOnInit(): void {
    this.admin.listManagers().subscribe((r) => this.managers.set(r.items));
    this.search$.pipe(debounceTime(300), distinctUntilChanged()).subscribe(() => {
      this.page.set(1);
      this.load(false);
    });
    this.route.queryParams.subscribe((q) => {
      if (q['orphan'] === 'true') {
        this.filterMode = 'orphan';
      } else if (q['manager_id']) {
        this.filterMode = 'm:' + q['manager_id'];
      } else {
        this.filterMode = 'all';
      }
      this.page.set(1);
      this.load(false);
    });
  }

  onSearch(val: string): void {
    this.searchText = val;
    this.search$.next(val);
  }

  clearSearch(): void {
    this.searchText = '';
    this.page.set(1);
    this.load(false);
  }

  onFilterChange(): void {
    const qp: Record<string, string> = {};
    if (this.filterMode === 'orphan') {
      qp['orphan'] = 'true';
    } else if (this.filterMode.startsWith('m:')) {
      qp['manager_id'] = this.filterMode.slice(2);
    }
    void this.router.navigate([], { queryParams: qp });
    this.page.set(1);
    this.load(false);
  }

  refresh(): void {
    this.page.set(1);
    this.load(true);
  }

  goTo(p: number): void {
    this.page.set(p);
    this.load(false);
  }

  totalPages(): number {
    return Math.max(1, Math.ceil(this.total() / this.pageSize));
  }

  load(doRefresh: boolean): void {
    const opts: { refresh?: boolean; manager_id?: number; orphan?: boolean; q?: string; page: number; page_size: number } = {
      refresh: doRefresh, page: this.page(), page_size: this.pageSize,
      q: this.searchText || undefined,
    };
    if (this.filterMode === 'orphan') {
      opts.orphan = true;
    } else if (this.filterMode.startsWith('m:')) {
      opts.manager_id = Number(this.filterMode.slice(2));
    }
    this.vpn.list(opts).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of({ items: [], page: 1, page_size: this.pageSize, total: 0 });
      }),
    ).subscribe((res) => {
      this.items.set(res.items);
      this.total.set(res.total);
    });
  }

  stateOf(u: AdminVpnListItem): string {
    return primaryProfileState(u.profiles ?? []);
  }
}
