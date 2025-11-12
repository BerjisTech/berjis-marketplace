import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import {
  CreateReturnPayload,
  OrderItemSummary,
  OrderReturn,
  OrderService,
  ShopOrder,
} from '../app/core/services/order.service';
import { ApiResponse } from '../app/core/services/product.service';
import { environment } from '../environments/environment';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
}

interface ReturnForm {
  reason: string;
  notes: string;
  refundAmount: string;
  restock: boolean;
}

interface ReturnFormItem extends OrderItemSummary {
  selected: boolean;
  returnQuantity: number;
  reason: string;
  condition: string;
  maxQuantity: number;
}

const RETURN_STATUSES = ['requested', 'approved', 'received', 'restocked', 'rejected', 'refunded'] as const;

@Component({
  standalone: true,
  selector: 'app-orders-returns-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './orders-returns-page.component.html',
  styleUrls: ['./orders-returns-page.component.css'],
})
export class OrdersReturnsPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly ordersService = inject(OrderService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly returns = signal<OrderReturn[]>([]);
  readonly orders = signal<ShopOrder[]>([]);
  readonly formItems = signal<ReturnFormItem[]>([]);
  readonly loading = signal<boolean>(false);
  readonly createBusy = signal<boolean>(false);
  readonly updatingReturn = signal<string>('');
  readonly error = signal<string>('');
  readonly message = signal<string>('');
  readonly selectedOrderUuid = signal<string>('');
  readonly returnForm = signal<ReturnForm>({
    reason: '',
    notes: '',
    refundAmount: '',
    restock: true,
  });

  readonly returnStatuses = RETURN_STATUSES;

  readonly openReturns = computed(() => this.returns().filter((r) => r.status !== 'restocked' && r.status !== 'refunded' && r.status !== 'rejected').length);

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
            this.reloadAll();
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
    this.selectedOrderUuid.set('');
    this.formItems.set([]);
    this.returnForm.set({
      reason: '',
      notes: '',
      refundAmount: '',
      restock: true,
    });
    this.reloadAll();
  }

  reloadAll(): void {
    this.loadReturns();
    this.loadOrders();
  }

  loadReturns(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      return;
    }
    this.ordersService.listReturns(slug).subscribe({
      next: (response: ApiResponse<OrderReturn[]>) => {
        this.returns.set(response?.data ?? []);
      },
      error: () => {
        this.error.set('Unable to load returns.');
        setTimeout(() => this.error.set(''), 3000);
      },
    });
  }

  loadOrders(): void {
    const slug = this.selectedShopSlug();
    if (!slug) {
      return;
    }
    this.ordersService.listShopOrders(slug).subscribe({
      next: (response) => {
        this.orders.set(response?.data ?? []);
      },
      error: () => {
        this.error.set('Unable to load shop orders.');
        setTimeout(() => this.error.set(''), 3000);
      },
    });
  }

  selectOrder(orderUuid: string): void {
    if (!orderUuid) {
      this.selectedOrderUuid.set('');
      this.formItems.set([]);
      return;
    }
    this.selectedOrderUuid.set(orderUuid);
    this.ordersService.getOrderItems(orderUuid).subscribe({
      next: (response: ApiResponse<OrderItemSummary[]>) => {
        const items = (response?.data ?? []).map<ReturnFormItem>((item) => ({
          ...item,
          selected: true,
          returnQuantity: 1,
          reason: '',
          condition: '',
          maxQuantity: item.quantity ?? 1,
        }));
        this.formItems.set(items);
      },
      error: () => {
        this.error.set('Could not load order items.');
        setTimeout(() => this.error.set(''), 3000);
        this.formItems.set([]);
      },
    });
  }

  toggleItem(index: number, selected: boolean): void {
    const items = [...this.formItems()];
    if (!items[index]) {
      return;
    }
    items[index] = { ...items[index], selected };
    this.formItems.set(items);
  }

  updateItemQuantity(index: number, value: string | number): void {
    const items = [...this.formItems()];
    const target = items[index];
    if (!target) {
      return;
    }
    const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10);
    const safe = Number.isFinite(parsed) ? Math.min(Math.max(parsed, 1), target.maxQuantity) : 1;
    items[index] = { ...target, returnQuantity: safe };
    this.formItems.set(items);
  }

  updateItemReason(index: number, reason: string): void {
    const items = [...this.formItems()];
    if (!items[index]) {
      return;
    }
    items[index] = { ...items[index], reason };
    this.formItems.set(items);
  }

  updateItemCondition(index: number, condition: string): void {
    const items = [...this.formItems()];
    if (!items[index]) {
      return;
    }
    items[index] = { ...items[index], condition };
    this.formItems.set(items);
  }

  updateReturnReason(value: string): void {
    const current = this.returnForm();
    this.returnForm.set({ ...current, reason: value });
  }

  updateReturnNotes(value: string): void {
    const current = this.returnForm();
    this.returnForm.set({ ...current, notes: value });
  }

  updateReturnRefund(value: string): void {
    const current = this.returnForm();
    this.returnForm.set({ ...current, refundAmount: value });
  }

  updateReturnRestock(value: boolean): void {
    const current = this.returnForm();
    this.returnForm.set({ ...current, restock: value });
  }

  submitReturn(): void {
    const slug = this.selectedShopSlug();
    const orderUuid = this.selectedOrderUuid();
    if (!slug || !orderUuid) {
      this.error.set('Select a shop and order before creating a return.');
      setTimeout(() => this.error.set(''), 3000);
      return;
    }
    const items = this.formItems().filter((item) => item.selected && item.returnQuantity > 0);
    if (items.length === 0) {
      this.error.set('Select at least one line item to return.');
      setTimeout(() => this.error.set(''), 3000);
      return;
    }
    const form = this.returnForm();
    const payload: CreateReturnPayload = {
      items: items.map((item) => ({
        orderItemUuid: item.uuid,
        quantity: item.returnQuantity,
        reason: item.reason || undefined,
        condition: item.condition || undefined,
      })),
      reason: form.reason || undefined,
      notes: form.notes || undefined,
      restock: form.restock,
    };
    const refundCents = this.parseCurrency(form.refundAmount);
    if (refundCents > 0) {
      payload.refundAmountCents = refundCents;
    }

    this.createBusy.set(true);
    this.ordersService.createReturn(orderUuid, payload).subscribe({
      next: () => {
        this.createBusy.set(false);
        this.message.set('Return request submitted.');
        setTimeout(() => this.message.set(''), 3000);
        this.resetFormState();
        this.loadReturns();
      },
      error: (err) => {
        this.createBusy.set(false);
        const reason = err?.error?.message || 'Could not create return request.';
        this.error.set(reason);
        setTimeout(() => this.error.set(''), 4000);
      },
    });
  }

  resetFormState(): void {
    this.selectedOrderUuid.set('');
    this.formItems.set([]);
    this.returnForm.set({
      reason: '',
      notes: '',
      refundAmount: '',
      restock: true,
    });
  }

  setReturnStatus(ret: OrderReturn, status: string): void {
    if (this.updatingReturn() === ret.uuid || status === ret.status) {
      return;
    }
    this.updatingReturn.set(ret.uuid);
    this.ordersService.updateReturn(ret.uuid, { status }).subscribe({
      next: () => {
        this.updatingReturn.set('');
        this.message.set(`Return ${status.replace('_', ' ')}.`);
        setTimeout(() => this.message.set(''), 3000);
        this.loadReturns();
      },
      error: (err) => {
        this.updatingReturn.set('');
        const reason = err?.error?.message || 'Could not update return.';
        this.error.set(reason);
        setTimeout(() => this.error.set(''), 4000);
      },
    });
  }

  toggleReturnRestock(ret: OrderReturn, value: boolean): void {
    if (this.updatingReturn() === ret.uuid || ret.restock === value) {
      return;
    }
    this.updatingReturn.set(ret.uuid);
    this.ordersService.updateReturn(ret.uuid, { restock: value }).subscribe({
      next: () => {
        this.updatingReturn.set('');
        this.loadReturns();
      },
      error: () => {
        this.updatingReturn.set('');
        this.error.set('Could not update restock preference.');
        setTimeout(() => this.error.set(''), 3000);
      },
    });
  }

  parseCurrency(value: string): number {
    if (!value) {
      return 0;
    }
    const normalized = value.replace(/[^0-9.,-]/g, '').replace(',', '.');
    const parsed = Number.parseFloat(normalized);
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return 0;
    }
    return Math.round(parsed * 100);
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

  trackReturn(_index: number, ret: OrderReturn): string {
    return ret.uuid;
  }

  trackFormItem(_index: number, item: ReturnFormItem): string {
    return item.uuid;
  }
}
