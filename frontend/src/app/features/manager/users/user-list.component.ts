import { Component, inject, OnInit, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { VpnListItem } from '../../../core/models';
import { VpnUserService } from '../../../core/services/vpn-user.service';
import { ProfileStateChipComponent } from '../../../shared/components/profile-state-chip/profile-state-chip.component';
import { primaryProfileState } from '../../../shared/utils/profile-active';
import { UI_MESSAGES } from '../../../core/i18n/messages';

@Component({
  selector: 'app-user-list',
  standalone: true,
  imports: [RouterLink, ProfileStateChipComponent],
  template: `
    <div class="toolbar">
      <h2 class="page-title">کاربران VPN</h2>
      <div class="toolbar-actions">
        <button type="button" class="btn" (click)="load(true)" [disabled]="loading()">بروزرسانی</button>
        <a routerLink="/users/new" class="btn primary">کاربر جدید</a>
      </div>
    </div>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    @if (!loading() && items().length === 0) {
      <div class="card empty">
        <p>{{ msg.emptyUsers }}</p>
        <a routerLink="/users/new" class="btn primary">ایجاد کاربر</a>
      </div>
    } @else {
      <div class="card table-wrap">
        <table>
          <thead>
            <tr>
              <th>نام</th>
              <th>فعال</th>
              <th>اتصال همزمان</th>
              <th>پروفایل</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            @for (u of items(); track u.mikrotik_name) {
              <tr [class.row-disabled]="u.disabled">
                <td>{{ u.mikrotik_name }}</td>
                <td>
                  <span class="router-status" [class.off]="u.disabled">
                    {{ u.disabled ? 'غیرفعال' : 'فعال' }}
                  </span>
                </td>
                <td>{{ u.shared_users }}</td>
                <td>
                  <app-profile-state-chip [state]="stateOf(u)" />
                </td>
                <td>
                  @if (u.id) {
                    <a [routerLink]="['/users', u.id]">جزئیات</a>
                  } @else {
                    <span class="muted">بدون متادیتا</span>
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
export class UserListComponent implements OnInit {
  private readonly vpn = inject(VpnUserService);
  readonly msg = UI_MESSAGES;

  readonly items = signal<VpnListItem[]>([]);
  readonly loading = signal(false);
  readonly error = signal('');

  ngOnInit(): void {
    this.load(false);
  }

  load(refresh: boolean): void {
    this.loading.set(true);
    this.error.set('');
    this.vpn
      .list(refresh)
      .pipe(
        catchError((e) => {
          this.error.set(ApiClient.mapError(e));
          return of({ items: [] });
        }),
      )
      .subscribe((res) => {
        this.items.set(res.items);
        this.loading.set(false);
      });
  }

  stateOf(u: VpnListItem): string {
    return primaryProfileState(u.profiles ?? []);
  }
}
