import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface SalesSeriesPoint {
  date: string;
  revenueCents: number;
  ordersCount: number;
}

export interface SalesData {
  totalRevenueCents: number;
  ordersCount: number;
  averageOrderValueCents: number;
  series: SalesSeriesPoint[];
}

export interface TopProduct {
  productUuid: string;
  title: string;
  unitsSold: number;
  revenueCents: number;
}

export interface CustomerData {
  newCustomers: number;
  returningCustomers: number;
  totalCustomers: number;
}

export interface ConversionData {
  productViews: number;
  addToCarts: number;
  checkoutsStarted: number;
  checkoutsCompleted: number;
  conversionRate: number;
}

export interface AnalyticsEvent {
  uuid: string;
  eventName: string;
  sessionId?: string;
  payload?: Record<string, unknown>;
  occurredAt: string;
}

@Injectable({ providedIn: 'root' })
export class AnalyticsService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  trackEvent(
    shopUuid: string,
    eventType: string,
    metadata?: Record<string, unknown>,
  ): Observable<ApiResponse<{ uuid: string }>> {
    return this.http.post<ApiResponse<{ uuid: string }>>(
      `${this.api}/v1/analytics/event`,
      { shopUuid, eventType, metadata },
    );
  }

  getSales(
    shopSlug: string,
    from: string,
    to: string,
    granularity: string = 'day',
  ): Observable<ApiResponse<SalesData>> {
    const params = new HttpParams()
      .set('from', from)
      .set('to', to)
      .set('granularity', granularity);
    return this.http.get<ApiResponse<SalesData>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/analytics/sales`,
      { params, withCredentials: true },
    );
  }

  getTopProducts(
    shopSlug: string,
    from: string,
    to: string,
    limit: number = 10,
  ): Observable<ApiResponse<TopProduct[]>> {
    const params = new HttpParams()
      .set('from', from)
      .set('to', to)
      .set('limit', limit.toString());
    return this.http.get<ApiResponse<TopProduct[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/analytics/top-products`,
      { params, withCredentials: true },
    );
  }

  getCustomers(
    shopSlug: string,
    from: string,
    to: string,
  ): Observable<ApiResponse<CustomerData>> {
    const params = new HttpParams()
      .set('from', from)
      .set('to', to);
    return this.http.get<ApiResponse<CustomerData>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/analytics/customers`,
      { params, withCredentials: true },
    );
  }

  getConversions(
    shopSlug: string,
    from: string,
    to: string,
  ): Observable<ApiResponse<ConversionData>> {
    const params = new HttpParams()
      .set('from', from)
      .set('to', to);
    return this.http.get<ApiResponse<ConversionData>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/analytics/conversions`,
      { params, withCredentials: true },
    );
  }

  getRecentEvents(
    shopSlug: string,
    limit: number = 50,
  ): Observable<ApiResponse<AnalyticsEvent[]>> {
    const params = new HttpParams().set('limit', limit.toString());
    return this.http.get<ApiResponse<AnalyticsEvent[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/analytics/events`,
      { params, withCredentials: true },
    );
  }
}
