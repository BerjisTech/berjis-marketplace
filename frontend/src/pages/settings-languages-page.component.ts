import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../environments/environment';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface LanguageOption {
  code: string;
  name: string;
  enabled: boolean;
}

@Component({
  standalone: true,
  selector: 'app-settings-languages-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-languages-page.component.html',
  styleUrls: ['./settings-languages-page.component.css']
})
export class SettingsLanguagesPageComponent implements OnInit {
  private http = inject(HttpClient);
  private api = environment.apiBase;
  private shop = inject(ShopStateService);

  loading = signal(true);
  saving = signal(false);
  saved = signal(false);
  error = signal('');

  primaryLanguage = 'en';
  availableLanguages: LanguageOption[] = [
    { code: 'en', name: 'English', enabled: true },
    { code: 'fr', name: 'French', enabled: false },
    { code: 'es', name: 'Spanish', enabled: false },
    { code: 'de', name: 'German', enabled: false },
    { code: 'pt', name: 'Portuguese', enabled: false },
    { code: 'ar', name: 'Arabic', enabled: false },
    { code: 'sw', name: 'Swahili', enabled: false },
    { code: 'zh', name: 'Chinese', enabled: false },
    { code: 'ja', name: 'Japanese', enabled: false },
  ];

  ngOnInit() { this.load(); }

  async load() {
    this.loading.set(true);
    const slug = this.shop.activeShopSlug();
    if (!slug) { this.loading.set(false); return; }
    try {
      const res: any = await firstValueFrom(
        this.http.get(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/language-settings`, { withCredentials: true })
      );
      const d = res?.data ?? res;
      this.primaryLanguage = d?.primaryLanguage ?? 'en';
      if (d?.languages?.length) {
        this.availableLanguages = d.languages;
      }
    } catch {
      this.error.set('Unable to load language settings.');
    } finally {
      this.loading.set(false);
    }
  }

  toggleLanguage(code: string) {
    this.availableLanguages = this.availableLanguages.map(l =>
      l.code === code ? { ...l, enabled: !l.enabled } : l
    );
  }

  async save() {
    this.saving.set(true);
    this.saved.set(false);
    this.error.set('');
    const slug = this.shop.activeShopSlug();
    try {
      await firstValueFrom(
        this.http.put(`${this.api}/v1/my/shops/${encodeURIComponent(slug)}/language-settings`, {
          primaryLanguage: this.primaryLanguage,
          languages: this.availableLanguages,
        }, { withCredentials: true })
      );
      this.saved.set(true);
    } catch {
      this.error.set('Unable to save language settings.');
    } finally {
      this.saving.set(false);
    }
  }
}
