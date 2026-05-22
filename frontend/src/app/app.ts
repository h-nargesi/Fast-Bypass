import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { UI_MESSAGES } from './core/i18n/messages';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: `
    <div class="shell">
      <header>
        <h1>{{ title }}</h1>
      </header>
      <main>
        <router-outlet />
      </main>
    </div>
  `,
  styles: `
    .shell {
      min-height: 100vh;
    }
    header {
      background: #1565c0;
      color: #fff;
      padding: 0.75rem 1.25rem;
    }
    h1 {
      margin: 0;
      font-size: 1.15rem;
      font-weight: 600;
    }
    main {
      padding: 1rem 1.25rem;
    }
  `,
})
export class App {
  readonly title = UI_MESSAGES.appTitle;
}
