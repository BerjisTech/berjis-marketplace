import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface FulfillmentLocation {
  id: string;
  name: string;
  address: string;
  city: string;
  country: string;
}

@Component({
  standalone: true,
  selector: 'app-settings-locations-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-locations-page.component.html',
  styleUrls: ['./settings-locations-page.component.css']
})
export class SettingsLocationsPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  error = signal('');
  message = signal('');
  showForm = signal(false);

  locations = signal<FulfillmentLocation[]>([]);
  editingId = signal('');

  formName = '';
  formAddress = '';
  formCity = '';
  formCountry = '';

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/locations`, { withCredentials: true })
      );
      this.locations.set(res?.data ?? []);
    } catch {
      this.error.set('Unable to load locations.');
    } finally {
      this.loading.set(false);
    }
  }

  openAdd() {
    this.editingId.set('');
    this.formName = '';
    this.formAddress = '';
    this.formCity = '';
    this.formCountry = '';
    this.showForm.set(true);
  }

  openEdit(loc: FulfillmentLocation) {
    this.editingId.set(loc.id);
    this.formName = loc.name;
    this.formAddress = loc.address;
    this.formCity = loc.city;
    this.formCountry = loc.country;
    this.showForm.set(true);
  }

  cancelForm() {
    this.showForm.set(false);
    this.editingId.set('');
  }

  async saveLocation() {
    if (!this.formName.trim()) return;
    this.saving.set(true);
    this.error.set('');
    this.message.set('');
    const slug = this.shop.activeShopSlug();
    const payload = {
      name: this.formName.trim(),
      address: this.formAddress.trim(),
      city: this.formCity.trim(),
      country: this.formCountry.trim(),
    };

    try {
      if (this.editingId()) {
        await firstValueFrom(
          this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/locations/${this.editingId()}`, payload, { withCredentials: true })
        );
        this.locations.set(this.locations().map(l => l.id === this.editingId() ? { ...l, ...payload } : l));
        this.message.set('Location updated.');
      } else {
        const res: any = await firstValueFrom(
          this.http.post(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/locations`, payload, { withCredentials: true })
        );
        const loc = res?.data ?? { id: crypto.randomUUID(), ...payload };
        this.locations.set([...this.locations(), loc]);
        this.message.set('Location added.');
      }
      this.showForm.set(false);
      this.editingId.set('');
    } catch {
      this.error.set('Unable to save location.');
    } finally {
      this.saving.set(false);
    }
  }

  async removeLocation(id: string) {
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.delete(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/locations/${id}`, { withCredentials: true })
      );
      this.locations.set(this.locations().filter(l => l.id !== id));
      this.message.set('Location removed.');
    } catch {
      this.error.set('Unable to remove location.');
    }
  }
}
