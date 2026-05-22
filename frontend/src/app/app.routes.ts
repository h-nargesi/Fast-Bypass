import { Routes } from '@angular/router';
import { adminGuard, authGuard, guestGuard, managerGuard } from './core/auth/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/auth/login/login.component').then((m) => m.LoginComponent),
  },
  {
    path: '',
    canActivate: [authGuard, managerGuard],
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./features/manager/dashboard/dashboard.component').then((m) => m.DashboardComponent),
      },
      {
        path: 'users',
        loadComponent: () =>
          import('./features/manager/users/user-list.component').then((m) => m.UserListComponent),
      },
      {
        path: 'users/new',
        loadComponent: () =>
          import('./features/manager/users/user-form.component').then((m) => m.UserFormComponent),
      },
      {
        path: 'users/:id',
        loadComponent: () =>
          import('./features/manager/users/user-detail.component').then((m) => m.UserDetailComponent),
      },
      {
        path: 'renewals',
        loadComponent: () =>
          import('./features/manager/renewals/renewals.component').then((m) => m.ManagerRenewalsComponent),
      },
      {
        path: 'settings',
        loadComponent: () =>
          import('./features/settings/settings.component').then((m) => m.SettingsComponent),
      },
    ],
  },
  {
    path: 'admin',
    canActivate: [authGuard, adminGuard],
    children: [
      {
        path: '',
        loadComponent: () =>
          import('./features/admin/dashboard/admin-dashboard.component').then(
            (m) => m.AdminDashboardComponent,
          ),
      },
      {
        path: 'managers',
        loadComponent: () =>
          import('./features/admin/managers/managers.component').then((m) => m.ManagersComponent),
      },
      {
        path: 'users',
        loadComponent: () =>
          import('./features/admin/users/admin-user-list.component').then((m) => m.AdminUserListComponent),
      },
      {
        path: 'users/new',
        loadComponent: () =>
          import('./features/admin/users/admin-user-form.component').then((m) => m.AdminUserFormComponent),
      },
      {
        path: 'users/:id',
        loadComponent: () =>
          import('./features/admin/users/admin-user-detail.component').then(
            (m) => m.AdminUserDetailComponent,
          ),
      },
      {
        path: 'renewals',
        loadComponent: () =>
          import('./features/admin/renewals/admin-renewals.component').then(
            (m) => m.AdminRenewalsComponent,
          ),
      },
      {
        path: 'settings',
        loadComponent: () =>
          import('./features/settings/settings.component').then((m) => m.SettingsComponent),
      },
    ],
  },
  { path: '**', redirectTo: '' },
];
