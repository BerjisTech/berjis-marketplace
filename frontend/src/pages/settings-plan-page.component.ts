import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Plan {
  id: string;
  name: string;
  price: string;
  features: string[];
  current: boolean;
}

@Component({
  standalone: true,
  selector: 'app-settings-plan-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-plan-page.component.html',
  styleUrls: ['./settings-plan-page.component.css']
})
export class SettingsPlanPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  switching = signal(false);
  error = signal('');
  message = signal('');

  currentPlan = signal('free');
  plans = signal<Plan[]>([
    { id: 'free', name: 'Free', price: '$0/mo', features: ['Up to 10 products', 'Basic analytics', 'Email support'], current: true },
    { id: 'pro', name: 'Pro', price: '$29/mo', features: ['Unlimited products', 'Advanced analytics', 'Priority support', 'Custom domain'], current: false },
    { id: 'enterprise', name: 'Enterprise', price: '$99/mo', features: ['Everything in Pro', 'Dedicated account manager', 'API access', 'Custom integrations', 'SLA guarantee'], current: false },
  ]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/plan`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      const activePlan = d?.planId ?? 'free';
      this.currentPlan.set(activePlan);
      this.plans.set(this.plans().map(p => ({ ...p, current: p.id === activePlan })));
    } catch {
      this.error.set('Unable to load plan information.');
    } finally {
      this.loading.set(false);
    }
  }

  async switchPlan(planId: string) {
    if (planId === this.currentPlan()) return;
    this.switching.set(true);
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/plan`, {
          planId,
        }, { withCredentials: true })
      );
      this.currentPlan.set(planId);
      this.plans.set(this.plans().map(p => ({ ...p, current: p.id === planId })));
      this.message.set('Plan updated successfully.');
    } catch {
      this.error.set('Unable to change plan.');
    } finally {
      this.switching.set(false);
    }
  }
}
