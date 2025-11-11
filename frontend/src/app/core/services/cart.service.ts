import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import {
  CartApiResponse,
  CartApiService,
  CartItemResponse,
  CartPreviewPayload,
  CartPreviewResponse,
  CartPreviewTotals,
} from './cart-api.service';

export interface CartItem {
  productId: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  variantId?: string | null;
  quantity: number;
}

type CartTotals = CartPreviewTotals;

@Injectable({ providedIn: 'root' })
export class CartService {
  private key = 'cart_items_v1';
  private discountKey = 'cart_discount_code_v1';
  private giftCardKey = 'cart_gift_card_code_v1';
  private readonly api = inject(CartApiService);
  items = signal<CartItem[]>(this.load());
  syncing = signal(false);
  discountCode = signal<string>(this.loadCode(this.discountKey));
  giftCardCode = signal<string>(this.loadCode(this.giftCardKey));
  preview = signal<CartTotals>(this.createBaselineTotals());
  previewLoading = signal(false);
  previewError = signal('');

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

  private loadCode(key: string): string {
    try {
      return localStorage.getItem(key) ?? '';
    } catch {
      return '';
    }
  }

  private saveCode(key: string, value: string): void {
    try {
      if (value) {
        localStorage.setItem(key, value);
      } else {
        localStorage.removeItem(key);
      }
    } catch {
      return;
    }
  }

  private createBaselineTotals(): CartTotals {
    const subtotal = this.totalCents();
    const currency = this.items()[0]?.currency ?? 'USD';
    return {
      subtotalCents: subtotal,
      discountAmountCents: 0,
      giftCardAmountCents: 0,
      totalCents: subtotal,
      currency,
    };
  }

  private normalizeCode(code: string): string {
    return code ? code.trim().toUpperCase() : '';
  }

  private resetPreview(clearError = true): void {
    this.preview.set(this.createBaselineTotals());
    this.previewLoading.set(false);
    if (clearError) {
      this.previewError.set('');
    }
  }

  private hasItems(): boolean {
    return this.items().length > 0;
  }

  private buildPreviewPayload(): CartPreviewPayload {
    const payload: CartPreviewPayload = {};
    const discount = this.normalizeCode(this.discountCode());
    const gift = this.normalizeCode(this.giftCardCode());
    if (discount) {
      payload.discountCode = discount;
    }
    if (gift) {
      payload.giftCardCode = gift;
    }
    return payload;
  }

  private normalizePreview(response: CartPreviewResponse): CartTotals {
    const currency = response?.data?.currency ?? this.items()[0]?.currency ?? 'USD';
    const subtotal = Number(response?.data?.subtotalCents ?? this.totalCents());
    const discountAmount = Number(response?.data?.discountAmountCents ?? 0);
    const giftCardAmount = Number(response?.data?.giftCardAmountCents ?? 0);
    const total = Number(
      response?.data?.totalCents ?? Math.max(0, subtotal - discountAmount - giftCardAmount),
    );
    return {
      subtotalCents: subtotal,
      discountAmountCents: discountAmount,
      giftCardAmountCents: giftCardAmount,
      totalCents: total,
      currency,
    };
  }

  private resolvePreviewError(err: unknown): string {
    const status =
      (err as { status?: number }).status ??
      (err as { statusCode?: number }).statusCode ??
      (err as { error?: { status?: number } }).error?.status;
    const message = (err as { error?: { message?: string } }).error?.message;
    if (status === 400) {
      return message || 'Discount or gift card code not accepted.';
    }
    if (status === 401) {
      return 'Sign in to apply discounts or gift cards.';
    }
    if (status === 404) {
      return message || 'Code not found for this shop.';
    }
    if (status === 409) {
      return message || 'Code cannot be applied to this cart.';
    }
    return message || 'Could not refresh pricing.';
  }

  clear(): void {
    this.items.set([]);
    this.save();
    this.resetPreview();
    if (this.authed()) {
      this.syncToServer();
    }
  }
  count(): number { return this.items().reduce((a,b)=> a + (b.quantity||0), 0); }
  totalCents(): number { return this.items().reduce((a,b)=> a + b.priceCents * b.quantity, 0); }

  add(item: Omit<CartItem,'quantity'>, quantity = 1): void {
    const q = Math.max(1, quantity|0);
    const key = (i: CartItem) => i.productId + '::' + (i.variantId || '');
    const items = [...this.items()];
    const idx = items.findIndex(i => key(i) === (item.productId + '::' + (item.variantId || '')));
    if (idx >= 0) items[idx] = { ...items[idx], quantity: items[idx].quantity + q };
    else items.push({ ...item, quantity: q });
    this.items.set(items);
    this.save();
    this.resetPreview();
    if (this.authed()) {
      this.syncToServer();
    }
  }
  update(productId: string, variantId: string | null, quantity: number): void {
    const items = this.items()
      .map(i => (i.productId === productId && (i.variantId||null) === (variantId||null))
        ? { ...i, quantity: Math.max(0, quantity|0) }
        : i)
      .filter(i => i.quantity > 0);
    this.items.set(items);
    this.save();
    this.resetPreview();
    if (this.authed()) {
      this.syncToServer();
    }
  }
  remove(productId: string, variantId: string | null): void { this.update(productId, variantId, 0); }
  constructor() {
    this.resetPreview();
    if (this.authed()) {
      this.syncFromServer();
    }
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
  syncFromServer(): void {
    if (!this.authed()) return;
    this.api.getCart().subscribe({
      next: (response) => {
        const raw = response?.data?.items ?? response?.items ?? [];
        const mapped = this.mapServerItems(raw ?? []);
        this.items.set(mapped);
        this.save();
        this.resetPreview();
        if (mapped.length > 0) {
          void this.refreshPreview();
        }
      },
      error: () => {
        this.resetPreview();
      },
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
  handleAuthLogout(): void {
    this.syncing.set(false);
    this.resetPreview();
  }
  syncToServer(): void {
    if (!this.authed()) return;
    if (this.syncing()) return;
    this.syncing.set(true);
    const payload = this.items().map(i => ({ productId: i.productId, quantity: i.quantity }));
    this.api.putCart(payload).subscribe({
      next: () => undefined,
      error: () => {
        this.syncing.set(false);
        this.previewError.set('Could not sync cart.');
        this.resetPreview(false);
      },
      complete: () => {
        this.syncing.set(false);
        void this.refreshPreview();
      },
    });
  }

  async refreshPreview(): Promise<CartTotals> {
    const payload = this.buildPreviewPayload();
    const hasCodes = !!payload.discountCode || !!payload.giftCardCode;
    if (!this.hasItems()) {
      this.previewError.set('');
      this.resetPreview();
      return this.preview();
    }
    if (!this.authed()) {
      if (hasCodes) {
        this.previewError.set('Sign in to apply discounts or gift cards.');
      } else {
        this.previewError.set('');
      }
      this.resetPreview(!hasCodes);
      return this.preview();
    }
    this.previewLoading.set(true);
    this.previewError.set('');
    try {
      const response: CartPreviewResponse = await firstValueFrom(
        this.api.previewCart(payload),
      );
      const totals = this.normalizePreview(response);
      this.preview.set(totals);
      return totals;
    } catch (err) {
      this.previewError.set(this.resolvePreviewError(err));
      this.resetPreview(false);
      return this.preview();
    } finally {
      this.previewLoading.set(false);
    }
  }

  async applyDiscountCode(code: string): Promise<CartTotals> {
    const value = this.normalizeCode(code);
    this.discountCode.set(value);
    this.saveCode(this.discountKey, value);
    return this.refreshPreview();
  }

  async clearDiscountCode(): Promise<CartTotals> {
    this.discountCode.set('');
    this.saveCode(this.discountKey, '');
    return this.refreshPreview();
  }

  async applyGiftCardCode(code: string): Promise<CartTotals> {
    const value = this.normalizeCode(code);
    this.giftCardCode.set(value);
    this.saveCode(this.giftCardKey, value);
    return this.refreshPreview();
  }

  async clearGiftCardCode(): Promise<CartTotals> {
    this.giftCardCode.set('');
    this.saveCode(this.giftCardKey, '');
    return this.refreshPreview();
  }

  clearPreviewError(): void {
    this.previewError.set('');
  }
}
