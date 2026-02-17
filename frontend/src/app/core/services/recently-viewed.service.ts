import { Injectable, signal } from '@angular/core';

export interface RecentlyViewedItem {
  uuid: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  shopName?: string;
  shopSlug?: string;
  viewedAt: number;
}

const STORAGE_KEY = 'berjis_recently_viewed';
const MAX_ITEMS = 12;

@Injectable({ providedIn: 'root' })
export class RecentlyViewedService {
  readonly items = signal<RecentlyViewedItem[]>(this.load());

  add(product: Omit<RecentlyViewedItem, 'viewedAt'>): void {
    const current = this.items().filter(i => i.uuid !== product.uuid);
    const updated = [{ ...product, viewedAt: Date.now() }, ...current].slice(0, MAX_ITEMS);
    this.items.set(updated);
    this.save(updated);
  }

  getExcluding(uuid: string, limit = 6): RecentlyViewedItem[] {
    return this.items().filter(i => i.uuid !== uuid).slice(0, limit);
  }

  private load(): RecentlyViewedItem[] {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? JSON.parse(raw) : [];
    } catch {
      return [];
    }
  }

  private save(items: RecentlyViewedItem[]): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
    } catch { /* quota exceeded — ignore */ }
  }
}
