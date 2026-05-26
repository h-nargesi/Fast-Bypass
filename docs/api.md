# REST API

Base URL: `http://localhost:8080/api/v1`  
Auth: `Authorization: Bearer <access_token>`  
Content-Type: `application/json`  
Errors: `{ "error": { "code": "QUOTA_EXCEEDED", "message": "..." } }`

## Auth


| Method | Path            | Body                         | پاسخ                                                        |
| ------ | --------------- | ---------------------------- | ----------------------------------------------------------- |
| POST   | `/auth/login`   | `{ "username", "password" }` | `{ access_token, refresh_token, role, manager_id?, slug?, name_prefix? }` |
| POST   | `/auth/refresh` | `{ "refresh_token" }`        | access + refresh جدید                                       |
| POST   | `/auth/logout`  | —                            | 204 (invalidate refresh اختیاری)                            |
| GET    | `/me`           | —                            | پروفایل خواندنی — پاسخ نمونه زیر |
| PATCH  | `/me`           | `{ "display_name" }`         | فقط نام نمایشی — مدیر/ادمین |
| POST   | `/me/password`  | `{ "current_password", "new_password" }` | 204 — تغییر رمز **خود** |

`PATCH /me` هر فیلد دیگری (`username`, `slug`, `quota`, `password`) → `400`.

### GET `/me` — پاسخ (مدیر)

```json
{
  "username": "ali",
  "display_name": "علی احمدی",
  "slug": "ali",
  "name_prefix": "ali-",
  "quota": 10,
  "used_quota": 5
}
```

`name_prefix` = `slug` + `USERNAME_PREFIX_SEPARATOR` از env سرور — برای نمایش قبل از textbox نام کاربر VPN در UI.

پاسخ login مدیر همان فیلدهای `slug` و `name_prefix` را در body یا از `/me` پس از ورود بارگذاری می‌کند.

## Manager — VPN users


| Method | Path                                    | توضیح                                            |
| ------ | --------------------------------------- | ------------------------------------------------ |
| GET    | `/vpn-users`                            | لیست کاربران خود (کش MikroTik + enrich DB؛ `?refresh=true` اجبار fetch) |
| GET    | `/vpn-users/:id`                        | جزئیات + user-profiles + `activations` (شامل `shared_users`) + `connection_bundle` |
| GET    | `/vpn-users/:id/connection`             | همان `connection_bundle` (اختیاری — برای lazy-load کارت اتصال) |
| GET    | `/vpn-users/:id/ovpn`                   | دانلود فایل پیکربندی OpenVPN (`application/x-openvpn-profile`) |
| POST   | `/vpn-users`                            | ایجاد — body زیر                                 |
| PATCH  | `/vpn-users/:id`                        | `password`, `shared_users`, `contact_*`, `notes` — **بدون** `comment` روتر |
| DELETE | `/vpn-users/:id`                        | حذف روتر + DB                                    |
| POST   | `/vpn-users/:id/assign-profile`         | assign پروفایل                                   |
| DELETE | `/vpn-users/:id/profiles/:profileRowId` | حذف رزرو                                         |


### POST `/vpn-users`

```json
{
  "local_name": "reza01",
  "password": "Secret123",
  "shared_users": 2,
  "contact_info": "تلگرام @x",
  "notes": "مشتری فروشگاه",
  "disabled": false,
  "assign_profile": true,
  "profile_name": "profile-open-2M-30d",
  "amount_paid": 150000,
  "currency": "IRR"
}
```

`amount_paid` و `currency` **اختیاری** (حذف یا `null` = بدون ثبت مبلغ؛ `0` مجاز است).

سرور پس از `user/add` در روتر: `name = name_prefix + local_name` و `comment = panel={slug}` (مدیر این فیلدها را ارسال نمی‌کند).

پاسخ 201 (بدون `mikrotik_comment`):

```json
{
  "id": 1,
  "mikrotik_name": "ali_reza01",
  "shared_users": 2,
  "disabled": false,
  "profiles": [{ "profile": "profile-open-2M-30d", "state": "active", "end_time": "..." }]
}
```

`disabled` (اختیاری، پیش‌فرض `false`): `true` = غیرفعال در **User Manager روتر**؛ در SQLite ذخیره نمی‌شود.

### PATCH `/vpn-users/:id`

فیلدهای اختیاری: `password`, `shared_users`, `disabled`, `contact_info`, `notes`.

پس از `PATCH` با `shared_users`، آخرین activation تسویه‌نشده همان کاربر در DB به‌روز می‌شود (همان قانون هم‌خوانی `shared_users` در [business-rules.md](business-rules.md#هم‌خوانی-shared_users-روتر--دفتر-تمدید)).

### `activations` در `GET /vpn-users/:id`

آرایهٔ تاریخچه تمدید/assign. هر عنصر حداقل:

| فیلد | UI (جدول تاریخچه در `/users/:id`) |
|------|-----------------------------------|
| `created_at` | تاریخ |
| `profile_name` | پروفایل |
| `shared_users` | اتصال — برای آخرین ردیف تسویه‌نشده از MikroTik overlay |
| `amount_paid` | مبلغ (اختیاری) |
| `is_settled` | تسویه ✓/✗ |

### POST `/vpn-users/:id/assign-profile`

```json
{
  "profile_name": "profile-open-2M-30d",
  "amount_paid": 150000,
  "currency": "IRR",
  "paid_at": "2026-05-21T10:00:00Z",
  "note": "تمدید اردیبهشت"
}
```

بدنه می‌تواند فقط `{ "profile_name": "..." }` باشد (بدون مبلغ).

### `connection_bundle` — اطلاعات اتصال برای مشتری

در `GET /vpn-users/:id` (و در صورت نیاز `GET /vpn-users/:id/connection`) آبجکت زیر برگردانده می‌شود. فقط **مدیر مالک** (همان قوانین `NOT_OWNER`).

```json
{
  "username": "ali-reza01",
  "password": "Secret123",
  "openvpn_key_password": "KeyPassFromEnv",
  "l2tp_ipsec_secret": "SharedSecretFromEnv",
  "l2tp_server": "vpn.nimbaha.info",
  "openvpn_download_url": "http://dl.nimbaha.info/dl/"
}
```

| فیلد | منبع |
|------|------|
| `username` | `mikrotik_name` |
| `password` | MikroTik User Manager — در صورت عدم دسترسی در کش، `null` و UI پیام بروزرسانی نشان می‌دهد |
| `openvpn_key_password` | اولویت: `vpn_user_meta.cert_key_pass` → `managers.cert_key_pass` (مالک) → env `OPENVPN_KEY_PASSWORD` — [certificates.md](certificates.md) |
| `l2tp_ipsec_secret` | env `L2TP_IPSEC_SECRET` |
| `l2tp_server` | env `L2TP_SERVER` |
| `openvpn_download_url` | env `OPENVPN_DOWNLOAD_URL` |

**امنیت:** این endpointها فقط برای نقش `manager` مالک کاربر؛ ادمین از مسیر `/admin/vpn-users/:id` می‌تواند همان bundle را ببیند. رمزها و `cert_key_pass` در لاگ API نوشته نمی‌شوند.

### GET `/vpn-users/:id/ovpn`

- پاسخ: `Content-Type: application/x-openvpn-profile` (یا `text/plain`)
- `Content-Disposition: attachment; filename="{mikrotik_name}.ovpn"`
- بدنه (بدون اجرای اسکریپت در زمان دانلود):
  - اگر کاربر یا مدیر مالک `cert_title` دارد: محتوای `config-{mikrotik_name}.ovpn` از فایل‌سیستم روتر (ساخته‌شده در زمان ایجاد کاربر)
  - وگرنه: قالب `OPENVPN_TEMPLATE_PATH` + `username` / `password` / `openvpn_key_password` از env
- خطا: `503` اگر قالب legacy پیکربندی نشده و مسیر گواهی هم در دسترس نیست؛ `404` اگر کاربر وجود ندارد

جزئیات گواهی: [certificates.md](certificates.md).

### GET `/vpn-users` query


| Param               | توضیح                         |
| ------------------- | ----------------------------- |
| `q`                 | جستجو در mikrotik_name / contact_info |
| `active_only`       | فقط با پروفایل فعال           |
| `page`, `page_size` | پیش‌فرض 1, 20                 |
| `refresh`           | `true` = نادیده گرفتن کش MikroTik |

لیست مدیر: کاربرانی که `resolve_owner` = مدیر جاری (نام با `name_prefix` **یا** `comment="panel={slug}"`). فیلد `mikrotik_comment` در پاسخ **مدیر وجود ندارد**.

## Manager — renewals ledger (read-only)

| Method | Path | توضیح |
| ------ | ---- | ----- |
| GET | `/renewals` | دفتر تمدید — ستون ۱: `renewed_at`؛ **فقط** کاربران متعلق به مدیر جاری؛ بدون تسویه |

مدیر **نمی‌تواند** `manager_id` یا `settled` را برای تغییر محدودهٔ لیست ارسال کند (سرور نادیده می‌گیرد). `POST .../settle-through` برای نقش manager → `403`.

### GET `/renewals`

Query مجاز: `from`, `to`, `q`, `page`, `page_size`, `refresh` (همان معنی لیست VPN).

پاسخ همان ساختار [GET `/admin/renewals`](#get-adminrenewals) با این تفاوت‌ها:

- فقط activations کاربرانی که `resolve_owner` = `manager_id` توکن
- `summary` فقط روی همان محدوده محاسبه می‌شود
- `can_settle: false` در سطح پاسخ (UI دکمه تسویه نشان ندهد)
- ستون `is_settled` **خواندنی** نمایش داده می‌شود

**ترتیب ستون‌های UI (مدیر و ادمین):**  
`renewed_at` (تاریخ تمدید) → `mikrotik_name` → `shared_users` → `profile_name` → `profile_state` → `mikrotik_end_time` → `is_settled` → (ادمین) عمل تسویه.

### GET `/me/quota`

```json
{
  "quota": 10,
  "used": 5,
  "available": 5
}
```

## Admin — managers


| Method | Path                   | نقش                                      |
| ------ | ---------------------- | ---------------------------------------- |
| GET    | `/admin/stats`         | آمار VPN — کاربران فعال و اتصال، orphan، به تفکیک مدیر (`?refresh=true`) |
| GET    | `/admin/managers`      | admin                                    |
| POST   | `/admin/managers`      | admin — بدنه اختیاری `cert_title` (ساخت گواهی در همان درخواست) |
| PATCH  | `/admin/managers/:id`  | admin                                    |
| GET    | `/admin/vpn-users`     | همه — `manager_id`, `orphan=true`؛ هر ردیف شامل مالک + `mikrotik_comment` + `owner_mismatch` |
| POST   | `/admin/vpn-users`     | ایجاد کاربر VPN — بدنه همان `POST /vpn-users` + **`manager_id` اختیاری** + **`cert_title` اختیاری** (ساخت گواهی و `config-{mikrotik_name}.ovpn` در همان درخواست) |
| GET    | `/admin/vpn-users/:id` | جزئیات enrich + `connection_bundle` + activations + profiles |
| GET    | `/admin/vpn-users/:id/connection` | همان `connection_bundle` |
| GET    | `/admin/vpn-users/:id/ovpn` | دانلود `.ovpn` — بدون `NOT_OWNER` |
| PATCH  | `/admin/vpn-users/:id` | همان فیلدهای مدیر (`password`, `shared_users`, `contact_info`, `notes`) + اختیاری `manager_id` برای هم‌خوان‌سازی DB وقتی `resolve_owner` مالک دارد |
| DELETE | `/admin/vpn-users/:id` | حذف روتر + DB |
| POST   | `/admin/vpn-users/:id/assign-profile` | assign / تمدید — quota **مدیر مالک** (`resolve_owner`) اعمال می‌شود |
| DELETE | `/admin/vpn-users/:id/profiles/:profileRowId` | حذف رزرو |
| GET    | `/admin/renewals`      | دفتر تمدید — فیلتر **اجباری** مدیر یا orphan — [جزئیات](#admin--renewals-ledger) |
| POST   | `/admin/renewals/settle-through` | تسویه دسته‌ای در **همان محدوده فیلتر** — فقط admin |

**دسترسی:** نقش `admin` روی همه ردیف‌ها — **بدون** `NOT_OWNER`. عملیات write که `shared_users` را افزایش می‌دهد همچنان سقف **مدیر مالک** (`resolve_owner`) را چک می‌کند؛ برای orphan تا زمان تخصیص مالک، assign با افزایش مصرف ممکن است `403` یا نیاز به تخصیص مالک اول باشد (طبق [business-rules.md](business-rules.md)).

### GET `/admin/stats`

آمار یک‌جا برای داشبورد ادمین. منبع: لیست کاربران MikroTik + `resolve_owner` + پروفایل‌های فعال (همان قانون `used_quota`).

Query: `refresh=true` (اختیاری) — نادیده گرفتن کش روتر.

```json
{
  "manager_count": 2,
  "totals": { "vpn_users": 12, "connections": 18 },
  "orphan": { "vpn_users": 1, "connections": 2 },
  "by_manager": [
    {
      "manager_id": 1,
      "display_name": "علی احمدی",
      "username": "ali",
      "quota": 10,
      "vpn_users": 5,
      "connections": 8
    }
  ]
}
```

| فیلد | توضیح |
|------|--------|
| `totals.vpn_users` | تعداد کاربران **فعال** — غیرغیرفعال در روتر + پروفایل فعال |
| `totals.connections` | جمع `shared_users` برای همان کاربران فعال |
| `orphan.*` | کاربرانی که `resolve_owner` = بدون مدیر (فقط فعال‌ها) |
| `by_manager[].connections` | مصرف فعال همان مدیر (برابر `used_quota` در `GET /admin/managers`) |
| `by_manager[].vpn_users` | تعداد کاربران فعال متعلق به مدیر |

### GET `/admin/vpn-users` — query

همان پارامترهای `GET /vpn-users` مدیر، به‌علاوه:

| Param | توضیح |
|-------|--------|
| `manager_id` | فقط کاربرانی که `resolve_owner` = این مدیر |
| `orphan` | `true` = فقط بدون مدیر (`resolve_owner` = null) |

### پاسخ نمونه — لیست / جزئیات (فیلدهای مالک)

```json
{
  "id": 12,
  "mikrotik_name": "ali-reza01",
  "mikrotik_comment": "panel=ali",
  "contact_info": "تلگرام @x",
  "notes": "یادداشت",
  "manager_id": 1,
  "manager_display_name": "علی احمدی",
  "manager_username": "ali",
  "manager_slug": "ali",
  "owner_mismatch": false,
  "profiles": [],
  "activations": []
}
```

orphan:

```json
{
  "mikrotik_name": "reza",
  "mikrotik_comment": "",
  "manager_id": null,
  "manager_display_name": null,
  "manager_username": null,
  "manager_slug": null,
  "owner_mismatch": false
}
```

`manager_*` از JOIN `managers` روی `resolve_owner` (نه لزوماً `vpn_user_meta.manager_id` اگر ناهماهنگ باشد — در آن صورت `owner_mismatch: true` و UI مالک واقعی را از `resolve_owner` نشان می‌دهد).

### POST `/admin/vpn-users`

```json
{
  "manager_id": 1,
  "local_name": "reza01",
  "password": "Secret123",
  "shared_users": 2,
  "assign_profile": true,
  "profile_name": "profile-open-2M-30d",
  "amount_paid": 150000,
  "currency": "IRR"
}
```

- با **`manager_id`**: سقف `shared_users` و نام/comment روتر مطابق مدیر (`name_prefix + local_name`، `comment="panel={slug}"`).
- **بدون `manager_id`**: `local_name` = نام کامل روتر (حداکثر ۳۲ کاراکتر)؛ `comment` خالی؛ `manager_id` در DB = `NULL` (orphan).

### PATCH `/admin/vpn-users/:id`

Body ترکیبی:

```json
{
  "password": "NewSecret1",
  "contact_info": "تلگرام @x",
  "notes": "یادداشت پشتیبانی",
  "manager_id": 1
}
```

| فیلد | قانون |
|------|--------|
| `password`, `shared_users`, `contact_*`, `notes` | همان اعتبارسنجی `PATCH /vpn-users/:id` |
| `manager_id` | فقط وقتی `resolve_owner` از قبل مالک دارد — هم‌خوان‌سازی SQLite؛ **orphan را برطرف نمی‌کند** بدون rename یا `panel={slug}` در روتر |
| `cert_title` | اختیاری — [certificates.md](certificates.md): خالی→جدید = صدور گواهی؛ پر→عنوان دیگر = صدور گواهی جدید؛ پر→خالی = پاک meta (تأیید UI در فرانت) |

پاسخ جزئیات شامل `cert_title` (بدون `cert_key_pass`). GET همان endpoint.


### POST `/admin/managers`

```json
{
  "username": "ali",
  "password": "ManagerPass1",
  "display_name": "علی",
  "slug": "ali",
  "quota": 10
}
```

### PATCH `/admin/managers/:id`

ادمین می‌تواند `quota` را ویرایش و مدیر را غیرفعال کند:

```json
{
  "username": "ali.new",
  "display_name": "علی احمدی",
  "quota": 15,
  "is_active": false,
  "password": "NewManager1"
}
```


| فیلد        | قانون                                                     |
| ----------- | --------------------------------------------------------- |
| `username`  | اختیاری — نام کاربری ورود؛ غیرخالی؛ یکتا (NOCASE)؛ `409 USERNAME_IN_USE` |
| `quota`     | اگر کمتر از `used_quota` فعلی → `409 QUOTA_BELOW_USAGE`   |
| `is_active` | `false` = ورود مدیر مسدود؛ کاربران VPN در روتر بدون تغییر |
| `slug`      | فقط اگر آن مدیر هیچ VPN user ندارد؛ بدون همپوشانی با slugهای دیگر |
| `password`  | اختیاری — reset رمز ورود پنل (بدون `current_password`)؛ همان اعتبارسنجی پنل |
| `cert_title` | اختیاری — اگر نسبت به مقدار فعلی **تغییر** کند: گواهی جدید روی MikroTik + به‌روز `cert_key_pass`؛ رشته خالی = پاک کردن فیلدها در DB (بدون حذف فایل روی روتر) |

`GET /admin/managers` هر ردیف شامل `cert_title` (بدون `cert_key_pass`) است.

### Admin — renewals ledger

هر `profile_activations` (ایجاد + تمدید). لیست **همیشه** در یک محدودهٔ مالکیت است؛ بدون انتخاب مدیر = فقط **بدون مدیر (orphan)**.

#### GET `/admin/renewals`

Query:

| Param | توضیح |
|-------|--------|
| `manager_id` | شناسه مدیر — اگر **حذف/خالی** باشد → فقط تمدیدهای کاربران **orphan** (`resolve_owner` = بدون مدیر) |
| `settled` | `unsettled` (پیش‌فرض فیلتر جدول)، `settled`، `all` — فقط داخل همان محدودهٔ `manager_id`/orphan |
| `from`, `to` | بازه `created_at` (ISO8601، اختیاری) |
| `q` | جستجو در `mikrotik_name` / `contact_info` |
| `page`, `page_size` | پیش‌فرض 1, 50 |

**فیلتر مالک (سرور):**

| `manager_id` در query | ردیف‌های مجاز |
|----------------------|----------------|
| حذف شده / خالی | `vpn_user_meta` که `resolve_owner` = orphan |
| `manager_id=3` | کاربرانی که `resolve_owner` = مدیر ۳ |

جمع‌ها (`summary`) و `total` فقط روی همین محدوده (نه کل سیستم).

پاسخ:

```json
{
  "scope": { "manager_id": 3, "manager_display_name": "علی احمدی", "orphan": false },
  "can_settle": true,
  "summary": {
    "unsettled_shared_users_sum": 45,
    "all_shared_users_sum": 120
  },
  "items": [
    {
      "id": 42,
      "renewed_at": "2026-05-21T10:00:00+03:30",
      "mikrotik_name": "ali-reza01",
      "manager_id": 1,
      "manager_display_name": "علی احمدی",
      "shared_users": 2,
      "profile_name": "profile-open-2M-30d",
      "profile_state": "active",
      "mikrotik_end_time": "2026-06-20T23:59:59+03:30",
      "is_settled": false,
      "amount_paid": null,
      "currency": "IRR"
    }
  ],
  "page": 1,
  "page_size": 50,
  "total": 200
}
```

| # | فیلد آیتم | منبع | ستون UI |
|---|-----------|------|---------|
| 1 | `renewed_at` | `profile_activations.created_at` | تاریخ تمدید |
| 2 | `mikrotik_name` | `vpn_user_meta` | نام کاربر |
| — | `manager_*` | JOIN `managers` | (فقط در scope/header ادمین، نه ستون جدول) |
| 3 | `shared_users` | MikroTik برای `is_settled=0`؛ DB ثابت برای تسویه‌شده | تعداد اتصالات همزمان |
| 4 | `profile_name` | DB | نام پروفایل |
| 5 | `profile_state` | MikroTik `user-profile` | وضعیت پروفایل |
| 6 | `mikrotik_end_time` | DB | تاریخ اعتبار |
| 7 | `is_settled` | DB | تسویه شده |

`summary.unsettled_shared_users_sum`: جمع `shared-users` زنده از MikroTik برای ردیف‌های `is_settled=0` در **همان scope** (بدون توجه به `page`).  
`summary.all_shared_users_sum`: unsettled از روتر + settled از DB، در همان scope.

**هم‌خوانی `shared_users`:** `PATCH /vpn-users/:id` با `shared_users` پس از `SetUser` روی روتر، ستون `shared_users` **آخرین** `profile_activations` با `is_settled=0` همان کاربر را به‌روز می‌کند. در `GET /renewals` و `activations` جزئیات کاربر، فقط **آخرین** ردیف تسویه‌نشده هر کاربر از MikroTik overlay می‌شود (heal در DB در صورت اختلاف). ردیف‌های تسویه‌شده و unsettledهای قدیمی‌تر ثابت می‌مانند.

مرتب‌سازی پیش‌فرض: `renewed_at` نزولی (جدیدترین بالا).

پاسخ orphan (بدون `manager_id`):

```json
{
  "scope": { "manager_id": null, "orphan": true },
  "can_settle": true,
  "summary": { "unsettled_shared_users_sum": 8, "all_shared_users_sum": 12 }
}
```

#### POST `/admin/renewals/settle-through`

فقط **admin**. علامت‌گذاری تسویه برای unsettledهای **قدیمی‌تر یا هم‌زمان** با رکورد انتخاب‌شده، **فقط اگر activation در scope فعلی باشد**.

```json
{
  "through_activation_id": 42,
  "manager_id": 3
}
```

| فیلد body | توضیح |
|-----------|--------|
| `through_activation_id` | اجباری |
| `manager_id` | اختیاری — باید با scope لیست هم‌خوان باشد؛ حذف = scope orphan |

قانون سرور:

1. بارگذاری activation `42`؛ اگر مالک کاربر در scope نباشد → `403 NOT_IN_SCOPE`
2. `T = created_at`, `I = id` از همان activation
3. `UPDATE ... WHERE is_settled=0 AND (created_at < T OR (created_at = T AND id <= I))` **و** همان شرط مالک scope (مدیر N یا orphan)
4. پاسخ: `{ "updated_count": 17, "scope": {...}, "summary": {...} }`

خطا: `404` اگر id وجود نداشته باشد؛ `403` برای manager یا activation خارج از scope.

**ثبت `shared_users` در assign:** در `POST /vpn-users` (با assign) و `POST /vpn-users/:id/assign-profile` پس از موفقیت روتر، مقدار فعلی `shared-users` کاربر در ستون `profile_activations.shared_users` ذخیره شود (شروع دورهٔ جاری).

## Health


| Method | Path      |
| ------ | --------- |
| GET    | `/health` |


```json
{ "status": "ok", "db": true, "mikrotik": true }
```

## کدهای HTTP


| Status | کاربرد                                        |
| ------ | --------------------------------------------- |
| 400    | validation                                    |
| 401    | token نامعتبر؛ `INVALID_CURRENT_PASSWORD`     |
| 403    | NOT_OWNER / نقش / `MANAGER_DISABLED`          |
| 409    | NAME_TAKEN, QUOTA_EXCEEDED, QUOTA_BELOW_USAGE, SLUG_OVERLAPS |
| 503    | MIKROTIK_UNAVAILABLE                          |


## Angular — سرویس‌ها


| سرویس            | endpoints                 |
| ---------------- | ------------------------- |
| `AuthService`    | login, refresh, logout, changePassword |
| `ProfileService` | getMe, patchMe (`display_name`)       |
| `VpnUserService` | CRUD + assign + `getConnection`, `downloadOvpn` |
| `RenewalsService`| `GET /renewals` (manager) |
| `QuotaService`   | `/me/quota` (اختیاری اگر در `/me` ادغام شود) |
| `AdminService`   | managers, `GET /admin/stats`, admin vpn-users, `GET/POST /admin/renewals` |


Interceptor: افزودن Bearer؛ در 401 تلاش refresh یک‌بار.