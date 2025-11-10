import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';

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
}
