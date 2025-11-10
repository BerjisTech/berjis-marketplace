import { Component, EventEmitter, Input, Output, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { CartService } from '../../../core/services/cart.service';

@Component({
  standalone: true,
  selector: 'app-mini-cart',
  imports: [CommonModule, RouterLink],
  templateUrl: './mini-cart.component.html',
  styleUrls: ['./mini-cart.component.css']
})
export class MiniCartComponent {
  @Input() open = false;
  @Output() openChange = new EventEmitter<boolean>();
  private readonly cart = inject(CartService);
  items = this.cart.items;
  subtotalCents = computed(() => this.cart.totalCents());
  close(){ this.openChange.emit(false); }
  dec(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity - 1); }
  inc(i: number){ const it = this.items()[i]; this.cart.update(it.productId, it.variantId||null, it.quantity + 1); }
  remove(i: number){ const it = this.items()[i]; this.cart.remove(it.productId, it.variantId||null); }
}
