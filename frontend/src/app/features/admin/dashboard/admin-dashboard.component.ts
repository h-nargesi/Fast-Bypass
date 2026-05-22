import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, forkJoin, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { AdminService, AdminVpnService } from '../../../core/services/vpn-user.service';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  imports: [RouterLink],
  template: `
    <h2 class="page-title">داشبورد ادمین</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <div class="stats">
      <div class="card stat">
        <span class="label">تعداد مدیران</span>
        <span class="value">{{ managerCount() }}</span>
        <a routerLink="/admin/managers">مدیریت مدیران</a>
      </div>
      <div class="card stat">
        <span class="label">کاربران بدون مدیر (orphan)</span>
        <span class="value">{{ orphanCount() }}</span>
        <a routerLink="/admin/users" [queryParams]="{ orphan: true }">مشاهده</a>
      </div>
    </div>
    <div class="links">
      <a routerLink="/admin/users" class="btn">همه کاربران VPN</a>
      <a routerLink="/admin/renewals" class="btn">دفتر تمدید</a>
    </div>
  `,
  styles: `
    .stats {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
      gap: 1rem;
    }
    .stat .value {
      display: block;
      font-size: 2rem;
      font-weight: 700;
      margin: 0.35rem 0;
    }
    .stat a {
      font-size: 0.88rem;
      color: #1565c0;
    }
    .links {
      margin-top: 1.25rem;
      display: flex;
      gap: 0.5rem;
      flex-wrap: wrap;
    }
  `,
})
export class AdminDashboardComponent implements OnInit {
  private readonly admin = inject(AdminService);
  private readonly vpn = inject(AdminVpnService);

  readonly managerCount = signal(0);
  readonly orphanCount = signal(0);
  readonly error = signal('');

  ngOnInit(): void {
    forkJoin({
      managers: this.admin.listManagers(),
      orphans: this.vpn.list({ orphan: true }),
    })
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of(null);
        }),
      )
      .subscribe((data) => {
        if (!data) return;
        this.managerCount.set(data.managers.items.length);
        this.orphanCount.set(data.orphans.items.length);
      });
  }
}
