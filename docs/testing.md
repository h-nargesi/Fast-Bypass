# تست و محیط توسعه

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
/ip address add address=192.168.56.2/24 interface=ether2
```

4. در `.env` پنل: `MIKROTIK_HOST=192.168.56.2`

**گزینه B — Bridged**

- Adapter به کارت فیزیکی Wi‑Fi/Ethernet میزبان bridge شود؛ روتر IP از همان subnet DHCP می‌گیرد (مثلاً `192.168.1.x`).

### ۴. اولین بالا آمدن روتر

1. VM را Start کنید؛ با **Winbox** (Neighbours) یا serial به روتر وصل شوید.
2. نسخه RouterOS را **7.21+** نگه دارید.
3. پکیج **User Manager** را نصب/فعال کنید (`/system package print`).

### ۵. آماده‌سازی برای پنل

```routeros
# کاربر API (در production گروه محدودتر)
/user add name=api-panel group=full password=StrongApiPass

# فعال‌سازی API-SSL
/ip service set api-ssl disabled=no port=8729

# پروفایل نمونه (نام را با DEFAULT_PROFILE در .env هماهنگ کنید)
/user-manager profile add name=profile-open-2M-30d

# منطقه زمانی — هم‌خوان با سرور پنل (تهران)
/system clock set time-zone-name=Asia/Tehran
```

از میزبان تست اتصال:

```bash
# اگر openssl در دسترس است — یا فقط پنل را با .env اجرا کنید
nc -zv 192.168.56.2 8729
```

### ۶. تنظیم `.env` پنل (توسعه)

```bash
cp .env.example .env
```

| متغیر | مقدار نمونه (VirtualBox Host-Only) |
|--------|-------------------------------------|
| `MIKROTIK_HOST` | `192.168.56.2` |
| `MIKROTIK_PORT` | `8729` |
| `MIKROTIK_USERNAME` | `api-panel` |
| `MIKROTIK_PASSWORD` | همان رمز روتر |
| `MIKROTIK_TLS_INSECURE` | `true` در dev اگر گواهی self-signed است |

پس از پیاده‌سازی backend/frontend، همان [اجرای سریع](../README.md#اجرای-سریع-پس-از-پیاده‌سازی) در README.

## تست بدون روتر (اختیاری)

برای unit testهای Go می‌توان `MIKROTIK_FAKE=true` (فاز پیاده‌سازی) و mock کلاینت RouterOS استفاده کرد — جایگزین تست یکپارچه با User Manager واقعی نیست.

## عیب‌یابی رایج VirtualBox

| مشکل | بررسی |
|------|--------|
| Winbox روتر را نمی‌بیند | کابل مجازی؛ فایروال میزبان |
| پنل `MIKROTIK_UNAVAILABLE` | IP/pورت؛ `api-ssl` فعال؛ ping از میزبان |
| TLS خطا | `MIKROTIK_TLS_INSECURE=true` در dev |
| User Manager خالی | پکیج user-manager نصب؛ نسخه ≥ 7.21 |

## پذیرش دستی در VM

- [ ] از میزبان به `MIKROTIK_HOST:8729` دسترسی TCP
- [ ] `/user-manager/user print` از API یا Winbox
- [ ] پروفایل `profile-open-2M-30d` (یا نام env) وجود دارد
- [ ] ایجاد کاربر با پیشوند (مثلاً `ali-test01`)؛ در روتر `comment=panel:ali` بررسی شود
- [ ] legacy: کاربر `reza` + `comment=panel:ali` در لیست مدیر `ali` دیده شود
- [ ] مدیر در API پاسخ `mikrotik_comment` نداشته باشد
