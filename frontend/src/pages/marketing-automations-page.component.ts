import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface AutomationRule {
  id: string;
  trigger: string;
  action: string;
  status: 'active' | 'inactive';
}

@Component({
  standalone: true,
  selector: 'app-marketing-automations-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './marketing-automations-page.component.html',
  styleUrls: ['./marketing-automations-page.component.css']
})
export class MarketingAutomationsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  rules = signal<AutomationRule[]>([]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/marketing/automations`, { withCredentials: true })
      );
      this.rules.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load automation rules.');
    } finally {
      this.loading.set(false);
    }
  }

  toggleRule(id: string) {
    this.rules.set(
      this.rules().map(r => r.id === id
        ? { ...r, status: r.status === 'active' ? 'inactive' as const : 'active' as const }
        : r
      )
    );
  }

  createRule() {
    this.message.set('Automation rule creation coming soon.');
  }
}
