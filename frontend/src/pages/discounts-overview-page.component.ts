import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';
import { DiscountReportRow, DiscountService } from '../app/core/services/discount.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

@Component({
  standalone: true,
  selector: 'app-discounts-overview-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './discounts-overview-page.component.html',
  styleUrls: ['./discounts-overview-page.component.css'],
})
export class DiscountsOverviewPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly discountsApi = inject(DiscountService);

  readonly api = environment.apiBase;
  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly loadingShops = signal<boolean>(false);
  readonly loadingReport = signal<boolean>(false);
  readonly shopError = signal<string>('');
  readonly reportError = signal<string>('');
  readonly report = signal<DiscountReportRow[]>([]);

  readonly hasReport = computed(() => !this.loadingReport() && this.report().length > 0);
  readonly totalDiscounts = computed(() => this.report().length);
  readonly activeDiscounts = computed(
    () => this.report().filter((row) => row.status?.toLowerCase() === 'active').length,
  );
  readonly totalRedemptions = computed(() =>
    this.report().reduce((sum, row) => sum + (row.redemptionCount || 0), 0),
  );
  readonly totalRedeemedCents = computed(() =>
    this.report().reduce((sum, row) => sum + (row.totalAmountCents || 0), 0),
  );

  ngOnInit(): void {
    this.loadShops();
  }

  loadShops(): void {
    this.loadingShops.set(true);
    this.shopError.set('');
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const list = response?.data ?? [];
          this.shops.set(list);
          const existingSlug = this.shopSlug();
          const initial =
            existingSlug && list.some((shop) => shop.slug === existingSlug)
              ? existingSlug
              : list[0]?.slug ?? '';
          this.shopSlug.set(initial);
          this.loadingShops.set(false);
          if (initial) {
            this.loadReport();
          }
        },
        error: () => {
          this.loadingShops.set(false);
          this.shopError.set('Could not load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopSlug.set(slug);
    this.report.set([]);
    this.reportError.set('');
    if (slug) {
      this.loadReport();
    }
  }

  refresh(): void {
    if (!this.shopSlug()) {
      return;
    }
    this.loadReport(true);
  }

  loadReport(force = false): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.loadingReport() && !force) {
      return;
    }
    this.loadingReport.set(true);
    this.reportError.set('');
    this.discountsApi.report(slug).subscribe({
      next: (response) => {
        this.report.set(response?.data ?? []);
        this.loadingReport.set(false);
      },
      error: () => {
        this.loadingReport.set(false);
        this.reportError.set('Could not load discount metrics.');
      },
    });
  }

  trackShopBy(_index: number, shop: ShopSummary): string {
    return shop.uuid;
  }

  trackDiscountBy(_index: number, row: DiscountReportRow): string {
    return row.uuid;
  }
}


