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
  discountCode?: string;
  giftCardCode?: string;
  [key: string]: unknown;
}

export interface ManualOrderItemPayload {
  productUuid: string;
  quantity: number;
}

export interface CreateManualOrderPayload {
  customerUuid: string;
  items: ManualOrderItemPayload[];
  status?: string;
  discountCode?: string;
  giftCardCode?: string;
  shippingAddress?: string;
  paymentMethod?: string;
}

export interface CreateManualOrderResponse {
  uuid: string;
  subtotalCents: number;
  totalCents: number;
  discountAmountCents: number;
  giftCardAmountCents: number;
  currency: string;
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
  cancelledAt?: string;
  refundedAt?: string;
  refundTotalCents: number;
  draftSourceUuid?: string;
  customerUuid?: string;
  customerEmail: string;
  customerName: string;
}

export interface OrderTimelineEvent {
  uuid: string;
  orderUuid: string;
  eventType: string;
  message: string;
  metadata?: Record<string, unknown> | null;
  createdBy?: string;
  createdAt: string;
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

export interface DraftOrderItem {
  uuid: string;
  draftUuid: string;
  productUuid: string;
  quantity: number;
  priceCents: number;
  lineTotalCents: number;
  currency: string;
  title: string;
  createdAt: string;
  updatedAt: string;
}

export interface DraftOrder {
  uuid: string;
  shopUuid: string;
  customerUuid?: string;
  customerEmail?: string;
  customerName?: string;
  currency: string;
  subtotalCents: number;
  discountCode?: string;
  discountAmountCents: number;
  giftCardCode?: string;
  giftCardAmountCents: number;
  shippingAddress?: string;
  paymentMethod?: string;
  notes?: string;
  status: string;
  expiresAt?: string;
  createdBy?: string;
  updatedBy?: string;
  createdAt: string;
  updatedAt: string;
  items: DraftOrderItem[];
}

export interface CreateDraftOrderPayload {
  customerUuid?: string;
  customerEmail?: string;
  customerName?: string;
  notes?: string;
  discountCode?: string;
  giftCardCode?: string;
  shippingAddress?: string;
  paymentMethod?: string;
  status?: string;
  expiresAt?: string;
  items: ManualOrderItemPayload[];
}

export interface UpdateDraftOrderPayload {
  customerUuid?: string;
  customerEmail?: string;
  customerName?: string;
  notes?: string;
  discountCode?: string;
  giftCardCode?: string;
  shippingAddress?: string;
  paymentMethod?: string;
  status?: string;
  expiresAt?: string;
  items?: ManualOrderItemPayload[];
}

export interface CommitDraftOrderPayload {
  status?: string;
  discountCode?: string;
  giftCardCode?: string;
  shippingAddress?: string;
  paymentMethod?: string;
}

export interface CommitDraftOrderResponse {
  orderUuid: string;
  draftUuid: string;
  status: string;
  subtotalCents: number;
  discountAmountCents: number;
  giftCardAmountCents: number;
  totalCents: number;
  currency: string;
}

@Injectable({ providedIn: 'root' })
export class OrderService {
  private readonly api = environment.apiBase;
  private readonly http = inject(HttpClient);

  async createOrder(body: CreateOrderPayload): Promise<{ id: string }> {
    const res = await firstValueFrom(
      this.http.post<OrderApiResponse>(`${this.api}/v1/orders`, body, { withCredentials: true })
    );
    const apiId = res?.data?.uuid ?? res?.data?.id ?? res?.id;
    if (!apiId) {
      throw new Error('Order creation failed');
    }
    return { id: apiId };
  }

  listShopOrders(shopSlug: string): Observable<ApiResponse<ShopOrder[]>> {
    return this.http.get<ApiResponse<ShopOrder[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/orders`,
      { withCredentials: true },
    );
  }

  listDraftOrders(shopSlug: string): Observable<ApiResponse<DraftOrder[]>> {
    return this.http.get<ApiResponse<DraftOrder[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/orders/drafts`,
      { withCredentials: true },
    );
  }

  createDraftOrder(
    shopSlug: string,
    payload: CreateDraftOrderPayload,
  ): Observable<ApiResponse<DraftOrder>> {
    return this.http.post<ApiResponse<DraftOrder>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/orders/drafts`,
      payload,
      { withCredentials: true },
    );
  }

  updateDraftOrder(
    draftUuid: string,
    payload: UpdateDraftOrderPayload,
  ): Observable<ApiResponse<DraftOrder>> {
    return this.http.patch<ApiResponse<DraftOrder>>(
      `${this.api}/v1/orders/drafts/${encodeURIComponent(draftUuid)}`,
      payload,
      { withCredentials: true },
    );
  }

  deleteDraftOrder(draftUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/orders/drafts/${encodeURIComponent(draftUuid)}`,
      { withCredentials: true },
    );
  }

  commitDraftOrder(
    draftUuid: string,
    payload?: CommitDraftOrderPayload,
  ): Observable<ApiResponse<CommitDraftOrderResponse>> {
    return this.http.post<ApiResponse<CommitDraftOrderResponse>>(
      `${this.api}/v1/orders/drafts/${encodeURIComponent(draftUuid)}/commit`,
      payload ?? {},
      { withCredentials: true },
    );
  }

  createManualOrder(
    shopSlug: string,
    payload: CreateManualOrderPayload,
  ): Observable<ApiResponse<CreateManualOrderResponse>> {
    return this.http.post<ApiResponse<CreateManualOrderResponse>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/orders`,
      payload,
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

  cancelOrder(orderUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.post<ApiResponse<unknown>>(
      `${this.api}/v1/orders/${encodeURIComponent(orderUuid)}/cancel`,
      {},
      { withCredentials: true },
    );
  }

  refundOrder(
    orderUuid: string,
    payload?: { amountCents?: number; reason?: string },
  ): Observable<ApiResponse<unknown>> {
    return this.http.post<ApiResponse<unknown>>(
      `${this.api}/v1/orders/${encodeURIComponent(orderUuid)}/refunds`,
      payload ?? {},
      { withCredentials: true },
    );
  }

  getShopMetrics(shopSlug: string): Observable<ApiResponse<ShopMetrics>> {
    return this.http.get<ApiResponse<ShopMetrics>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/metrics`,
      { withCredentials: true },
    );
  }

  timeline(orderUuid: string): Observable<ApiResponse<OrderTimelineEvent[]>> {
    return this.http.get<ApiResponse<OrderTimelineEvent[]>>(
      `${this.api}/v1/orders/${encodeURIComponent(orderUuid)}/events`,
      { withCredentials: true },
    );
  }

  addNote(orderUuid: string, message: string): Observable<ApiResponse<OrderTimelineEvent>> {
    return this.http.post<ApiResponse<OrderTimelineEvent>>(
      `${this.api}/v1/orders/${encodeURIComponent(orderUuid)}/notes`,
      { message },
      { withCredentials: true },
    );
  }
}
