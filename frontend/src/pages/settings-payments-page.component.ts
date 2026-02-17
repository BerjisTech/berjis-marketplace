import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface PaymentProvider {
  id: string;
  name: string;
  enabled: boolean;
  apiKey: string;
  secretKey: string;
}

@Component({
  standalone: true,
  selector: 'app-settings-payments-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-payments-page.component.html',
  styleUrls: ['./settings-payments-page.component.css']
})
export class SettingsPaymentsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  providers = signal<PaymentProvider[]>([
    { id: 'stripe', name: 'Stripe', enabled: false, apiKey: '', secretKey: '' },
    { id: 'flutterwave', name: 'Flutterwave', enabled: false, apiKey: '', secretKey: '' },
    { id: 'mpesa', name: 'M-Pesa', enabled: false, apiKey: '', secretKey: '' },
  ]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/payment-settings`, { withCredentials: true })
      );
      const data = res?.data ?? [];
      if (data.length) {
        this.providers.set(data);
      }
    } catch {
      this.error.set('Unable to load payment settings.');
    } finally {
      this.loading.set(false);
    }
  }

  toggleProvider(id: string) {
    this.providers.set(
      this.providers().map(p => p.id === id ? { ...p, enabled: !p.enabled } : p)
    );
  }

  updateProvider(id: string, field: 'apiKey' | 'secretKey', value: string) {
    this.providers.set(
      this.providers().map(p => p.id === id ? { ...p, [field]: value } : p)
    );
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/payment-settings`, {
          providers: this.providers(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save payment settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
