import { Component, signal, computed, inject, OnInit, DoCheck } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, NgForm, AbstractControl } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { CartService, CartItem } from '../app/core/services/cart.service';
import { CreateOrderPayload, OrderService } from '../app/core/services/order.service';
import { PricingService } from '../app/core/services/pricing.service';

@Component({
  standalone: true,
  selector: 'app-checkout-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './checkout-page.component.html',
  styleUrls: ['./checkout-page.component.css']
})
export class CheckoutPageComponent implements OnInit, DoCheck {
  step = signal<'shipping'|'payment'|'review'|'done'>('shipping');
  shipping = { name: '', email: '', address: '', city: '', country: '', zip: '' };
  payment = { cardName: '', cardNumber: '', exp: '', cvc: '' };
  placing = signal(false);
  orderId = signal<string>('');
  orderError = signal('');
  private readonly cart = inject(CartService);
  private readonly orders = inject(OrderService);
  private readonly router = inject(Router);
  private readonly pricing = inject(PricingService);
  items = this.cart.items;
  pricingConfig = this.pricing.config;
  preview = this.cart.preview;
  previewLoading = this.cart.previewLoading;
  previewError = this.cart.previewError;
  discountCode = this.cart.discountCode;
  giftCardCode = this.cart.giftCardCode;
  discountInput = this.discountCode();
  giftCardInput = this.giftCardCode();
  currency = computed(() => this.preview().currency || this.items()[0]?.currency || 'USD');
  subtotalCents = computed(()=> this.preview().subtotalCents ?? this.cart.totalCents());
  discountCents = computed(()=> this.preview().discountAmountCents ?? 0);
  giftCardCents = computed(()=> this.preview().giftCardAmountCents ?? 0);
  netCents = computed(()=> Math.max(0, this.preview().totalCents ?? (this.subtotalCents() - this.discountCents() - this.giftCardCents())));
  taxCents = computed(()=> Math.round(this.netCents() * (this.pricingConfig().taxRatePercent / 100)));
  shippingCents = computed(()=> (this.items().length > 0 ? this.pricingConfig().shippingFlatCents : 0));
  totalCents = computed(()=> this.netCents() + this.taxCents() + this.shippingCents());

  advanceFromShipping(form: NgForm){
    if (!this.validateForm(form)) { return; }
    this.step.set('payment');
  }
  advanceFromPayment(form: NgForm){
    if (!this.validateForm(form)) { return; }
    this.step.set('review');
    this.orderError.set('');
    void this.cart.refreshPreview();
  }
  back(){ if (this.step()==='payment') this.step.set('shipping'); else if (this.step()==='review') this.step.set('payment'); }

  async placeOrder(){
    this.orderError.set('');
    this.cart.clearPreviewError();
    this.placing.set(true);
    await this.cart.refreshPreview();
    const lineItems = this.items().map(item => this.toOrderItem(item));
    const body: CreateOrderPayload = {
      shipping: { ...this.shipping },
      items: lineItems,
      payment: { method: 'mock' },
      subtotalCents: this.subtotalCents(),
      discountCode: this.discountCode() || undefined,
      giftCardCode: this.giftCardCode() || undefined,
    };
    try {
      const res = await this.orders.createOrder(body);
      this.orderId.set(res.id);
      this.cart.clear();
      await this.cart.clearDiscountCode();
      await this.cart.clearGiftCardCode();
      this.placing.set(false);
      try { localStorage.removeItem('checkout_progress'); } catch { /* ignore */ }
      this.router.navigate(['/checkout/confirmation', res.id]);
    } catch (err: unknown) {
      this.placing.set(false);
      let message = 'Could not place order. Please review your codes and try again.';
      if (typeof err === 'object' && err !== null) {
        const httpError = err as { error?: { message?: string } };
        if (httpError.error && typeof httpError.error.message === 'string' && httpError.error.message.trim()) {
          message = httpError.error.message.trim();
        } else if ('message' in err && typeof (err as { message?: string }).message === 'string' && (err as { message?: string }).message) {
          message = (err as { message: string }).message;
        }
      } else if (err instanceof Error && err.message) {
        message = err.message;
      }
      this.orderError.set(message);
    }
  }
  // Persist checkout progress
  ngOnInit(){
    try {
      const saved = JSON.parse(localStorage.getItem('checkout_progress')||'null');
      if (saved) { this.shipping = saved.shipping||this.shipping; this.payment = saved.payment||this.payment; if (saved.step) this.step.set(saved.step); }
    } catch {
      return;
    }
    void this.cart.refreshPreview();
  }
  ngDoCheck(){
    try { localStorage.setItem('checkout_progress', JSON.stringify({ shipping: this.shipping, payment: this.payment, step: this.step() })); } catch {
      return;
    }
  }

  private toOrderItem(item: CartItem) {
    return {
      productId: item.productId,
      quantity: item.quantity,
    };
  }

  fieldInvalid(form: NgForm, control: string): boolean {
    const ctrl = form.controls[control];
    return !!ctrl && ctrl.invalid && (ctrl.touched || form.submitted);
  }

  hasError(form: NgForm, control: string, key: string): boolean {
    const ctrl = form.controls[control];
    return !!ctrl && !!ctrl.errors?.[key] && (ctrl.touched || form.submitted);
  }

  private validateForm(form: NgForm): boolean {
    if (form.valid) {
      return true;
    }
    Object.values(form.controls).forEach(control => {
      const inner: AbstractControl | undefined = (control as { control?: AbstractControl }).control;
      inner?.markAsTouched();
      inner?.updateValueAndValidity();
    });
    return false;
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

