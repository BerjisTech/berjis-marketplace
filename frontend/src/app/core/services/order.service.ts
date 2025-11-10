import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface CreateOrderPayload {
  shipping: Record<string, unknown>;
  items: { productId: string; quantity: number; [key: string]: unknown }[];
  payment: Record<string, unknown>;
  subtotalCents: number;
  [key: string]: unknown;
}

interface OrderResponseData {
  uuid?: string;
  id?: string;
}

interface OrderApiResponse {
  data?: OrderResponseData;
  id?: string;
}

export interface ShopOrder {
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
  customerUuid?: string;
  customerEmail: string;
  customerName: string;
}

export interface UpdateTrackingPayload {
  status?: string;
  trackingNumber?: string;
  trackingUrl?: string;
  shippingCarrier?: string;
}

export interface ShopMetrics {
  totalSalesCents: number;
  ordersCount: number;
  averageOrderValueCents: number;
  customersCount: number;
  salesSeries: { date: string; totalCents: number }[];
}

@Injectable({ providedIn: 'root' })
export class OrderService {
  private readonly api = environment.apiBase;
  private readonly http = inject(HttpClient);

  async createOrder(body: CreateOrderPayload): Promise<{ id: string }> {
    try {
      const res = await firstValueFrom(
        this.http.post<OrderApiResponse>(`${this.api}/v1/orders`, body, { withCredentials: true })
      );
      const apiId = res?.data?.uuid ?? res?.data?.id ?? res?.id;
      return { id: apiId ?? Date.now().toString() };
    } catch {
      return { id: Date.now().toString() };
    }
  }

  listShopOrders(shopSlug: string): Observable<ApiResponse<ShopOrder[]>> {
    return this.http.get<ApiResponse<ShopOrder[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/orders`,
      { withCredentials: true },
    );
  }

  updateTracking(orderUuid: string, payload: UpdateTrackingPayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/orders/${encodeURIComponent(orderUuid)}/tracking`,
      payload,
      { withCredentials: true },
    );
  }

  getShopMetrics(shopSlug: string): Observable<ApiResponse<ShopMetrics>> {
    return this.http.get<ApiResponse<ShopMetrics>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/metrics`,
      { withCredentials: true },
    );
  }
}
