import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { ProductService } from '../app/core/services/product.service';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-products-new-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './products-new-page.component.html',
  styleUrls: ['./products-new-page.component.css']
})
export class ProductsNewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly productsApi = inject(ProductService);
  private readonly api = environment.apiBase;
  // Basic product model for skeleton form
  model: ProductDraft = {
    title: 'Short sleeve t-shirt',
    description: '',
    mediaFiles: [] as string[],
    category: '',
    price: 0,
    compareAt: null,
    unitPrice: null,
    chargeTax: true,
    costPerItem: null,
    inventoryTracked: true,
    quantity: 0,
    sku: '',
    barcode: '',
    allowOversell: false,
    physicalProduct: true,
    packagePreset: 'default',
    weight: 0,
    weightUnit: 'kg',
    countryOfOrigin: '',
    hsCode: '',
    seoTitle: '',
    seoDescription: '',
    publishOnlineStore: true,
    publishPOS: true,
    type: '',
    vendor: '',
    collections: '',
    tags: '',
    themeTemplate: 'default'
  };
  submitting = signal(false);
  shops = signal<ShopSummary[]>([]);
  shopSlug = signal<string>('');
  chosenFileName = signal<string>('No file chosen');

  ngOnInit(): void {
    this.http.get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => {
      const arr = r?.data || []; this.shops.set(arr);
      if (arr.length && !this.shopSlug()) this.shopSlug.set(arr[0].slug);
    });
  }

  onFile(ev: Event){
    const input = ev.target as HTMLInputElement; const file = input.files?.[0]; if(!file) return;
    this.chosenFileName.set(file.name);
    this.productsApi.upload(file).subscribe(res => {
      const url = res?.data?.url || '';
      if (!this.model.mediaFiles) this.model.mediaFiles = [];
      if (url) this.model.mediaFiles.push(url);
    });
  }

  submit() {
    this.submitting.set(true);
    const payload = { ...this.model, shopSlug: this.shopSlug() };
    this.productsApi.createProduct(payload).subscribe({
      next: () => this.submitting.set(false),
      error: () => this.submitting.set(false)
    });
  }
}

export interface ProductDraft {
  title: string;
  description: string;
  mediaFiles: string[];
  category: string;
  price: number;
  compareAt: number | null;
  unitPrice: number | null;
  chargeTax: boolean;
  costPerItem: number | null;
  inventoryTracked: boolean;
  quantity: number;
  sku: string;
  barcode: string;
  allowOversell: boolean;
  physicalProduct: boolean;
  packagePreset: string;
  weight: number;
  weightUnit: string;
  countryOfOrigin: string;
  hsCode: string;
  seoTitle: string;
  seoDescription: string;
  publishOnlineStore: boolean;
  publishPOS: boolean;
  type: string;
  vendor: string;
  collections: string;
  tags: string;
  themeTemplate: string;
  [key: string]: unknown;
}

export interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

