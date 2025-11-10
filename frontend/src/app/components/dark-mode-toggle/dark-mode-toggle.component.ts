import { Component, inject } from '@angular/core';
import { ThemeService } from '../../core/services/theme.service';

@Component({
  selector: 'app-dark-mode-toggle',
  standalone: true,
  imports: [],
  templateUrl: './dark-mode-toggle.component.html',
  styleUrl: './dark-mode-toggle.component.css'
})
export class DarkModeToggleComponent {
  private readonly theme = inject(ThemeService);
  readonly isDark = this.theme.isDark;

  toggleTheme(): void {
    this.theme.toggle();
  }
}
