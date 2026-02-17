import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface FileItem {
  id: string;
  name: string;
  size: string;
  date: string;
  url: string;
}

@Component({
  standalone: true,
  selector: 'app-content-files-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './content-files-page.component.html',
  styleUrls: ['./content-files-page.component.css']
})
export class ContentFilesPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');

  files = signal<FileItem[]>([]);
  viewMode = signal<'grid' | 'list'>('grid');

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/content/files`, { withCredentials: true })
      );
      this.files.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load files.');
    } finally {
      this.loading.set(false);
    }
  }

  toggleView() {
    this.viewMode.set(this.viewMode() === 'grid' ? 'list' : 'grid');
  }

  upload() {
    this.message.set('File upload coming soon.');
  }
}
