import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-settings-checkout-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-checkout-page.component.html',
  styleUrls: ['./settings-checkout-page.component.css']
})
export class SettingsCheckoutPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  guestCheckout = signal(true);
  requireEmail = signal(true);
  requirePhone = signal(false);
  orderNotes = signal(true);
  termsAcceptance = signal(false);

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/checkout-settings`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.guestCheckout.set(d?.guestCheckout ?? true);
      this.requireEmail.set(d?.requireEmail ?? true);
      this.requirePhone.set(d?.requirePhone ?? false);
      this.orderNotes.set(d?.orderNotes ?? true);
      this.termsAcceptance.set(d?.termsAcceptance ?? false);
    } catch {
      this.error.set('Unable to load checkout settings.');
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
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/checkout-settings`, {
          guestCheckout: this.guestCheckout(),
          requireEmail: this.requireEmail(),
          requirePhone: this.requirePhone(),
          orderNotes: this.orderNotes(),
          termsAcceptance: this.termsAcceptance(),
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save checkout settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
