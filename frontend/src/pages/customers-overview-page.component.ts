import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  CreateCustomerPayload,
  CustomerOrder,
  CustomerService,
  CustomerSummary,
  UpdateCustomerPayload,
} from '../app/core/services/customer.service';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import { ApiResponse } from '../app/core/services/product.service';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'app-customers-overview-page',
  imports: [CommonModule, FormsModule, RouterLink, ModalComponent],
  templateUrl: './customers-overview-page.component.html',
  styleUrls: ['./customers-overview-page.component.css'],
})
export class CustomersOverviewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly customerService = inject(CustomerService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly customersList = signal<CustomerSummary[]>([]);
  readonly loading = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly search = signal<string>('');
  readonly tagFilter = signal<string>('');

  readonly newCustomer = signal<NewCustomerForm>({
    email: '',
    firstName: '',
    lastName: '',
    phone: '',
    tags: '',
    notes: '',
    marketingOptIn: false,
  });
  readonly createBusy = signal<boolean>(false);

  readonly detailOpen = signal<boolean>(false);
  readonly detailLoading = signal<boolean>(false);
  readonly detailError = signal<string>('');
  readonly detailOrdersLoading = signal<boolean>(false);
  readonly detailOrdersError = signal<string>('');
  readonly detailOrders = signal<CustomerOrder[]>([]);
  readonly selectedCustomer = signal<CustomerSummary | null>(null);
  readonly detailTimeline = signal<CustomerTimelineEvent[]>([]);

  readonly editModalOpen = signal<boolean>(false);
  readonly editLoading = signal<boolean>(false);
  readonly editError = signal<string>('');
  readonly editBusy = signal<boolean>(false);
  readonly editingCustomer = signal<CustomerSummary | null>(null);
  readonly editForm = signal<EditCustomerForm>({
    email: '',
    firstName: '',
    lastName: '',
    phone: '',
    notes: '',
    tags: '',
    marketingOptIn: false,
  });

  readonly hasCustomers = computed(() => !this.loading() && this.customersList().length > 0);

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
          const slug = current && shops.some((shop) => shop.slug === current) ? current : shops[0]?.slug ?? '';
          this.selectedShopSlug.set(slug);
          if (slug) {
            this.loadCustomers();
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

  loadCustomers(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.customersList.set([]);
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.customerService
      .list(slug, {
        q: this.search().trim(),
        tag: this.tagFilter().trim(),
      })
      .subscribe({
        next: (response) => {
          this.customersList.set(response?.data ?? []);
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Could not load customers for this shop.');
        },
      });
  }

  changeShop(slug: string): void {
    this.selectedShopSlug.set(slug);
    this.search.set('');
    this.tagFilter.set('');
    this.message.set('');
    this.loadCustomers();
  }

  updateSearch(value: string): void {
    this.search.set(value);
    this.loadCustomers();
  }

  updateTagFilter(value: string): void {
    this.tagFilter.set(value);
    this.loadCustomers();
  }

  createCustomer(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      return;
    }
    const draft = this.newCustomer();
    const payload: CreateCustomerPayload = {
      email: draft.email.trim(),
      firstName: draft.firstName.trim(),
      lastName: draft.lastName.trim(),
      phone: draft.phone.trim(),
      notes: draft.notes.trim(),
      marketingOptIn: draft.marketingOptIn,
      tags: this.parseTags(draft.tags),
    };
    if (!payload.email) {
      this.error.set('Email is required to create a customer.');
      return;
    }
    this.createBusy.set(true);
    this.customerService.create(slug, payload).subscribe({
      next: (response) => {
        const created = response?.data;
        if (created) {
          this.customersList.set([created, ...this.customersList()]);
          this.message.set('Customer created.');
        }
        this.resetNewCustomerForm();
        this.createBusy.set(false);
      },
      error: () => {
        this.createBusy.set(false);
        this.error.set('Could not create customer.');
      },
    });
  }

  openDetailDrawer(customer: CustomerSummary): void {
    this.selectedCustomer.set(customer);
    this.detailOpen.set(true);
    this.detailError.set('');
    this.detailOrdersError.set('');
    this.detailLoading.set(true);
    this.detailOrdersLoading.set(true);
    this.detailTimeline.set(this.buildTimeline(customer, this.detailOrders()));

    this.customerService.get(customer.uuid).subscribe({
      next: (response) => {
        const latest = response?.data ?? customer;
        this.selectedCustomer.set(latest);
        this.detailLoading.set(false);
        this.detailTimeline.set(this.buildTimeline(latest, this.detailOrders()));
      },
      error: () => {
        this.detailLoading.set(false);
        this.detailError.set('Could not load customer details.');
      },
    });

    this.customerService.listOrders(customer.uuid).subscribe({
      next: (response) => {
        const orders = response?.data ?? [];
        this.detailOrders.set(orders);
        this.detailOrdersLoading.set(false);
        this.detailTimeline.set(this.buildTimeline(this.selectedCustomer(), orders));
      },
      error: () => {
        this.detailOrders.set([]);
        this.detailOrdersLoading.set(false);
        this.detailOrdersError.set('Could not load order history.');
      },
    });
  }

  closeDetailDrawer(): void {
    this.detailOpen.set(false);
    this.detailLoading.set(false);
    this.detailOrdersLoading.set(false);
    this.detailError.set('');
    this.detailOrdersError.set('');
    this.detailOrders.set([]);
    this.selectedCustomer.set(null);
    this.detailTimeline.set([]);
  }

  openEditModal(customer: CustomerSummary): void {
    this.editingCustomer.set(customer);
    this.editError.set('');
    this.editForm.set(this.toEditForm(customer));
    this.editModalOpen.set(true);
    this.editLoading.set(true);
    this.customerService.get(customer.uuid).subscribe({
      next: (response) => {
        const latest = response?.data;
        if (latest) {
          this.editingCustomer.set(latest);
          this.editForm.set(this.toEditForm(latest));
        }
        this.editLoading.set(false);
      },
      error: () => {
        this.editLoading.set(false);
        this.editError.set('Could not load latest customer details. Editing cached data.');
      },
    });
  }

  closeEditModal(): void {
    this.editModalOpen.set(false);
    this.editLoading.set(false);
    this.editBusy.set(false);
    this.editError.set('');
    this.editingCustomer.set(null);
    this.resetEditForm();
  }

  updateCustomer(): void {
    const customer = this.editingCustomer();
    if (!customer) {
      return;
    }
    const draft = this.editForm();
    const email = draft.email?.trim() ?? '';
    if (!email) {
      this.editError.set('Email is required to save a customer.');
      return;
    }
    this.editError.set('');
    const payload: UpdateCustomerPayload = {
      email,
      firstName: draft.firstName?.trim() ?? '',
      lastName: draft.lastName?.trim() ?? '',
      phone: draft.phone?.trim() ?? '',
      notes: draft.notes?.trim() ?? '',
      marketingOptIn: draft.marketingOptIn,
      tags: this.parseTags(draft.tags),
    };
    this.editBusy.set(true);
    this.customerService.update(customer.uuid, payload).subscribe({
      next: (response) => {
        const updated = response?.data;
        if (updated) {
          this.customersList.set(
            this.customersList().map((existing) => (existing.uuid === updated.uuid ? updated : existing)),
          );
          this.message.set('Customer updated.');
          this.error.set('');
          this.editingCustomer.set(updated);
          const selected = this.selectedCustomer();
          if (selected && selected.uuid === updated.uuid) {
            this.selectedCustomer.set(updated);
            this.detailTimeline.set(this.buildTimeline(updated, this.detailOrders()));
          }
        }
        this.editBusy.set(false);
        this.closeEditModal();
      },
      error: () => {
        this.editBusy.set(false);
        this.editError.set('Could not update customer.');
      },
    });
  }

  customerName(customer: CustomerSummary): string {
    if (customer.firstName || customer.lastName) {
      return [customer.firstName, customer.lastName].filter(Boolean).join(' ');
    }
    return customer.email;
  }

  tags(customer: CustomerSummary): string[] {
    return Array.isArray(customer.tags) ? customer.tags.filter(Boolean) : [];
  }

  marketingStatus(customer: CustomerSummary | null): string {
    if (!customer) {
      return 'Unknown';
    }
    return customer.marketingOptIn ? 'Opted in to marketing emails' : 'Not opted into marketing emails';
  }

  hasLinkedAccount(customer: CustomerSummary | null): boolean {
    return !!customer?.userUuid;
  }

  averageOrderValue(customer: CustomerSummary | null): string {
    if (!customer || customer.ordersCount === 0) {
      return this.formatCurrency(0, 'USD');
    }
    const avg = Math.round(customer.totalSpentCents / Math.max(customer.ordersCount, 1));
    return this.formatCurrency(avg, 'USD');
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

  timelineIcon(event: CustomerTimelineEvent): string {
    switch (event.type) {
      case 'order':
        return 'receipt_long';
      case 'shipping':
        return 'local_shipping';
      case 'marketing':
        return 'campaign';
      case 'note':
        return 'sticky_note_2';
      default:
        return 'person';
    }
  }

  timelineAccent(event: CustomerTimelineEvent): string {
    switch (event.type) {
      case 'order':
        return 'bg-blue-500/15 text-blue-500';
      case 'shipping':
        return 'bg-amber-500/15 text-amber-500';
      case 'marketing':
        return 'bg-emerald-500/15 text-emerald-500';
      case 'note':
        return 'bg-purple-500/15 text-purple-500';
      default:
        return 'bg-slate-500/15 text-slate-600 dark:text-slate-300';
    }
  }

  private toEditForm(customer: CustomerSummary): EditCustomerForm {
    return {
      email: customer.email ?? '',
      firstName: customer.firstName ?? '',
      lastName: customer.lastName ?? '',
      phone: customer.phone ?? '',
      notes: customer.notes ?? '',
      tags: this.joinTags(customer.tags),
      marketingOptIn: customer.marketingOptIn ?? false,
    };
  }

  private joinTags(tags: string[] | undefined): string {
    if (!Array.isArray(tags) || tags.length === 0) {
      return '';
    }
    return tags.filter(Boolean).join(', ');
  }

  private parseTags(value: string): string[] {
    if (!value) {
      return [];
    }
    return value
      .split(',')
      .map((tag) => tag.trim().toLowerCase())
      .filter((tag, index, arr) => tag.length > 0 && arr.indexOf(tag) === index)
      .slice(0, 10);
  }

  private resetNewCustomerForm(): void {
    this.newCustomer.set({
      email: '',
      firstName: '',
      lastName: '',
      phone: '',
      notes: '',
      tags: '',
      marketingOptIn: false,
    });
  }

  private buildTimeline(summary: CustomerSummary | null, orders: CustomerOrder[]): CustomerTimelineEvent[] {
    const events: CustomerTimelineEvent[] = [];
    if (summary) {
      events.push({
        type: 'created',
        title: 'Customer added',
        subtitle: summary.email,
        timestamp: summary.createdAt,
      });
      if (summary.marketingOptIn) {
        events.push({
          type: 'marketing',
          title: 'Opted into marketing emails',
          timestamp: summary.updatedAt ?? summary.createdAt,
        });
      }
      if (summary.notes) {
        events.push({
          type: 'note',
          title: 'Internal note updated',
          subtitle: summary.notes.slice(0, 120),
          timestamp: summary.updatedAt ?? summary.createdAt,
        });
      }
      if (summary.lastOrderAt) {
        events.push({
          type: 'order',
          title: 'Most recent order placed',
          subtitle: this.formatDate(summary.lastOrderAt),
          timestamp: summary.lastOrderAt,
        });
      }
    }
    orders.forEach((order) => {
      events.push({
        type: 'order',
        title: `Order ${order.uuid.slice(0, 8)} placed`,
        subtitle: `${this.formatCurrency(order.totalCents ?? 0, order.currency ?? 'USD')} • ${order.status.toUpperCase()}`,
        timestamp: order.createdAt,
      });
      if (order.shippedAt) {
        events.push({
          type: 'shipping',
          title: `Order ${order.uuid.slice(0, 8)} shipped${order.shippingCarrier ? ` via ${order.shippingCarrier}` : ''}`,
          subtitle: order.trackingNumber ? `Tracking ${order.trackingNumber}` : undefined,
          timestamp: order.shippedAt,
        });
      }
      if (order.deliveredAt) {
        events.push({
          type: 'shipping',
          title: `Order ${order.uuid.slice(0, 8)} delivered`,
          timestamp: order.deliveredAt,
        });
      }
    });

    return events
      .filter((event) => !!event.timestamp)
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }

  private resetEditForm(): void {
    this.editForm.set({
      email: '',
      firstName: '',
      lastName: '',
      phone: '',
      notes: '',
      tags: '',
      marketingOptIn: false,
    });
  }
}

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
}

interface NewCustomerForm {
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  notes: string;
  tags: string;
  marketingOptIn: boolean;
}

interface EditCustomerForm {
  email: string;
  firstName: string;
  lastName: string;
  phone: string;
  notes: string;
  tags: string;
  marketingOptIn: boolean;
}

interface CustomerTimelineEvent {
  type: 'created' | 'order' | 'shipping' | 'marketing' | 'note';
  title: string;
  subtitle?: string;
  timestamp: string;
}
