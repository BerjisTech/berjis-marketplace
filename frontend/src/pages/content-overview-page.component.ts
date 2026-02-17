import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface ContentCount {
  label: string;
  count: number;
  link: string;
}

@Component({
  standalone: true,
  selector: 'app-content-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './content-overview-page.component.html',
  styleUrls: ['./content-overview-page.component.css']
})
export class ContentOverviewPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');

  counts = signal<ContentCount[]>([
    { label: 'Pages', count: 0, link: '/dashboard/content' },
    { label: 'Blog posts', count: 0, link: '/dashboard/content/blog-posts' },
    { label: 'Files', count: 0, link: '/dashboard/content/files' },
    { label: 'Menus', count: 0, link: '/dashboard/content/menus' },
  ]);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/content/overview`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      if (d?.counts) {
        this.counts.set(d.counts);
      }
    } catch {
      this.error.set('Unable to load content overview.');
    } finally {
      this.loading.set(false);
    }
  }
}
