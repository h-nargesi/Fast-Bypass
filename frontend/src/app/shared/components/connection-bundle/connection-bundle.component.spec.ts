import { ComponentFixture, TestBed } from '@angular/core/testing';
import { beforeEach, describe, expect, it } from 'vitest';
import { ConnectionBundleComponent } from './connection-bundle.component';
import { ConnectionBundle } from '../../../core/models';

const bundle: ConnectionBundle = {
  username: 'ali-test',
  password: 'Secret1',
  openvpn_key_password: 'key',
  l2tp_ipsec_secret: 'l2tp',
  l2tp_server: 'vpn.example.com',
  openvpn_download_url: 'http://dl.example/dl/',
};

describe('ConnectionBundleComponent (کارت اتصال مشتری)', () => {
  let fixture: ComponentFixture<ConnectionBundleComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [ConnectionBundleComponent] }).compileComponents();
    fixture = TestBed.createComponent(ConnectionBundleComponent);
    fixture.componentRef.setInput('bundle', bundle);
    fixture.detectChanges();
  });

  it('shows Persian card title', () => {
    expect(fixture.nativeElement.textContent).toContain('اطلاعات اتصال برای مشتری');
    expect(fixture.nativeElement.textContent).toMatch(/کانال امن/);
  });

  it('has OpenVPN and L2TP tabs', () => {
    const tabs = fixture.nativeElement.querySelectorAll('.tabs button');
    expect(tabs.length).toBe(2);
    expect(tabs[0].textContent).toContain('OpenVPN');
    expect(tabs[1].textContent).toContain('L2TP');
  });

  it('preview text includes username in LTR block', () => {
    const pre = fixture.nativeElement.querySelector('pre') as HTMLPreElement;
    expect(pre.getAttribute('dir')).toBe('ltr');
    expect(pre.textContent).toContain('ali-test');
  });
});
