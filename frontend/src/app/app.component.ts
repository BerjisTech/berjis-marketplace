import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterOutlet } from '@angular/router';
import { AuthSyncService } from './core/services/auth-sync.service';
import { ThemeService } from './core/services/theme.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterOutlet],
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  // Kick off auth-driven cart/wishlist sync on bootstrap.
  private readonly authSync = inject(AuthSyncService);
  // Ensure theme preferences hydrate immediately.
  private readonly theme = inject(ThemeService);
}
