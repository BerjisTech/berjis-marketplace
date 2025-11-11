import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';
import { InventoryLocation, InventoryService } from '../app/core/services/inventory.service';
import { Transfer, TransferService } from '../app/core/services/transfer.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

interface TransferItemDraft {
  productUuid: string;
  quantity: number;
}

@Component({
  standalone: true,
  selector: 'app-products-transfers-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-transfers-page.component.html',
  styleUrls: ['./products-transfers-page.component.css'],
})
export class ProductsTransfersPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly transfersApi = inject(TransferService);
  private readonly inventory = inject(InventoryService);
  readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly transfers = signal<Transfer[]>([]);
  readonly transfersLoading = signal<boolean>(false);
  readonly transfersError = signal<string>('');

  readonly locations = signal<InventoryLocation[]>([]);
  readonly locationLoading = signal<boolean>(false);

  readonly transferForm = signal({
    sourceLocationUuid: '',
    destinationLocationUuid: '',
    notes: '',
  });
  readonly transferItems = signal<TransferItemDraft[]>([]);
  readonly newItem = signal<TransferItemDraft>({ productUuid: '', quantity: 1 });
  readonly submittingTransfer = signal<boolean>(false);
  readonly transferMessage = signal<string>('');
  readonly transferError = signal<string>('');

  readonly commitTarget = signal<Transfer | null>(null);
  readonly commitNote = signal<string>('');
  readonly commitBusy = signal<boolean>(false);
  readonly commitError = signal<string>('');

  ngOnInit(): void {
    this.bootstrap();
  }

  bootstrap(): void {
    this.transfersLoading.set(true);
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
            this.loadTransfers();
            this.loadLocations();
          } else {
            this.transfersLoading.set(false);
          }
        },
        error: () => {
          this.transfersLoading.set(false);
          this.transfersError.set('Could not load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopSlug.set(slug);
    this.transfersError.set('');
    this.transferMessage.set('');
    this.transferError.set('');
    this.resetTransferForm();
    this.commitTarget.set(null);
    this.loadTransfers();
    this.loadLocations();
  }

  loadTransfers(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.transfers.set([]);
      return;
    }
    this.transfersLoading.set(true);
    this.transfersError.set('');
    this.transfersApi.list(slug).subscribe({
      next: (response) => {
        this.transfers.set(response?.data ?? []);
        this.transfersLoading.set(false);
      },
      error: () => {
        this.transfersLoading.set(false);
        this.transfersError.set('Could not load transfers.');
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
    if (!productUuid) {
      this.transferError.set('Product UUID is required.');
      return;
    }
    if (!Number.isFinite(quantity) || quantity <= 0) {
      this.transferError.set('Quantity must be greater than zero.');
      return;
    }
    this.transferItems.update((items) => [...items, { productUuid, quantity: Math.round(quantity) }]);
    this.newItem.set({ productUuid: '', quantity: 1 });
    this.transferError.set('');
  }

  removeItem(index: number): void {
    this.transferItems.update((items) => items.filter((_, idx) => idx !== index));
  }

  resetTransferForm(): void {
    this.transferForm.set({
      sourceLocationUuid: '',
      destinationLocationUuid: '',
      notes: '',
    });
    this.transferItems.set([]);
    this.newItem.set({ productUuid: '', quantity: 1 });
  }

  submitTransfer(): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.transferItems().length === 0) {
      this.transferError.set('Add at least one product to transfer.');
      return;
    }
    const form = this.transferForm();
    const source = form.sourceLocationUuid || undefined;
    const destination = form.destinationLocationUuid || undefined;
    if (!source && !destination) {
      this.transferError.set('Select a source or destination location.');
      return;
    }
    if (source && destination && source === destination) {
      this.transferError.set('Source and destination must be different.');
      return;
    }
    const payload = {
      sourceLocationUuid: source,
      destinationLocationUuid: destination,
      notes: form.notes?.trim() || undefined,
      items: this.transferItems().map((item) => ({
        productUuid: item.productUuid.trim(),
        quantity: Math.round(item.quantity),
      })),
    };
    this.submittingTransfer.set(true);
    this.transferError.set('');
    this.transfersApi.create(slug, payload).subscribe({
      next: () => {
        this.submittingTransfer.set(false);
        this.transferMessage.set('Transfer created.');
        this.resetTransferForm();
        this.loadTransfers();
      },
      error: (err) => {
        this.submittingTransfer.set(false);
        const status = err?.status ?? 500;
        if (status === 400) {
          this.transferError.set('Transfer rejected. Check the values and try again.');
        } else {
          this.transferError.set('Could not create transfer.');
        }
      },
    });
  }

  openCommit(transfer: Transfer): void {
    this.commitTarget.set(transfer);
    this.commitNote.set('');
    this.commitError.set('');
  }

  cancelCommit(): void {
    this.commitTarget.set(null);
    this.commitNote.set('');
    this.commitError.set('');
    this.commitBusy.set(false);
  }

  confirmCommit(): void {
    const transfer = this.commitTarget();
    const slug = this.shopSlug();
    if (!transfer || !slug) {
      return;
    }
    this.commitBusy.set(true);
    this.commitError.set('');
    this.transfersApi.commit(slug, transfer.uuid, { note: this.commitNote().trim() || undefined }).subscribe({
      next: () => {
        this.commitBusy.set(false);
        this.transferMessage.set('Transfer completed.');
        this.cancelCommit();
        this.loadTransfers();
      },
      error: (err) => {
        this.commitBusy.set(false);
        const status = err?.status ?? 500;
        if (status === 400) {
          this.commitError.set('Transfer could not be completed. Check inventory levels.');
        } else {
          this.commitError.set('Could not commit transfer.');
        }
      },
    });
  }
}
