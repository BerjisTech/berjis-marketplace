import { Component, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { CartService } from '../app/core/services/cart.service';

@Component({
  standalone: true,
  selector: 'cart-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './cart-page.component.html',
  styleUrls: ['./cart-page.component.css']
})
export class CartPageComponent {
  constructor(public cart: CartService) {}
  items = this.cart.items;
  subtotalCents = computed(() => this.cart.totalCents());

  dec(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity - 1); }
  inc(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity + 1); }
  remove(i: number){ const it = this.items()[i]; this.cart.remove(it.productId, it.variantId||null); }
  clear(){ this.cart.clear(); }
}

