import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { CartService } from '../app/core/services/cart.service';
import { WishlistService } from '../app/core/services/wishlist.service';
import { ToastService } from '../app/shared/components/toast/toast.service';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'product-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './product-page.component.html',
  styleUrls: ['./product-page.component.css']
})
export class ProductPageComponent implements OnInit {
  api = environment.apiBase;
  product = signal<any>(null);
  loading = signal(true);
  quantity = signal<number>(1);
  related = signal<any[]>([]);

  selectedImage = signal<string | null>(null);
  selectedVariantId = signal<string | null>(null);

  constructor(private route: ActivatedRoute, private http: HttpClient, private cart: CartService, private wishlist: WishlistService, private toasts: ToastService) {}
  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.http.get<any>(`${this.api}/v1/products/${id}`).subscribe(r => {
      const p = r.data; this.product.set(p); this.selectedImage.set(p?.imageUrl||null); this.loading.set(false);
      const cat = p?.category; if (cat) {
        this.http.get<any>(`${this.api}/v1/products?category=${encodeURIComponent(cat)}&limit=8`).subscribe(rr => {
          const items = (rr?.data || []).filter((x: any)=> x.uuid !== p.uuid).slice(0, 8);
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
    const p = this.product(); if (!p) return 0;
    let price = p.priceCents;
    const list = p.variants as any[] | undefined;
    if (list && list.length) {
      const v = list.find(x => x.id === this.selectedVariantId());
      if (v && typeof v.priceCents === 'number') price = v.priceCents;
    }
    return price;
  }
}
