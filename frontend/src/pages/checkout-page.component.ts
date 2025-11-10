import { Component, signal, computed, inject, OnInit, DoCheck } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { CartService } from '../app/core/services/cart.service';
import { OrderService } from '../app/core/services/order.service';

@Component({
  standalone: true,
  selector: 'checkout-page',
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

  next(){ if (this.step()==='shipping') this.step.set('payment'); else if (this.step()==='payment') this.step.set('review'); }
  back(){ if (this.step()==='payment') this.step.set('shipping'); else if (this.step()==='review') this.step.set('payment'); }

  async placeOrder(){
    this.placing.set(true);
    const body = { shipping: this.shipping, items: this.items(), payment: { method: 'mock' }, subtotalCents: this.subtotalCents() };
    const res = await this.orders.createOrder(body);
    this.orderId.set(res.id);
    this.cart.clear();
    this.placing.set(false);
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
}
