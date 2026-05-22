import { Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter, map, startWith } from 'rxjs';
import { UI_MESSAGES } from './core/i18n/messages';
import { AuthService } from './core/auth/auth.service';
import { ToastComponent } from './shared/components/toast/toast.component';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet, RouterLink, RouterLinkActive, ToastComponent],
  template: `
    <div class="shell">
      <header>
        <h1>{{ title }}</h1>
        @if (showNav()) {
          <nav>
            @if (auth.isManager()) {
              <a routerLink="/" routerLinkActive="active" [routerLinkActiveOptions]="{ exact: true }">داشبورد</a>
              <a routerLink="/users" routerLinkActive="active">کاربران</a>
              <a routerLink="/renewals" routerLinkActive="active">تمدیدها</a>
              <a routerLink="/settings" routerLinkActive="active">تنظیمات</a>
            }
            @if (auth.isAdmin()) {
              <a routerLink="/admin" routerLinkActive="active" [routerLinkActiveOptions]="{ exact: true }">ادمین</a>
              <a routerLink="/admin/managers" routerLinkActive="active">مدیران</a>
              <a routerLink="/admin/users" routerLinkActive="active">کاربران VPN</a>
              <a routerLink="/admin/renewals" routerLinkActive="active">تمدیدها</a>
              <a routerLink="/admin/settings" routerLinkActive="active">تنظیمات</a>
            }
            <button type="button" class="logout" (click)="auth.logout()">{{ msg.logout }}</button>
          </nav>
        }
      </header>
      <main [class.full]="!showNav()">
        <router-outlet />
      </main>
      <app-toast />
    </div>
  `,
  styles: `
    .shell {
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }
    header {
      background: #1565c0;
      color: #fff;
      padding: 0.65rem 1.25rem;
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 0.75rem 1.5rem;
    }
    h1 {
      margin: 0;
      font-size: 1.15rem;
      font-weight: 600;
    }
    nav {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 0.35rem 0.85rem;
      flex: 1;
    }
    nav a {
      color: rgba(255, 255, 255, 0.92);
      text-decoration: none;
      font-size: 0.9rem;
      padding: 0.2rem 0.35rem;
      border-radius: 4px;
    }
    nav a.active {
      background: rgba(255, 255, 255, 0.2);
    }
    .logout {
      margin-right: auto;
      background: rgba(255, 255, 255, 0.15);
      border: 1px solid rgba(255, 255, 255, 0.35);
      color: #fff;
      padding: 0.3rem 0.75rem;
      border-radius: 6px;
      cursor: pointer;
      font: inherit;
      font-size: 0.85rem;
    }
    main {
      flex: 1;
      padding: 1rem 1.25rem 2rem;
      max-width: 1100px;
      width: 100%;
      margin: 0 auto;
      box-sizing: border-box;
    }
    main.full {
      max-width: none;
    }
  `,
})
export class App {
  readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  readonly title = UI_MESSAGES.appTitle;
  readonly msg = UI_MESSAGES;

  private readonly currentUrl = toSignal(
    this.router.events.pipe(
      filter((e): e is NavigationEnd => e instanceof NavigationEnd),
      map(() => this.router.url),
      startWith(this.router.url),
    ),
    { initialValue: this.router.url },
  );

  readonly showNav = computed(
    () => this.auth.loggedIn() && !this.currentUrl().startsWith('/login'),
  );
}
