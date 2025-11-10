import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { CartApiResponse, CartApiService, CartItemResponse } from './cart-api.service';

export interface CartItem {
  productId: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  variantId?: string | null;
  quantity: number;
}

@Injectable({ providedIn: 'root' })
export class CartService {
  private key = 'cart_items_v1';
  private readonly api = inject(CartApiService);
  items = signal<CartItem[]>(this.load());
  syncing = signal(false);

  private load(): CartItem[] {
    try {
      const raw = localStorage.getItem(this.key);
      return raw ? (JSON.parse(raw) as CartItem[]) : [];
    } catch {
      return [];
    }
  }
  private save(): void {
    try {
      localStorage.setItem(this.key, JSON.stringify(this.items()));
    } catch {
      return;
    }
  }

  clear(){ this.items.set([]); this.save(); }
  count(){ return this.items().reduce((a,b)=> a + (b.quantity||0), 0); }
  totalCents(){ return this.items().reduce((a,b)=> a + b.priceCents * b.quantity, 0); }

  add(item: Omit<CartItem,'quantity'>, quantity = 1){
    const q = Math.max(1, quantity|0);
    const key = (i: CartItem) => i.productId + '::' + (i.variantId || '');
    const items = [...this.items()];
    const idx = items.findIndex(i => key(i) === (item.productId + '::' + (item.variantId || '')));
    if (idx >= 0) items[idx] = { ...items[idx], quantity: items[idx].quantity + q };
    else items.push({ ...item, quantity: q });
    this.items.set(items); this.save(); this.syncToServer();
  }
  update(productId: string, variantId: string | null, quantity: number){
    const items = this.items().map(i => (i.productId === productId && (i.variantId||null) === (variantId||null)) ? { ...i, quantity: Math.max(0, quantity|0) } : i).filter(i => i.quantity > 0);
    this.items.set(items); this.save(); this.syncToServer();
  }
  remove(productId: string, variantId: string | null){ this.update(productId, variantId, 0); }
  constructor() {
    if (this.authed()) this.syncFromServer();
  }

  private authed(){
    try { const t = localStorage.getItem('auth_token'); return !!(t && t.length > 10); } catch { return false; }
  }
  private mapServerItems(raw: CartItemResponse[]): CartItem[] {
    return Array.isArray(raw) ? raw.map((item) => ({
      productId: item.productId || item.productUuid || item.product_id || '',
      title: item.title ?? '',
      priceCents: Number(item.priceCents ?? item.price_cents ?? 0),
      currency: item.currency ?? 'USD',
      imageUrl: item.imageUrl ?? item.image_url ?? undefined,
      variantId: item.variantId ?? item.variant_id ?? null,
      quantity: Math.max(1, Number(item.quantity ?? 1))
    })).filter(i => i.productId) : [];
  }
  private mergeItems(server: CartItem[], local: CartItem[]): CartItem[] {
    const combined = new Map<string, CartItem>();
    const keyOf = (item: CartItem) => item.productId + '::' + (item.variantId || '');
    const include = (item: CartItem) => {
      const key = keyOf(item);
      const existing = combined.get(key);
      if (existing) {
        combined.set(key, { ...existing, quantity: existing.quantity + item.quantity });
      } else {
        combined.set(key, { ...item });
      }
    };
    server.forEach(include);
    local.forEach(include);
    return Array.from(combined.values());
  }
  syncFromServer(){
    if (!this.authed()) return;
    this.api.getCart().subscribe((response) => {
      const raw = response?.data?.items ?? response?.items ?? [];
      const mapped = this.mapServerItems(raw ?? []);
      this.items.set(mapped);
      this.save();
    });
  }
  async reconcileAfterLogin(): Promise<void> {
    if (!this.authed()) return;
    const localItems = [...this.items()];
    try {
      const response: CartApiResponse = await firstValueFrom(this.api.getCart());
      const serverItems = this.mapServerItems(response?.data?.items ?? response?.items ?? []);
      const merged = this.mergeItems(serverItems, localItems);
      this.items.set(merged);
      this.save();
      await firstValueFrom(this.api.putCart(merged.map(i => ({ productId: i.productId, quantity: i.quantity }))));
    } catch {
      await this.pushLocalState();
    } finally {
      this.syncing.set(false);
      this.syncFromServer();
    }
  }
  private async pushLocalState(): Promise<void> {
    try {
      await firstValueFrom(this.api.putCart(this.items().map(i => ({ productId: i.productId, quantity: i.quantity }))));
    } catch {
      return;
    }
  }
  handleAuthLogout(){ this.syncing.set(false); }
  syncToServer(){
    if (!this.authed()) return; if (this.syncing()) return; this.syncing.set(true);
    const payload = this.items().map(i => ({ productId: i.productId, quantity: i.quantity }));
    this.api.putCart(payload).subscribe({ complete: () => this.syncing.set(false), error: () => this.syncing.set(false) });
  }
}
