import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import {
  WishlistApiResponse,
  WishlistApiService,
  WishlistItemResponse,
  WishlistPayloadItem,
} from './wishlist-api.service';
import { CartService } from './cart.service';

export interface WishlistItem {
  productId: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  shopName?: string;
  shopSlug?: string;
  addedAt?: string;
}

@Injectable({ providedIn: 'root' })
export class WishlistService {
  private readonly storageKey = 'wishlist_items_v1';
  private readonly api = inject(WishlistApiService);
  private readonly cart = inject(CartService);
  items = signal<WishlistItem[]>(this.load());
  syncing = signal(false);

  constructor() {
    if (this.authed()) {
      this.syncFromServer();
    }
  }

  private load(): WishlistItem[] {
    try {
      const raw = localStorage.getItem(this.storageKey);
      return raw ? (JSON.parse(raw) as WishlistItem[]) : [];
    } catch {
      return [];
    }
  }

  private save(): void {
    try {
      localStorage.setItem(this.storageKey, JSON.stringify(this.items()));
    } catch {
      return;
    }
  }

  private authed(): boolean {
    try {
      const token = localStorage.getItem('auth_token');
      return !!(token && token.length > 10);
    } catch {
      return false;
    }
  }

  private payload(): WishlistPayloadItem[] {
    return this.items().map(item => ({ productId: item.productId }));
  }

  private mapServerItems(raw: WishlistItemResponse[]): WishlistItem[] {
    return Array.isArray(raw)
      ? raw
          .map(item => ({
            productId: item.productUuid || item.productId || '',
            title: item.title ?? '',
            priceCents: item.priceCents ?? 0,
            currency: item.currency ?? 'USD',
            imageUrl: item.imageUrl ?? undefined,
            shopName: item.shopName,
            shopSlug: item.shopSlug,
            addedAt: item.addedAt,
          }))
          .filter(item => item.productId)
      : [];
  }

  private mergeItems(server: WishlistItem[], local: WishlistItem[]): WishlistItem[] {
    const combined = new Map<string, WishlistItem>();
    server.forEach(item => combined.set(item.productId, { ...item }));
    local.forEach(item => {
      if (!combined.has(item.productId)) {
        combined.set(item.productId, { ...item });
      }
    });
    return Array.from(combined.values());
  }

  add(item: WishlistItem): void {
    if (this.items().some(i => i.productId === item.productId)) {
      return;
    }
    const next = [...this.items(), { ...item, addedAt: new Date().toISOString() }];
    this.items.set(next);
    this.save();
    this.syncToServer();
  }

  remove(productId: string): void {
    this.items.set(this.items().filter(item => item.productId !== productId));
    this.save();
    this.syncToServer();
  }

  clear(): void {
    this.items.set([]);
    this.save();
    this.syncToServer();
  }

  moveToCart(productId: string): void {
    const item = this.items().find(i => i.productId === productId);
    if (!item) {
      return;
    }
    this.cart.add(
      {
        productId: item.productId,
        title: item.title,
        priceCents: item.priceCents,
        currency: item.currency,
        imageUrl: item.imageUrl,
        variantId: null,
      },
      1,
    );
    this.remove(productId);
  }

  syncFromServer(): void {
    if (!this.authed()) {
      return;
    }
    this.api.getWishlist().subscribe(response => {
      const raw = response?.data?.items ?? response?.items ?? [];
      const mapped = this.mapServerItems(raw ?? []);
      this.items.set(mapped);
      this.save();
    });
  }

  syncToServer(): void {
    if (!this.authed() || this.syncing()) {
      return;
    }
    this.syncing.set(true);
    this.api.putWishlist(this.payload()).subscribe({
      complete: () => this.syncing.set(false),
      error: () => this.syncing.set(false),
    });
  }

  async reconcileAfterLogin(): Promise<void> {
    if (!this.authed()) {
      return;
    }
    const localItems = [...this.items()];
    try {
      const response: WishlistApiResponse = await firstValueFrom(this.api.getWishlist());
      const serverItems = this.mapServerItems(response?.data?.items ?? response?.items ?? []);
      const merged = this.mergeItems(serverItems, localItems);
      this.items.set(merged);
      this.save();
      await firstValueFrom(this.api.putWishlist(merged.map(item => ({ productId: item.productId }))));
    } catch {
      await this.pushLocalState();
    } finally {
      this.syncing.set(false);
      this.syncFromServer();
    }
  }

  private async pushLocalState(): Promise<void> {
    try {
      await firstValueFrom(this.api.putWishlist(this.payload()));
    } catch {
      return;
    }
  }

  handleAuthLogout(): void {
    this.syncing.set(false);
  }
}
