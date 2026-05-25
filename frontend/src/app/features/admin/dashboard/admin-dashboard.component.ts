import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { AdminStatsResponse } from '../../../core/models';
import { AdminService } from '../../../core/services/vpn-user.service';

@Component({
  selector: 'app-admin-dashboard',
  standalone: true,
  imports: [RouterLink],
  template: `
    <h2 class="page-title">داشبورد ادمین</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (stats(); as s) {
      <div class="stats">
        <div class="card stat">
          <span class="label">تعداد مدیران</span>
          <span class="value">{{ s.manager_count }}</span>
          <a routerLink="/admin/managers">مدیریت مدیران</a>
        </div>
        <div class="card stat">
          <span class="label">کاربران فعال</span>
          <span class="value">{{ s.totals.vpn_users }}</span>
          <span class="hint">غیرغیرفعال در روتر + پروفایل فعال</span>
          <a routerLink="/admin/users">همه کاربران</a>
        </div>
        <div class="card stat">
          <span class="label">جمع اتصال همزمان (فعال)</span>
          <span class="value">{{ s.totals.connections }}</span>
          <span class="hint">جمع shared-users کاربران با پروفایل فعال</span>
        </div>
        <div class="card stat">
          <span class="label">بدون مدیر (orphan)</span>
          <span class="value">{{ s.orphan.connections }}</span>
          <span class="hint">{{ s.orphan.vpn_users }} کاربر فعال</span>
          <a routerLink="/admin/users" [queryParams]="{ orphan: true }">مشاهده</a>
        </div>
      </div>
      @if (s.by_manager.length) {
        <section class="card breakdown">
          <h3>به تفکیک مدیر</h3>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>مدیر</th>
                  <th>کاربران فعال</th>
                  <th>اتصال همزمان</th>
                  <th>سقف</th>
                </tr>
              </thead>
              <tbody>
                @for (m of s.by_manager; track m.manager_id) {
                  <tr>
                    <td>
                      <a [routerLink]="['/admin/users']" [queryParams]="{ manager_id: m.manager_id }">
                        {{ m.display_name }}
                      </a>
                    </td>
                    <td>{{ m.vpn_users }}</td>
                    <td>{{ m.connections }}</td>
                    <td>{{ m.connections }} / {{ m.quota }}</td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        </section>
      }
    }
    <div class="links">
      <button type="button" class="btn" (click)="load(true)">بروزرسانی</button>
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
    .stat .hint {
      display: block;
      font-size: 0.82rem;
      color: #666;
      margin-bottom: 0.25rem;
    }
    .stat a {
      font-size: 0.88rem;
      color: #1565c0;
    }
    .breakdown {
      margin-top: 1.25rem;
    }
    .breakdown h3 {
      margin: 0 0 0.75rem;
      font-size: 1rem;
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

  readonly stats = signal<AdminStatsResponse | null>(null);
  readonly error = signal('');

  ngOnInit(): void {
    this.load(false);
  }

  load(refresh: boolean): void {
    this.admin
      .getStats(refresh)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of(null);
        }),
      )
      .subscribe((data) => {
        if (data) {
          this.stats.set(data);
          this.error.set('');
        }
      });
  }
}
