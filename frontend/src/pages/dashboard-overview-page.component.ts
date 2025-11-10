import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-dashboard-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './dashboard-overview-page.component.html',
  styleUrls: ['./dashboard-overview-page.component.css']
})
export class DashboardOverviewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;
  shops = signal<ShopSummary[]>([]);
  selectedShop: ShopSummary | null = null;
  products = signal<ProductSummary[]>([]);

  ngOnInit(): void {
    this.load();
  }

  private load(){
    this.http.get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: response => {
          const shops = response?.data ?? [];
          this.shops.set(shops);
          this.selectedShop = shops[0] ?? null;
          if (this.selectedShop) {
            this.loadProducts(this.selectedShop.slug);
          } else {
            this.products.set([]);
          }
        },
      });
  }

  private loadProducts(slug: string){
    this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/my/shops/${slug}/products`, { withCredentials: true })
      .subscribe({
        next: response => this.products.set(response?.data ?? []),
      });
  }

  isGenericShopName(name?: string): boolean {
    if (!name) return true;
    const n = name.trim().toLowerCase();
    return /(my\s+shop|new\s+shop|untitled|demo\s+shop|store|shop\s*\d+)/.test(n);
  }
}

export interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}


