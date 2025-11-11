import { Component, OnInit, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { CartService } from '../app/core/services/cart.service';
import { PricingService } from '../app/core/services/pricing.service';

@Component({
  standalone: true,
  selector: 'app-cart-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './cart-page.component.html',
  styleUrls: ['./cart-page.component.css']
})
export class CartPageComponent implements OnInit {
  private readonly cart = inject(CartService);
  private readonly pricing = inject(PricingService);
  readonly items = this.cart.items;
  readonly pricingConfig = this.pricing.config;
  readonly preview = this.cart.preview;
  readonly previewLoading = this.cart.previewLoading;
  readonly previewError = this.cart.previewError;
  readonly discountCode = this.cart.discountCode;
  readonly giftCardCode = this.cart.giftCardCode;
  discountInput = this.discountCode();
  giftCardInput = this.giftCardCode();

  readonly currency = computed(() => this.preview().currency || this.items()[0]?.currency || 'USD');
  readonly subtotalCents = computed(() => this.preview().subtotalCents ?? this.cart.totalCents());
  readonly discountCents = computed(() => this.preview().discountAmountCents ?? 0);
  readonly giftCardCents = computed(() => this.preview().giftCardAmountCents ?? 0);
  readonly netCents = computed(() =>
    Math.max(0, (this.preview().totalCents ?? (this.subtotalCents() - this.discountCents() - this.giftCardCents()))));
  readonly taxCents = computed(() =>
    Math.round(this.netCents() * (this.pricingConfig().taxRatePercent / 100)));
  readonly shippingCents = computed(() =>
    (this.items().length > 0 ? this.pricingConfig().shippingFlatCents : 0));
  readonly totalCents = computed(() => this.netCents() + this.taxCents() + this.shippingCents());

  ngOnInit(): void {
    void this.cart.refreshPreview();
  }

  dec(i: number): void {
    this.cart.clearPreviewError();
    const it = this.items()[i];
    this.cart.update(it.productId, it.variantId || null, it.quantity - 1);
  }
  inc(i: number): void {
    this.cart.clearPreviewError();
    const it = this.items()[i];
    this.cart.update(it.productId, it.variantId || null, it.quantity + 1);
  }
  remove(i: number): void {
    this.cart.clearPreviewError();
    const it = this.items()[i];
    this.cart.remove(it.productId, it.variantId || null);
  }
  clear(): void {
    this.cart.clearPreviewError();
    this.cart.clear();
  }

  async applyDiscount(): Promise<void> {
    this.cart.clearPreviewError();
    await this.cart.applyDiscountCode(this.discountInput);
    this.discountInput = this.discountCode();
  }

  async clearDiscount(): Promise<void> {
    this.cart.clearPreviewError();
    await this.cart.clearDiscountCode();
    this.discountInput = '';
  }

  async applyGiftCard(): Promise<void> {
    this.cart.clearPreviewError();
    await this.cart.applyGiftCardCode(this.giftCardInput);
    this.giftCardInput = this.giftCardCode();
  }

  async clearGiftCard(): Promise<void> {
    this.cart.clearPreviewError();
    await this.cart.clearGiftCardCode();
    this.giftCardInput = '';
  }
}

