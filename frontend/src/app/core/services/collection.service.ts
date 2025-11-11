import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse, ProductSummary } from './product.service';

export interface CollectionRule {
  field: string;
  operator: string;
  value?: string | number | boolean | null;
  min?: number | null;
  max?: number | null;
}

export interface Collection {
  uuid: string;
  shopUuid: string;
  title: string;
  slug: string;
  description: string;
  isAutomatic: boolean;
  rules?: CollectionRule[] | null;
  sortOrder: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CollectionSummary {
  uuid: string;
  title: string;
  slug: string;
  description: string;
  isAutomatic: boolean;
  sortOrder: number;
  productCount: number;
}

export interface CreateCollectionPayload {
  title: string;
  slug: string;
  description?: string;
  isAutomatic: boolean;
  sortOrder?: number;
  isActive?: boolean;
  rules?: CollectionRule[] | null;
  productUuids?: string[];
}

export type UpdateCollectionPayload = Partial<CreateCollectionPayload>;

@Injectable({ providedIn: 'root' })
export class CollectionService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  listMyCollections(shopSlug: string): Observable<ApiResponse<Collection[]>> {
    return this.http.get<ApiResponse<Collection[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections`,
      { withCredentials: true },
    );
  }

  createCollection(shopSlug: string, body: CreateCollectionPayload): Observable<ApiResponse<{ uuid: string }>> {
    return this.http.post<ApiResponse<{ uuid: string }>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections`,
      body,
      { withCredentials: true },
    );
  }

  updateCollection(shopSlug: string, collectionUuid: string, body: UpdateCollectionPayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections/${collectionUuid}`,
      body,
      { withCredentials: true },
    );
  }

  deleteCollection(shopSlug: string, collectionUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections/${collectionUuid}`,
      { withCredentials: true },
    );
  }

  rebuildCollection(shopSlug: string, collectionUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.post<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections/${collectionUuid}/rebuild`,
      {},
      { withCredentials: true },
    );
  }

  listCollectionProducts(shopSlug: string, collectionUuid: string): Observable<ApiResponse<ProductSummary[]>> {
    return this.http.get<ApiResponse<ProductSummary[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/collections/${collectionUuid}/products`,
      { withCredentials: true },
    );
  }

  listPublicCollections(shopSlug: string): Observable<ApiResponse<CollectionSummary[]>> {
    return this.http.get<ApiResponse<CollectionSummary[]>>(
      `${this.api}/v1/shops/${shopSlug}/collections`,
    );
  }
}
