import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  standalone: true,
  selector: 'products-new-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './products-new-page.component.html',
  styleUrls: ['./products-new-page.component.css']
})
export class ProductsNewPageComponent {
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

  submit() {
    this.submitting.set(true);
    // In real app, call API; for skeleton just simulate
    setTimeout(() => this.submitting.set(false), 600);
  }
}

