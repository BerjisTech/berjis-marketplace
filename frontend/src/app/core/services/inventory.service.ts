import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface InventoryEntry {
  uuid: string;
  shopUuid: string;
  productUuid: string;
  productTitle: string;
  productSlug: string;
  locationUuid?: string | null;
  locationName?: string | null;
  locationCode?: string | null;
  quantity: number;
  reserved: number;
  safetyStock: number;
  createdAt: string;
  updatedAt: string;
}

export interface InventoryLocation {
  uuid: string;
  shopUuid: string;
  name: string;
  code: string;
  description: string;
  isPrimary: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface InventoryAdjustment {
  uuid: string;
  inventoryLevelUuid: string;
  shopUuid: string;
  productUuid: string;
  locationUuid?: string | null;
  userUuid?: string | null;
  deltaQuantity: number;
  deltaReserved: number;
  resultingQuantity: number;
  resultingReserved: number;
  reason: string;
  note: string;
  adjustmentSource: string;
  createdAt: string;
}

export interface InventoryAlert {
  uuid: string;
  inventoryLevelUuid: string;
  shopUuid: string;
  productUuid: string;
  locationUuid?: string | null;
  productTitle: string;
  productSlug: string;
  locationName?: string | null;
  locationCode?: string | null;
  quantity: number;
  safetyStock: number;
  status: string;
  triggeredAt: string;
  resolvedAt?: string | null;
  note: string;
}

interface AdjustmentResponse {
  inventory: InventoryEntry;
  adjustment: InventoryAdjustment;
}

@Injectable({ providedIn: 'root' })
export class InventoryService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  listInventory(shopSlug: string): Observable<ApiResponse<InventoryEntry[]>> {
    return this.http.get<ApiResponse<InventoryEntry[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory`,
      { withCredentials: true },
    );
  }

  updateLevel(
    shopSlug: string,
    levelUuid: string,
    body: Partial<Pick<InventoryEntry, 'quantity' | 'reserved' | 'safetyStock'>>,
  ): Observable<ApiResponse<InventoryEntry>> {
    return this.http.patch<ApiResponse<InventoryEntry>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/${levelUuid}`,
      body,
      { withCredentials: true },
    );
  }

  createAdjustment(
    shopSlug: string,
    levelUuid: string,
    body: { deltaQuantity?: number | null; deltaReserved?: number | null; reason?: string; note?: string },
  ): Observable<ApiResponse<AdjustmentResponse>> {
    return this.http.post<ApiResponse<AdjustmentResponse>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/${levelUuid}/adjustments`,
      body,
      { withCredentials: true },
    );
  }

  getHistory(
    shopSlug: string,
    levelUuid: string,
    limit = 50,
  ): Observable<ApiResponse<InventoryAdjustment[]>> {
    return this.http.get<ApiResponse<InventoryAdjustment[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/${levelUuid}/history?limit=${limit}`,
      { withCredentials: true },
    );
  }

  listAlerts(
    shopSlug: string,
    status: 'open' | 'resolved' | 'all' = 'open',
  ): Observable<ApiResponse<InventoryAlert[]>> {
    const query = status ? `?status=${encodeURIComponent(status)}` : '';
    return this.http.get<ApiResponse<InventoryAlert[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/alerts${query}`,
      { withCredentials: true },
    );
  }

  resolveAlert(
    shopSlug: string,
    alertUuid: string,
    body: { note?: string } = {},
  ): Observable<ApiResponse<InventoryAlert>> {
    return this.http.post<ApiResponse<InventoryAlert>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/alerts/${alertUuid}/resolve`,
      body,
      { withCredentials: true },
    );
  }

  listLocations(shopSlug: string): Observable<ApiResponse<InventoryLocation[]>> {
    return this.http.get<ApiResponse<InventoryLocation[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/locations`,
      { withCredentials: true },
    );
  }

  createLocation(
    shopSlug: string,
    payload: { name: string; code: string; description?: string; isPrimary?: boolean },
  ): Observable<ApiResponse<InventoryLocation>> {
    return this.http.post<ApiResponse<InventoryLocation>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/locations`,
      {
        name: payload.name,
        code: payload.code,
        description: payload.description ?? '',
        isPrimary: !!payload.isPrimary,
      },
      { withCredentials: true },
    );
  }

  updateLocation(
    shopSlug: string,
    locationUuid: string,
    payload: Partial<{ name: string; code: string; description: string; isPrimary: boolean }>,
  ): Observable<ApiResponse<InventoryLocation>> {
    return this.http.patch<ApiResponse<InventoryLocation>>(
      `${this.api}/v1/my/shops/${shopSlug}/inventory/locations/${locationUuid}`,
      payload,
      { withCredentials: true },
    );
  }
}
