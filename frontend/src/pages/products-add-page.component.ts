import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, ElementRef, OnInit, ViewChild, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ProductService } from '../app/core/services/product.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-products-add-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-add-page.component.html',
  styleUrls: ['./products-add-page.component.css']
})
export class ProductsAddPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly products = inject(ProductService);
  private readonly shopState = inject(ShopStateService);
  readonly templateUrl = '/products-import-template.csv';
  readonly api = environment.apiBase;

  readonly shops = this.shopState.shops;
  readonly activeShopSlug = this.shopState.activeShopSlug;
  readonly loadingShops = this.shopState.loading;
  message = signal<string>('');
  error = signal<string>('');
  uploading = signal<boolean>(false);
  importingDemo = signal<boolean>(false);

  @ViewChild('fileInput') fileInput?: ElementRef<HTMLInputElement>;

  readonly createProductQuery = computed(() => {
    const slug = this.shopState.activeShopSlug();
    return slug ? { shop: slug } : null;
  });

  ngOnInit(): void {
    this.shopState.ensureLoaded();
  }

  changeShop(slug: string): void {
    this.shopState.setActiveShopSlug(slug);
    this.message.set('');
    this.error.set('');
  }

  openFilePicker(): void {
    this.fileInput?.nativeElement.click();
  }

  handleFile(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) {
      return;
    }
    this.uploadCsv(file);
    input.value = '';
  }

  importDemoProducts(): void {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.error.set('Select a shop first.');
      return;
    }
    this.importingDemo.set(true);
    this.message.set('');
    this.error.set('');
    this.products.importDemoProducts(slug).subscribe({
      next: (response) => {
        const created = response?.data?.created ?? 0;
        this.message.set(`Imported ${created} demo products into ${slug}.`);
        this.importingDemo.set(false);
      },
      error: (err) => {
        this.error.set(this.resolveError(err, 'Unable to import demo products.'));
        this.importingDemo.set(false);
      },
    });
  }

  private uploadCsv(file: File): void {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.error.set('Select a shop first.');
      return;
    }
    this.uploading.set(true);
    this.message.set('');
    this.error.set('');
    this.products.importProductsCsv(slug, file).subscribe({
      next: (response) => {
        const created = this.safeNumber(response?.data, 'created');
        const imported = created ?? 'some';
        this.message.set(`Imported ${imported} products from your CSV.`);
        this.uploading.set(false);
      },
      error: (err) => {
        this.error.set(this.resolveError(err, 'Could not import that CSV. Double-check the template and try again.'));
        this.uploading.set(false);
      },
    });
  }

  private safeNumber(payload: Record<string, unknown> | undefined, key: string): number | null {
    if (!payload) {
      return null;
    }
    const value = payload[key];
    if (typeof value === 'number') {
      return value;
    }
    if (typeof value === 'string') {
      const parsed = Number(value);
      if (!Number.isNaN(parsed)) {
        return parsed;
      }
    }
    return null;
  }

  private resolveError(err: unknown, fallback: string): string {
    if (!err) {
      return fallback;
    }
    const httpErr = err as { error?: { message?: string }; message?: string };
    if (httpErr?.error?.message) {
      return httpErr.error.message;
    }
    if (httpErr?.message) {
      return httpErr.message;
    }
    return fallback;
  }
}
