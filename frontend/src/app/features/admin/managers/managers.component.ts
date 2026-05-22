import { Component, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, of } from 'rxjs';
import { ApiClient } from '../../../core/api/api-client.service';
import { ManagerRow } from '../../../core/models';
import { AdminService } from '../../../core/services/vpn-user.service';
@Component({
  selector: 'app-managers',
  standalone: true,
  imports: [FormsModule],
  template: `
    <h2 class="page-title">مدیران</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <section class="card form">
      <h3>مدیر جدید</h3>
      <form (ngSubmit)="create()">
        <div class="grid">
          <label>نام کاربری <input [(ngModel)]="newUser.username" name="nu" required /></label>
          <label>رمز <input type="password" [(ngModel)]="newUser.password" name="np" required /></label>
          <label>نام نمایشی <input [(ngModel)]="newUser.display_name" name="nd" required /></label>
          <label>slug <input [(ngModel)]="newUser.slug" name="ns" required /></label>
          <label>سقف <input type="number" min="1" [(ngModel)]="newUser.quota" name="nq" /></label>
        </div>
        <button type="submit" class="btn primary">ایجاد</button>
      </form>
    </section>
    <div class="card table-wrap">
      <table>
        <thead>
          <tr>
            <th>نام</th>
            <th>نام کاربری</th>
            <th>slug</th>
            <th>سقف</th>
            <th>مصرف</th>
            <th>فعال</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          @for (m of items(); track m.id) {
            <tr [class.inactive]="!m.is_active">
              <td>{{ m.display_name }}</td>
              <td>
                @if (editId() === m.id) {
                  <input dir="ltr" [(ngModel)]="editUsername" [name]="'u' + m.id" required />
                } @else {
                  <span dir="ltr">{{ m.username }}</span>
                }
              </td>
              <td dir="ltr">{{ m.slug }}</td>
              <td>
                @if (editId() === m.id) {
                  <input type="number" min="1" [(ngModel)]="editQuota" [name]="'q' + m.id" />
                } @else {
                  {{ m.quota }}
                }
              </td>
              <td>{{ m.used_quota }}</td>
              <td>
                @if (editId() === m.id) {
                  <label class="inline-check">
                    <input type="checkbox" [(ngModel)]="editActive" [name]="'a' + m.id" />
                    فعال
                  </label>
                } @else {
                  {{ m.is_active ? 'بله' : 'خیر' }}
                }
              </td>
              <td class="actions">
                @if (editId() === m.id) {
                  <label class="pw-field">
                    رمز جدید
                    <input
                      type="password"
                      dir="ltr"
                      [(ngModel)]="editPassword"
                      [name]="'p' + m.id"
                      placeholder="خالی = بدون تغییر"
                      autocomplete="new-password"
                    />
                  </label>
                  <button type="button" class="link" (click)="saveEdit(m)">ذخیره</button>
                  <button type="button" class="link" (click)="editId.set(null)">انصراف</button>
                } @else {
                  <button type="button" class="link" (click)="startEdit(m)">ویرایش</button>
                }
              </td>
            </tr>
          }
        </tbody>
      </table>
    </div>
  `,
  styles: `
    .grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
      gap: 0.75rem;
    }
    tr.inactive {
      opacity: 0.55;
    }
    .inline-check {
      display: flex;
      align-items: center;
      gap: 0.35rem;
      margin: 0;
      font-size: 0.85rem;
    }
    .inline-check input {
      width: auto;
      margin: 0;
    }
    .actions {
      min-width: 12rem;
    }
    .pw-field {
      display: block;
      margin: 0 0 0.35rem;
      font-size: 0.8rem;
    }
    .pw-field input {
      width: 100%;
      margin-top: 0.2rem;
    }
  `,
})
export class ManagersComponent implements OnInit {
  private readonly admin = inject(AdminService);

  readonly items = signal<ManagerRow[]>([]);
  readonly error = signal('');
  readonly editId = signal<number | null>(null);
  editUsername = '';
  editPassword = '';
  editQuota = 10;
  editActive = true;

  newUser = { username: '', password: '', display_name: '', slug: '', quota: 10 };

  ngOnInit(): void {
    this.reload();
  }

  reload(): void {
    this.admin.listManagers().pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of({ items: [] });
      }),
    ).subscribe((res) => this.items.set(res.items));
  }

  create(): void {
    this.admin.createManager(this.newUser).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }),
    ).subscribe((res) => {
      if (res) {
        this.newUser = { username: '', password: '', display_name: '', slug: '', quota: 10 };
        this.reload();
      }
    });
  }

  startEdit(m: ManagerRow): void {
    this.editId.set(m.id);
    this.editUsername = m.username;
    this.editPassword = '';
    this.editQuota = m.quota;
    this.editActive = m.is_active;
  }

  saveEdit(m: ManagerRow): void {
    const body: {
      username: string;
      quota: number;
      is_active: boolean;
      password?: string;
    } = {
      username: this.editUsername.trim(),
      quota: this.editQuota,
      is_active: this.editActive,
    };
    const pw = this.editPassword.trim();
    if (pw) {
      body.password = pw;
    }
    this.admin.patchManager(m.id, body).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }),
    ).subscribe((res) => {
      if (res) {
        this.editId.set(null);
        this.reload();
      }
    });
  }
}
