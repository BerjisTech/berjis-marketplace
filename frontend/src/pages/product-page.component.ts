import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { CartService } from '../app/core/services/cart.service';
import { WishlistService } from '../app/core/services/wishlist.service';
import { ToastService } from '../app/shared/components/toast/toast.service';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-product-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './product-page.component.html',
  styleUrls: ['./product-page.component.css']
})
export class ProductPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly http = inject(HttpClient);
  private readonly cart = inject(CartService);
  private readonly wishlist = inject(WishlistService);
  private readonly toasts = inject(ToastService);
  readonly api = environment.apiBase;
  product = signal<ProductDetail | null>(null);
  loading = signal(true);
  quantity = signal<number>(1);
  related = signal<ProductSummary[]>([]);

  selectedImage = signal<string | null>(null);
  selectedVariantId = signal<string | null>(null);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.http.get<ApiResponse<ProductDetail>>(`${this.api}/v1/products/${id}`, { withCredentials: true }).subscribe(r => {
      const p = r.data;
      this.product.set(p ?? null);
      this.selectedImage.set(p?.imageUrl ?? null);
      this.loading.set(false);
      const cat = p?.category;
      if (p && cat) {
        this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/products?category=${encodeURIComponent(cat)}&limit=8`, { withCredentials: true }).subscribe(rr => {
          const items = (rr?.data ?? []).filter(x => x.uuid !== p.uuid).slice(0, 8);
          this.related.set(items);
        });
      }
    });
  }
  addToCart(){
    const p = this.product(); if (!p) return;
    this.cart.add({ productId: p.uuid, title: p.title, priceCents: p.priceCents, currency: p.currency, imageUrl: p.imageUrl, variantId: this.selectedVariantId() }, this.quantity());
    this.toasts.show('Added to cart');
  }
  addToWishlist(){
    const p = this.product(); if (!p) return;
    this.wishlist.add({
      productId: p.uuid,
      title: p.title,
      priceCents: this.displayPriceCents(),
      currency: p.currency,
      imageUrl: p.imageUrl ?? undefined,
      shopName: p.shopName,
      shopSlug: p.shopSlug
    });
    this.toasts.show('Saved to wishlist');
  }
  displayPriceCents(): number {
    const p = this.product();
    if (!p) {
      return 0;
    }
    const variants = p.variants ?? [];
    const match = variants.find(x => x.id === this.selectedVariantId());
    if (match && typeof match.priceCents === 'number') {
      return match.priceCents;
    }
    return p.priceCents;
  }
}

export interface ProductVariant {
  id: string;
  title: string;
  priceCents?: number;
}

export interface ProductDetail extends ProductSummary {
  summary?: string;
  description?: string;
  category?: string;
  variants?: ProductVariant[];
  images?: string[];
}

