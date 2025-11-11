import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, Signal, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';
import {
  InventoryAlert,
  InventoryAdjustment,
  InventoryEntry,
  InventoryLocation,
  InventoryService,
} from '../app/core/services/inventory.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

interface AdjustmentFormModel {
  deltaQuantity: number;
  deltaReserved: number;
  reason: string;
  note: string;
}

interface SafetyFormModel {
  safetyStock: number;
}

@Component({
  standalone: true,
  selector: 'app-products-inventory-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-inventory-page.component.html',
  styleUrls: ['./products-inventory-page.component.css'],
})
export class ProductsInventoryPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly inventoryApi = inject(InventoryService);
  readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly inventory = signal<InventoryEntry[]>([]);
  readonly alerts = signal<InventoryAlert[]>([]);
  readonly alertStatus = signal<'open' | 'resolved' | 'all'>('open');
  readonly loading = signal<boolean>(false);
  readonly alertLoading = signal<boolean>(false);
  readonly historyLoading = signal<boolean>(false);
  readonly adjustmentBusy = signal<boolean>(false);
  readonly safetyBusy = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly alertError = signal<string>('');
  readonly historyError = signal<string>('');
  readonly selectedLevel = signal<InventoryEntry | null>(null);
  readonly history = signal<InventoryAdjustment[]>([]);
  readonly adjustmentForm = signal<AdjustmentFormModel>({
    deltaQuantity: 0,
    deltaReserved: 0,
    reason: '',
    note: '',
  });
  readonly safetyForm = signal<SafetyFormModel>({
    safetyStock: 0,
  });
  readonly locations = signal<InventoryLocation[]>([]);
  readonly locationLoading = signal<boolean>(false);
  readonly locationBusy = signal<boolean>(false);
  readonly locationError = signal<string>('');
  readonly editingLocation = signal<InventoryLocation | null>(null);
  readonly locationForm = signal<{ name: string; code: string; description: string; isPrimary: boolean }>({
    name: '',
    code: '',
    description: '',
    isPrimary: false,
  });

  readonly hasInventory: Signal<boolean> = computed(
    () => !this.loading() && this.inventory().length > 0,
  );

  ngOnInit(): void {
    this.bootstrap();
  }

  bootstrap(): void {
    this.loading.set(true);
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
            this.loadInventory();
            this.refreshAlerts();
            this.loadLocations();
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
    this.shopSlug.set(slug);
    this.message.set('');
    this.error.set('');
    this.selectedLevel.set(null);
    this.history.set([]);
    this.loadInventory();
    this.refreshAlerts();
    this.resetLocationForm();
    this.loadLocations();
  }

  loadInventory(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.inventory.set([]);
      return;
    }
    this.loading.set(true);
    this.inventoryApi.listInventory(slug).subscribe({
      next: (response) => {
        this.inventory.set(response?.data ?? []);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Could not load inventory for this shop.');
      },
    });
  }

  refreshAlerts(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.alerts.set([]);
      return;
    }
    this.alertLoading.set(true);
    this.alertError.set('');
    this.inventoryApi.listAlerts(slug, this.alertStatus()).subscribe({
      next: (response) => {
        this.alerts.set(response?.data ?? []);
        this.alertLoading.set(false);
      },
      error: () => {
        this.alertLoading.set(false);
        this.alertError.set('Could not load alerts.');
      },
    });
  }

  openLevel(entry: InventoryEntry): void {
    this.selectedLevel.set(entry);
    this.message.set('');
    this.error.set('');
    this.historyError.set('');
    this.adjustmentForm.set({
      deltaQuantity: 0,
      deltaReserved: 0,
      reason: '',
      note: '',
    });
    this.safetyForm.set({
      safetyStock: entry.safetyStock ?? 0,
    });
    this.loadHistory(entry);
  }

  loadHistory(entry: InventoryEntry | null): void {
    const slug = this.shopSlug();
    if (!slug || !entry) {
      this.history.set([]);
      return;
    }
    this.historyLoading.set(true);
    this.historyError.set('');
    this.inventoryApi.getHistory(slug, entry.uuid, 50).subscribe({
      next: (response) => {
        this.history.set(response?.data ?? []);
        this.historyLoading.set(false);
      },
      error: () => {
        this.historyLoading.set(false);
        this.historyError.set('Could not load adjustment history.');
      },
    });
  }

  submitAdjustment(): void {
    const slug = this.shopSlug();
    const level = this.selectedLevel();
    if (!slug || !level) {
      return;
    }
    const form = this.adjustmentForm();
    const deltaQuantity = Number(form.deltaQuantity) || 0;
    const deltaReserved = Number(form.deltaReserved) || 0;
    if (deltaQuantity === 0 && deltaReserved === 0) {
      this.error.set('Enter an adjustment to quantity or reserved.');
      return;
    }
    this.adjustmentBusy.set(true);
    this.inventoryApi
      .createAdjustment(slug, level.uuid, {
        deltaQuantity,
        deltaReserved,
        reason: form.reason?.trim() || undefined,
        note: form.note?.trim() || undefined,
      })
      .subscribe({
        next: (response) => {
          const payload = response?.data;
          const updated = payload?.inventory;
          if (updated) {
            this.updateInventoryEntry(updated);
            this.selectedLevel.set(updated);
            this.safetyForm.set({ safetyStock: updated.safetyStock ?? 0 });
          }
          if (payload?.adjustment) {
            this.history.set([payload.adjustment, ...this.history()]);
          } else {
            this.loadHistory(updated ?? level);
          }
          this.adjustmentForm.set({
            deltaQuantity: 0,
            deltaReserved: 0,
            reason: '',
            note: '',
          });
          this.adjustmentBusy.set(false);
          this.message.set('Inventory adjusted.');
          this.error.set('');
          this.refreshAlerts();
        },
        error: (err) => {
          this.adjustmentBusy.set(false);
          const status = err?.status ?? 500;
          if (status === 400) {
            this.error.set('Adjustment rejected. Check the values and try again.');
          } else {
            this.error.set('Could not save adjustment.');
          }
        },
      });
  }

  submitSafetyStock(): void {
    const slug = this.shopSlug();
    const level = this.selectedLevel();
    if (!slug || !level) {
      return;
    }
    const target = Number(this.safetyForm().safetyStock);
    if (!Number.isFinite(target) || target < 0) {
      this.error.set('Safety stock must be zero or greater.');
      return;
    }
    this.safetyBusy.set(true);
    this.inventoryApi
      .updateLevel(slug, level.uuid, { safetyStock: Math.floor(target) })
      .subscribe({
        next: (response) => {
          const updated = response?.data;
          if (updated) {
            this.updateInventoryEntry(updated);
            this.selectedLevel.set(updated);
          } else {
            this.loadInventory();
          }
          this.safetyBusy.set(false);
          this.message.set('Safety stock saved.');
          this.error.set('');
          this.refreshAlerts();
        },
        error: () => {
          this.safetyBusy.set(false);
          this.error.set('Could not update safety stock.');
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
    this.locationError.set('');
    this.inventoryApi.listLocations(slug).subscribe({
      next: (response) => {
        this.locations.set(response?.data ?? []);
        this.locationLoading.set(false);
      },
      error: () => {
        this.locationLoading.set(false);
        this.locationError.set('Could not load locations.');
      },
    });
  }

  submitLocationForm(): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    const form = this.locationForm();
    const name = form.name.trim();
    const code = form.code.trim();
    if (!name || !code) {
      this.locationError.set('Name and code are required.');
      return;
    }
    this.locationBusy.set(true);
    this.locationError.set('');
    const payload = {
      name,
      code,
      description: form.description.trim(),
      isPrimary: form.isPrimary,
    };
    const editing = this.editingLocation();
    const request = editing
      ? this.inventoryApi.updateLocation(slug, editing.uuid, payload)
      : this.inventoryApi.createLocation(slug, payload);
    request.subscribe({
      next: () => {
        this.locationBusy.set(false);
        this.message.set(editing ? 'Location updated.' : 'Location created.');
        this.resetLocationForm();
        this.loadLocations();
      },
      error: (err) => {
        this.locationBusy.set(false);
        const status = err?.status ?? 500;
        if (status === 409) {
          this.locationError.set('A location with that code already exists.');
        } else if (status === 400) {
          this.locationError.set('Location rejected. Check the values and try again.');
        } else {
          this.locationError.set('Could not save location.');
        }
      },
    });
  }

  editLocation(location: InventoryLocation): void {
    this.editingLocation.set(location);
    this.locationForm.set({
      name: location.name,
      code: location.code,
      description: location.description ?? '',
      isPrimary: location.isPrimary,
    });
    this.locationError.set('');
  }

  cancelLocationEdit(): void {
    this.resetLocationForm();
  }

  setPrimaryLocation(location: InventoryLocation): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    this.locationBusy.set(true);
    this.locationError.set('');
    this.inventoryApi.updateLocation(slug, location.uuid, { isPrimary: true }).subscribe({
      next: () => {
        this.locationBusy.set(false);
        this.message.set('Primary location updated.');
        this.loadLocations();
      },
      error: () => {
        this.locationBusy.set(false);
        this.locationError.set('Could not update primary location.');
      },
    });
  }

  resetLocationForm(): void {
    this.editingLocation.set(null);
    this.locationForm.set({
      name: '',
      code: '',
      description: '',
      isPrimary: false,
    });
    this.locationError.set('');
    this.locationBusy.set(false);
  }

  resolveAlert(alert: InventoryAlert): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    this.alertLoading.set(true);
    this.inventoryApi.resolveAlert(slug, alert.uuid, {}).subscribe({
      next: () => {
        this.message.set('Alert resolved.');
        this.alertError.set('');
        this.refreshAlerts();
      },
      error: () => {
        this.alertLoading.set(false);
        this.alertError.set('Could not resolve alert.');
      },
    });
  }

  changeAlertStatus(status: 'open' | 'resolved' | 'all'): void {
    this.alertStatus.set(status);
    this.refreshAlerts();
  }

  isLowStock(entry: InventoryEntry): boolean {
    const threshold = entry.safetyStock ?? 0;
    if (threshold <= 0) {
      return false;
    }
    return entry.quantity <= threshold;
  }

  availableUnits(entry: InventoryEntry): number {
    return Math.max(0, (entry.quantity ?? 0) - (entry.reserved ?? 0));
  }

  deltaLabel(value: number): string {
    if (value > 0) {
      return `+${value}`;
    }
    if (value < 0) {
      return `${value}`;
    }
    return '0';
  }

  private updateInventoryEntry(updated: InventoryEntry): void {
    const list = this.inventory().map((item) => (item.uuid === updated.uuid ? updated : item));
    this.inventory.set(list);
  }
}
