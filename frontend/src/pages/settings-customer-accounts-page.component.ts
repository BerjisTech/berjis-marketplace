import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-customer-accounts-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-customer-accounts-page.component.html',
  styleUrls: ['./settings-customer-accounts-page.component.css']
})
export class SettingsCustomerAccountsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  loginRequired = signal(false);
  socialLogin = signal(false);
  minPasswordLength = 8;

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/customer-account-settings`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.loginRequired.set(d?.loginRequired ?? false);
      this.socialLogin.set(d?.socialLogin ?? false);
      this.minPasswordLength = d?.minPasswordLength ?? 8;
    } catch {
      this.error.set('Unable to load customer account settings.');
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
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/customer-account-settings`, {
          loginRequired: this.loginRequired(),
          socialLogin: this.socialLogin(),
          minPasswordLength: this.minPasswordLength,
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save customer account settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
