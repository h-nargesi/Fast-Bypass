/** مبلغ با جداکننده هزارگان فارسی + پسوند ریال */
export function formatRial(amount: number | null | undefined): string {
  if (amount == null) {
    return 'ثبت نشده';
  }
  const formatted = new Intl.NumberFormat('fa-IR').format(amount);
  return `${formatted} ریال`;
}
