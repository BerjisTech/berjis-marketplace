import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { AnalyticsService, SalesData, TopProduct, CustomerData, ConversionData } from '../app/core/services/analytics.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-analytics-overview-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './analytics-overview-page.component.html',
  styleUrls: ['./analytics-overview-page.component.css']
})
export class AnalyticsOverviewPageComponent implements OnInit {
  private analytics = inject(AnalyticsService);
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');

  sales = signal<SalesData | null>(null);
  topProducts = signal<TopProduct[]>([]);
  customers = signal<CustomerData | null>(null);
  conversions = signal<ConversionData | null>(null);

  dateFrom = '';
  dateTo = '';
  granularity = 'day';

  ngOnInit() {
    const today = new Date();
    const thirtyDaysAgo = new Date(today.getTime() - 30 * 24 * 60 * 60 * 1000);
    this.dateTo = today.toISOString().split('T')[0];
    this.dateFrom = thirtyDaysAgo.toISOString().split('T')[0];
    this.load();
  }

  load() {
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); this.error.set('No active shop selected.'); return; }
    this.loading.set(true);
    this.error.set('');

    let pending = 4;
    const done = () => { pending--; if (pending <= 0) this.loading.set(false); };

    this.analytics.getSales(slug, this.dateFrom, this.dateTo, this.granularity).subscribe({
      next: res => { this.sales.set(res?.data ?? null); done(); },
      error: () => { this.error.set('Unable to load sales data.'); done(); }
    });

    this.analytics.getTopProducts(slug, this.dateFrom, this.dateTo).subscribe({
      next: res => { this.topProducts.set(res?.data ?? []); done(); },
      error: () => { done(); }
    });

    this.analytics.getCustomers(slug, this.dateFrom, this.dateTo).subscribe({
      next: res => { this.customers.set(res?.data ?? null); done(); },
      error: () => { done(); }
    });

    this.analytics.getConversions(slug, this.dateFrom, this.dateTo).subscribe({
      next: res => { this.conversions.set(res?.data ?? null); done(); },
      error: () => { done(); }
    });
  }

  fmt(cents: number): string { return (cents / 100).toFixed(2); }

  maxRevenue(): number {
    const s = this.sales();
    if (!s?.series?.length) return 1;
    return Math.max(...s.series.map(p => p.revenueCents), 1);
  }

  barHeight(revenueCents: number): number {
    const max = this.maxRevenue();
    return max > 0 ? (revenueCents / max) * 100 : 0;
  }
}
