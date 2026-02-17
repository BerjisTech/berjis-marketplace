import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface ConnectedApp {
  id: string;
  name: string;
  description: string;
  status: 'connected' | 'disconnected' | 'error';
  icon: string;
}

@Component({
  standalone: true,
  selector: 'app-settings-apps-and-sales-channels-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-apps-and-sales-channels-page.component.html',
  styleUrls: ['./settings-apps-and-sales-channels-page.component.css']
})
export class SettingsAppsAndSalesChannelsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  apps = signal<ConnectedApp[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/apps`, { withCredentials: true })
      );
      this.apps.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load connected apps.');
    } finally {
      this.loading.set(false);
    }
  }

  statusClass(status: string): string {
    switch (status) {
      case 'connected': return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300';
      case 'error': return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300';
      default: return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
    }
  }

  async disconnect(id: string) {
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.delete(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/apps/${id}`, { withCredentials: true })
      );
      this.apps.set(this.apps().map(a => a.id === id ? { ...a, status: 'disconnected' as const } : a));
      this.message.set('App disconnected.');
    } catch {
      this.error.set('Unable to disconnect app.');
    }
  }
}
