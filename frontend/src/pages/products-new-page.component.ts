import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { ProductService } from '../app/core/services/product.service';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'products-new-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './products-new-page.component.html',
  styleUrls: ['./products-new-page.component.css']
})
export class ProductsNewPageComponent implements OnInit {
  private api = environment.apiBase;
  // Basic product model for skeleton form
  model: any = {
    title: 'Short sleeve t-shirt',
    description: '',
    mediaFiles: [],
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
  shops = signal<any[]>([]);
  shopSlug = signal<string>('');
  chosenFileName = signal<string>('No file chosen');

  constructor(private http: HttpClient, private productsApi: ProductService) {}

  ngOnInit(): void {
    this.http.get<any>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => {
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
