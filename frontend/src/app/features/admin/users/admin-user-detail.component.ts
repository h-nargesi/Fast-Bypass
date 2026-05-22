import { Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { VpnUserDetail } from '../../../core/models';
import { AdminVpnService } from '../../../core/services/vpn-user.service';
import { ConnectionBundleComponent } from '../../../shared/components/connection-bundle/connection-bundle.component';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { JalaliDatePipe } from '../../../shared/pipes/jalali-date.pipe';
import { formatRial } from '../../../core/format/currency';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-admin-user-detail',
  standalone: true,
  imports: [RouterLink, ConnectionBundleComponent, ProfileStateChipComponent, JalaliDatePipe],
  template: `
    <a routerLink="/admin/users" class="back">← بازگشت</a>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (user(); as u) {
      <h2 class="page-title">{{ u.mikrotik_name }}</h2>
      <section class="card owner">
        <h3>مالکیت</h3>
        <dl>
          <dt>مدیر</dt>
          <dd>
            @if (u.manager_display_name) {
              {{ u.manager_display_name }} ({{ u.manager_username }})
            } @else {
              {{ msg.orphanLabel }}
            }
          </dd>
          <dt>comment روتر</dt>
          <dd dir="ltr">{{ u.mikrotik_comment || '—' }}</dd>
          @if (u.owner_mismatch) {
            <p class="banner warn">ناهماهنگی مالکیت — مالک واقعی از نام/comment مشخص است.</p>
          }
        </dl>
        <p class="hint">ویرایش VPN از API ادمین هنوز فعال نیست؛ نمایش اطلاعات و اتصال مشتری.</p>
      </section>
      <app-connection-bundle [bundle]="u.connection_bundle" [allowOvpn]="false" />
      <section class="card">
        <h3>پروفایل‌ها</h3>
        <table>
          <thead><tr><th>پروفایل</th><th>وضعیت</th><th>اعتبار</th></tr></thead>
          <tbody>
            @for (p of u.profiles; track p.id) {
              <tr>
                <td>{{ p.profile }}</td>
                <td><app-profile-state-chip [state]="p.state" /></td>
                <td>{{ p.end_time | jalaliDate: 'datetime' }}</td>
              </tr>
            }
          </tbody>
        </table>
      </section>
      <section class="card">
        <h3>تاریخچه</h3>
        <table>
          <thead><tr><th>تاریخ</th><th>پروفایل</th><th>مبلغ</th></tr></thead>
          <tbody>
            @for (a of u.activations; track a.id) {
              <tr>
                <td>{{ a.created_at | jalaliDate }}</td>
                <td>{{ a.profile_name }}</td>
                <td>{{ formatAmount(a.amount_paid) }}</td>
              </tr>
            }
          </tbody>
        </table>
      </section>
    }
  `,
  styles: `
    .owner dl {
      display: grid;
      grid-template-columns: 7rem 1fr;
      gap: 0.35rem;
    }
    dt {
      color: #666;
      margin: 0;
    }
    dd {
      margin: 0;
    }
    .hint {
      font-size: 0.85rem;
      color: #666;
      margin-top: 0.75rem;
    }
  `,
})
export class AdminUserDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly vpn = inject(AdminVpnService);

  readonly msg = UI_MESSAGES;
  readonly user = signal<VpnUserDetail | null>(null);
  readonly error = signal('');
  readonly formatAmount = formatRial;

  ngOnInit(): void {
    this.route.paramMap.subscribe((p) => {
      const id = Number(p.get('id'));
      if (id) {
        this.vpn.get(id).pipe(
          catchError((e) => {
            this.error.set(ApiClient.mapError(e));
            return of(null);
          }),
        ).subscribe((u) => this.user.set(u));
      }
    });
  }
}
