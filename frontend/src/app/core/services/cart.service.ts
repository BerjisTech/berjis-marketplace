import { Injectable, signal } from '@angular/core';
import { CartApiService } from './cart-api.service';

export type CartItem = {
  productId: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  variantId?: string | null;
  quantity: number;
};

@Injectable({ providedIn: 'root' })
export class CartService {
  private key = 'cart_items_v1';
  items = signal<CartItem[]>(this.load());
  syncing = signal(false);

  private load(): CartItem[] {
    try { const raw = localStorage.getItem(this.key); return raw ? JSON.parse(raw) : []; } catch { return []; }
  }
  private save(){ try { localStorage.setItem(this.key, JSON.stringify(this.items())); } catch {}
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
  constructor(private api: CartApiService) { if (this.authed()) this.syncFromServer(); }

  private authed(){
    try { const t = localStorage.getItem('auth_token'); return !!(t && t.length > 10); } catch { return false; }
  }
  syncFromServer(){
    if (!this.authed()) return;
    this.api.getCart().subscribe({ next: (r) => {
      const serverItems = (r?.data?.items || r?.items || []) as CartItem[];
      if (Array.isArray(serverItems) && serverItems.length){ this.items.set(serverItems); this.save(); }
    }, error: () => {} });
  }
  syncToServer(){
    if (!this.authed()) return; if (this.syncing()) return; this.syncing.set(true);
    const payload = this.items().map(i => ({ productId: i.productId, variantId: i.variantId||null, quantity: i.quantity }));
    this.api.putCart(payload).subscribe({ complete: () => this.syncing.set(false), error: () => this.syncing.set(false) });
  }
}
