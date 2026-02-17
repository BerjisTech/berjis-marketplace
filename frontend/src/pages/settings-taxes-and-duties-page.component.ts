import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface TaxZone {
  uuid?: string;
  countryCode: string;
  regionCode: string;
  ratePercent: number;
  name: string;
}

interface TaxSettings {
  autoCalculate: boolean;
  defaultRatePercent: number;
  pricesIncludeTax: boolean;
}

@Component({
  standalone: true,
  selector: 'app-settings-taxes-and-duties-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-taxes-and-duties-page.component.html',
  styleUrls: ['./settings-taxes-and-duties-page.component.css']
})
export class SettingsTaxesAndDutiesPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  autoCalculate = signal(false);
  defaultRatePercent = signal(0);
  pricesIncludeTax = signal(false);
  zones = signal<TaxZone[]>([]);

  newCountryCode = '';
  newRegionCode = '';
  newRate = 0;
  newName = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/tax-settings`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      const settings = d?.settings ?? {};
      this.autoCalculate.set(settings?.autoCalculate ?? false);
      this.defaultRatePercent.set(settings?.defaultRatePercent ?? 0);
      this.pricesIncludeTax.set(settings?.pricesIncludeTax ?? false);
      const rawZones = d?.zones ?? [];
      this.zones.set(rawZones.map((z: any) => ({
        uuid: z.uuid ?? '',
        countryCode: z.countryCode ?? '',
        regionCode: z.regionCode ?? '',
        ratePercent: z.ratePercent ?? 0,
        name: z.name ?? '',
      })));
    } catch {
      this.error.set('Unable to load tax settings.');
    } finally {
      this.loading.set(false);
    }
  }

  addZone() {
    if (!this.newCountryCode.trim()) return;
    const zone: TaxZone = {
      countryCode: this.newCountryCode.trim().toUpperCase(),
      regionCode: this.newRegionCode.trim().toUpperCase(),
      ratePercent: this.newRate,
      name: this.newName.trim(),
    };
    this.zones.set([...this.zones(), zone]);
    this.newCountryCode = '';
    this.newRegionCode = '';
    this.newRate = 0;
    this.newName = '';
  }

  removeZone(index: number) {
    this.zones.set(this.zones().filter((_, i) => i !== index));
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      // Save settings
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/tax-settings`, {
          autoCalculate: this.autoCalculate(),
          defaultRatePercent: this.defaultRatePercent(),
          pricesIncludeTax: this.pricesIncludeTax(),
        }, { withCredentials: true })
      );
      // Save zones
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/tax-zones`, {
          zones: this.zones(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save tax settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
