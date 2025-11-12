import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface GiftCard {
  uuid: string;
  shopUuid: string;
  code: string;
  balanceCents: number;
  originalBalanceCents: number;
  currency: string;
  issuedToEmail?: string | null;
  note?: string;
  status: string;
  expiresAt?: string | null;
  issuedAt: string;
  redeemedAt?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface GiftCardReportSummary {
  totalCards: number;
  activeCards: number;
  redeemedCards: number;
  issuedCents: number;
  outstandingCents: number;
  redeemedCents: number;
}

export interface GiftCardTransaction {
  uuid: string;
  giftCardUuid: string;
  changeCents: number;
  reason: string;
  createdAt: string;
}

export interface CreateGiftCardPayload {
  code?: string;
  balanceCents: number;
  currency: string;
  issuedToEmail?: string;
  note?: string;
  status?: string;
  expiresAt?: string | null;
}

export interface CreateGiftCardResponse {
  uuid: string;
  code: string;
}

export type UpdateGiftCardPayload = Partial<CreateGiftCardPayload>;

@Injectable({ providedIn: 'root' })
export class GiftCardService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(shopSlug: string): Observable<ApiResponse<GiftCard[]>> {
    return this.http.get<ApiResponse<GiftCard[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/gift-cards`,
      { withCredentials: true },
    );
  }

  create(shopSlug: string, payload: CreateGiftCardPayload): Observable<ApiResponse<CreateGiftCardResponse>> {
    return this.http.post<ApiResponse<CreateGiftCardResponse>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/gift-cards`,
      payload,
      { withCredentials: true },
    );
  }

  update(shopSlug: string, cardUuid: string, payload: UpdateGiftCardPayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/gift-cards/${encodeURIComponent(cardUuid)}`,
      payload,
      { withCredentials: true },
    );
  }

  report(shopSlug: string): Observable<ApiResponse<GiftCardReportSummary>> {
    return this.http.get<ApiResponse<GiftCardReportSummary>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/gift-cards/report`,
      { withCredentials: true },
    );
  }

  transactions(shopSlug: string, cardUuid: string): Observable<ApiResponse<GiftCardTransaction[]>> {
    return this.http.get<ApiResponse<GiftCardTransaction[]>>(
      `${this.api}/v1/my/shops/${encodeURIComponent(shopSlug)}/gift-cards/${encodeURIComponent(cardUuid)}/transactions`,
      { withCredentials: true },
    );
  }
}
