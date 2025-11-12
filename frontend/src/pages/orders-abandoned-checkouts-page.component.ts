import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  AbandonedCheckout,
  CheckoutRecoveryResponse,
  OrderService,
} from '../app/core/services/order.service';
import { ApiResponse } from '../app/core/services/product.service';
import { environment } from '../environments/environment';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
}

@Component({
  standalone: true,
  selector: 'app-orders-abandoned-checkouts-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './orders-abandoned-checkouts-page.component.html',
  styleUrls: ['./orders-abandoned-checkouts-page.component.css'],
})
export class OrdersAbandonedCheckoutsPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly ordersService = inject(OrderService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly abandonedCheckouts = signal<AbandonedCheckout[]>([]);
  readonly loading = signal<boolean>(false);
  readonly error = signal<string>('');
  readonly message = signal<string>('');
  readonly recoveryBusy = signal<string>('');
  readonly recoveryLinks = signal<Record<string, CheckoutRecoveryResponse>>({});

  readonly totalPotential = computed(() =>
    this.abandonedCheckouts().reduce((acc, checkout) => acc + (checkout.subtotalCents ?? 0), 0),
  );

  readonly averagePotential = computed(() => {
    const list = this.abandonedCheckouts();
    if (!list.length) {
      return 0;
    }
    return Math.round(this.totalPotential() / list.length);
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
            this.loadAbandoned();
          } else {
            this.loading.set(false);
          }
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Unable to load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    if (!slug || slug === this.selectedShopSlug()) {
      return;
    }
    this.selectedShopSlug.set(slug);
    this.message.set('');
    this.error.set('');
    this.abandonedCheckouts.set([]);
    this.loadAbandoned();
  }

  refresh(): void {
    this.loadAbandoned();
  }

  recoveryLinkFor(checkout: AbandonedCheckout): CheckoutRecoveryResponse | undefined {
    return this.recoveryLinks()[checkout.cartUuid];
  }

  loadAbandoned(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.loading.set(false);
      return;
    }
    this.loading.set(true);
    this.ordersService.listAbandonedCheckouts(slug).subscribe({
      next: (response: ApiResponse<AbandonedCheckout[]>) => {
        const data = response?.data ?? [];
        this.abandonedCheckouts.set(data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Unable to load abandoned checkouts.');
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

  formatDate(value?: string): string {
    if (!value) {
      return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
  }

  formatRelative(value?: string): string {
    if (!value) {
      return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }
    const diff = Date.now() - date.getTime();
    const hours = Math.round(diff / (1000 * 60 * 60));
    if (hours < 1) {
      return 'Just now';
    }
    if (hours < 24) {
      return `${hours}h ago`;
    }
    const days = Math.round(hours / 24);
    if (days < 7) {
      return `${days}d ago`;
    }
    const weeks = Math.round(days / 7);
    return `${weeks}w ago`;
  }

  potentialValueClass(cents: number): string {
    if (cents >= 50000) {
      return 'text-emerald-600 dark:text-emerald-300';
    }
    if (cents >= 20000) {
      return 'text-blue-600 dark:text-blue-300';
    }
    return 'text-slate-600 dark:text-slate-300';
  }

  copyEmail(email?: string): void {
    if (!email) {
      return;
    }
    this.copyText(email, 'Email copied to clipboard.');
  }

  checkoutContact(checkout: AbandonedCheckout): string {
    if (checkout.customerEmail) {
      return checkout.customerEmail;
    }
    if (checkout.customerName) {
      return checkout.customerName;
    }
    return checkout.userUuid.slice(0, 8);
  }

  generateRecovery(checkout: AbandonedCheckout): void {
    const slug = this.selectedShopSlug();
    if (!slug || this.recoveryBusy() === checkout.cartUuid) {
      return;
    }
    this.recoveryBusy.set(checkout.cartUuid);
    const payload = checkout.customerEmail
      ? { email: checkout.customerEmail }
      : undefined;
    this.ordersService.createCheckoutRecovery(slug, checkout.cartUuid, payload).subscribe({
      next: (response: ApiResponse<CheckoutRecoveryResponse>) => {
        const data = response?.data;
        if (data) {
          const current = { ...this.recoveryLinks() };
          current[checkout.cartUuid] = data;
          this.recoveryLinks.set(current);
          this.message.set('Recovery link generated.');
          setTimeout(() => this.message.set(''), 2000);
        }
        this.recoveryBusy.set('');
      },
      error: () => {
        this.recoveryBusy.set('');
        this.error.set('Could not generate recovery link.');
        setTimeout(() => this.error.set(''), 3000);
      },
    });
  }

  copyRecoveryLink(checkout: AbandonedCheckout): void {
    const recovery = this.recoveryLinkFor(checkout);
    if (!recovery) {
      return;
    }
    const value = recovery.recoveryUrl || recovery.token;
    this.copyText(value, 'Recovery link copied.');
  }

  private copyText(value: string, successMessage: string): void {
    if (!navigator?.clipboard?.writeText) {
      return;
    }
    navigator.clipboard
      .writeText(value)
      .then(() => {
        this.message.set(successMessage);
        setTimeout(() => this.message.set(''), 2000);
      })
      .catch(() => {
        this.error.set('Unable to copy text.');
        setTimeout(() => this.error.set(''), 2000);
      });
  }
}
