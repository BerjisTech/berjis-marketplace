import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Market {
  id: string;
  name: string;
  currency: string;
  languages: string[];
  active: boolean;
}

@Component({
  standalone: true,
  selector: 'app-markets-overview-page',
  imports: [CommonModule, RouterLink, FormsModule],
  templateUrl: './markets-overview-page.component.html',
  styleUrls: ['./markets-overview-page.component.css']
})
export class MarketsOverviewPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  markets = signal<Market[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/markets`, { withCredentials: true })
      );
      this.markets.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load markets.');
    } finally {
      this.loading.set(false);
    }
  }

  addMarket() {
    this.message.set('Market creation coming soon.');
  }
}
