import { Component, signal, computed, inject, OnInit, DoCheck } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, NgForm, AbstractControl } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { CartService, CartItem } from '../app/core/services/cart.service';
import { CreateOrderPayload, OrderService } from '../app/core/services/order.service';

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
  private readonly cart = inject(CartService);
  private readonly orders = inject(OrderService);
  private readonly router = inject(Router);
  items = this.cart.items;
  subtotalCents = computed(()=> this.cart.totalCents());

  advanceFromShipping(form: NgForm){
    if (!this.validateForm(form)) { return; }
    this.step.set('payment');
  }
  advanceFromPayment(form: NgForm){
    if (!this.validateForm(form)) { return; }
    this.step.set('review');
  }
  back(){ if (this.step()==='payment') this.step.set('shipping'); else if (this.step()==='review') this.step.set('payment'); }

  async placeOrder(){
    this.placing.set(true);
    const lineItems = this.items().map(item => this.toOrderItem(item));
    const body: CreateOrderPayload = {
      shipping: { ...this.shipping },
      items: lineItems,
      payment: { method: 'mock' },
      subtotalCents: this.subtotalCents(),
    };
    const res = await this.orders.createOrder(body);
    this.orderId.set(res.id);
    this.cart.clear();
    this.placing.set(false);
    try { localStorage.removeItem('checkout_progress'); } catch { /* ignore */ }
    this.router.navigate(['/checkout/confirmation', res.id]);
  }
  // Persist checkout progress
  ngOnInit(){
    try {
      const saved = JSON.parse(localStorage.getItem('checkout_progress')||'null');
      if (saved) { this.shipping = saved.shipping||this.shipping; this.payment = saved.payment||this.payment; if (saved.step) this.step.set(saved.step); }
    } catch {
      return;
    }
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
}

