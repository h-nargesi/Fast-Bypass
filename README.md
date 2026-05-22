# پنل مدیریت کاربران (User Manager Panel)

پنل وب برای مدیریت کاربران **MikroTik User Manager** از طریق `api-ssl`. هر **مدیر** فقط کاربران متعلق به خود را می‌بیند (پیشوند نام یا `comment=panel:{slug}`) و با قانون نام می‌سازد؛ **ادمین** کل سیستم و `comment` روتر را می‌بیند.

## پیش‌نیازها

| مورد | مقدار |
|------|--------|
| RouterOS | 7.21+ |
| پکیج | `user-manager` فعال |
| API | `MIKROTIK_API=api-ssl` (8729) یا `api` (8728) |
| زمان | همه سرویس‌ها `Asia/Tehran` |
| پروفایل پیش‌فرض | `profile-open-2M-30d` (در روتر از قبل تعریف شده باشد) |

## استک

- **Backend:** Go 1.22+، REST، JWT
- **Frontend:** Angular 19+، UI فارسی (RTL)
- **DB:** SQLite 3 (متادیتای پنل؛ کاربر VPN در MikroTik)

## مستندات پیاده‌سازی

| سند | محتوا |
|-----|--------|
| [docs/architecture.md](docs/architecture.md) | معماری، منبع حقیقت، جریان داده |
| [docs/data-model.md](docs/data-model.md) | اسکیما SQLite و نگاشت به MikroTik |
| [docs/business-rules.md](docs/business-rules.md) | سقف، پیشوند، تمدید، قوانین با مثال |
| [docs/mikrotik-api.md](docs/mikrotik-api.md) | اتصال RouterOS API و عملیات User Manager |
| [docs/api.md](docs/api.md) | قرارداد REST (endpointها) |
| [docs/user-flows.md](docs/user-flows.md) | سناریوها و صفحات UI |
| [db/schema.sql](db/schema.sql) | DDL اولیه SQLite |
| [.env.example](.env.example) | متغیرهای محیط |
| [docs/testing.md](docs/testing.md) | تست با VirtualBox و روتر MikroTik مجازی (CHR) |

## تست

- تست خودکار بدون روتر: `cd backend && make test`
- تست با VM MikroTik (VirtualBox): `make test-vm` — جزئیات env و `VBoxManage` در [docs/testing.md](docs/testing.md#تست-خودکار-با-vm-virtualbox)

برای توسعه دستی، می‌توان با **Oracle VirtualBox** یک روتر MikroTik (CHR) را بالا آورد و پنل را به IP همان VM وصل کرد (`MIKROTIK_FAKE=false`).

## اجرای سریع

```bash
cp .env.example .env
# ویرایش JWT_SECRET، ADMIN_PASSWORD و در صورت نیاز MIKROTIK_*
# برای توسعه بدون روتر: MIKROTIK_FAKE=true (پیش‌فرض در .env.example نیست — در .env اضافه کنید)

# backend (نیاز به Go 1.22+)
cd backend
go mod tidy
go run ./cmd/server
# API: http://localhost:8080/api/v1 — health: http://localhost:8080/health

# frontend
cd frontend && npm install && npm start
# تست واحد: npm run test:ci — جزئیات docs/testing.md
```

ساختار backend: `backend/cmd/server`، ماژول‌ها در `backend/internal/` (auth، store، mikrotik، app).

## خلاصه موجودیت‌های MikroTik

### کاربر — `/user-manager/user`

| فیلد RouterOS | کاربرد |
|---------------|--------|
| `name` | نام ورود VPN (در پنل: `{prefix}{local_name}`) |
| `password` | رمز VPN |
| `shared-users` | حداکثر اتصال همزمان این کاربر |
| `comment` | برچسب مالکیت پنل: `panel:{slug}` — سرور ست می‌کند؛ مدیر نمی‌بیند. یادداشت مشتری در SQLite (`notes`) |

### کاربر-پروفایل — `/user-manager/user-profile`

| فیلد | کاربرد |
|------|--------|
| `user` | نام کاربر |
| `profile` | نام پروفایل (مثلاً `profile-open-2M-30d`) |
| `state` | فقط خواندنی — وضعیت (فعال / منقضی / …) |
| `end-time` | فقط خواندنی — پایان دوره |

## امکانات (مرجع محصول)

- چند **مدیر** با سقف (`quota`) و پیشوند نام یکتا
- یک پروفایل قابل انتساب در فاز اول (قابل تنظیم در env)
- متادیتا در پنل: تماس، توضیحات؛ مبلغ پرداخت **اختیاری** به ازای هر دوره (۰ یا خالی مجاز)
- **تحویل به مشتری:** کارت اتصال در جزئیات کاربر (OpenVPN/L2TP، کپی، دانلود `.ovpn`) — [user-flows.md](docs/user-flows.md)
- **مالکیت:** `resolve_owner` از **نام** (`{slug}{separator}…`) یا **`comment=panel:{slug}`**؛ مدیر فقط با قانون نام می‌سازد
- **ادمین:** همه VPN + `mikrotik_comment`؛ orphan = بدون تطابق نام و comment؛ نسبت legacy با `panel:{slug}` در Winbox
- **دفتر تمدید:** ادمین با فیلتر مدیر (پیش‌فرض orphan) + تسویه؛ مدیر فقط لیست خود و وضعیت تسویه (بدون تغییر) — [user-flows.md](docs/user-flows.md)

جزئیات در [docs/business-rules.md](docs/business-rules.md).
