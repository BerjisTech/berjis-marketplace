import { Component, signal, computed, inject, OnInit, DoCheck, OnDestroy, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, NgForm, AbstractControl } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { CartService, CartItem } from '../app/core/services/cart.service';
import { CreateOrderPayload, OrderService } from '../app/core/services/order.service';
import { PricingService } from '../app/core/services/pricing.service';
import { StripeService } from '../app/core/services/stripe.service';
import { BreadcrumbsComponent } from '../app/shared/components/breadcrumbs/breadcrumbs.component';

@Component({
  standalone: true,
  selector: 'app-checkout-page',
  imports: [CommonModule, FormsModule, RouterLink, BreadcrumbsComponent],
  templateUrl: './checkout-page.component.html',
  styleUrls: ['./checkout-page.component.css']
})
export class CheckoutPageComponent implements OnInit, DoCheck, OnDestroy, AfterViewInit {
  step = signal<'shipping'|'payment'|'review'|'done'>('shipping');
  shipping = { name: '', email: '', address: '', city: '', country: '', zip: '' };
  placing = signal(false);
  orderId = signal<string>('');
  orderError = signal('');
  private readonly cart = inject(CartService);
  private readonly orders = inject(OrderService);
  private readonly router = inject(Router);
  private readonly pricing = inject(PricingService);
  readonly stripe = inject(StripeService);
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
  netCents = computed(()=> {
    const serverTotal = this.preview().totalCents;
    const serverTax = this.preview().taxCents ?? 0;
    // Server totalCents already includes tax, so subtract it to get the net (pre-tax) amount
    if (serverTotal != null && serverTax > 0) {
      return Math.max(0, serverTotal - serverTax);
    }
    return Math.max(0, serverTotal ?? (this.subtotalCents() - this.discountCents() - this.giftCardCents()));
  });
  taxCents = computed(()=> {
    // Prefer server-provided tax; fall back to client-side calculation
    const serverTax = this.preview().taxCents;
    if (serverTax != null && serverTax > 0) {
      return serverTax;
    }
    return Math.round(this.netCents() * (this.pricingConfig().taxRatePercent / 100));
  });
  shippingCents = computed(()=> (this.items().length > 0 ? this.pricingConfig().shippingFlatCents : 0));
  totalCents = computed(()=> {
    // When server totalCents already includes tax, just add shipping
    const serverTax = this.preview().taxCents ?? 0;
    if (serverTax > 0) {
      // Server total already includes tax but not shipping
      return (this.preview().totalCents ?? 0) + this.shippingCents();
    }
    return this.netCents() + this.taxCents() + this.shippingCents();
  });

  // Stripe integration state
  private stripeClientSecret = '';
  private stripePublishableKey = '';
  stripeReady = this.stripe.cardReady;
  stripeError = this.stripe.cardError;
  private stripeMounted = false;

  advanceFromShipping(form: NgForm){
    if (!this.validateForm(form)) { return; }
    this.step.set('payment');
    // Mount Stripe card element after view updates
    setTimeout(() => this.mountStripeCard(), 50);
  }

  advanceFromPayment(){
    this.step.set('review');
    this.orderError.set('');
    void this.cart.refreshPreview();
  }

  back(){
    if (this.step()==='payment') {
      this.stripe.unmountCardElement();
      this.stripeMounted = false;
      this.step.set('shipping');
    } else if (this.step()==='review') {
      this.step.set('payment');
      setTimeout(() => this.mountStripeCard(), 50);
    }
  }

  async placeOrder(){
    this.orderError.set('');
    this.cart.clearPreviewError();
    this.placing.set(true);
    await this.cart.refreshPreview();
    const lineItems = this.items().map(item => this.toOrderItem(item));
    const body: CreateOrderPayload = {
      shipping: { ...this.shipping },
      items: lineItems,
      payment: { method: 'stripe' },
      subtotalCents: this.subtotalCents(),
      discountCode: this.discountCode() || undefined,
      giftCardCode: this.giftCardCode() || undefined,
    };
    try {
      const res = await this.orders.createOrder(body);
      const clientSecret = res.clientSecret;
      const publishableKey = res.stripePublishableKey;

      if (clientSecret && publishableKey) {
        // Real Stripe payment flow
        this.stripeClientSecret = clientSecret;
        this.stripePublishableKey = publishableKey;

        // Ensure Stripe is loaded
        await this.stripe.loadStripe(publishableKey);

        // Confirm the payment with Stripe
        const paymentResult = await this.stripe.confirmCardPayment(clientSecret);
        if (!paymentResult.success) {
          this.placing.set(false);
          this.orderError.set(paymentResult.error || 'Payment failed. Please try again.');
          return;
        }
      }

      // Payment succeeded (or no Stripe configured — mock mode)
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

  ngOnInit(){
    try {
      const saved = JSON.parse(localStorage.getItem('checkout_progress')||'null');
      if (saved) { this.shipping = saved.shipping||this.shipping; if (saved.step) this.step.set(saved.step); }
    } catch {
      return;
    }
    void this.cart.refreshPreview();
  }

  ngAfterViewInit() {
    if (this.step() === 'payment') {
      setTimeout(() => this.mountStripeCard(), 100);
    }
  }

  ngDoCheck(){
    try { localStorage.setItem('checkout_progress', JSON.stringify({ shipping: this.shipping, step: this.step() })); } catch {
      return;
    }
  }

  ngOnDestroy() {
    this.stripe.unmountCardElement();
    this.stripeMounted = false;
  }

  private async mountStripeCard() {
    if (this.stripeMounted) return;
    try {
      // Try loading Stripe with a default key — the actual key comes from the order response
      // For now, mount the card element regardless (Stripe.js is loaded lazily)
      const key = this.stripePublishableKey || 'pk_test_placeholder';
      await this.stripe.loadStripe(key);
      const container = document.getElementById('card-element');
      if (container) {
        this.stripe.mountCardElement('#card-element');
        this.stripeMounted = true;
      }
    } catch {
      // Stripe not available — will fall back to mock mode
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
