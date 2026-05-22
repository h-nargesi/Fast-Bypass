# یکپارچه‌سازی MikroTik

## اتصال

| پارامتر | env |
|---------|-----|
| حالت | `MIKROTIK_API` — `api-ssl` (پیش‌فرض) یا `api` |
| Host | `MIKROTIK_HOST` |
| Port | `MIKROTIK_PORT` — پیش‌فرض 8729 برای api-ssl، 8728 برای api |
| User / Pass | `MIKROTIK_USERNAME`, `MIKROTIK_PASSWORD` |
| TLS | فقط در `api-ssl`؛ در dev `MIKROTIK_TLS_INSECURE=true` |

**`api-ssl`:** `DialTLS` روی پورت 8729 — production و VM با اسنپ‌شات `clean-test-state`.

**`api`:** API دودویی بدون TLS روی پورت 8728 — وقتی `/ip service` فقط `api` فعال است یا handshake روی 8729 خطا می‌دهد:

```env
MIKROTIK_API=api
MIKROTIK_PORT=8728
```

اگر `MIKROTIK_API` خالی باشد و فقط `MIKROTIK_PORT=8728` ست شده باشد، پنل خودکار حالت `api` را انتخاب می‌کند.

**کتابخانه Go:** `github.com/go-routeros/routeros` — binary API برای User Manager.

اتصال **یک pool** در سرور؛ هر درخواست HTTP یک یا چند دستور sequential. Timeout: `MIKROTIK_TIMEOUT`.

## مسیرهای User Manager

### لیست کاربران

```
/user-manager/user/print
```

فیلتر سمت پنل (بعد از `print` — تابع `resolve_owner` در [business-rules.md](business-rules.md)):

- مدیر `ali` (separator `-`): نگه داشتن کاربرانی که `name` با `ali-` شروع شود **یا** `comment=panel:ali`
- ادمین orphan: هیچ الگوی نام و هیچ `panel:{slug}`

### `comment` (برچسب مالکیت)

| موضوع | مقدار |
|--------|--------|
| فرمت | `panel:{slug}` — مثال `panel:ali` |
| ست توسط پنل | هر `user/add` و `user/set` موفق از API |
| legacy | ادمین می‌تواند در Winbox برای کاربر بدون پیشوند نام ست کند |
| API مدیر | `comment` در پاسخ REST **حذف** می‌شود |
| یادداشت مشتری | فقط `vpn_user_meta.notes` — **نه** در `comment` |

### ایجاد کاربر (از پنل — مدیر)

```
/user-manager/user/add
  name=ali-reza01
  password=***
  shared-users=2
  comment=panel:ali
```

`name` = `name_prefix + local_name`؛ مدیر فقط `local_name` به API می‌فرستد.

### ویرایش کاربر (از پنل)

```
/user-manager/user/set
  .id=*ID یا numbers=...
  password=***          # اختیاری
  shared-users=3        # اختیاری
  disabled=yes|no       # غیرفعال/فعال در User Manager (نه SQLite)
  comment=panel:ali     # همیشه بازنویسی مالک — حتی اگر فقط رمز عوض شده
```

در API پنل: `disabled: true` = کاربر در روتر غیرفعال (پرچم `X` در `user print`)؛ UI با چک‌باکس «فعال در روتر» نمایش داده می‌شود.

### حذف کاربر

```
/user-manager/user/remove
  .id=*ID
```

حذف user-profileهای وابسته معمولاً توسط User Manager cascade می‌شود؛ در صورت خطا ابتدا remove رزروها.

### لیست پروفایل‌های یک کاربر

```
/user-manager/user-profile/print
  ?user=ali_reza01
```

### نسبت پروفایل (تمدید / فعال‌سازی)

```
/user-manager/user-profile/add
  user=ali_reza01
  profile=profile-open-2M-30d
```

پروفایل باید از قبل در `/user-manager/profile` وجود داشته باشد.

### حذف پروفایل رزرو

```
/user-manager/user-profile/remove
  .id=*ID
```

فقط ردیف‌هایی که **فعال نیستند** (رزرو) — UI نباید حذف پروفایل فعال را پیشنهاد دهد مگر ادمین با تأیید صریح.

## تشخیص پروفایل فعال

پس از `print` روی user-profile:

```text
state contains "active" OR (end-time parsed > now)
```

(دقیقاً با خروجی RouterOS 7.21 در محیط شما یک بار در تست دستی تطبیق داده شود.)

## خواندن `shared-users` برای quota

از `/user-manager/user/print` فیلد `shared-users` برای هر `name` که پروفایل فعال دارد.

## خطا و retry

| وضعیت | رفتار API پنل |
|--------|----------------|
| timeout | 503 `MIKROTIK_UNAVAILABLE` |
| duplicate name | 409 `NAME_TAKEN` |
| unknown profile | 400 `PROFILE_NOT_FOUND` |

حداکثر ۱ retry برای دستور idempotent (print). برای `add`/`set`/`remove` بدون retry خودکار.

## حساب API در روتر

کاربر MikroTik با گروه محدود:

- فقط `user-manager` write/read لازم
- بدون دسترسی به firewall/system

نمونه (در Winbox/CLI روتر — خارج از پنل):

```
/user add name=api-panel group=full
# در production گروه custom با policy محدود به user-manager
```

## تست بدون روتر

لایه `internal/mikrotik` interface `Client` + پیاده‌سازی `FakeClient` برای تست واحد و dev (`MIKROTIK_FAKE=true` در env اختیاری).

## تست با روتر مجازی

برای تست یکپارچه، روتر MikroTik را در **Oracle VirtualBox** (CHR) بالا بیاورید و `MIKROTIK_HOST` را به IP ماشین مجازی تنظیم کنید: [testing.md](testing.md).
