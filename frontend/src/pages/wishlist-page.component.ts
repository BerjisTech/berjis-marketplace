import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { WishlistService } from '../app/core/services/wishlist.service';
import { ToastService } from '../app/shared/components/toast/toast.service';

@Component({
  standalone: true,
  selector: 'wishlist-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './wishlist-page.component.html',
  styleUrls: ['./wishlist-page.component.css']
})
export class WishlistPageComponent {
  items = this.wishlist.items;

  constructor(private wishlist: WishlistService, private toasts: ToastService) {}

  remove(productId: string) {
    this.wishlist.remove(productId);
    this.toasts.show('Removed from wishlist');
  }

  moveToCart(productId: string) {
    const item = this.items().find(i => i.productId === productId);
    if (!item) {
      return;
    }
    this.wishlist.moveToCart(productId);
    this.toasts.show('Moved to cart');
  }
}
