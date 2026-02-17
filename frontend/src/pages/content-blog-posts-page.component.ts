import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface BlogPost {
  id: string;
  title: string;
  status: 'published' | 'draft' | 'archived';
  date: string;
}

@Component({
  standalone: true,
  selector: 'app-content-blog-posts-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './content-blog-posts-page.component.html',
  styleUrls: ['./content-blog-posts-page.component.css']
})
export class ContentBlogPostsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  posts = signal<BlogPost[]>([]);
  statusFilter = 'all';
  statuses = ['all', 'published', 'draft', 'archived'];

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/content/blog-posts`, { withCredentials: true })
      );
      this.posts.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load blog posts.');
    } finally {
      this.loading.set(false);
    }
  }

  filteredPosts(): BlogPost[] {
    if (this.statusFilter === 'all') return this.posts();
    return this.posts().filter(p => p.status === this.statusFilter);
  }

  statusClass(status: string): string {
    switch (status) {
      case 'published': return 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300';
      case 'draft': return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-300';
      case 'archived': return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
      default: return 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300';
    }
  }

  createPost() {
    this.message.set('Blog post editor coming soon.');
  }
}
