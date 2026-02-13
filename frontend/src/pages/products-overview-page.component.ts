import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnDestroy, OnInit, computed, effect, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { firstValueFrom } from 'rxjs';
import { ApiResponse, ProductService, ProductSummary } from '../app/core/services/product.service';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-products-overview-page',
  imports: [CommonModule, FormsModule, RouterLink, ModalComponent],
  templateUrl: './products-overview-page.component.html',
  styleUrls: ['./products-overview-page.component.css'],
})
export class ProductsOverviewPageComponent implements OnInit, OnDestroy {
  private readonly http = inject(HttpClient);
  private readonly products = inject(ProductService);
  private readonly shopState = inject(ShopStateService);
  readonly api = environment.apiBase;
  readonly shops = this.shopState.shops;
  readonly activeShopSlug = this.shopState.activeShopSlug;
  items = signal<ProductSummary[]>([]);
  loading = signal<boolean>(false);
  q = signal<string>('');
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly editing = signal<boolean>(false);
  readonly editBusy = signal<boolean>(false);
  readonly selectedProduct = signal<ProductSummary | null>(null);
  readonly editModel = signal<ProductEditDraft>({
    title: '',
    summary: '',
    price: '',
    stock: 0,
    published: false,
    category: '',
  });

  readonly hasProducts = computed(() => !this.loading() && this.items().length > 0);

  private readonly syncActiveShop = effect(() => {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      return;
    }
    this.load(slug);
  });

  ngOnInit(): void {
    this.shopState.ensureLoaded();
  }

  ngOnDestroy(): void {
    this.syncActiveShop.destroy();
  }

  load(slug?: string): void {
    const target = slug ?? this.shopState.activeShopSlug();
    if (!target) {
      this.items.set([]);
      return;
    }
    this.loading.set(true);
    this.error.set('');
    const params = this.q() ? `?q=${encodeURIComponent(this.q())}` : '';
    this.http
      .get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/my/shops/${target}/products${params}`, {
        withCredentials: true,
      })
      .subscribe({
        next: (response) => {
          this.items.set(response?.data ?? []);
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Could not load products.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopState.setActiveShopSlug(slug);
    this.message.set('');
    this.error.set('');
  }

  changeQuery(value: string): void {
    this.q.set(value);
    this.message.set('');
    this.load();
  }

  openEdit(product: ProductSummary): void {
    this.selectedProduct.set(product);
    this.editModel.set({
      title: product.title,
      summary: product.summary ?? '',
      price: (product.priceCents / 100).toFixed(2),
      stock: product.stock ?? 0,
      published: !!product.published,
      category: product.category ?? '',
    });
    this.message.set('');
    this.error.set('');
    this.editing.set(true);
  }

  closeEdit(): void {
    this.editing.set(false);
    this.editBusy.set(false);
    this.selectedProduct.set(null);
    this.message.set('');
    this.error.set('');
  }

  async saveEdit(): Promise<void> {
    const product = this.selectedProduct();
    if (!product) {
      return;
    }
    const draft = this.editModel();
    const priceCents = this.toCents(draft.price);
    if (priceCents < 0) {
      this.error.set('Enter a valid price.');
      return;
    }
    this.editBusy.set(true);
    try {
      await firstValueFrom(
        this.products.updateProduct(product.uuid, {
          title: draft.title.trim(),
          summary: draft.summary.trim(),
          priceCents,
          stock: Number.isFinite(draft.stock) ? draft.stock : 0,
          published: draft.published,
          category: draft.category.trim(),
        }),
      );
      this.closeEdit();
      this.message.set('Product updated.');
      this.load();
    } catch {
      this.error.set('Could not update product.');
      this.editBusy.set(false);
    }
  }

  formatMoney(amountCents: number, currency: string): string {
    try {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: currency || 'USD',
        minimumFractionDigits: 2,
      }).format((amountCents ?? 0) / 100);
    } catch {
      return `${(amountCents ?? 0) / 100} ${currency}`;
    }
  }

  private toCents(value: string): number {
    const normalized = value?.trim();
    if (!normalized) {
      return 0;
    }
    const parsed = Number(normalized);
    if (Number.isNaN(parsed)) {
      return -1;
    }
    return Math.round(parsed * 100);
  }
}

interface ProductEditDraft {
  title: string;
  summary: string;
  price: string;
  stock: number;
  published: boolean;
  category: string;
}

