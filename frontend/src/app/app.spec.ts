import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { beforeEach, describe, expect, it } from 'vitest';
import { App } from './app';
import { UI_MESSAGES } from './core/i18n/messages';

describe('App (Persian shell)', () => {
  let fixture: ComponentFixture<App>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [App],
      providers: [provideRouter([])],
    }).compileComponents();
    fixture = TestBed.createComponent(App);
    fixture.detectChanges();
  });

  it('renders Persian app title in header', () => {
    const h1 = fixture.nativeElement.querySelector('h1');
    expect(h1?.textContent).toBe(UI_MESSAGES.appTitle);
    expect(h1?.textContent).toMatch(/[\u0600-\u06FF]/);
  });
});
