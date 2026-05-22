# تست و محیط توسعه

## تست خودکار backend (Go)

```bash
cd backend
go mod tidy
make test          # همه unit + integration
make test-unit     # فقط unit
make test-integration
```

| پکیج | نوع | محتوا |
|------|-----|--------|
| `internal/owner`, `quota`, `password`, `auth`, `mikrotik` | Unit | قوانین مالکیت، quota، رمز، JWT، FakeClient |
| `internal/store` | Unit | SQLite موقت، مدیر، activation، مایگریشن `contact_info` |
| `internal/integration` | Integration | HTTP با `httptest`، DB موقت، `FakeClient` |

Integration از `internal/testutil` برای bootstrap ادمین و seed مدیر استفاده می‌کند.

فایل‌های تست:

| فایل | نقش |
|------|-----|
| `internal/integration/api_test.go` | سناریوهای پایه API |
| `internal/integration/vpn_meta_test.go` | `contact_info` / حذف فیلدهای legacy / ایجاد orphan ادمین |
| `internal/integration/acceptance_test.go` | نگاشت به چک‌لیست [business-rules.md](business-rules.md) فاز ۱ |

### پوشش چک‌لیست پذیرش (backend)

| قانون | unit | integration |
|-------|:----:|:-------------:|
| `resolve_owner` (نام / comment / orphan) | ✓ owner | ✓ acceptance |
| همپوشانی slug | ✓ owner | ✓ api |
| quota / پروفایل فعال | ✓ quota | ✓ api + acceptance |
| مدیر فقط کاربران خود | — | ✓ acceptance |
| مدیر بدون `mikrotik_comment` | — | ✓ acceptance |
| `comment=panel:{slug}` روی روتر | — | ✓ acceptance |
| ایجاد + مبلغ activation | — | ✓ acceptance |
| تمدید با quota پر | — | ✓ api |
| رد ایجاد / افزایش shared-users | — | ✓ acceptance |
| ادمین quota / غیرفعال مدیر | — | ✓ acceptance |
| `QUOTA_BELOW_USAGE` | — | ✓ acceptance |
| bootstrap ادمین | ✓ store | ✓ acceptance |
| orphan / فیلدهای مالک ادمین | — | ✓ acceptance |
| `contact_info` / بدون `local_name` در پاسخ | ✓ store | ✓ vpn_meta + acceptance |
| ادمین ایجاد بدون `manager_id` | — | ✓ vpn_meta |
| مایگریشن DB (`local_name` → حذف) | ✓ store | — |
| `disabled` کاربر VPN در روتر | ✓ mikrotik + quota | ✓ vpn_meta + vpn_list_disabled |
| لیست کاربران — فیلد `disabled` / `active_only` | — | ✓ vpn_list_disabled |
| UI لیست — ستون فعال و `row-disabled` | — | ✓ user-list + admin-user-list spec |
| `owner_mismatch` | ✓ owner | ✓ acceptance |
| `NOT_OWNER` | — | ✓ acceptance |
| `PATCH /me` محدود | — | ✓ acceptance |
| دفتر تمدید / settle / ممنوعیت مدیر | — | ✓ api + acceptance |
| snapshot `shared_users` در activation | ✓ store | ✓ acceptance |
| کش + `?refresh=true` | ✓ mikrotik | ✓ acceptance |
| حذف کاربر / حذف رزرو | ✓ fake | ✓ acceptance |
| ادمین: ایجاد / PATCH / DELETE VPN | ✓ fake | ✓ acceptance |
| `connection_bundle` / `.ovpn` | — | ✓ acceptance |
| refresh JWT | ✓ auth | ✓ acceptance |
| تغییر رمز `/me/password` | ✓ password | ✓ acceptance |

**خارج پوشش تست backend (فاز ۱)** — در [تست frontend](#تست-خودکار-frontend-angular) پوشش داده می‌شود:

- UI فارسی، RTL، تاریخ شمسی، modal تأیید

---

## تست خودکار frontend (Angular)

```bash
cd frontend
npm install
npm test          # Vitest — حالت watch
npm run test:ci   # یک‌بار، برای CI
```

| مسیر تست | موضوع |
|----------|--------|
| `src/index-locale.spec.ts` | `index.html`: `lang=fa`, `dir=rtl` |
| `src/app/core/i18n/messages.spec.ts` | پیام‌های فارسی UI و نگاشت خطای API |
| `src/app/core/date/jalali.spec.ts` | تاریخ شمسی با `Asia/Tehran` |
| `src/app/core/format/currency.spec.ts` | مبلغ با جداکننده هزارگان + «ریال» |
| `src/app/core/api/api-client.spec.ts` | `ApiClient.mapError` — خطاهای فارسی |
| `src/app/core/auth/token-storage.spec.ts` | ذخیره/پاک‌سازی JWT در session |
| `src/app/core/auth/auth.service.spec.ts` | login و `homeRoute` |
| `src/app/core/auth/auth.guard.spec.ts` | گاردهای guest/auth/admin/manager |
| `src/app/core/services/vpn-user.service.spec.ts` | درخواست‌های VPN، renewals، managers |
| `src/app/shared/pipes/jalali-date.pipe.spec.ts` | pipe نمایش تاریخ |
| `src/app/shared/utils/profile-active.spec.ts` | وضعیت فعال پروفایل |
| `src/app/shared/services/toast.service.spec.ts` | toast فارسی |
| `src/app/shared/components/confirm-dialog/*.spec.ts` | modal تأیید (نقش `alertdialog`، دکمه تأیید/انصراف) |
| `src/app/shared/components/profile-state-chip/*.spec.ts` | chip وضعیت پروفایل |
| `src/app/shared/components/quota-badge/*.spec.ts` | نمایش سقف quota |
| `src/app/shared/components/copy-field/*.spec.ts` | کپی فیلد + toast |
| `src/app/shared/components/connection-bundle/*.spec.ts` | کارت اتصال مشتری (تب‌ها، پیش‌نمایش) |
| `src/app/features/auth/login/*.spec.ts` | فرم ورود و POST `/auth/login` |
| `src/app/features/manager/dashboard/*.spec.ts` | داشبورد مدیر + API |
| `src/app/app.spec.ts` | پوسته، nav مدیر/ادمین، مخفی در login |
| `src/app/core/layout/document-locale.spec.ts` | قرارداد `lang`/`dir` سند |

### پوشش چک‌لیست فاز ۱ (frontend)

| مورد | unit |
|------|:----:|
| UI فارسی (پیام‌ها، عنوان) | ✓ |
| RTL (`dir=rtl`, `lang=fa`) | ✓ |
| تاریخ شمسی | ✓ |
| modal تأیید حذف | ✓ |
| فرمت مبلغ ریال | ✓ |
| ورود / JWT / گارد مسیر | ✓ |
| داشبورد مدیر (quota + لیست) | ✓ |
| کارت اتصال مشتری | ✓ |
| سرویس‌های API (VPN، renewals، managers) | ✓ |

**خارج پوشش تست frontend (فاز ۱):**

- E2E UI با backend/روتر واقعی (تست backend+روتر در `test-vm` زیر)
- E2E تمام جریان‌های [user-flows.md](user-flows.md) با مرورگر (Playwright/Cypress)
- E2E UI ادمین برای ایجاد/ویرایش/حذف VPN (فرم `/admin/users/new` و جزئیات) — پوشش API در acceptance `TestAdmin_vpnUser_createPatchDelete`
- لاگ نکردن password — بازبینی دستی / lint
- timezone `Asia/Tehran` در assert — سرور تست `TZ=UTC`؛ رفتار parse در production با env
- `SLUG_HAS_USERS` هنگام تغییر slug — نیاز سناریو با کاربر VPN موجود
- فاز ۲: sync خودکار، audit، اعلان انقضا

---

برای توسعه و تست پنل به **روتر MikroTik واقعی** (یا مجازی) با User Manager نیاز دارید. راه پیشنهادی روی میز کار توسعه‌دهنده: **Oracle VirtualBox** و یک ماشین مجازی RouterOS.

## VirtualBox + MikroTik CHR

[MikroTik CHR](https://mikrotik.com/download) (Cloud Hosted Router) نسخه مجازی RouterOS است و برای VirtualBox مناسب است.

### ۱. نصب VirtualBox

- [Oracle VirtualBox](https://www.virtualbox.org/) را روی Linux/Windows/macOS نصب کنید.
- برای آداپتر شبکه **Host-Only** یا **Bridged** معمولاً به extension pack نیاز نیست؛ Host-Only برای ایزوله بودن تست کافی است.

### ۲. ساخت ماشین مجازی

| تنظیم | پیشنهاد |
|--------|---------|
| نوع | Other → Other/Unknown (64-bit) |
| RAM | حداقل ۲۵۶ MB (CHR سبک)؛ ۵۱۲ MB راحت‌تر |
| دیسک | فایل `.vdi` از پکیج CHR (مثلاً `chr-7.x.vdi`) |
| CPU | ۱ vCPU |

1. از سایت MikroTik فایل **Virtual Hard Disk** برای VirtualBox را دانلود کنید.
2. VM جدید بسازید و دیسک موجود (`.vdi`) را attach کنید.
3. در **Settings → System** گزینه **Enable EFI** را مطابق راهنمای نسخه CHR تنظیم کنید (برای برخی buildها لازم است).

### ۳. شبکه (دسترسی از میزبان به API)

هدف: ماشین توسعه (پنل Go) به IP روتر مجازی و پورت **8729** (`api-ssl`) برسد.

**گزینه A — Host-Only (پیشنهادی برای تست محلی)**

1. VirtualBox → **File → Host Network Manager** → آداپتر `vboxnet0` بسازید (مثلاً `192.168.56.1/24`).
2. در VM دو کارت شبکه:
   - **Adapter 1:** NAT (اختیاری — خروج اینترنت برای آپدیت روتر)
   - **Adapter 2:** Host-only → `vboxnet0`
3. پس از بالا آمدن روتر، در Winbox/CLI به کارت دوم IP بدهید، مثلاً:

```routeros
/ip address add address=192.168.56.11/24 interface=ether2
```

4. در `.env` پنل: `MIKROTIK_HOST=192.168.56.11`

**گزینه B — Bridged**

- Adapter به کارت فیزیکی Wi‑Fi/Ethernet میزبان bridge شود؛ روتر IP از همان subnet DHCP می‌گیرد (مثلاً `192.168.1.x`).

### ۴. اولین بالا آمدن روتر

1. VM را Start کنید؛ با **Winbox** (Neighbours) یا serial به روتر وصل شوید.
2. نسخه RouterOS را **7.21+** نگه دارید.
3. پکیج **User Manager** را نصب/فعال کنید (`/system package print`).

### VM آماده: Mikrotik-Base (`clean-test-state`)

اسنپ‌شات تست از قبل این تنظیمات را دارد:

| سرویس | مقدار |
|--------|--------|
| کاربر RouterOS | `admin` / `admin` |
| SSH (CLI دستی، Winbox) | پورت `22` — `ssh admin@192.168.56.11` |
| API-SSL (پنل و `make test-vm`) | پورت `8729` — `MIKROTIK_PORT` |

پنل **فقط** از **api-ssl** استفاده می‌کند؛ SSH برای عیب‌یابی یا تنظیم دستی روی روتر است.

### ۵. آماده‌سازی برای پنل (روتر تازه، بدون اسنپ‌شات)

اگر VM از صفر ساخته شده (نه `Mikrotik-Base`):

```routeros
# فعال‌سازی API-SSL
/ip service set api-ssl disabled=no port=8729

# پروفایل نمونه (نام را با DEFAULT_PROFILE در .env هماهنگ کنید)
/user-manager profile add name=profile-open-2M-30d

# منطقه زمانی — هم‌خوان با سرور پنل (تهران)
/system clock set time-zone-name=Asia/Tehran
```

در production ترجیحاً کاربر API جدا با دسترسی محدود به User Manager (مثلاً `api-panel`) بسازید؛ برای VM توسعه همان `admin` کافی است.

از میزبان تست اتصال:

```bash
nc -zv 192.168.56.11 8729
ssh -p 22 admin@192.168.56.11
```

### ۶. تنظیم `.env` پنل (توسعه)

```bash
cp .env.example .env
# برای VM: MIKROTIK_FAKE=false
```

| متغیر | مقدار (VM Mikrotik-Base) |
|--------|---------------------------|
| `MIKROTIK_HOST` | `192.168.56.11` |
| `MIKROTIK_API` | `api-ssl` (یا `api` + پورت `8728` اگر TLS شکست می‌خورد) |
| `MIKROTIK_PORT` | `8729` (api-ssl) / `8728` (api) |
| `MIKROTIK_USERNAME` | `admin` |
| `MIKROTIK_PASSWORD` | `admin` |
| `MIKROTIK_TLS_INSECURE` | `true` (فقط api-ssl) |

پس از پیاده‌سازی backend/frontend، همان [اجرای سریع](../README.md#اجرای-سریع-پس-از-پیاده‌سازی) در README.

## تست بدون روتر (اختیاری)

برای unit testهای Go می‌توان `MIKROTIK_FAKE=true` (فاز پیاده‌سازی) و mock کلاینت RouterOS استفاده کرد — جایگزین تست یکپارچه با User Manager واقعی نیست.

## عیب‌یابی رایج VirtualBox

| مشکل | بررسی |
|------|--------|
| Winbox روتر را نمی‌بیند | کابل مجازی؛ فایروال میزبان |
| پنل `MIKROTIK_UNAVAILABLE` | IP/pورت؛ `api-ssl` فعال؛ ping از میزبان |
| TLS خطا | `MIKROTIK_TLS_INSECURE=true` در dev؛ یا `MIKROTIK_API=api` و `MIKROTIK_PORT=8728` |
| User Manager خالی | پکیج user-manager نصب؛ نسخه ≥ 7.21 |

## تست خودکار با VM (VirtualBox)

تست‌های `internal/vmtest` با build tag **`vm`** به روتر واقعی در VirtualBox وصل می‌شوند (نه `FakeClient`). پیش‌نیاز: VM آماده (مثلاً `Mikrotik-Base` + اسنپ‌شات `clean-test-state`)، User Manager، کاربر API و پروفایل پیش‌فرض.

### متغیرهای محیط

| متغیر | پیش‌فرض | توضیح |
|--------|---------|--------|
| `MIKROTIK_FAKE` | (اجبار `false` در تست VM) | — |
| `MIKROTIK_HOST` | `192.168.56.11` | IP روتر روی `vboxnet0` |
| `MIKROTIK_PORT` | `8729` | api-ssl (نه SSH) |
| `MIKROTIK_USERNAME` | `admin` | همان کاربر VM |
| `MIKROTIK_PASSWORD` | `admin` | از `.env` یا پیش‌فرض اسنپ‌شات |
| `MIKROTIK_TLS_INSECURE` | `true` در تست VM | گواهی self-signed |
| `DEFAULT_PROFILE` | `profile-open-2M-30d` | باید روی روتر وجود داشته باشد |
| `MIKROTIK_VM_NAME` | `Mikrotik-Base` | نام VM در VirtualBox |
| `MIKROTIK_VM_SNAPSHOT` | `clean-test-state` | بازگردانی قبل از تست |
| `MIKROTIK_VM_MANAGE` | `true` | `false` = VM از قبل روشن است؛ فقط منتظر پورت |
| `MIKROTIK_VM_WAIT` | `90s` | زمان انتظار برای بالا آمدن api-ssl |

### اجرا

```bash
cd backend
cp ../.env.example ../.env   # admin/admin و MIKROTIK_FAKE=false برای VM
make test-vm
# یا:
go test -tags=vm ./internal/vmtest/... -count=1 -timeout=15m
```

`TestMain` در صورت `MIKROTIK_VM_MANAGE=true`: `VBoxManage snapshot restore` → `startvm --type headless` → انتظار TCP روی `MIKROTIK_HOST:8729` → اجرای تست‌ها → خاموش کردن VM.

### پوشش تست VM

| تست | محتوا |
|-----|--------|
| `TestRouterOS_pingAndProfile` | اتصال api-ssl |
| `TestRouterOS_createUser_panelComment` | `comment=panel:…` روی روتر |
| `TestRouterOS_assignProfile` | `user-profile/add` |
| `TestE2E_createVPNUser_onRouter` | `POST /vpn-users` + بررسی روتر |
| `TestE2E_managerListIsolation` | جداسازی لیست + legacy `comment` |
| `TestE2E_deleteVPNUser_removesFromRouter` | حذف از API و روتر |

## پذیرش دستی در VM

- [ ] از میزبان به `MIKROTIK_HOST:8729` دسترسی TCP
- [ ] `/user-manager/user print` از API یا Winbox
- [ ] پروفایل `profile-open-2M-30d` (یا نام env) وجود دارد
- [ ] ایجاد کاربر با پیشوند (مثلاً `ali-test01`)؛ در روتر `comment=panel:ali` بررسی شود
- [ ] legacy: کاربر `reza` + `comment=panel:ali` در لیست مدیر `ali` دیده شود
- [ ] مدیر در API پاسخ `mikrotik_comment` نداشته باشد
