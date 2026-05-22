import { Pipe, PipeTransform } from '@angular/core';
import { formatJalaliDate, formatJalaliDateTime } from '../../core/date/jalali';

@Pipe({ name: 'jalaliDate', standalone: true })
export class JalaliDatePipe implements PipeTransform {
  transform(value: string | null | undefined, mode: 'date' | 'datetime' = 'datetime'): string {
    return mode === 'date' ? formatJalaliDate(value) : formatJalaliDateTime(value);
  }
}
