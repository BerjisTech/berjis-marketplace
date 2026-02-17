import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Webhook {
  uuid: string;
  shopUuid: string;
  url: string;
  secret: string;
  events: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface WebhookDelivery {
  uuid: string;
  webhookUuid: string;
  eventType: string;
  payload: any;
  responseStatus?: number;
  responseBody?: string;
  attempt: number;
  status: string;
  nextRetryAt?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

const ALL_EVENTS = [
  'order.created',
  'order.paid',
  'order.cancelled',
  'order.refunded',
  'fulfillment.created',
  'fulfillment.shipped',
  'product.created',
  'product.updated',
  'product.deleted',
  'customer.created',
  'inventory.low_stock',
];

@Component({
  standalone: true,
  selector: 'app-settings-webhooks-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-webhooks-page.component.html',
  styleUrls: ['./settings-webhooks-page.component.css']
})
export class SettingsWebhooksPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  error = signal('');
  success = signal('');

  webhooks = signal<Webhook[]>([]);
  allEvents = ALL_EVENTS;

  // Form state
  showForm = signal(false);
  editingUuid = signal<string | null>(null);
  formUrl = '';
  formSecret = '';
  formEvents: Record<string, boolean> = {};
  formIsActive = true;

  // Delivery log
  expandedWebhookUuid = signal<string | null>(null);
  deliveries = signal<WebhookDelivery[]>([]);
  deliveriesTotal = signal(0);
  deliveriesLoading = signal(false);
  expandedDeliveryUuid = signal<string | null>(null);

  ngOnInit() {
    this.resetFormEvents();
    this.load();
  }

  resetFormEvents() {
    this.formEvents = {};
    for (const e of ALL_EVENTS) {
      this.formEvents[e] = false;
    }
  }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/webhooks`, { withCredentials: true })
      );
      this.webhooks.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load webhooks.');
    } finally {
      this.loading.set(false);
    }
  }

  openAddForm() {
    this.editingUuid.set(null);
    this.formUrl = '';
    this.formSecret = '';
    this.formIsActive = true;
    this.resetFormEvents();
    this.showForm.set(true);
    this.error.set('');
    this.success.set('');
  }

  openEditForm(wh: Webhook) {
    this.editingUuid.set(wh.uuid);
    this.formUrl = wh.url;
    this.formSecret = wh.secret;
    this.formIsActive = wh.isActive;
    this.resetFormEvents();
    for (const e of wh.events ?? []) {
      this.formEvents[e] = true;
    }
    this.showForm.set(true);
    this.error.set('');
    this.success.set('');
  }

  cancelForm() {
    this.showForm.set(false);
    this.editingUuid.set(null);
  }

  getSelectedEvents(): string[] {
    return Object.entries(this.formEvents)
      .filter(([, v]) => v)
      .map(([k]) => k);
  }

  async saveWebhook() {
    this.saving.set(true);
    this.error.set('');
    this.success.set('');
    const slug = this.shop.activeShopSlug();

    const payload = {
      url: this.formUrl.trim(),
      secret: this.formSecret,
      events: this.getSelectedEvents(),
      isActive: this.formIsActive,
    };

    if (!payload.url) {
      this.error.set('URL is required.');
      this.saving.set(false);
      return;
    }
    if (payload.events.length === 0) {
      this.error.set('Select at least one event.');
      this.saving.set(false);
      return;
    }

    try {
      if (this.editingUuid()) {
        await firstValueFrom(
          this.http.patch(`${this.api}/v1/webhooks/${this.editingUuid()}`, payload, { withCredentials: true })
        );
        this.success.set('Webhook updated.');
      } else {
        await firstValueFrom(
          this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/webhooks`, payload, { withCredentials: true })
        );
        this.success.set('Webhook created.');
      }
      this.showForm.set(false);
      this.editingUuid.set(null);
      await this.load();
    } catch {
      this.error.set('Failed to save webhook.');
    } finally {
      this.saving.set(false);
    }
  }

  async deleteWebhook(uuid: string) {
    if (!confirm('Delete this webhook? This cannot be undone.')) return;
    this.error.set('');
    this.success.set('');
    try {
      await firstValueFrom(
        this.http.delete(`${this.api}/v1/webhooks/${uuid}`, { withCredentials: true })
      );
      this.success.set('Webhook deleted.');
      await this.load();
    } catch {
      this.error.set('Failed to delete webhook.');
    }
  }

  async toggleActive(wh: Webhook) {
    this.error.set('');
    try {
      await firstValueFrom(
        this.http.patch(`${this.api}/v1/webhooks/${wh.uuid}`, { isActive: !wh.isActive }, { withCredentials: true })
      );
      await this.load();
    } catch {
      this.error.set('Failed to update webhook.');
    }
  }

  async toggleDeliveries(webhookUuid: string) {
    if (this.expandedWebhookUuid() === webhookUuid) {
      this.expandedWebhookUuid.set(null);
      this.deliveries.set([]);
      return;
    }
    this.expandedWebhookUuid.set(webhookUuid);
    this.expandedDeliveryUuid.set(null);
    await this.loadDeliveries(webhookUuid);
  }

  async loadDeliveries(webhookUuid: string) {
    this.deliveriesLoading.set(true);
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/webhooks/${webhookUuid}/deliveries?limit=20`, { withCredentials: true })
      );
      this.deliveries.set(res?.data?.deliveries ?? []);
      this.deliveriesTotal.set(res?.data?.total ?? 0);
    } catch {
      this.error.set('Failed to load delivery log.');
    } finally {
      this.deliveriesLoading.set(false);
    }
  }

  toggleDeliveryDetail(uuid: string) {
    this.expandedDeliveryUuid.set(this.expandedDeliveryUuid() === uuid ? null : uuid);
  }

  statusColor(status: string): string {
    switch (status) {
      case 'delivered': return 'text-green-600 dark:text-green-400';
      case 'failed': return 'text-red-600 dark:text-red-400';
      default: return 'text-yellow-600 dark:text-yellow-400';
    }
  }

  formatPayload(payload: any): string {
    try {
      return JSON.stringify(typeof payload === 'string' ? JSON.parse(payload) : payload, null, 2);
    } catch {
      return String(payload);
    }
  }
}
