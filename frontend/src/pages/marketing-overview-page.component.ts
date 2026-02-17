import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface MarketingMetric {
  label: string;
  value: string | number;
  link: string;
}

@Component({
  standalone: true,
  selector: 'app-marketing-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './marketing-overview-page.component.html',
  styleUrls: ['./marketing-overview-page.component.css']
})
export class MarketingOverviewPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');

  metrics = signal<MarketingMetric[]>([
    { label: 'Campaigns', value: 0, link: '/dashboard/marketing/campaigns' },
    { label: 'Impressions', value: 0, link: '/dashboard/marketing/campaigns' },
    { label: 'Clicks', value: 0, link: '/dashboard/marketing/attributions' },
    { label: 'Conversions', value: 0, link: '/dashboard/marketing/attributions' },
  ]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/marketing/overview`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      if (d?.metrics) {
        this.metrics.set(d.metrics);
      }
    } catch {
      this.error.set('Unable to load marketing overview.');
    } finally {
      this.loading.set(false);
    }
  }
}
