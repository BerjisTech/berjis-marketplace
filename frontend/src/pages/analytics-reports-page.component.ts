import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AnalyticsService, SalesData, TopProduct, CustomerData, ConversionData } from '../app/core/services/analytics.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-analytics-reports-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './analytics-reports-page.component.html',
  styleUrls: ['./analytics-reports-page.component.css']
})
export class AnalyticsReportsPageComponent implements OnInit {
  private analytics = inject(AnalyticsService);
  private shop = inject(ShopStateService);

  loading = signal(false);
  error = signal('');

  dateFrom = '';
  dateTo = '';
  reportType = 'sales';
  reportTypes = ['sales', 'top-products', 'customers', 'conversions'];

  salesData = signal<SalesData | null>(null);
  topProducts = signal<TopProduct[]>([]);
  customerData = signal<CustomerData | null>(null);
  conversionData = signal<ConversionData | null>(null);

  ngOnInit() {
    const today = new Date();
    const thirtyDaysAgo = new Date(today.getTime() - 30 * 24 * 60 * 60 * 1000);
    this.dateTo = today.toISOString().split('T')[0];
    this.dateFrom = thirtyDaysAgo.toISOString().split('T')[0];
  }

  generateReport() {
    this.loading.set(true);
    this.error.set('');
    this.salesData.set(null);
    this.topProducts.set([]);
    this.customerData.set(null);
    this.conversionData.set(null);

    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); this.error.set('No active shop selected.'); return; }

    const done = () => this.loading.set(false);

    switch (this.reportType) {
      case 'sales':
        this.analytics.getSales(slug, this.dateFrom, this.dateTo, 'day').subscribe({
          next: res => { this.salesData.set(res?.data ?? null); done(); },
          error: () => { this.error.set('Unable to load sales report.'); done(); }
        });
        break;
      case 'top-products':
        this.analytics.getTopProducts(slug, this.dateFrom, this.dateTo, 50).subscribe({
          next: res => { this.topProducts.set(res?.data ?? []); done(); },
          error: () => { this.error.set('Unable to load products report.'); done(); }
        });
        break;
      case 'customers':
        this.analytics.getCustomers(slug, this.dateFrom, this.dateTo).subscribe({
          next: res => { this.customerData.set(res?.data ?? null); done(); },
          error: () => { this.error.set('Unable to load customer report.'); done(); }
        });
        break;
      case 'conversions':
        this.analytics.getConversions(slug, this.dateFrom, this.dateTo).subscribe({
          next: res => { this.conversionData.set(res?.data ?? null); done(); },
          error: () => { this.error.set('Unable to load conversion report.'); done(); }
        });
        break;
      default:
        done();
    }
  }

  exportCsv() {
    let csvContent = '';
    switch (this.reportType) {
      case 'sales': {
        const s = this.salesData();
        if (!s) return;
        csvContent = 'Date,Revenue (cents),Orders\n';
        for (const pt of s.series) {
          csvContent += `${pt.date},${pt.revenueCents},${pt.ordersCount}\n`;
        }
        csvContent += `\nTotal,${s.totalRevenueCents},${s.ordersCount}\n`;
        csvContent += `Average Order Value (cents),,${s.averageOrderValueCents}\n`;
        break;
      }
      case 'top-products': {
        const tp = this.topProducts();
        if (!tp.length) return;
        csvContent = 'Product,Units Sold,Revenue (cents)\n';
        for (const p of tp) {
          csvContent += `"${p.title.replace(/"/g, '""')}",${p.unitsSold},${p.revenueCents}\n`;
        }
        break;
      }
      case 'customers': {
        const cd = this.customerData();
        if (!cd) return;
        csvContent = 'Metric,Value\n';
        csvContent += `New Customers,${cd.newCustomers}\n`;
        csvContent += `Returning Customers,${cd.returningCustomers}\n`;
        csvContent += `Total Customers,${cd.totalCustomers}\n`;
        break;
      }
      case 'conversions': {
        const cv = this.conversionData();
        if (!cv) return;
        csvContent = 'Metric,Value\n';
        csvContent += `Product Views,${cv.productViews}\n`;
        csvContent += `Add to Cart,${cv.addToCarts}\n`;
        csvContent += `Checkouts Started,${cv.checkoutsStarted}\n`;
        csvContent += `Checkouts Completed,${cv.checkoutsCompleted}\n`;
        csvContent += `Conversion Rate,${cv.conversionRate.toFixed(2)}%\n`;
        break;
      }
    }
    if (!csvContent) return;
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${this.reportType}-report-${this.dateFrom}-to-${this.dateTo}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  fmt(cents: number): string { return (cents / 100).toFixed(2); }
}
