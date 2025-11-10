import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface WishlistPayloadItem {
  productId: string;
}

export interface WishlistItemResponse {
  productUuid?: string;
  productId?: string;
  title?: string;
  priceCents?: number;
  currency?: string;
  imageUrl?: string | null;
  shopName?: string;
  shopSlug?: string;
  addedAt?: string;
}

export interface WishlistApiResponse {
  data?: { items?: WishlistItemResponse[]; [key: string]: unknown };
  items?: WishlistItemResponse[];
}

@Injectable({ providedIn: 'root' })
export class WishlistApiService {
  private readonly api = environment.apiBase;
  private readonly http = inject(HttpClient);

  getWishlist(): Observable<WishlistApiResponse> {
    return this.http.get<WishlistApiResponse>(`${this.api}/v1/wishlist`, { withCredentials: true });
  }

  putWishlist(items: WishlistPayloadItem[]): Observable<unknown> {
    return this.http.put<unknown>(`${this.api}/v1/wishlist`, { items }, { withCredentials: true });
  }
}
