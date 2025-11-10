import { Component, EventEmitter, Input, Output, WritableSignal, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Router, RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-new-shop-form',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './new-shop-form.component.html',
  styleUrls: ['./new-shop-form.component.css']
})
export class NewShopFormComponent {
  @Input() redirectTo: string | null = '/dashboard';
  @Output() created = new EventEmitter<{ name: string; slug: string }>();

  api = environment.apiBase;
  submitting = signal(false);
  errorMsg = signal<string>('');

  salesChannels = signal<string[]>([]);
  businessType = signal<'new'|'existing'|''>('');
  otherPlatforms = signal<string[]>([]);
  planToSell = signal<'own-products'|'digital'|'dropshipping'|'services'|'print-on-demand'|'later'|''>('');
  name = signal('');
  slug = signal('');
  slugEdited = signal(false);
  slugStatus = signal<'idle'|'checking'|'available'|'taken'|'error'>('idle');

  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);

  toggle(list: WritableSignal<string[]>, value: string) {
    const current = list();
    const next = new Set(current);
    if (next.has(value)) next.delete(value); else next.add(value);
    list.set(Array.from(next));
  }

  onNameChange(v: string) {
    this.name.set(v);
    if (!this.slugEdited()) {
      this.slug.set(this.slugify(v));
    }
  }

  slugify(v: string) {
    return (v||'').toLowerCase().trim().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'');
  }

  onSlugChange(v: string){
    this.slugEdited.set(true);
    const s = this.slugify(v);
    this.slug.set(s);
    this.checkSlug(s);
  }

  private checkSlug(s: string){
    if (!s) { this.slugStatus.set('idle'); return; }
    this.slugStatus.set('checking');
    // Try a conventional availability endpoint; ignore errors if backend differs
    this.http.get<ApiResponse<{ available: boolean }>>(`${this.api}/v1/shops/slug-availability?slug=${encodeURIComponent(s)}`, { withCredentials: true }).subscribe({
      next: (r) => {
        const ok = r?.data?.available ?? false;
        this.slugStatus.set(ok ? 'available' : 'taken');
      },
      error: () => { this.slugStatus.set('error'); }
    });
  }

  submit() {
    if (this.submitting()) return;
    const name = this.name().trim();
    const slug = (this.slug().trim() || this.slugify(name));
    if (!name || !slug) return;

    const meta: ShopMeta = {
      salesChannels: this.salesChannels(),
      businessType: this.businessType(),
      otherPlatforms: this.otherPlatforms(),
      planToSell: this.planToSell(),
    };

    const body: CreateShopPayload = { name, slug, description: '', meta };
    this.submitting.set(true);
    this.errorMsg.set('');
    this.http.post(`${this.api}/v1/shops`, body, { withCredentials: true }).subscribe({
      next: () => {
        this.submitting.set(false);
        this.created.emit({ name, slug });
        if (this.redirectTo) this.router.navigateByUrl(this.redirectTo);
      },
      error: (err: HttpErrorResponse) => {
        this.submitting.set(false);
        if (err.status === 401) {
          this.errorMsg.set('Please sign in to create a shop. Redirecting to login…');
          const redirect = encodeURIComponent(this.redirectTo || '/dashboard');
          setTimeout(()=> this.router.navigateByUrl(`/auth/login?redirect=${redirect}`), 600);
        } else if (err.status === 409) {
          this.errorMsg.set('That handle is already taken. Please choose another.');
          this.slugStatus.set('taken');
        } else {
          this.errorMsg.set('Could not create shop. Please try again.');
        }
      }
    });
  }
}

export interface ShopMeta {
  salesChannels: string[];
  businessType: 'new' | 'existing' | '';
  otherPlatforms: string[];
  planToSell: 'own-products' | 'digital' | 'dropshipping' | 'services' | 'print-on-demand' | 'later' | '';
}

export interface CreateShopPayload {
  name: string;
  slug: string;
  description: string;
  meta: ShopMeta;
}


