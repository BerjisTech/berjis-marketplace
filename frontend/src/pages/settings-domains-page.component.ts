import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface Domain {
  id: string;
  domain: string;
  verified: boolean;
  primary: boolean;
}

@Component({
  standalone: true,
  selector: 'app-settings-domains-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-domains-page.component.html',
  styleUrls: ['./settings-domains-page.component.css']
})
export class SettingsDomainsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  error = signal('');
  message = signal('');
  verifying = signal('');

  domains = signal<Domain[]>([]);
  newDomain = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/domains`, { withCredentials: true })
      );
      this.domains.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load domains.');
    } finally {
      this.loading.set(false);
    }
  }

  async addDomain() {
    if (!this.newDomain.trim()) return;
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    try {
      const res: any = await firstValueFrom(
        this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/domains`, {
          domain: this.newDomain.trim(),
        }, { withCredentials: true })
      );
      const domain = res?.data ?? { id: crypto.randomUUID(), domain: this.newDomain.trim(), verified: false, primary: false };
      this.domains.set([...this.domains(), domain]);
      this.newDomain = '';
      this.message.set('Domain added. Please verify DNS settings.');
    } catch {
      this.error.set('Unable to add domain.');
    }
  }

  async verifyDomain(id: string) {
    this.verifying.set(id);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/domains/${id}/verify`, {}, { withCredentials: true })
      );
      this.domains.set(this.domains().map(d => d.id === id ? { ...d, verified: true } : d));
      this.message.set('Domain verified successfully.');
    } catch {
      this.error.set('Verification failed. Please check DNS settings.');
    } finally {
      this.verifying.set('');
    }
  }

  async removeDomain(id: string) {
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.delete(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/domains/${id}`, { withCredentials: true })
      );
      this.domains.set(this.domains().filter(d => d.id !== id));
      this.message.set('Domain removed.');
    } catch {
      this.error.set('Unable to remove domain.');
    }
  }
}
