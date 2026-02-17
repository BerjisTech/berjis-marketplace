import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-general-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-general-page.component.html',
  styleUrls: ['./settings-general-page.component.css']
})
export class SettingsGeneralPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  name = '';
  contactEmail = '';
  description = '';
  currency = 'USD';
  timezone = 'UTC';

  currencies = ['USD', 'EUR', 'GBP', 'CAD', 'AUD', 'KES', 'NGN', 'ZAR'];
  timezones = ['UTC', 'America/New_York', 'America/Los_Angeles', 'Europe/London', 'Europe/Paris', 'Africa/Nairobi', 'Asia/Tokyo'];

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.name = d?.name ?? '';
      this.contactEmail = d?.contactEmail ?? d?.contact_email ?? '';
      this.description = d?.description ?? '';
      this.currency = d?.currency ?? 'USD';
      this.timezone = d?.timezone ?? 'UTC';
    } catch {
      this.error.set('Unable to load shop settings.');
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
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}`, {
          name: this.name.trim(),
          contactEmail: this.contactEmail.trim(),
          description: this.description.trim(),
          currency: this.currency,
          timezone: this.timezone,
        }, { withCredentials: true })
      );
      this.saved.set(true);
      this.shop.refresh();
    } catch {
      this.error.set('Unable to save settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
