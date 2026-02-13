import { HttpClient } from '@angular/common/http';
import { Injectable, Signal, inject, signal } from '@angular/core';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

@Injectable({ providedIn: 'root' })
export class ShopStateService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;
  private readonly storageKey = 'marketplace.activeShop';

  private readonly shopsSignal = signal<ShopSummary[]>([]);
  private readonly activeSlugSignal = signal<string>(this.restoreStoredSlug());
  private readonly loadingSignal = signal<boolean>(false);
  private loaded = false;

  readonly shops: Signal<ShopSummary[]> = this.shopsSignal.asReadonly();
  readonly activeShopSlug: Signal<string> = this.activeSlugSignal.asReadonly();
  readonly loading: Signal<boolean> = this.loadingSignal.asReadonly();

  /**
   * Ensure we have an up-to-date cache of the user's shops.
   * Safe to call multiple times; only triggers a fetch when needed.
   */
  ensureLoaded(): void {
    if (this.loaded || this.loadingSignal()) {
      return;
    }
    this.loadingSignal.set(true);
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          this.applyShops(response?.data ?? []);
          this.loadingSignal.set(false);
          this.loaded = true;
        },
        error: () => {
          this.loadingSignal.set(false);
        },
      });
  }

  /**
   * Replace the cached shops list (e.g. after guard or manual refresh)
   * and keep the active shop slug in sync. Returns the resolved slug.
   */
  applyShops(list: ShopSummary[]): string {
    this.shopsSignal.set(list);
    const nextSlug = this.resolveSlug(list, this.activeSlugSignal());
    this.setActiveShopSlug(nextSlug);
    this.loaded = true;
    return nextSlug;
  }

  setActiveShopSlug(slug: string): void {
    if (this.activeSlugSignal() === slug) {
      return;
    }
    this.activeSlugSignal.set(slug);
    this.persistSlug(slug);
  }

  hasStoredShop(): boolean {
    return !!this.activeSlugSignal();
  }

  /**
   * Force a fresh download of shops (e.g. after creating a store).
   */
  refresh(): void {
    this.loaded = false;
    this.ensureLoaded();
  }

  private resolveSlug(list: ShopSummary[], current: string): string {
    if (current && list.some((shop) => shop.slug === current)) {
      return current;
    }
    return list[0]?.slug ?? '';
  }

  private persistSlug(value: string): void {
    if (typeof window === 'undefined' || !window?.localStorage) {
      return;
    }
    try {
      if (value) {
        window.localStorage.setItem(this.storageKey, value);
      } else {
        window.localStorage.removeItem(this.storageKey);
      }
    } catch {
      // ignore storage errors (private mode, etc.)
    }
  }

  private restoreStoredSlug(): string {
    if (typeof window === 'undefined' || !window?.localStorage) {
      return '';
    }
    try {
      return window.localStorage.getItem(this.storageKey) ?? '';
    } catch {
      return '';
    }
  }
}
