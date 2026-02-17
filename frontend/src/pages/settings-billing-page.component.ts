import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Invoice {
  id: string;
  date: string;
  amount: string;
  status: 'paid' | 'pending' | 'overdue';
  downloadUrl: string;
}

@Component({
  standalone: true,
  selector: 'app-settings-billing-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-billing-page.component.html',
  styleUrls: ['./settings-billing-page.component.css']
})
export class SettingsBillingPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');

  planName = signal('Free');
  billingCycle = signal('Monthly');
  nextBillingDate = signal('');
  invoices = signal<Invoice[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/billing`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.planName.set(d?.planName ?? 'Free');
      this.billingCycle.set(d?.billingCycle ?? 'Monthly');
      this.nextBillingDate.set(d?.nextBillingDate ?? '');
      this.invoices.set(d?.invoices ?? []);
    } catch {
      this.error.set('Unable to load billing information.');
    } finally {
      this.loading.set(false);
    }
  }

  statusClass(status: string): string {
    switch (status) {
      case 'paid': return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300';
      case 'pending': return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300';
      case 'overdue': return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300';
      default: return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
    }
  }
}
