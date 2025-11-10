import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';

export interface PricingConfig {
  taxRatePercent: number;
  shippingFlatCents: number;
}

interface PricingConfigDto {
  taxRatePercent?: number | string;
  shippingFlatCents?: number | string;
}

interface PricingApiResponse {
  data?: PricingConfigDto;
}

@Injectable({ providedIn: 'root' })
export class PricingService {
  config = signal<PricingConfig>({ taxRatePercent: 8, shippingFlatCents: 1500 });
  private loading = false;
  private loaded = false;
  private readonly api = environment.apiBase;
  private readonly http = inject(HttpClient);

  constructor() {
    void this.refresh();
  }

  async refresh(force = false): Promise<void> {
    if (this.loading || (this.loaded && !force)) {
      return;
    }
    this.loading = true;
    try {
      const response = await firstValueFrom(
        this.http.get<PricingApiResponse>(`${this.api}/v1/settings/pricing`, { withCredentials: true })
      );
      const data = response?.data ?? {};
      const tax = this.normalizeNumber(data.taxRatePercent, this.config().taxRatePercent);
      const shipping = this.normalizeNumber(data.shippingFlatCents, this.config().shippingFlatCents);
      this.config.set({
        taxRatePercent: tax,
        shippingFlatCents: Math.max(0, Math.round(shipping))
      });
      this.loaded = true;
    } catch {
      this.loaded = false;
    } finally {
      this.loading = false;
    }
  }

  private normalizeNumber(input: unknown, fallback: number): number {
    if (typeof input === 'number' && !Number.isNaN(input)) {
      return input;
    }
    if (typeof input === 'string') {
      const parsed = parseFloat(input);
      if (!Number.isNaN(parsed)) {
        return parsed;
      }
    }
    return fallback;
  }
}
