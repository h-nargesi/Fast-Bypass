import { Component, computed, inject, OnInit, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { catchError, finalize, of } from 'rxjs';
import { MatButtonModule } from '@angular/material/button';
import { ApiClient } from '../../../core/api/api-client.service';
import { ManagerRow } from '../../../core/models';
import { AdminService } from '../../../core/services/vpn-user.service';
import { ConfirmDialogComponent } from '../../../shared/components/confirm-dialog/confirm-dialog.component';
import { MATERIAL_FORM } from '../../../shared/ui/material-form';

@Component({
  selector: 'app-managers',
  standalone: true,
  imports: [FormsModule, MatButtonModule, ConfirmDialogComponent, ...MATERIAL_FORM],
  template: `
    <h2 class="page-title">مدیران</h2>
    @if (error()) {
      <p class="banner err">{{ error() }}</p>
    }
    <section class="card">
      <h3>مدیر جدید</h3>
      <form class="form-grid" (ngSubmit)="create()">
        <mat-form-field appearance="outline">
          <mat-label>نام کاربری</mat-label>
          <input matInput class="ltr-input" [(ngModel)]="newUser.username" name="nu" required [disabled]="creating()" />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>رمز</mat-label>
          <input matInput type="password" class="ltr-input" [(ngModel)]="newUser.password" name="np" required [disabled]="creating()" />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>نام نمایشی</mat-label>
          <input matInput [(ngModel)]="newUser.display_name" name="nd" required [disabled]="creating()" />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>slug</mat-label>
          <input matInput class="ltr-input" [(ngModel)]="newUser.slug" name="ns" required [disabled]="creating()" />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>سقف</mat-label>
          <input matInput type="number" min="1" [(ngModel)]="newUser.quota" name="nq" [disabled]="creating()" />
        </mat-form-field>
        <mat-form-field appearance="outline">
          <mat-label>عنوان گواهی (اختیاری)</mat-label>
          <input matInput class="ltr-input" [(ngModel)]="newUser.cert_title" name="nct" placeholder="manager-cert" [disabled]="creating()" />
        </mat-form-field>
        <div class="form-grid-submit">
          <button type="submit" mat-flat-button color="primary" [disabled]="creating()">
            {{ createButtonLabel() }}
          </button>
        </div>
      </form>
    </section>
    <div class="card table-wrap">
      <table>
        <thead>
          <tr>
            <th>نام</th>
            <th>نام کاربری</th>
            <th>slug</th>
            <th>عنوان گواهی</th>
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
                  <mat-form-field appearance="outline" class="form-inline-field">
                    <input matInput class="ltr-input" [(ngModel)]="editUsername" [name]="'u' + m.id" required [disabled]="savingEdit()" />
                  </mat-form-field>
                } @else {
                  <span dir="ltr">{{ m.username }}</span>
                }
              </td>
              <td dir="ltr">{{ m.slug }}</td>
              <td>
                @if (editId() === m.id) {
                  <mat-form-field appearance="outline" class="form-inline-field">
                    <input
                      matInput
                      class="ltr-input"
                      [(ngModel)]="editCertTitle"
                      [name]="'ct' + m.id"
                      placeholder="خالی = حذف"
                      [disabled]="savingEdit()"
                    />
                  </mat-form-field>
                } @else {
                  <span dir="ltr">{{ m.cert_title || '—' }}</span>
                }
              </td>
              <td>
                @if (editId() === m.id) {
                  <mat-form-field appearance="outline" class="form-inline-field">
                    <input matInput type="number" min="1" [(ngModel)]="editQuota" [name]="'q' + m.id" [disabled]="savingEdit()" />
                  </mat-form-field>
                } @else {
                  {{ m.quota }}
                }
              </td>
              <td>{{ m.used_quota }}</td>
              <td>
                @if (editId() === m.id) {
                  <mat-checkbox [(ngModel)]="editActive" [name]="'a' + m.id" [disabled]="savingEdit()">فعال</mat-checkbox>
                } @else {
                  {{ m.is_active ? 'بله' : 'خیر' }}
                }
              </td>
              <td class="actions">
                @if (editId() === m.id) {
                  <mat-form-field appearance="outline" class="form-inline-field">
                    <mat-label>رمز جدید</mat-label>
                    <input
                      matInput
                      type="password"
                      class="ltr-input"
                      [(ngModel)]="editPassword"
                      [name]="'p' + m.id"
                      placeholder="خالی = بدون تغییر"
                      autocomplete="new-password"
                      [disabled]="savingEdit()"
                    />
                  </mat-form-field>
                  <button type="button" class="link" (click)="saveEdit(m)" [disabled]="savingEdit()">
                    {{ savingEdit() ? 'در حال ذخیره…' : 'ذخیره' }}
                  </button>
                  <button type="button" class="link" (click)="cancelEdit()" [disabled]="savingEdit()">انصراف</button>
                } @else {
                  <button type="button" class="link" (click)="startEdit(m)" [disabled]="savingEdit() || creating()">ویرایش</button>
                }
              </td>
            </tr>
          }
        </tbody>
      </table>
    </div>
    <app-confirm-dialog
      [open]="confirmCertRegenerate()"
      title="صدور گواهی جدید"
      message="عنوان گواهی این مدیر تغییر می‌کند. گواهی جدید روی روتر ساخته می‌شود و رمز کلید قبلی دیگر معتبر نیست. ادامه می‌دهید؟"
      confirmLabel="صدور گواهی جدید"
      (confirmed)="onConfirmCertRegenerate()"
      (cancelled)="onCancelCertRegenerate()"
    />
  `,
  styles: `
    .form-grid-submit {
      display: flex;
      align-items: center;
      grid-column: 1 / -1;
    }
    tr.inactive {
      opacity: 0.55;
    }
    .actions {
      min-width: 12rem;
    }
    .link:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  `,
})
export class ManagersComponent implements OnInit {
  private readonly admin = inject(AdminService);

  readonly items = signal<ManagerRow[]>([]);
  readonly error = signal('');
  readonly editId = signal<number | null>(null);
  readonly creating = signal(false);
  readonly savingEdit = signal(false);
  readonly confirmCertRegenerate = signal(false);

  readonly createButtonLabel = computed(() => {
    if (!this.creating()) {
      return 'ایجاد';
    }
    return this.newUser.cert_title.trim() ? 'در حال ایجاد و ساخت گواهی…' : 'در حال ایجاد…';
  });

  editUsername = '';
  editPassword = '';
  editCertTitle = '';
  editQuota = 10;
  editActive = true;
  private editCertTitleInitial = '';
  private pendingSaveManager: ManagerRow | null = null;

  newUser = { username: '', password: '', display_name: '', slug: '', quota: 10, cert_title: '' };

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
    this.creating.set(true);
    this.error.set('');
    const { cert_title, ...base } = this.newUser;
    const ct = cert_title.trim();
    const payload = ct ? { ...base, cert_title: ct } : base;
    this.admin.createManager(payload).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }),
      finalize(() => this.creating.set(false)),
    ).subscribe((res) => {
      if (res) {
        this.newUser = { username: '', password: '', display_name: '', slug: '', quota: 10, cert_title: '' };
        this.reload();
      }
    });
  }

  startEdit(m: ManagerRow): void {
    this.editId.set(m.id);
    this.editUsername = m.username;
    this.editPassword = '';
    this.editCertTitle = m.cert_title ?? '';
    this.editCertTitleInitial = this.editCertTitle.trim();
    this.editQuota = m.quota;
    this.editActive = m.is_active;
  }

  cancelEdit(): void {
    this.confirmCertRegenerate.set(false);
    this.pendingSaveManager = null;
    this.editId.set(null);
  }

  saveEdit(m: ManagerRow): void {
    if (this.needsCertRegenerateConfirm()) {
      this.pendingSaveManager = m;
      this.confirmCertRegenerate.set(true);
      return;
    }
    this.performSaveEdit(m);
  }

  onConfirmCertRegenerate(): void {
    this.confirmCertRegenerate.set(false);
    const m = this.pendingSaveManager;
    this.pendingSaveManager = null;
    if (m) {
      this.performSaveEdit(m);
    }
  }

  onCancelCertRegenerate(): void {
    this.confirmCertRegenerate.set(false);
    this.pendingSaveManager = null;
  }

  private needsCertRegenerateConfirm(): boolean {
    const old = this.editCertTitleInitial;
    const neu = this.editCertTitle.trim();
    return old !== '' && neu !== '' && neu !== old;
  }

  private performSaveEdit(m: ManagerRow): void {
    this.savingEdit.set(true);
    this.error.set('');
    const body: {
      username: string;
      quota: number;
      is_active: boolean;
      password?: string;
      cert_title?: string | null;
    } = {
      username: this.editUsername.trim(),
      quota: this.editQuota,
      is_active: this.editActive,
    };
    const pw = this.editPassword.trim();
    if (pw) {
      body.password = pw;
    }
    const ct = this.editCertTitle.trim();
    if (ct !== this.editCertTitleInitial) {
      body.cert_title = ct || null;
    }
    this.admin.patchManager(m.id, body).pipe(
      catchError((e) => {
        this.error.set(ApiClient.mapError(e));
        return of(null);
      }),
      finalize(() => this.savingEdit.set(false)),
    ).subscribe((res) => {
      if (res) {
        this.editId.set(null);
        this.reload();
      }
    });
  }
}
