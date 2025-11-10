import { Injectable, NgZone, inject } from '@angular/core';
import { CartService } from './cart.service';
import { WishlistService } from './wishlist.service';

@Injectable({ providedIn: 'root' })
export class AuthSyncService {
  private lastToken = this.currentToken();
  private syncing = false;

  private readonly cart = inject(CartService);
  private readonly wishlist = inject(WishlistService);
  private readonly zone = inject(NgZone);

  constructor() {
    this.zone.run(() => this.detectChanges(true));
    window.addEventListener('storage', (event) => {
      if (event.key === 'auth_token') {
        this.zone.run(() => this.detectChanges(true));
      }
    });
    window.addEventListener('focus', () => this.zone.run(() => this.detectChanges(true)));
    document.addEventListener('visibilitychange', () => {
      if (!document.hidden) {
        this.zone.run(() => this.detectChanges(true));
      }
    });
  }

  private currentToken(): string {
    try {
      return localStorage.getItem('auth_token') || '';
    } catch {
      return '';
    }
  }

  private detectChanges(forceRefresh = false): void {
    if (this.syncing) {
      return;
    }
    this.syncing = true;
    Promise.resolve().then(async () => {
      const currentToken = this.currentToken();
      const authed = currentToken.length > 10;
      const previouslyAuthed = this.lastToken.length > 10;
      if (authed !== previouslyAuthed) {
        this.lastToken = currentToken;
        if (authed) {
          await Promise.allSettled([
            this.cart.reconcileAfterLogin(),
            this.wishlist.reconcileAfterLogin()
          ]);
        } else {
          this.cart.handleAuthLogout();
          this.wishlist.handleAuthLogout();
        }
      } else if (authed && forceRefresh) {
        this.cart.syncFromServer();
        this.wishlist.syncFromServer();
      }
    }).finally(() => {
      this.syncing = false;
    });
  }
}
