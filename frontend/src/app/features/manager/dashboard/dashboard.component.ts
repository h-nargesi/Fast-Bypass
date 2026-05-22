import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, forkJoin, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ProfileService } from '../../../core/auth/auth.service';
import { ManagerProfile } from '../../../core/models';
import { VpnUserService } from '../../../core/services/vpn-user.service';
import { QuotaBadgeComponent } from '../../../shared/components/quota-badge/quota-badge.component';
import { isProfileActive } from '../../../shared/utils/profile-active';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [RouterLink, QuotaBadgeComponent],
  template: `
    <h2 class="page-title">داشبورد</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (profile(); as p) {
      <p class="welcome">سلام، {{ p.display_name || p.username }}</p>
      @if (quota(); as q) {
        <app-quota-badge [quota]="q.quota" [used]="q.used" [available]="q.available" />
        @if (q.available <= 0) {
          <p class="banner warn">سقف تعداد کاربران (اتصال همزمان) پر شده است.</p>
        }
      }
      <div class="stat card">
        <span class="stat-label">کاربران با پروفایل فعال</span>
        <span class="stat-value">{{ activeCount() }}</span>
      </div>
      <div class="actions">
        <a routerLink="/users" class="btn">لیست کاربران</a>
        <a routerLink="/users/new" class="btn primary">کاربر جدید</a>
      </div>
    }
  `,
  styles: `
    .welcome {
      margin: 0 0 1rem;
      color: #444;
    }
    .stat {
      margin-top: 1rem;
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
    .stat-value {
      font-size: 1.5rem;
      font-weight: 700;
    }
    .actions {
      margin-top: 1.25rem;
      display: flex;
      gap: 0.5rem;
      flex-wrap: wrap;
    }
  `,
})
export class DashboardComponent implements OnInit {
  private readonly profileSvc = inject(ProfileService);
  private readonly vpn = inject(VpnUserService);

  readonly profile = signal<ManagerProfile | null>(null);
  readonly quota = signal<{ quota: number; used: number; available: number } | null>(null);
  readonly activeCount = signal(0);
  readonly error = signal('');

  ngOnInit(): void {
    forkJoin({
      me: this.profileSvc.getMe(),
      quota: this.profileSvc.getQuota(),
      users: this.vpn.list(),
    })
      .pipe(catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }))
      .subscribe((data) => {
        if (!data) return;
        this.profile.set(data.me as ManagerProfile);
        this.quota.set(data.quota);
        const n = data.users.items.filter((u) => u.profiles?.some(isProfileActive)).length;
        this.activeCount.set(n);
      });
  }
}
