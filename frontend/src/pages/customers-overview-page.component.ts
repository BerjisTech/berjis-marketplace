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

  readonly ordersModalOpen = signal<boolean>(false);
  readonly ordersLoading = signal<boolean>(false);
  readonly ordersError = signal<string>('');
  readonly selectedCustomer = signal<CustomerSummary | null>(null);
  readonly customerOrders = signal<CustomerOrder[]>([]);

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

  openOrdersModal(customer: CustomerSummary): void {
    this.selectedCustomer.set(customer);
    this.ordersModalOpen.set(true);
    this.ordersLoading.set(true);
    this.ordersError.set('');
    this.customerService.listOrders(customer.uuid).subscribe({
      next: (response) => {
        this.customerOrders.set(response?.data ?? []);
        this.ordersLoading.set(false);
      },
      error: () => {
        this.ordersLoading.set(false);
        this.ordersError.set('Could not load order history.');
      },
    });
  }

  closeOrdersModal(): void {
    this.ordersModalOpen.set(false);
    this.ordersLoading.set(false);
    this.ordersError.set('');
    this.customerOrders.set([]);
    this.selectedCustomer.set(null);
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
      return '—';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
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
