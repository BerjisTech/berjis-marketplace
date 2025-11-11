import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';
import { InventoryLocation, InventoryService } from '../app/core/services/inventory.service';
import {
  PurchaseOrder,
  PurchaseOrderService,
  Supplier,
} from '../app/core/services/purchase-order.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

interface OrderItemDraft {
  productUuid: string;
  quantity: number;
  costCents: number;
}

interface ReceiveItemDraft {
  productUuid: string;
  quantity: number;
}

@Component({
  standalone: true,
  selector: 'app-products-purchase-orders-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-purchase-orders-page.component.html',
  styleUrls: ['./products-purchase-orders-page.component.css'],
})
export class ProductsPurchaseOrdersPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly purchaseOrders = inject(PurchaseOrderService);
  private readonly inventory = inject(InventoryService);
  readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly orders = signal<PurchaseOrder[]>([]);
  readonly ordersLoading = signal<boolean>(false);
  readonly ordersError = signal<string>('');

  readonly suppliers = signal<Supplier[]>([]);
  readonly suppliersLoading = signal<boolean>(false);

  readonly locations = signal<InventoryLocation[]>([]);
  readonly locationLoading = signal<boolean>(false);

  readonly orderForm = signal({
    supplierUuid: '',
    supplierName: '',
    contactEmail: '',
    phone: '',
    expectedAt: '',
    notes: '',
    status: 'pending',
  });
  readonly orderItems = signal<OrderItemDraft[]>([]);
  readonly newItem = signal<OrderItemDraft>({ productUuid: '', quantity: 1, costCents: 0 });
  readonly submittingOrder = signal<boolean>(false);
  readonly orderMessage = signal<string>('');
  readonly orderError = signal<string>('');

  readonly receivingOrder = signal<PurchaseOrder | null>(null);
  readonly receiveForm = signal<{ locationUuid: string; note: string; items: ReceiveItemDraft[] }>({
    locationUuid: '',
    note: '',
    items: [],
  });
  readonly receiveBusy = signal<boolean>(false);
  readonly receiveError = signal<string>('');

  ngOnInit(): void {
    this.bootstrap();
  }

  bootstrap(): void {
    this.ordersLoading.set(true);
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const list = response?.data ?? [];
          this.shops.set(list);
          const current = this.shopSlug();
          const initial = current && list.some((shop) => shop.slug === current) ? current : list[0]?.slug ?? '';
          this.shopSlug.set(initial);
          if (initial) {
            this.loadOrders();
            this.loadSuppliers();
            this.loadLocations();
          } else {
            this.ordersLoading.set(false);
          }
        },
        error: () => {
          this.ordersLoading.set(false);
          this.ordersError.set('Could not load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopSlug.set(slug);
    this.ordersError.set('');
    this.orderMessage.set('');
    this.orderError.set('');
    this.resetOrderForm();
    this.receivingOrder.set(null);
    this.receiveError.set('');
    this.loadOrders();
    this.loadSuppliers();
    this.loadLocations();
  }

  loadOrders(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.orders.set([]);
      return;
    }
    this.ordersLoading.set(true);
    this.ordersError.set('');
    this.purchaseOrders.list(slug).subscribe({
      next: (response) => {
        this.orders.set(response?.data ?? []);
        this.ordersLoading.set(false);
      },
      error: () => {
        this.ordersLoading.set(false);
        this.ordersError.set('Could not load purchase orders.');
      },
    });
  }

  loadSuppliers(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.suppliers.set([]);
      return;
    }
    this.suppliersLoading.set(true);
    this.purchaseOrders.listSuppliers(slug).subscribe({
      next: (response) => {
        this.suppliers.set(response?.data ?? []);
        this.suppliersLoading.set(false);
      },
      error: () => {
        this.suppliersLoading.set(false);
      },
    });
  }

  loadLocations(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.locations.set([]);
      return;
    }
    this.locationLoading.set(true);
    this.inventory.listLocations(slug).subscribe({
      next: (response) => {
        this.locations.set(response?.data ?? []);
        this.locationLoading.set(false);
      },
      error: () => {
        this.locationLoading.set(false);
      },
    });
  }

  addItem(): void {
    const item = this.newItem();
    const productUuid = item.productUuid.trim();
    const quantity = Number(item.quantity);
    const costCents = Number(item.costCents);
    if (!productUuid) {
      this.orderError.set('Product UUID is required.');
      return;
    }
    if (!Number.isFinite(quantity) || quantity <= 0) {
      this.orderError.set('Quantity must be greater than zero.');
      return;
    }
    if (!Number.isFinite(costCents) || costCents < 0) {
      this.orderError.set('Cost must be zero or greater.');
      return;
    }
    this.orderItems.update((items) => [...items, { productUuid, quantity: Math.round(quantity), costCents: Math.round(costCents) }]);
    this.newItem.set({ productUuid: '', quantity: 1, costCents: 0 });
    this.orderError.set('');
  }

  removeItem(index: number): void {
    this.orderItems.update((items) => items.filter((_, idx) => idx !== index));
  }

  resetOrderForm(): void {
    this.orderForm.set({
      supplierUuid: '',
      supplierName: '',
      contactEmail: '',
      phone: '',
      expectedAt: '',
      notes: '',
      status: 'pending',
    });
    this.orderItems.set([]);
    this.newItem.set({ productUuid: '', quantity: 1, costCents: 0 });
  }

  onSupplierSelect(uuid: string): void {
    const supplier = this.suppliers().find((item) => item.uuid === uuid);
    this.orderForm.update((form) => ({
      ...form,
      supplierUuid: uuid,
      supplierName: supplier ? supplier.name : '',
      contactEmail: supplier?.contactEmail ?? '',
      phone: supplier?.phone ?? '',
    }));
  }

  submitOrder(): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.orderItems().length === 0) {
      this.orderError.set('Add at least one item to the purchase order.');
      return;
    }
    const form = this.orderForm();
    const payload = {
      supplierUuid: form.supplierUuid?.trim() || undefined,
      supplierName: form.supplierName?.trim() || undefined,
      contactEmail: form.contactEmail?.trim() || undefined,
      phone: form.phone?.trim() || undefined,
      expectedAt: form.expectedAt?.trim() || undefined,
      notes: form.notes?.trim() || undefined,
      status: form.status?.trim() || undefined,
      items: this.orderItems().map((item) => ({
        productUuid: item.productUuid.trim(),
        quantity: Math.round(item.quantity),
        costCents: Math.round(item.costCents),
      })),
    };
    this.submittingOrder.set(true);
    this.orderError.set('');
    this.purchaseOrders.create(slug, payload).subscribe({
      next: () => {
        this.submittingOrder.set(false);
        this.orderMessage.set('Purchase order created.');
        this.resetOrderForm();
        this.loadOrders();
        this.loadSuppliers();
      },
      error: (err) => {
        this.submittingOrder.set(false);
        const status = err?.status ?? 500;
        if (status === 400) {
          this.orderError.set('Order rejected. Check the values and try again.');
        } else {
          this.orderError.set('Could not create purchase order.');
        }
      },
    });
  }

  pendingQuantity(item: { quantity: number; receivedQuantity: number }): number {
    return Math.max(0, item.quantity - item.receivedQuantity);
  }

  startReceive(order: PurchaseOrder): void {
    const pending = order.items
      .map<ReceiveItemDraft>((item) => ({
        productUuid: item.productUuid,
        quantity: this.pendingQuantity(item),
      }))
      .filter((item) => item.quantity > 0);
    this.receivingOrder.set(order);
    this.receiveForm.set({
      locationUuid: '',
      note: '',
      items: pending.length > 0 ? pending : order.items.map((item) => ({ productUuid: item.productUuid, quantity: 0 })),
    });
    this.receiveError.set('');
  }

  cancelReceive(): void {
    this.receivingOrder.set(null);
    this.receiveForm.set({ locationUuid: '', note: '', items: [] });
    this.receiveError.set('');
  }

  completeReceive(): void {
    const order = this.receivingOrder();
    const slug = this.shopSlug();
    if (!order || !slug) {
      return;
    }
    const form = this.receiveForm();
    const items = form.items
      .map((item) => ({ productUuid: item.productUuid, quantity: Math.round(Number(item.quantity) || 0) }))
      .filter((item) => item.quantity > 0);
    if (items.length === 0) {
      this.receiveError.set('Enter at least one quantity to receive.');
      return;
    }
    this.receiveBusy.set(true);
    this.receiveError.set('');
    this.purchaseOrders
      .receive(slug, order.uuid, {
        locationUuid: form.locationUuid || undefined,
        note: form.note?.trim() || undefined,
        items,
      })
      .subscribe({
        next: () => {
          this.receiveBusy.set(false);
          this.orderMessage.set('Purchase order received.');
          this.cancelReceive();
          this.loadOrders();
        },
        error: (err) => {
          this.receiveBusy.set(false);
          const status = err?.status ?? 500;
          if (status === 400) {
            this.receiveError.set('Receiving rejected. Check the quantities and try again.');
          } else {
            this.receiveError.set('Could not record receiving.');
          }
        },
      });
  }

  outstandingItems(order: PurchaseOrder): boolean {
    return order.items.some((item) => this.pendingQuantity(item) > 0);
  }
}
