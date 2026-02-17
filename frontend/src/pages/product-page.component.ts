import { Component, OnInit, OnDestroy, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { CartService } from '../app/core/services/cart.service';
import { WishlistService } from '../app/core/services/wishlist.service';
import { ToastService } from '../app/shared/components/toast/toast.service';
import { RecentlyViewedService } from '../app/core/services/recently-viewed.service';
import { SeoService } from '../app/core/services/seo.service';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';
import { ImageLightboxComponent } from '../app/shared/components/image-lightbox/image-lightbox.component';
import { StarRatingComponent } from '../app/shared/components/star-rating/star-rating.component';
import { BreadcrumbsComponent, BreadcrumbItem } from '../app/shared/components/breadcrumbs/breadcrumbs.component';
import { SkeletonComponent } from '../app/shared/components/skeleton/skeleton.component';

interface ReviewData {
  uuid: string;
  userName: string;
  rating: number;
  title: string;
  body: string;
  isVerifiedPurchase: boolean;
  createdAt: string;
}

@Component({
  standalone: true,
  selector: 'app-product-page',
  imports: [CommonModule, FormsModule, RouterLink, ImageLightboxComponent, StarRatingComponent, BreadcrumbsComponent, SkeletonComponent],
  templateUrl: './product-page.component.html',
  styleUrls: ['./product-page.component.css']
})
export class ProductPageComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly http = inject(HttpClient);
  private readonly cart = inject(CartService);
  private readonly wishlist = inject(WishlistService);
  private readonly toasts = inject(ToastService);
  private readonly recentlyViewed = inject(RecentlyViewedService);
  private readonly seo = inject(SeoService);
  readonly api = environment.apiBase;

  product = signal<ProductDetail | null>(null);
  loading = signal(true);
  quantity = signal<number>(1);
  related = signal<ProductSummary[]>([]);
  recentItems = signal<{ uuid: string; title: string; priceCents: number; currency: string; imageUrl?: string }[]>([]);

  selectedImage = signal<string | null>(null);
  selectedVariantId = signal<string | null>(null);

  // Lightbox
  lightboxOpen = signal(false);
  lightboxIndex = signal(0);

  // Reviews
  reviews = signal<ReviewData[]>([]);
  avgRating = signal(0);
  reviewCount = signal(0);
  reviewPage = signal(1);
  reviewHasNext = signal(false);
  newReviewRating = signal(0);
  newReviewTitle = '';
  newReviewBody = '';
  submittingReview = signal(false);

  // Breadcrumbs
  breadcrumbs = signal<BreadcrumbItem[]>([]);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.http.get<ApiResponse<ProductDetail>>(`${this.api}/v1/products/${id}`, { withCredentials: true }).subscribe(r => {
      const p = r.data;
      this.product.set(p ?? null);
      this.selectedImage.set(p?.imageUrl ?? null);
      this.loading.set(false);

      if (p) {
        // Track recently viewed
        this.recentlyViewed.add({
          uuid: p.uuid, title: p.title, priceCents: p.priceCents,
          currency: p.currency, imageUrl: p.imageUrl, shopName: p.shopName, shopSlug: p.shopSlug,
        });
        this.recentItems.set(this.recentlyViewed.getExcluding(p.uuid, 6));

        // SEO
        this.seo.set({
          title: p.title,
          description: p.summary || `Buy ${p.title} on Berjis Marketplace`,
          image: p.imageUrl,
        });

        // Breadcrumbs
        const crumbs: BreadcrumbItem[] = [{ label: 'Home', url: '/' }];
        if (p.category) crumbs.push({ label: p.category, url: `/?category=${encodeURIComponent(p.category)}` });
        if (p.shopName && p.shopSlug) crumbs.push({ label: p.shopName, url: `/shop/${p.shopSlug}` });
        crumbs.push({ label: p.title });
        this.breadcrumbs.set(crumbs);

        // Load reviews
        this.loadReviews(p.uuid);
      }

      const cat = p?.category;
      if (p && cat) {
        this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/products?category=${encodeURIComponent(cat)}&limit=8`, { withCredentials: true }).subscribe(rr => {
          const items = (rr?.data ?? []).filter(x => x.uuid !== p.uuid).slice(0, 8);
          this.related.set(items);
        });
      }
    });
  }

  ngOnDestroy(): void {
    this.seo.reset();
  }

  addToCart() {
    const p = this.product(); if (!p) return;
    this.cart.add({ productId: p.uuid, title: p.title, priceCents: p.priceCents, currency: p.currency, imageUrl: p.imageUrl, variantId: this.selectedVariantId() }, this.quantity());
    this.toasts.show('Added to cart');
  }

  addToWishlist() {
    const p = this.product(); if (!p) return;
    this.wishlist.add({
      productId: p.uuid, title: p.title, priceCents: this.displayPriceCents(),
      currency: p.currency, imageUrl: p.imageUrl ?? undefined, shopName: p.shopName, shopSlug: p.shopSlug,
    });
    this.toasts.show('Saved to wishlist');
  }

  displayPriceCents(): number {
    const p = this.product();
    if (!p) return 0;
    const variants = p.variants ?? [];
    const match = variants.find(x => x.id === this.selectedVariantId());
    if (match && typeof match.priceCents === 'number') return match.priceCents;
    return p.priceCents;
  }

  openLightbox(index: number) {
    this.lightboxIndex.set(index);
    this.lightboxOpen.set(true);
  }

  get allImages(): string[] {
    const p = this.product();
    if (!p) return [];
    const imgs: string[] = [];
    if (p.imageUrl) imgs.push(p.imageUrl);
    if (p.images) {
      for (const img of p.images) {
        if (!imgs.includes(img)) imgs.push(img);
      }
    }
    return imgs;
  }

  // Reviews
  loadReviews(productId: string) {
    const page = this.reviewPage();
    this.http.get<any>(`${this.api}/v1/products/${productId}/reviews?page=${page}&limit=10`, { withCredentials: true }).subscribe(r => {
      this.reviews.set(r?.data ?? []);
      this.avgRating.set(r?.avgRating ?? 0);
      this.reviewCount.set(r?.total ?? 0);
      this.reviewHasNext.set((r?.data ?? []).length >= 10);
    });
  }

  submitReview() {
    const p = this.product();
    if (!p || this.newReviewRating() === 0) return;
    this.submittingReview.set(true);
    this.http.post(`${this.api}/v1/products/${p.uuid}/reviews`, {
      rating: this.newReviewRating(),
      title: this.newReviewTitle,
      body: this.newReviewBody,
    }, { withCredentials: true }).subscribe({
      next: () => {
        this.toasts.show('Review submitted');
        this.newReviewRating.set(0);
        this.newReviewTitle = '';
        this.newReviewBody = '';
        this.submittingReview.set(false);
        this.loadReviews(p.uuid);
      },
      error: () => {
        this.toasts.show('Failed to submit review');
        this.submittingReview.set(false);
      },
    });
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
  avgRating?: number;
  reviewCount?: number;
}
