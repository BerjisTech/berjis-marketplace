import { Component, OnInit, signal } from '@angular/core';

@Component({
  selector: 'app-dark-mode-toggle',
  standalone: true,
  imports: [],
  templateUrl: './dark-mode-toggle.component.html',
  styleUrl: './dark-mode-toggle.component.css'
})
export class DarkModeToggleComponent implements OnInit {
  isDark = signal(false);
  copyRightYear = new Date().getFullYear();
  constructor() { }

  ngOnInit(): void {
    const persisted = (localStorage.getItem('theme') || '').toLowerCase();
    this.setTheme(persisted === 'dark' ? 'dark' : 'light');
  }

  toggleTheme() { this.setTheme(this.isDark() ? 'light' : 'dark'); }
  private setTheme(mode: 'light' | 'dark') { this.isDark.set(mode === 'dark'); document.documentElement.classList.toggle('dark', mode === 'dark'); localStorage.setItem('theme', mode); }
}
