import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { CartService } from '../app/core/services/cart.service';
import { PricingService } from '../app/core/services/pricing.service';

@Component({
  standalone: true,
  selector: 'cart-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './cart-page.component.html',
  styleUrls: ['./cart-page.component.css']
})
export class CartPageComponent {
  private readonly cart = inject(CartService);
  private readonly pricing = inject(PricingService);
  items = this.cart.items;
  pricingConfig = this.pricing.config;
  subtotalCents = computed(() => this.cart.totalCents());
  taxCents = computed(() => Math.round(this.subtotalCents() * (this.pricingConfig().taxRatePercent / 100)));
  shippingCents = computed(() => (this.items().length > 0 ? this.pricingConfig().shippingFlatCents : 0));
  totalCents = computed(() => this.subtotalCents() + this.taxCents() + this.shippingCents());

  dec(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity - 1); }
  inc(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity + 1); }
  remove(i: number){ const it = this.items()[i]; this.cart.remove(it.productId, it.variantId||null); }
  clear(){ this.cart.clear(); }
}
