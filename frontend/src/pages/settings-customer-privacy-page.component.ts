import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-customer-privacy-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-customer-privacy-page.component.html',
  styleUrls: ['./settings-customer-privacy-page.component.css']
})
export class SettingsCustomerPrivacyPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  cookieConsent = signal(true);
  dataCollectionNotice = signal(true);
  rightToDelete = signal(false);
  privacyPolicyUrl = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/privacy-settings`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.cookieConsent.set(d?.cookieConsent ?? true);
      this.dataCollectionNotice.set(d?.dataCollectionNotice ?? true);
      this.rightToDelete.set(d?.rightToDelete ?? false);
      this.privacyPolicyUrl = d?.privacyPolicyUrl ?? '';
    } catch {
      this.error.set('Unable to load privacy settings.');
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
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/privacy-settings`, {
          cookieConsent: this.cookieConsent(),
          dataCollectionNotice: this.dataCollectionNotice(),
          rightToDelete: this.rightToDelete(),
          privacyPolicyUrl: this.privacyPolicyUrl.trim(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save privacy settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
