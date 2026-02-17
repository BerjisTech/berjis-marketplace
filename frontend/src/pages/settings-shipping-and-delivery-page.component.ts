import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface ShippingRate {
  uuid: string;
  name: string;
  priceCents: number;
  minOrderCents?: number;
  maxOrderCents?: number;
  freeAboveCents?: number;
  estimatedDaysMin?: number;
  estimatedDaysMax?: number;
}

interface ShippingZone {
  uuid: string;
  name: string;
  countries: string[];
  isRestOfWorld: boolean;
  rates: ShippingRate[];
}

@Component({
  standalone: true,
  selector: 'app-settings-shipping-and-delivery-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-shipping-and-delivery-page.component.html',
  styleUrls: ['./settings-shipping-and-delivery-page.component.css']
})
export class SettingsShippingAndDeliveryPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  zones = signal<ShippingZone[]>([]);

  newZoneName = '';
  newZoneCountries = '';
  newZoneIsROW = false;

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/shipping-zones`, { withCredentials: true })
      );
      this.zones.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load shipping zones.');
    } finally {
      this.loading.set(false);
    }
  }

  addZone() {
    if (!this.newZoneName.trim()) return;
    const countries = this.newZoneCountries.split(',').map(c => c.trim().toUpperCase()).filter(Boolean);
    const zone: ShippingZone = {
      uuid: crypto.randomUUID(),
      name: this.newZoneName.trim(),
      countries,
      isRestOfWorld: this.newZoneIsROW,
      rates: [{ uuid: crypto.randomUUID(), name: 'Standard', priceCents: 1500 }],
    };
    this.zones.set([...this.zones(), zone]);
    this.newZoneName = '';
    this.newZoneCountries = '';
    this.newZoneIsROW = false;
  }

  removeZone(uuid: string) {
    this.zones.set(this.zones().filter(z => z.uuid !== uuid));
  }

  addRate(zoneUuid: string) {
    this.zones.set(this.zones().map(z => {
      if (z.uuid !== zoneUuid) return z;
      return { ...z, rates: [...z.rates, { uuid: crypto.randomUUID(), name: 'New rate', priceCents: 0 }] };
    }));
  }

  removeRate(zoneUuid: string, rateUuid: string) {
    this.zones.set(this.zones().map(z => {
      if (z.uuid !== zoneUuid) return z;
      return { ...z, rates: z.rates.filter(r => r.uuid !== rateUuid) };
    }));
  }

  updateRate(zoneUuid: string, rateUuid: string, field: string, value: unknown) {
    this.zones.set(this.zones().map(z => {
      if (z.uuid !== zoneUuid) return z;
      return {
        ...z,
        rates: z.rates.map(r => r.uuid === rateUuid ? { ...r, [field]: value } : r),
      };
    }));
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/shipping-zones`, {
          zones: this.zones(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save shipping zones.');
    } finally {
      this.saving.set(false);
    }
  }
}
