import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnDestroy, OnInit, computed, effect, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';
import { OrderService, ShopMetrics } from '../app/core/services/order.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-dashboard-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './dashboard-overview-page.component.html',
  styleUrls: ['./dashboard-overview-page.component.css'],
})
export class DashboardOverviewPageComponent implements OnInit, OnDestroy {
  private readonly http = inject(HttpClient);
  private readonly ordersService = inject(OrderService);
  private readonly api = environment.apiBase;
  private readonly shopState = inject(ShopStateService);

  readonly shops = this.shopState.shops;
  readonly activeShopSlug = this.shopState.activeShopSlug;
  readonly products = signal<ProductSummary[]>([]);
  readonly metrics = signal<ShopMetrics | null>(null);
  readonly loading = signal<boolean>(false);
  readonly error = signal<string>('');

  readonly chartHeight = 120;
  readonly chartWidth = 420;

  readonly salesSeries = computed(() => this.metrics()?.salesSeries ?? []);

  readonly chartPath = computed(() => {
    const points = this.salesSeries();
    if (!points.length) {
      return '';
    }
    const max = Math.max(...points.map((point) => point.totalCents), 1);
    const step = points.length > 1 ? this.chartWidth / (points.length - 1) : this.chartWidth;
    return points
      .map((point, index) => {
        const x = index * step;
        const y = this.chartHeight - (point.totalCents / max) * this.chartHeight;
        return `${index === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  });

  readonly chartGradient = computed(() => {
    const path = this.chartPath();
    if (!path) {
      return '';
    }
    return `${path} L${this.chartWidth},${this.chartHeight} L0,${this.chartHeight} Z`;
  });

  private readonly syncActiveShop = effect(() => {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.products.set([]);
      this.metrics.set(null);
      this.loading.set(false);
      return;
    }
    this.loadData(slug);
  });

  ngOnInit(): void {
    this.shopState.ensureLoaded();
  }

  ngOnDestroy(): void {
    this.syncActiveShop.destroy();
  }

  changeShop(slug: string): void {
    if (!slug) {
      return;
    }
    this.shopState.setActiveShopSlug(slug);
  }

  private loadData(slug: string): void {
    this.loading.set(true);
    this.error.set('');
    this.loadProducts(slug);
    this.ordersService.getShopMetrics(slug).subscribe({
      next: (response) => {
        this.metrics.set(response?.data ?? null);
        this.loading.set(false);
      },
      error: () => {
        this.metrics.set(null);
        this.loading.set(false);
        this.error.set('Could not load sales metrics.');
      },
    });
  }

  private loadProducts(slug: string): void {
    this.http
      .get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/my/shops/${slug}/products`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const items = response?.data ?? [];
          this.products.set(items.slice(0, 3));
        },
        error: () => {
          this.products.set([]);
        },
      });
  }

  formatCurrency(cents: number, currency: string): string {
    try {
      return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(
        (cents ?? 0) / 100,
      );
    } catch {
      return `${(cents ?? 0) / 100} ${currency}`;
    }
  }

  isGenericShopName(name?: string): boolean {
    if (!name) {
      return true;
    }
    const sanitized = name.trim().toLowerCase();
    return /(my\s+shop|new\s+shop|untitled|demo\s+shop|store|shop\s*\d+)/.test(sanitized);
  }
}
