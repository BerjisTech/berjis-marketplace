import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface Supplier {
  uuid: string;
  shopUuid: string;
  name: string;
  contactEmail?: string | null;
  phone?: string | null;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface PurchaseOrderItem {
  purchaseOrderUuid: string;
  productUuid: string;
  productTitle: string;
  quantity: number;
  costCents: number;
  receivedQuantity: number;
}

export interface PurchaseOrder {
  uuid: string;
  shopUuid: string;
  supplierUuid?: string | null;
  supplierName?: string | null;
  supplierEmail?: string | null;
  supplierPhone?: string | null;
  status: string;
  expectedAt?: string | null;
  notes: string;
  createdBy?: string | null;
  createdAt: string;
  updatedAt: string;
  items: PurchaseOrderItem[];
}

export interface CreatePurchaseOrderItem {
  productUuid: string;
  quantity: number;
  costCents: number;
}

export interface CreatePurchaseOrderPayload {
  supplierUuid?: string;
  supplierName?: string;
  contactEmail?: string;
  phone?: string;
  notes?: string;
  expectedAt?: string;
  status?: string;
  items: CreatePurchaseOrderItem[];
}

export interface ReceivePurchaseOrderPayload {
  locationUuid?: string;
  note?: string;
  items: { productUuid: string; quantity: number }[];
}

@Injectable({ providedIn: 'root' })
export class PurchaseOrderService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(shopSlug: string): Observable<ApiResponse<PurchaseOrder[]>> {
    return this.http.get<ApiResponse<PurchaseOrder[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/purchase-orders`,
      { withCredentials: true },
    );
  }

  create(shopSlug: string, payload: CreatePurchaseOrderPayload): Observable<ApiResponse<PurchaseOrder>> {
    return this.http.post<ApiResponse<PurchaseOrder>>(
      `${this.api}/v1/my/shops/${shopSlug}/purchase-orders`,
      payload,
      { withCredentials: true },
    );
  }

  receive(
    shopSlug: string,
    orderUuid: string,
    payload: ReceivePurchaseOrderPayload,
  ): Observable<ApiResponse<PurchaseOrder>> {
    return this.http.post<ApiResponse<PurchaseOrder>>(
      `${this.api}/v1/my/shops/${shopSlug}/purchase-orders/${orderUuid}/receive`,
      payload,
      { withCredentials: true },
    );
  }

  listSuppliers(shopSlug: string): Observable<ApiResponse<Supplier[]>> {
    return this.http.get<ApiResponse<Supplier[]>>(
      `${this.api}/v1/my/shops/${shopSlug}/suppliers`,
      { withCredentials: true },
    );
  }

  createSupplier(
    shopSlug: string,
    payload: { name: string; contactEmail?: string; phone?: string; notes?: string },
  ): Observable<ApiResponse<Supplier>> {
    return this.http.post<ApiResponse<Supplier>>(
      `${this.api}/v1/my/shops/${shopSlug}/suppliers`,
      payload,
      { withCredentials: true },
    );
  }
}
