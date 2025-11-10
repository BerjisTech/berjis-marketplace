import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface CartPayloadItem {
  productId: string;
  quantity: number;
}

export interface CartItemResponse {
  productUuid?: string;
  productId?: string;
  product_id?: string;
  title?: string;
  priceCents?: number;
  price_cents?: number;
  currency?: string;
  imageUrl?: string | null;
  image_url?: string | null;
  variantId?: string | null;
  variant_id?: string | null;
  quantity?: number;
}

export interface CartApiResponse {
  data?: { items?: CartItemResponse[]; [key: string]: unknown };
  items?: CartItemResponse[];
}

@Injectable({ providedIn: 'root' })
export class CartApiService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  getCart(): Observable<CartApiResponse> {
    return this.http.get<CartApiResponse>(`${this.api}/v1/cart`, { withCredentials: true });
  }

  putCart(items: CartPayloadItem[]): Observable<unknown> {
    return this.http.put<unknown>(`${this.api}/v1/cart`, { items }, { withCredentials: true });
  }
}
