import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface ProductSummary {
  uuid: string;
  title: string;
  priceCents: number;
  currency: string;
  imageUrl?: string;
  shopName?: string;
  shopSlug?: string;
  published?: boolean;
  stock?: number;
  summary?: string;
  category?: string;
  images?: string[];
  avgRating?: number;
  reviewCount?: number;
}

export interface ApiResponse<T> {
  data: T;
  total?: number;
}

export type CreateProductPayload = Record<string, unknown>;

export interface UploadResponse {
  url: string;
}

@Injectable({ providedIn: 'root' })
export class ProductService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  listMyShopProducts(shopSlug: string): Observable<ApiResponse<ProductSummary[]>> {
    return this.http.get<ApiResponse<ProductSummary[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/products`,
      { withCredentials: true },
    );
  }

  createProduct(body: CreateProductPayload): Observable<ApiResponse<ProductSummary>> {
    return this.http.post<ApiResponse<ProductSummary>>(`${this.api}/v1/products`, body, {
      withCredentials: true,
    });
  }

  updateProduct(productUuid: string, body: Record<string, unknown>): Observable<ApiResponse<ProductSummary>> {
    return this.http.patch<ApiResponse<ProductSummary>>(
      `${this.api}/v1/products/${encodeURIComponent(productUuid)}`,
      body,
      { withCredentials: true },
    );
  }

  upload(file: File): Observable<ApiResponse<UploadResponse>> {
    const formData = new FormData();
    formData.append('file', file);
    return this.http.post<ApiResponse<UploadResponse>>(`${this.api}/v1/uploads`, formData, {
      withCredentials: true,
    });
  }

  importProductsCsv(shopSlug: string, file: File): Observable<ApiResponse<Record<string, unknown>>> {
    const formData = new FormData();
    formData.append('file', file);
    return this.http.post<ApiResponse<Record<string, unknown>>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/products/import`,
      formData,
      { withCredentials: true },
    );
  }

  importDemoProducts(shopSlug: string): Observable<ApiResponse<{ created: number }>> {
    return this.http.post<ApiResponse<{ created: number }>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/products/import-demo`,
      {},
      { withCredentials: true },
    );
  }
}
