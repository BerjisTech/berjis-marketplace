import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface Discount {
  uuid: string;
  shopUuid: string;
  name: string;
  code: string;
  description?: string;
  discountType: 'amount' | 'percentage';
  amountCents: number;
  percentage: number;
  startsAt?: string | null;
  endsAt?: string | null;
  usageLimitTotal?: number | null;
  usageLimitPerCustomer?: number | null;
  autoApply: boolean;
  status: string;
  appliesTo?: unknown;
  createdAt: string;
  updatedAt: string;
}

export interface DiscountReportRow extends Discount {
  redemptionCount: number;
  uniqueCustomers: number;
  totalAmountCents: number;
}

export interface CreateDiscountPayload {
  name: string;
  code: string;
  description?: string;
  discountType: 'amount' | 'percentage';
  amountCents?: number | null;
  percentage?: number | null;
  startsAt?: string | null;
  endsAt?: string | null;
  usageLimitTotal?: number | null;
  usageLimitPerCustomer?: number | null;
  autoApply?: boolean;
  status?: string;
  appliesTo?: unknown;
  productUuids?: string[];
}

export type UpdateDiscountPayload = Partial<CreateDiscountPayload>;

@Injectable({ providedIn: 'root' })
export class DiscountService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(shopSlug: string): Observable<ApiResponse<Discount[]>> {
    return this.http.get<ApiResponse<Discount[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/discounts`,
      { withCredentials: true },
    );
  }

  report(shopSlug: string): Observable<ApiResponse<DiscountReportRow[]>> {
    return this.http.get<ApiResponse<DiscountReportRow[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/discounts/report`,
      { withCredentials: true },
    );
  }

  create(shopSlug: string, payload: CreateDiscountPayload): Observable<ApiResponse<{ uuid: string }>> {
    return this.http.post<ApiResponse<{ uuid: string }>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/discounts`,
      payload,
      { withCredentials: true },
    );
  }

  update(shopSlug: string, discountUuid: string, payload: UpdateDiscountPayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/discounts/${encodeURIComponent(discountUuid)}`,
      payload,
      { withCredentials: true },
    );
  }

  archive(shopSlug: string, discountUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/discounts/${encodeURIComponent(discountUuid)}`,
      { withCredentials: true },
    );
  }
}
