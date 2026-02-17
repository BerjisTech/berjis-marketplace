import { Component, Input, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { WishlistService } from '../../../core/services/wishlist.service';
import { ToastService } from '../toast/toast.service';
import { ProductSummary } from '../../../core/services/product.service';
import { StarRatingComponent } from '../star-rating/star-rating.component';

@Component({
  standalone: true,
  selector: 'app-product-card',
  imports: [CommonModule, RouterLink, StarRatingComponent],
  templateUrl: './product-card.component.html',
  styleUrls: ['./product-card.component.css']
})
export class ProductCardComponent {
  @Input() product?: ProductSummary;
  @Input() showShop = false;

  private readonly wishlist = inject(WishlistService);
  private readonly toasts = inject(ToastService);

  addToWishlist(event: MouseEvent){
    event.stopPropagation();
    event.preventDefault();
    const p = this.product;
    if (!p || !p.uuid) return;
    this.wishlist.add({
      productId: p.uuid,
      title: p.title,
      priceCents: p.priceCents,
      currency: p.currency,
      imageUrl: p.imageUrl ?? undefined,
      shopName: p.shopName,
      shopSlug: p.shopSlug
    });
    this.toasts.show('Saved to wishlist');
  }
}
