import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import {
  CommitDraftOrderPayload,
  CommitDraftOrderResponse,
  CreateDraftOrderPayload,
  DraftOrder,
  ManualOrderItemPayload,
  OrderService,
  UpdateDraftOrderPayload,
} from '../app/core/services/order.service';
import { ApiResponse, ProductService, ProductSummary } from '../app/core/services/product.service';
import { CustomerService, CustomerSummary } from '../app/core/services/customer.service';
import { environment } from '../environments/environment';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
}

interface DraftFormItem {
  productUuid: string;
  quantity: number;
}

interface DraftForm {
  customerUuid: string;
  customerEmail: string;
  customerName: string;
  notes: string;
  discountCode: string;
  giftCardCode: string;
  shippingAddress: string;
  paymentMethod: string;
  status: string;
  expiresAt: string;
  items: DraftFormItem[];
}

const DRAFT_STATUSES = ['open', 'pending_payment', 'invoice_sent', 'finalized'] as const;

@Component({
  standalone: true,
  selector: 'app-orders-draft-page',
  imports: [CommonModule, FormsModule, ModalComponent],
  templateUrl: './orders-draft-page.component.html',
  styleUrls: ['./orders-draft-page.component.css'],
})
export class OrdersDraftPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly ordersService = inject(OrderService);
  private readonly customersService = inject(CustomerService);
  private readonly productsService = inject(ProductService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly drafts = signal<DraftOrder[]>([]);
  readonly loading = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');

  readonly dependenciesLoading = signal<boolean>(false);
  readonly draftCustomers = signal<CustomerSummary[]>([]);
  readonly draftProducts = signal<ProductSummary[]>([]);

  readonly draftModalOpen = signal<boolean>(false);
  readonly draftBusy = signal<boolean>(false);
  readonly draftError = signal<string>('');
  readonly selectedDraft = signal<DraftOrder | null>(null);
  readonly draftForm = signal<DraftForm>(this.createEmptyDraftForm());

  readonly filteredDrafts = computed(() => {
    return this.drafts();
  });

  readonly draftSummary = computed(() => {
    const products = this.draftProducts();
    const productMap = new Map(products.map((product) => [product.uuid, product]));
    const form = this.draftForm();
    let subtotal = 0;
    let currency = '';
    form.items.forEach((item) => {
      const product = productMap.get(item.productUuid);
      if (!product) {
        return;
      }
      const quantity = Number.isFinite(item.quantity) && item.quantity > 0 ? Math.floor(item.quantity) : 1;
      subtotal += (product.priceCents ?? 0) * quantity;
      if (!currency) {
        currency = product.currency ?? 'USD';
      }
    });
    return { subtotalCents: subtotal, currency: currency || 'USD' };
  });

  readonly draftStatuses = DRAFT_STATUSES;

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
            this.loadDependencies(slug);
            this.loadDrafts();
          } else {
            this.loading.set(false);
          }
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Could not load shops.');
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
    this.drafts.set([]);
    this.loadDependencies(slug);
    this.loadDrafts();
  }

  refresh(): void {
    if (!this.selectedShopSlug()) {
      return;
    }
    this.loadDrafts();
  }

  loadDrafts(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      return;
    }
    this.loading.set(true);
    this.error.set('');
    this.ordersService.listDraftOrders(slug).subscribe({
      next: (response) => {
        this.drafts.set(response?.data ?? []);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Unable to load draft orders.');
      },
    });
  }

  loadDependencies(slug: string): void {
    this.dependenciesLoading.set(true);
    let pending = 2;
    const finalize = () => {
      pending -= 1;
      if (pending <= 0) {
        this.dependenciesLoading.set(false);
      }
    };
    this.customersService.list(slug).subscribe({
      next: (response) => {
        this.draftCustomers.set(response?.data ?? []);
        finalize();
      },
      error: () => {
        this.draftCustomers.set([]);
        finalize();
      },
    });
    this.productsService.listMyShopProducts(slug).subscribe({
      next: (response) => {
        this.draftProducts.set(response?.data ?? []);
        finalize();
      },
      error: () => {
        this.draftProducts.set([]);
        finalize();
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
      return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString();
  }

  trackDraftBy(_index: number, draft: DraftOrder): string {
    return draft.uuid;
  }

  trackItemBy(_index: number, item: DraftFormItem): string {
    return `${item.productUuid}-${_index}`;
  }

  openCreateModal(): void {
    if (!this.selectedShopSlug()) {
      this.error.set('Select a shop before creating a draft order.');
      return;
    }
    this.selectedDraft.set(null);
    this.draftError.set('');
    this.draftForm.set(this.createEmptyDraftForm());
    this.draftModalOpen.set(true);
  }

  openEditModal(draft: DraftOrder): void {
    this.selectedDraft.set(draft);
    this.draftError.set('');
    this.draftForm.set({
      customerUuid: draft.customerUuid ?? '',
      customerEmail: draft.customerEmail ?? '',
      customerName: draft.customerName ?? '',
      notes: draft.notes ?? '',
      discountCode: draft.discountCode ?? '',
      giftCardCode: draft.giftCardCode ?? '',
      shippingAddress: draft.shippingAddress ?? '',
      paymentMethod: draft.paymentMethod ?? '',
      status: draft.status ?? 'open',
      expiresAt: draft.expiresAt ? draft.expiresAt.slice(0, 10) : '',
      items: draft.items.map((item) => ({
        productUuid: item.productUuid,
        quantity: item.quantity,
      })),
    });
    if (this.draftForm().items.length === 0) {
      this.addDraftItem();
    }
    this.draftModalOpen.set(true);
  }

  closeDraftModal(): void {
    this.draftModalOpen.set(false);
    this.draftBusy.set(false);
    this.draftError.set('');
    this.selectedDraft.set(null);
    this.draftForm.set(this.createEmptyDraftForm());
  }

  addDraftItem(): void {
    const current = this.draftForm();
    this.draftForm.set({
      ...current,
      items: [...current.items, { productUuid: '', quantity: 1 }],
    });
  }

  removeDraftItem(index: number): void {
    const current = this.draftForm();
    if (current.items.length <= 1) {
      return;
    }
    this.draftForm.set({
      ...current,
      items: current.items.filter((_, i) => i !== index),
    });
  }

  updateDraftItemProduct(index: number, productUuid: string): void {
    const current = this.draftForm();
    if (index < 0 || index >= current.items.length) {
      return;
    }
    const items = current.items.map((item, i) =>
      i === index ? { ...item, productUuid } : item,
    );
    this.draftForm.set({ ...current, items });
  }

  updateDraftItemQuantity(index: number, value: string | number): void {
    const current = this.draftForm();
    if (index < 0 || index >= current.items.length) {
      return;
    }
    const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10);
    const quantity = Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : 1;
    const items = current.items.map((item, i) =>
      i === index ? { ...item, quantity } : item,
    );
    this.draftForm.set({ ...current, items });
  }

  updateDraftCustomer(uuid: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, customerUuid: uuid });
  }

  updateDraftCustomerEmail(email: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, customerEmail: email });
  }

  updateDraftCustomerName(name: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, customerName: name });
  }

  updateDraftNotes(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, notes: value });
  }

  updateDraftDiscountCode(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, discountCode: value.toUpperCase() });
  }

  updateDraftGiftCardCode(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, giftCardCode: value.toUpperCase() });
  }

  updateDraftShippingAddress(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, shippingAddress: value });
  }

  updateDraftPaymentMethod(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, paymentMethod: value });
  }

  updateDraftStatus(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, status: value || 'open' });
  }

  updateDraftExpiresAt(value: string): void {
    const current = this.draftForm();
    this.draftForm.set({ ...current, expiresAt: value });
  }

  submitDraft(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      this.draftError.set('Select a shop before saving a draft.');
      return;
    }
    const form = this.draftForm();
    const items = this.prepareItems(form.items);
    if (items.length === 0) {
      this.draftError.set('Add at least one product to the draft.');
      return;
    }
    this.draftBusy.set(true);
    this.draftError.set('');
    const selected = this.selectedDraft();
    if (selected) {
      const payload: UpdateDraftOrderPayload = {
        customerUuid: form.customerUuid,
        customerEmail: form.customerEmail,
        customerName: form.customerName,
        notes: form.notes,
        discountCode: form.discountCode,
        giftCardCode: form.giftCardCode,
        shippingAddress: form.shippingAddress,
        paymentMethod: form.paymentMethod,
        status: form.status || 'open',
        items,
      };
      if (form.expiresAt) {
        payload.expiresAt = new Date(form.expiresAt).toISOString();
      }
      this.ordersService.updateDraftOrder(selected.uuid, payload).subscribe({
        next: () => {
          this.draftBusy.set(false);
          this.message.set('Draft order updated.');
          this.closeDraftModal();
          this.loadDrafts();
        },
        error: (err) => {
          this.draftBusy.set(false);
          this.draftError.set(err?.error?.message || 'Could not update draft order.');
        },
      });
      return;
    }

    const payload: CreateDraftOrderPayload = {
      items,
    };
    if (form.customerUuid) {
      payload.customerUuid = form.customerUuid;
    } else {
      if (form.customerEmail) {
        payload.customerEmail = form.customerEmail;
      }
      if (form.customerName) {
        payload.customerName = form.customerName;
      }
    }
    if (form.notes) {
      payload.notes = form.notes;
    }
    if (form.discountCode) {
      payload.discountCode = form.discountCode;
    }
    if (form.giftCardCode) {
      payload.giftCardCode = form.giftCardCode;
    }
    if (form.shippingAddress) {
      payload.shippingAddress = form.shippingAddress;
    }
    if (form.paymentMethod) {
      payload.paymentMethod = form.paymentMethod;
    }
    if (form.status) {
      payload.status = form.status;
    }
    if (form.expiresAt) {
      payload.expiresAt = new Date(form.expiresAt).toISOString();
    }

    this.ordersService.createDraftOrder(slug, payload).subscribe({
      next: () => {
        this.draftBusy.set(false);
        this.message.set('Draft order created.');
        this.closeDraftModal();
        this.loadDrafts();
      },
      error: (err) => {
        this.draftBusy.set(false);
        this.draftError.set(err?.error?.message || 'Could not create draft order.');
      },
    });
  }

  deleteDraft(draft: DraftOrder): void {
    const confirmed = window.confirm('Delete this draft order? This action cannot be undone.');
    if (!confirmed) {
      return;
    }
    this.ordersService.deleteDraftOrder(draft.uuid).subscribe({
      next: () => {
        this.message.set('Draft order deleted.');
        this.loadDrafts();
      },
      error: () => {
        this.error.set('Could not delete draft order.');
      },
    });
  }

  commitDraft(draft: DraftOrder): void {
    if (!draft.customerUuid) {
      this.error.set('Link a customer to the draft before converting it to an order.');
      return;
    }
    const confirmed = window.confirm('Convert this draft into an order?');
    if (!confirmed) {
      return;
    }
    const payload: CommitDraftOrderPayload = {
      status: 'pending',
      discountCode: draft.discountCode ?? undefined,
      giftCardCode: draft.giftCardCode ?? undefined,
      shippingAddress: draft.shippingAddress ?? undefined,
      paymentMethod: draft.paymentMethod ?? undefined,
    };
    this.ordersService.commitDraftOrder(draft.uuid, payload).subscribe({
      next: (response) => {
        const info: CommitDraftOrderResponse | undefined = response?.data;
        if (info) {
          this.message.set(
            `Draft converted. Created order ${info.orderUuid.slice(0, 12)}… with status ${info.status}.`,
          );
        } else {
          this.message.set('Draft converted into an order.');
        }
        this.loadDrafts();
      },
      error: (err) => {
        this.error.set(err?.error?.message || 'Could not convert draft into an order.');
      },
    });
  }

  canCommitDraft(draft: DraftOrder): boolean {
    return !!draft.customerUuid;
  }

  private prepareItems(items: DraftFormItem[]): ManualOrderItemPayload[] {
    return items
      .filter((item) => !!item.productUuid)
      .map((item) => ({
        productUuid: item.productUuid,
        quantity: Number.isFinite(item.quantity) && item.quantity > 0 ? Math.floor(item.quantity) : 1,
      }));
  }

  private createEmptyDraftForm(): DraftForm {
    return {
      customerUuid: '',
      customerEmail: '',
      customerName: '',
      notes: '',
      discountCode: '',
      giftCardCode: '',
      shippingAddress: '',
      paymentMethod: '',
      status: 'open',
      expiresAt: '',
      items: [{ productUuid: '', quantity: 1 }],
    };
  }
}
