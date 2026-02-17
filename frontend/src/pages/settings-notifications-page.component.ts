import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface NotificationSetting {
  id: string;
  label: string;
  description: string;
  enabled: boolean;
}

@Component({
  standalone: true,
  selector: 'app-settings-notifications-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-notifications-page.component.html',
  styleUrls: ['./settings-notifications-page.component.css']
})
export class SettingsNotificationsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');
  testSending = signal('');

  notifications = signal<NotificationSetting[]>([
    { id: 'order_confirmation', label: 'Order confirmation', description: 'Send email when an order is placed.', enabled: true },
    { id: 'shipping_update', label: 'Shipping update', description: 'Send email when shipping status changes.', enabled: true },
    { id: 'refund', label: 'Refund notification', description: 'Send email when a refund is issued.', enabled: true },
    { id: 'abandoned_cart', label: 'Abandoned cart', description: 'Send reminder for abandoned carts.', enabled: false },
  ]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/notification-settings`, { withCredentials: true })
      );
      const data = res?.data ?? [];
      if (data.length) {
        this.notifications.set(data);
      }
    } catch {
      this.error.set('Unable to load notification settings.');
    } finally {
      this.loading.set(false);
    }
  }

  toggleNotification(id: string) {
    this.notifications.set(
      this.notifications().map(n => n.id === id ? { ...n, enabled: !n.enabled } : n)
    );
  }

  async sendTest(id: string) {
    this.testSending.set(id);
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/notifications/${id}/test`, {}, { withCredentials: true })
      );
      this.error.set('');
    } catch {
      this.error.set('Unable to send test notification.');
    } finally {
      this.testSending.set('');
    }
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/notification-settings`, {
          notifications: this.notifications(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save notification settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
