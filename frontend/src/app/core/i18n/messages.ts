/** پیام‌های ثابت UI — فارسی */
export const UI_MESSAGES = {
  appTitle: 'پنل سبلان',
  login: 'ورود',
  logout: 'خروج',
  save: 'ذخیره',
  cancel: 'انصراف',
  delete: 'حذف',
  confirmDeleteTitle: 'تأیید حذف',
  confirmDeleteBody: 'آیا از حذف این کاربر مطمئن هستید؟',
  confirm: 'تأیید',
  copyDone: 'کپی شد',
  emptyUsers: 'هنوز کاربری ثبت نکرده‌اید',
  emptyRenewals: 'هنوز تمدیدی ثبت نشده',
  quotaExceeded: 'سقف تعداد اتصال همزمان پر شده است',
  managerDisabled: 'حساب مدیر غیرفعال است',
  unauthorized: 'نام کاربری یا رمز اشتباه است',
  orphanLabel: 'بدون مدیر',
  currencySuffix: 'ریال',
} as const;

export const API_ERROR_MESSAGES: Record<string, string> = {
  QUOTA_EXCEEDED: 'سقف تعداد اتصال همزمان پر شده است',
  QUOTA_BELOW_USAGE: 'سقف کمتر از مصرف فعلی است',
  NAME_TAKEN: 'این نام کاربری قبلاً ثبت شده است',
  SLUG_OVERLAPS: 'پیشوند با مدیر دیگر همپوشانی دارد',
  NOT_OWNER: 'کاربر متعلق به شما نیست',
  MANAGER_DISABLED: 'حساب مدیر غیرفعال است',
  MIKROTIK_UNAVAILABLE: 'ارتباط با روتر برقرار نشد',
  INVALID_CURRENT_PASSWORD: 'رمز فعلی اشتباه است',
  UNAUTHORIZED: 'نام کاربری یا رمز اشتباه است',
};

export function apiErrorMessage(code: string, fallback?: string): string {
  return API_ERROR_MESSAGES[code] ?? fallback ?? 'خطای ناشناخته';
}
