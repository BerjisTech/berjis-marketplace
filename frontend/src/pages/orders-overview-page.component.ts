import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  OrderService,
  ShopMetrics,
  ShopOrder,
  UpdateTrackingPayload,
} from '../app/core/services/order.service';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import { ApiResponse } from '../app/core/services/product.service';
import { environment } from '../environments/environment';

const ORDER_STATUSES = ['pending', 'processing', 'shipped', 'delivered', 'cancelled'] as const;

@Component({
  standalone: true,
  selector: 'app-orders-overview-page',
  imports: [CommonModule, FormsModule, RouterLink, ModalComponent],
  templateUrl: './orders-overview-page.component.html',
  styleUrls: ['./orders-overview-page.component.css'],
})
export class OrdersOverviewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly ordersService = inject(OrderService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly orders = signal<ShopOrder[]>([]);
  readonly metrics = signal<ShopMetrics | null>(null);
  readonly loading = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly statusFilter = signal<string>('all');
  readonly search = signal<string>('');

  readonly editModalOpen = signal<boolean>(false);
  readonly editOrder = signal<ShopOrder | null>(null);
  readonly editBusy = signal<boolean>(false);
  readonly editForm = signal<TrackingForm>({
    status: 'processing',
    trackingNumber: '',
    trackingUrl: '',
    shippingCarrier: '',
  });

  readonly filteredOrders = computed(() => {
    const orders = this.orders();
    const status = this.statusFilter();
    const search = this.search().trim().toLowerCase();
    return orders.filter((order) => {
      const matchesStatus = status === 'all' || order.status.toLowerCase() === status;
      const matchesSearch =
        !search ||
        order.uuid.toLowerCase().includes(search) ||
        (order.customerEmail ?? '').toLowerCase().includes(search);
      return matchesStatus && matchesSearch;
    });
  });

  ngOnInit(): void {
    this.bootstrap();
  }

  bootstrap(): void {
    this.loading.set(true);
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const shops = response?.data ?? [];
          this.shops.set(shops);
          const current = this.selectedShopSlug();
          const slug = current && shops.some((s) => s.slug === current) ? current : shops[0]?.slug ?? '';
          this.selectedShopSlug.set(slug);
          if (slug) {
            this.loadOrders();
          } else {
            this.loading.set(false);
          }
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Could not load shops or permissions.');
        },
      });
  }

  loadOrders(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.orders.set([]);
      this.metrics.set(null);
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.ordersService.listShopOrders(slug).subscribe({
      next: (response) => {
        this.orders.set(response?.data ?? []);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Could not load orders for this shop.');
      },
    });
    this.ordersService.getShopMetrics(slug).subscribe({
      next: (response) => {
        this.metrics.set(response?.data ?? null);
      },
      error: () => {
        this.metrics.set(null);
      },
    });
  }

  changeShop(slug: string): void {
    this.selectedShopSlug.set(slug);
    this.message.set('');
    this.statusFilter.set('all');
    this.search.set('');
    this.loadOrders();
  }

  changeStatusFilter(value: string): void {
    this.statusFilter.set(value);
  }

  changeSearch(value: string): void {
    this.search.set(value);
  }

  openEdit(order: ShopOrder): void {
    this.editOrder.set(order);
    this.editForm.set({
      status: order.status.toLowerCase(),
      trackingNumber: order.trackingNumber ?? '',
      trackingUrl: order.trackingUrl ?? '',
      shippingCarrier: order.shippingCarrier ?? '',
    });
    this.editModalOpen.set(true);
    this.message.set('');
    this.error.set('');
  }

  closeEdit(): void {
    this.editModalOpen.set(false);
    this.editBusy.set(false);
    this.editOrder.set(null);
    this.editForm.set({
      status: 'processing',
      trackingNumber: '',
      trackingUrl: '',
      shippingCarrier: '',
    });
  }

  saveEdit(): void {
    const order = this.editOrder();
    if (!order) {
      return;
    }
    const form = this.editForm();
    const payload: UpdateTrackingPayload = {
      status: form.status,
      trackingNumber: form.trackingNumber?.trim() || undefined,
      trackingUrl: form.trackingUrl?.trim() || undefined,
      shippingCarrier: form.shippingCarrier?.trim() || undefined,
    };
    this.editBusy.set(true);
    this.ordersService.updateTracking(order.uuid, payload).subscribe({
      next: () => {
        this.message.set('Order updated.');
        this.closeEdit();
        this.loadOrders();
      },
      error: () => {
        this.editBusy.set(false);
        this.error.set('Could not update order.');
      },
    });
  }

  formatCurrency(cents: number, currency: string): string {
    try {
      return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(
        (cents ?? 0) / 100,
      );
    } catch {
      return `${(cents ?? 0) / 100} ${currency}`;
    }
  }

  formatDate(value: string): string {
    if (!value) {
      return '—';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
  }

  statusOptions(): readonly string[] {
    return ORDER_STATUSES;
  }
}

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
}

interface TrackingForm {
  status: string;
  trackingNumber: string;
  trackingUrl: string;
  shippingCarrier: string;
}
