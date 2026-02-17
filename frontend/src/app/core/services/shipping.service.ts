import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface ShippingRate {
  uuid: string;
  zoneName: string;
  rateName: string;
  priceCents: number;
  freeAboveCents?: number;
  estimatedDays?: string;
  isFree: boolean;
}

interface ShippingRatesResponse {
  success: boolean;
  data?: ShippingRate[];
}

@Injectable({ providedIn: 'root' })
export class ShippingService {
  private readonly api = environment.apiBase;
  private readonly http = inject(HttpClient);

  async calculateRates(shopUuid: string, country: string, subtotalCents: number): Promise<ShippingRate[]> {
    const res = await firstValueFrom(
      this.http.post<ShippingRatesResponse>(`${this.api}/v1/shipping/rates`, {
        shopUuid,
        country,
        subtotalCents,
      })
    );
    return res?.data ?? [];
  }
}
