import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Campaign {
  id: string;
  name: string;
  status: 'active' | 'paused' | 'draft' | 'completed';
  budget: string;
  clicks: number;
  conversions: number;
}

@Component({
  standalone: true,
  selector: 'app-marketing-campaigns-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './marketing-campaigns-page.component.html',
  styleUrls: ['./marketing-campaigns-page.component.css']
})
export class MarketingCampaignsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  campaigns = signal<Campaign[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/marketing/campaigns`, { withCredentials: true })
      );
      this.campaigns.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load campaigns.');
    } finally {
      this.loading.set(false);
    }
  }

  statusClass(status: string): string {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300';
      case 'paused': return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300';
      case 'completed': return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300';
      default: return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
    }
  }

  createCampaign() {
    this.message.set('Campaign creation coming soon.');
  }
}
