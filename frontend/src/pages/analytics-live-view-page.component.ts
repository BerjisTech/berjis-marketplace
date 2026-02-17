import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AnalyticsService, AnalyticsEvent } from '../app/core/services/analytics.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-analytics-live-view-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './analytics-live-view-page.component.html',
  styleUrls: ['./analytics-live-view-page.component.css']
})
export class AnalyticsLiveViewPageComponent implements OnInit, OnDestroy {
  private analytics = inject(AnalyticsService);
  private shop = inject(ShopStateService);
  private pollInterval: ReturnType<typeof setInterval> | null = null;

  loading = signal(true);
  error = signal('');

  events = signal<AnalyticsEvent[]>([]);
  lastUpdated = signal('');

  ngOnInit() { this.load(); this.startPolling(); }

  ngOnDestroy() { this.stopPolling(); }

  load() {
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); this.error.set('No active shop selected.'); return; }
    this.analytics.getRecentEvents(slug, 50).subscribe({
      next: res => {
        this.events.set(res?.data ?? []);
        this.lastUpdated.set(new Date().toLocaleTimeString());
        this.error.set('');
        this.loading.set(false);
      },
      error: () => {
        this.error.set('Unable to load live data.');
        this.loading.set(false);
      }
    });
  }

  private startPolling() {
    this.pollInterval = setInterval(() => this.load(), 10000);
  }

  private stopPolling() {
    if (this.pollInterval) {
      clearInterval(this.pollInterval);
      this.pollInterval = null;
    }
  }

  formatTime(iso: string): string {
    if (!iso) return '';
    const d = new Date(iso);
    return d.toLocaleTimeString();
  }

  formatEventName(name: string): string {
    return name.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
  }

  eventCounts(): { name: string; count: number }[] {
    const m = new Map<string, number>();
    for (const e of this.events()) {
      m.set(e.eventName, (m.get(e.eventName) ?? 0) + 1);
    }
    return Array.from(m.entries())
      .map(([name, count]) => ({ name, count }))
      .sort((a, b) => b.count - a.count);
  }
}
