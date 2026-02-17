import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface ChannelBreakdown {
  channel: string;
  visits: number;
  conversions: number;
  revenue: string;
}

@Component({
  standalone: true,
  selector: 'app-marketing-attributions-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './marketing-attributions-page.component.html',
  styleUrls: ['./marketing-attributions-page.component.css']
})
export class MarketingAttributionsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');

  attributionModel = 'last-click';
  models = ['first-click', 'last-click', 'linear'];
  channels = signal<ChannelBreakdown[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/marketing/attributions`, {
          params: { model: this.attributionModel },
          withCredentials: true,
        })
      );
      this.channels.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load attribution data.');
    } finally {
      this.loading.set(false);
    }
  }

  onModelChange() {
    this.load();
  }
}
