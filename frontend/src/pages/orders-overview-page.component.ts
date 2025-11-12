import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  CreateManualOrderPayload,
  OrderService,
  OrderTimelineEvent,
  ShopMetrics,
  ShopOrder,
  UpdateTrackingPayload,
} from '../app/core/services/order.service';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import { ApiResponse, ProductService, ProductSummary } from '../app/core/services/product.service';
import { CustomerService, CustomerSummary } from '../app/core/services/customer.service';
import { environment } from '../environments/environment';

const ORDER_STATUSES = ['pending', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'] as const;

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
  private readonly customersService = inject(CustomerService);
  private readonly productsService = inject(ProductService);
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

  readonly manualModalOpen = signal<boolean>(false);
  readonly manualBusy = signal<boolean>(false);
  readonly manualOptionsLoading = signal<boolean>(false);
  readonly manualError = signal<string>('');
  readonly manualCustomers = signal<CustomerSummary[]>([]);
  readonly manualProducts = signal<ProductSummary[]>([]);
  readonly manualForm = signal<ManualOrderForm>(this.createDefaultManualForm());
  readonly manualSummary = computed(() => this.calculateManualSubtotal());
  readonly manualSubtotalCents = computed(() => this.manualSummary().subtotalCents);
  readonly manualCurrency = computed(() => this.manualSummary().currency);
  readonly cancellingOrder = signal<string>('');
  readonly refundingOrder = signal<string>('');
  readonly timelineModalOpen = signal<boolean>(false);
  readonly timelineOrder = signal<ShopOrder | null>(null);
  readonly timelineEvents = signal<OrderTimelineEvent[]>([]);
  readonly timelineLoading = signal<boolean>(false);
  readonly timelineError = signal<string>('');
  noteMessage = '';

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
        const data = response?.data ?? [];
        this.orders.set(data);
        this.loading.set(false);
        const current = this.timelineOrder();
        if (current) {
          const updated = data.find((order) => order.uuid === current.uuid);
          if (updated) {
            this.timelineOrder.set(updated);
          }
        }
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
    if (!this.canEdit(order)) {
      return;
    }
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

  remainingRefundCents(order: ShopOrder): number {
    const total = order?.totalCents ?? 0;
    const refunded = order?.refundTotalCents ?? 0;
    return Math.max(0, total - refunded);
  }

  canEdit(order: ShopOrder): boolean {
    const status = order.status?.toLowerCase();
    return status !== 'cancelled' && status !== 'refunded';
  }

  canCancel(order: ShopOrder): boolean {
    const status = order.status?.toLowerCase();
    return status === 'pending' || status === 'processing';
  }

  cancelOrder(order: ShopOrder): void {
    if (!this.canCancel(order) || this.cancellingOrder()) {
      return;
    }
    const confirmed = window.confirm(
      'Cancel this order? Any applied discounts or gift cards will be reversed.',
    );
    if (!confirmed) {
      return;
    }
    this.message.set('');
    this.error.set('');
    this.cancellingOrder.set(order.uuid);
    this.ordersService.cancelOrder(order.uuid).subscribe({
      next: () => {
        this.cancellingOrder.set('');
        this.message.set('Order cancelled.');
        this.loadOrders();
        if (this.timelineOrder()?.uuid === order.uuid) {
          this.loadTimeline(order.uuid);
        }
      },
      error: () => {
        this.cancellingOrder.set('');
        this.error.set('Could not cancel order.');
      },
    });
  }

  canRefund(order: ShopOrder): boolean {
    const status = order.status?.toLowerCase();
    return status !== 'cancelled' && this.remainingRefundCents(order) > 0;
  }

  refundOrder(order: ShopOrder): void {
    if (!this.canRefund(order) || this.refundingOrder()) {
      return;
    }
    const remainingCents = this.remainingRefundCents(order);
    const currency = order.currency ?? 'USD';
    let amountCents = remainingCents;
    const promptDefault = (remainingCents / 100).toFixed(2);
    const input = window.prompt(
      `Enter refund amount (${this.formatCurrency(remainingCents, currency)} available). Leave blank for a full refund.`,
      promptDefault,
    );
    if (input !== null) {
      const trimmed = input.trim();
      if (trimmed !== '') {
        const parsed = Number.parseFloat(trimmed);
        if (!Number.isFinite(parsed) || parsed <= 0) {
          this.error.set('Invalid refund amount.');
          return;
        }
        amountCents = Math.round(parsed * 100);
      }
    }
    if (amountCents <= 0 || amountCents > remainingCents) {
      this.error.set('Invalid refund amount.');
      return;
    }
    this.message.set('');
    this.error.set('');
    this.refundingOrder.set(order.uuid);
    this.ordersService.refundOrder(order.uuid, { amountCents, reason: 'Manual refund' }).subscribe({
      next: () => {
        this.refundingOrder.set('');
        this.message.set('Order refunded.');
        this.loadOrders();
        if (this.timelineOrder()?.uuid === order.uuid) {
          this.loadTimeline(order.uuid);
        }
      },
      error: () => {
        this.refundingOrder.set('');
        this.error.set('Could not refund order.');
      },
    });
  }

  openTimeline(order: ShopOrder): void {
    this.timelineOrder.set(order);
    this.timelineModalOpen.set(true);
    this.timelineError.set('');
    this.noteMessage = '';
    this.timelineEvents.set([]);
    this.loadTimeline(order.uuid);
  }

  closeTimeline(): void {
    this.timelineModalOpen.set(false);
    this.timelineOrder.set(null);
    this.timelineEvents.set([]);
    this.timelineLoading.set(false);
    this.timelineError.set('');
    this.noteMessage = '';
  }

  loadTimeline(orderUuid: string): void {
    this.timelineLoading.set(true);
    this.timelineError.set('');
    this.ordersService.timeline(orderUuid).subscribe({
      next: (response) => {
        this.timelineEvents.set(response?.data ?? []);
        this.timelineLoading.set(false);
      },
      error: () => {
        this.timelineLoading.set(false);
        this.timelineError.set('Unable to load order timeline.');
      },
    });
  }

  submitNote(): void {
    const order = this.timelineOrder();
    if (!order) {
      return;
    }
    const message = this.noteMessage.trim();
    if (!message) {
      this.timelineError.set('Please enter a note.');
      return;
    }
    this.timelineError.set('');
    this.ordersService.addNote(order.uuid, message).subscribe({
      next: () => {
        this.noteMessage = '';
        this.loadTimeline(order.uuid);
      },
      error: () => {
        this.timelineError.set('Could not add note.');
      },
    });
  }

  trackTimelineEventBy(_index: number, event: OrderTimelineEvent): string {
    return event.uuid;
  }

  openManual(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.error.set('Select a shop before creating an order.');
      return;
    }
    this.manualForm.set(this.createDefaultManualForm());
    this.manualError.set('');
    this.manualBusy.set(false);
    this.manualModalOpen.set(true);
    this.loadManualOptions(slug);
  }

  closeManual(): void {
    this.manualModalOpen.set(false);
    this.manualBusy.set(false);
    this.manualOptionsLoading.set(false);
    this.manualError.set('');
    this.manualForm.set(this.createDefaultManualForm());
  }

  loadManualOptions(slug: string): void {
    this.manualOptionsLoading.set(true);
    let pending = 2;
    const finalize = () => {
      pending -= 1;
      if (pending <= 0) {
        this.manualOptionsLoading.set(false);
      }
    };
    this.customersService.list(slug).subscribe({
      next: (response) => {
        const customers = (response?.data ?? []).filter((customer) => !!customer.userUuid);
        this.manualCustomers.set(customers);
        if (customers.length === 0) {
          this.manualError.set(
            'No customers with linked accounts found. Link a customer to a platform account before creating a manual order.',
          );
        }
        finalize();
      },
      error: () => {
        this.manualCustomers.set([]);
        this.manualError.set('Could not load customers for manual orders.');
        finalize();
      },
    });
    this.productsService.listMyShopProducts(slug).subscribe({
      next: (response) => {
        this.manualProducts.set(response?.data ?? []);
        finalize();
      },
      error: () => {
        this.manualProducts.set([]);
        if (!this.manualError()) {
          this.manualError.set('Could not load products for manual orders.');
        }
        finalize();
      },
    });
  }

  addManualItem(): void {
    const current = this.manualForm();
    this.manualForm.set({
      ...current,
      items: [...current.items, { productUuid: '', quantity: 1 }],
    });
  }

  removeManualItem(index: number): void {
    const current = this.manualForm();
    if (current.items.length <= 1) {
      return;
    }
    const items = current.items.filter((_, i) => i !== index);
    this.manualForm.set({
      ...current,
      items,
    });
  }

  updateManualCustomer(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, customerUuid: value });
  }

  updateManualStatus(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, status: value });
  }

  updateManualItemProduct(index: number, productUuid: string): void {
    const form = this.manualForm();
    if (index < 0 || index >= form.items.length) {
      return;
    }
    const items = form.items.map((item, i) => (i === index ? { ...item, productUuid } : item));
    this.manualForm.set({ ...form, items });
  }

  updateManualItemQuantity(index: number, value: string | number): void {
    const form = this.manualForm();
    if (index < 0 || index >= form.items.length) {
      return;
    }
    const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10);
    const quantity = Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 1;
    const items = form.items.map((item, i) => (i === index ? { ...item, quantity } : item));
    this.manualForm.set({ ...form, items });
  }

  updateManualDiscountCode(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, discountCode: value });
  }

  updateManualGiftCardCode(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, giftCardCode: value });
  }

  updateManualShippingAddress(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, shippingAddress: value });
  }

  updateManualPaymentMethod(value: string): void {
    const form = this.manualForm();
    this.manualForm.set({ ...form, paymentMethod: value });
  }

  submitManual(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.manualError.set('Select a shop before creating an order.');
      return;
    }
    const form = this.manualForm();
    if (!form.customerUuid) {
      this.manualError.set('Select a customer linked to a platform account.');
      return;
    }
    const preparedItems = form.items
      .filter((item) => !!item.productUuid)
      .map((item) => ({
        productUuid: item.productUuid,
        quantity: item.quantity > 0 ? item.quantity : 1,
      }));
    if (preparedItems.length === 0) {
      this.manualError.set('Add at least one product to the order.');
      return;
    }

    const payload: CreateManualOrderPayload = {
      customerUuid: form.customerUuid,
      items: preparedItems,
    };
    const status = form.status?.trim();
    if (status) {
      payload.status = status;
    }
    const discount = form.discountCode?.trim();
    if (discount) {
      payload.discountCode = discount;
    }
    const giftCard = form.giftCardCode?.trim();
    if (giftCard) {
      payload.giftCardCode = giftCard;
    }
    const shippingAddress = form.shippingAddress?.trim();
    if (shippingAddress) {
      payload.shippingAddress = shippingAddress;
    }
    const paymentMethod = form.paymentMethod?.trim();
    if (paymentMethod) {
      payload.paymentMethod = paymentMethod;
    }

    this.manualBusy.set(true);
    this.manualError.set('');
    this.ordersService.createManualOrder(slug, payload).subscribe({
      next: () => {
        this.manualBusy.set(false);
        this.manualModalOpen.set(false);
        this.manualForm.set(this.createDefaultManualForm());
        this.message.set('Order created.');
        this.loadOrders();
      },
      error: (err) => {
        this.manualBusy.set(false);
        const reason =
          err?.error?.message ||
          err?.message ||
          'Could not create manual order.';
        this.manualError.set(reason);
      },
    });
  }

  private createDefaultManualForm(): ManualOrderForm {
    return {
      customerUuid: '',
      status: 'pending',
      discountCode: '',
      giftCardCode: '',
      shippingAddress: '',
      paymentMethod: '',
      items: [{ productUuid: '', quantity: 1 }],
    };
  }

  private calculateManualSubtotal(): ManualSubtotal {
    const products = this.manualProducts();
    const productMap = new Map(products.map((product) => [product.uuid, product]));
    const form = this.manualForm();
    let subtotal = 0;
    let currency: string | null = null;
    for (const item of form.items) {
      const product = productMap.get(item.productUuid);
      if (!product) {
        continue;
      }
      const quantity = item.quantity > 0 ? item.quantity : 1;
      subtotal += (product.priceCents ?? 0) * quantity;
      if (!currency) {
        currency = product.currency ?? 'USD';
      }
    }
    return { subtotalCents: subtotal, currency };
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

interface ManualOrderItemForm {
  productUuid: string;
  quantity: number;
}

interface ManualOrderForm {
  customerUuid: string;
  status: string;
  discountCode: string;
  giftCardCode: string;
  shippingAddress: string;
  paymentMethod: string;
  items: ManualOrderItemForm[];
}

interface ManualSubtotal {
  subtotalCents: number;
  currency: string | null;
}
