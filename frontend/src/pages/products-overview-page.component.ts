import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-products-overview-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-overview-page.component.html',
  styleUrls: ['./products-overview-page.component.css']
})
export class ProductsOverviewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  readonly api = environment.apiBase;
  shops = signal<ShopSummary[]>([]);
  shopSlug = signal<string>('');
  items = signal<ProductSummary[]>([]);
  loading = signal<boolean>(false);
  q = signal<string>('');

  ngOnInit(): void {
    this.http.get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => {
      const arr = r?.data || []; this.shops.set(arr); if (arr.length) { this.shopSlug.set(arr[0].slug); this.load(); }
    });
  }
  load(){
    if (!this.shopSlug()) return; this.loading.set(true);
    const qs = this.q() ? `?q=${encodeURIComponent(this.q())}` : '';
    this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/my/shops/${this.shopSlug()}/products${qs}`, { withCredentials: true })
      .subscribe({ next: r => { this.items.set(r?.data || []); this.loading.set(false); }, error: () => this.loading.set(false) });
  }
}

export interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

