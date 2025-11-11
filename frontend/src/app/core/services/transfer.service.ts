import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface TransferItem {
  transferUuid: string;
  productUuid: string;
  productTitle: string;
  quantity: number;
}

export interface Transfer {
  uuid: string;
  shopUuid: string;
  sourceLocationUuid?: string | null;
  sourceLocationName?: string | null;
  sourceLocationCode?: string | null;
  destinationLocationUuid?: string | null;
  destinationLocationName?: string | null;
  destinationLocationCode?: string | null;
  status: string;
  notes: string;
  createdBy?: string | null;
  createdAt: string;
  updatedAt: string;
  items: TransferItem[];
}

export interface CreateTransferPayload {
  sourceLocationUuid?: string;
  destinationLocationUuid?: string;
  notes?: string;
  items: { productUuid: string; quantity: number }[];
}

@Injectable({ providedIn: 'root' })
export class TransferService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(shopSlug: string): Observable<ApiResponse<Transfer[]>> {
    return this.http.get<ApiResponse<Transfer[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/transfers`,
      { withCredentials: true },
    );
  }

  create(shopSlug: string, payload: CreateTransferPayload): Observable<ApiResponse<Transfer>> {
    return this.http.post<ApiResponse<Transfer>>(
      `${this.api}/v1/my/shops/${shopSlug}/transfers`,
      payload,
      { withCredentials: true },
    );
  }

  commit(shopSlug: string, transferUuid: string, payload: { note?: string }): Observable<ApiResponse<Transfer>> {
    return this.http.post<ApiResponse<Transfer>>(
      `${this.api}/v1/my/shops/${shopSlug}/transfers/${transferUuid}/commit`,
      payload,
      { withCredentials: true },
    );
  }
}
