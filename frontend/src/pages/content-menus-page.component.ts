import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface MenuItem {
  id: string;
  label: string;
  url: string;
}

interface NavMenu {
  id: string;
  name: string;
  items: MenuItem[];
}

@Component({
  standalone: true,
  selector: 'app-content-menus-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './content-menus-page.component.html',
  styleUrls: ['./content-menus-page.component.css']
})
export class ContentMenusPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  error = signal('');
  message = signal('');

  menus = signal<NavMenu[]>([]);
  activeMenuId = signal('');

  newItemLabel = '';
  newItemUrl = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/content/menus`, { withCredentials: true })
      );
      this.menus.set(res?.data ?? []);
      if (this.menus().length && !this.activeMenuId()) {
        this.activeMenuId.set(this.menus()[0].id);
      }
    } catch {
      this.error.set('Unable to load menus.');
    } finally {
      this.loading.set(false);
    }
  }

  activeMenu(): NavMenu | undefined {
    return this.menus().find(m => m.id === this.activeMenuId());
  }

  addItem() {
    if (!this.newItemLabel.trim()) return;
    const menu = this.activeMenu();
    if (!menu) return;
    const item: MenuItem = {
      id: crypto.randomUUID(),
      label: this.newItemLabel.trim(),
      url: this.newItemUrl.trim(),
    };
    this.menus.set(
      this.menus().map(m => m.id === menu.id ? { ...m, items: [...m.items, item] } : m)
    );
    this.newItemLabel = '';
    this.newItemUrl = '';
  }

  removeItem(menuId: string, itemId: string) {
    this.menus.set(
      this.menus().map(m => m.id === menuId ? { ...m, items: m.items.filter(i => i.id !== itemId) } : m)
    );
  }

  moveItem(menuId: string, itemId: string, direction: 'up' | 'down') {
    this.menus.set(
      this.menus().map(m => {
        if (m.id !== menuId) return m;
        const items = [...m.items];
        const idx = items.findIndex(i => i.id === itemId);
        if (idx < 0) return m;
        const newIdx = direction === 'up' ? idx - 1 : idx + 1;
        if (newIdx < 0 || newIdx >= items.length) return m;
        [items[idx], items[newIdx]] = [items[newIdx], items[idx]];
        return { ...m, items };
      })
    );
  }

  async save() {
    this.saving.set(true);
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/content/menus`, {
          menus: this.menus(),
        }, { withCredentials: true })
      );
      this.message.set('Menus saved.');
    } catch {
      this.error.set('Unable to save menus.');
    } finally {
      this.saving.set(false);
    }
  }
}
