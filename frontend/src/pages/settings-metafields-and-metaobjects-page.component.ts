import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface MetafieldDefinition {
  id: string;
  namespace: string;
  key: string;
  type: string;
}

@Component({
  standalone: true,
  selector: 'app-settings-metafields-and-metaobjects-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-metafields-and-metaobjects-page.component.html',
  styleUrls: ['./settings-metafields-and-metaobjects-page.component.css']
})
export class SettingsMetafieldsAndMetaobjectsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  error = signal('');
  message = signal('');

  definitions = signal<MetafieldDefinition[]>([]);

  newNamespace = '';
  newKey = '';
  newType = 'single_line_text';

  fieldTypes = ['single_line_text', 'multi_line_text', 'integer', 'decimal', 'boolean', 'date', 'url', 'json', 'color'];

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/metafield-definitions`, { withCredentials: true })
      );
      this.definitions.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load metafield definitions.');
    } finally {
      this.loading.set(false);
    }
  }

  async addDefinition() {
    if (!this.newNamespace.trim() || !this.newKey.trim()) return;
    this.saving.set(true);
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    const payload = {
      namespace: this.newNamespace.trim(),
      key: this.newKey.trim(),
      type: this.newType,
    };
    try {
      const res: any = await firstValueFrom(
        this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/metafield-definitions`, payload, { withCredentials: true })
      );
      const def = res?.data ?? { id: crypto.randomUUID(), ...payload };
      this.definitions.set([...this.definitions(), def]);
      this.newNamespace = '';
      this.newKey = '';
      this.newType = 'single_line_text';
      this.message.set('Metafield definition added.');
    } catch {
      this.error.set('Unable to add metafield definition.');
    } finally {
      this.saving.set(false);
    }
  }

  async removeDefinition(id: string) {
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.delete(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/metafield-definitions/${id}`, { withCredentials: true })
      );
      this.definitions.set(this.definitions().filter(d => d.id !== id));
      this.message.set('Metafield definition removed.');
    } catch {
      this.error.set('Unable to remove metafield definition.');
    }
  }
}
