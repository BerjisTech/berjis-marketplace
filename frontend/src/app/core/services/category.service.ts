import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface Category {
  uuid: string;
  name: string;
  slug: string;
  description: string;
  sortOrder: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateCategoryPayload {
  name: string;
  slug: string;
  description?: string;
  parentUuid?: string | null;
  sortOrder?: number | null;
  isActive?: boolean;
}

export interface UpdateCategoryPayload {
  name?: string;
  slug?: string;
  description?: string;
  parentUuid?: string | null;
  sortOrder?: number | null;
  isActive?: boolean;
}

@Injectable({ providedIn: 'root' })
export class CategoryService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(shopSlug: string): Observable<ApiResponse<Category[]>> {
    return this.http.get<ApiResponse<Category[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/categories`,
      { withCredentials: true },
    );
  }

  create(shopSlug: string, payload: CreateCategoryPayload): Observable<ApiResponse<{ uuid: string }>> {
    return this.http.post<ApiResponse<{ uuid: string }>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/categories`,
      payload,
      { withCredentials: true },
    );
  }

  update(shopSlug: string, categoryUuid: string, payload: UpdateCategoryPayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/categories/${encodeURIComponent(categoryUuid)}`,
      payload,
      { withCredentials: true },
    );
  }

  archive(shopSlug: string, categoryUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/categories/${encodeURIComponent(categoryUuid)}`,
      { withCredentials: true },
    );
  }
}
