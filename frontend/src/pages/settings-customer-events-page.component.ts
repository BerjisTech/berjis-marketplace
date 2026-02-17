import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-customer-events-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-customer-events-page.component.html',
  styleUrls: ['./settings-customer-events-page.component.css']
})
export class SettingsCustomerEventsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  facebookPixel = '';
  googleAnalytics = '';
  tiktokPixel = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/tracking-pixels`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.facebookPixel = d?.facebookPixel ?? '';
      this.googleAnalytics = d?.googleAnalytics ?? '';
      this.tiktokPixel = d?.tiktokPixel ?? '';
    } catch {
      this.error.set('Unable to load tracking pixel settings.');
    } finally {
      this.loading.set(false);
    }
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/tracking-pixels`, {
          facebookPixel: this.facebookPixel.trim(),
          googleAnalytics: this.googleAnalytics.trim(),
          tiktokPixel: this.tiktokPixel.trim(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save tracking pixel settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
