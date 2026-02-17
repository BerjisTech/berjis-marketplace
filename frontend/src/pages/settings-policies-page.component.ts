import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-policies-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-policies-page.component.html',
  styleUrls: ['./settings-policies-page.component.css']
})
export class SettingsPoliciesPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  refundPolicy = '';
  privacyPolicy = '';
  termsOfService = '';
  shippingPolicy = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/policies`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.refundPolicy = d?.refundPolicy ?? '';
      this.privacyPolicy = d?.privacyPolicy ?? '';
      this.termsOfService = d?.termsOfService ?? '';
      this.shippingPolicy = d?.shippingPolicy ?? '';
    } catch {
      this.error.set('Unable to load policies.');
    } finally {
      this.loading.set(false);
    }
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/policies`, {
          refundPolicy: this.refundPolicy,
          privacyPolicy: this.privacyPolicy,
          termsOfService: this.termsOfService,
          shippingPolicy: this.shippingPolicy,
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save policies.');
    } finally {
      this.saving.set(false);
    }
  }
}
