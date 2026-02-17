import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface MarketOption {
  id: string;
  name: string;
}

interface CatalogOverride {
  productId: string;
  productName: string;
  visible: boolean;
  priceOverride: string;
}

@Component({
  standalone: true,
  selector: 'app-markets-catalogs-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './markets-catalogs-page.component.html',
  styleUrls: ['./markets-catalogs-page.component.css']
})
export class MarketsCatalogsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  error = signal('');
  message = signal('');

  markets = signal<MarketOption[]>([]);
  selectedMarket = '';
  overrides = signal<CatalogOverride[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/markets`, { withCredentials: true })
      );
      this.markets.set(res?.data ?? []);
      if (this.markets().length && !this.selectedMarket) {
        this.selectedMarket = this.markets()[0].id;
        await this.loadOverrides();
      }
    } catch {
      this.error.set('Unable to load markets.');
    } finally {
      this.loading.set(false);
    }
  }

  async onMarketChange() {
    await this.loadOverrides();
  }

  async loadOverrides() {
    if (!this.selectedMarket) return;
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/markets/${this.selectedMarket}/catalog`, { withCredentials: true })
      );
      this.overrides.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load catalog overrides.');
    }
  }

  toggleVisibility(productId: string) {
    this.overrides.set(
      this.overrides().map(o => o.productId === productId ? { ...o, visible: !o.visible } : o)
    );
  }

  updatePrice(productId: string, price: string) {
    this.overrides.set(
      this.overrides().map(o => o.productId === productId ? { ...o, priceOverride: price } : o)
    );
  }

  async save() {
    this.saving.set(true);
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/markets/${this.selectedMarket}/catalog`, {
          overrides: this.overrides(),
        }, { withCredentials: true })
      );
      this.message.set('Catalog overrides saved.');
    } catch {
      this.error.set('Unable to save catalog overrides.');
    } finally {
      this.saving.set(false);
    }
  }
}
