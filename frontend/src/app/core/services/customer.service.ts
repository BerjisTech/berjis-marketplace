import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface CustomerSummary {
  uuid: string;
  shopUuid: string;
  userUuid?: string;
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  tags: string[];
  notes: string;
  marketingOptIn: boolean;
  createdAt: string;
  updatedAt: string;
  totalSpentCents: number;
  ordersCount: number;
  lastOrderAt?: string;
  customerName?: string;
}

export interface CustomerOrder {
  uuid: string;
  totalCents: number;
  currency: string;
  status: string;
  createdAt: string;
  updatedAt: string;
  trackingNumber?: string;
  trackingUrl?: string;
  shippingCarrier?: string;
  shippedAt?: string;
  deliveredAt?: string;
}

export interface CreateCustomerPayload {
  email: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  notes?: string;
  tags?: string[];
  marketingOptIn?: boolean;
  userUuid?: string;
}

export interface UpdateCustomerPayload {
  email?: string;
  firstName?: string;
  lastName?: string;
  phone?: string;
  notes?: string;
  tags?: string[];
  marketingOptIn?: boolean;
  userUuid?: string;
}

@Injectable({ providedIn: 'root' })
export class CustomerService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(
    shopSlug: string,
    filters?: { q?: string; tag?: string }
  ): Observable<ApiResponse<CustomerSummary[]>> {
    let params = new HttpParams();
    if (filters?.q) {
      params = params.set('q', filters.q);
    }
    if (filters?.tag) {
      params = params.set('tag', filters.tag);
    }
    return this.http.get<ApiResponse<CustomerSummary[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/customers`,
      { params, withCredentials: true }
    );
  }

  create(shopSlug: string, payload: CreateCustomerPayload): Observable<ApiResponse<CustomerSummary>> {
    return this.http.post<ApiResponse<CustomerSummary>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/customers`,
      payload,
      { withCredentials: true }
    );
  }

  get(customerUuid: string): Observable<ApiResponse<CustomerSummary>> {
    return this.http.get<ApiResponse<CustomerSummary>>(
      `${this.api}/v1/customers/${encodeURIComponent(customerUuid)}`,
      { withCredentials: true }
    );
  }

  update(customerUuid: string, payload: UpdateCustomerPayload): Observable<ApiResponse<CustomerSummary>> {
    return this.http.patch<ApiResponse<CustomerSummary>>(
      `${this.api}/v1/customers/${encodeURIComponent(customerUuid)}`,
      payload,
      { withCredentials: true }
    );
  }

  listOrders(customerUuid: string): Observable<ApiResponse<CustomerOrder[]>> {
    return this.http.get<ApiResponse<CustomerOrder[]>>(
      `${this.api}/v1/customers/${encodeURIComponent(customerUuid)}/orders`,
      { withCredentials: true }
    );
  }
}
