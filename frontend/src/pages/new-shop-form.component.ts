import { Component, EventEmitter, Input, Output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { Router, RouterLink } from '@angular/router';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'new-shop-form',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './new-shop-form.component.html',
  styleUrls: ['./new-shop-form.component.css']
})
export class NewShopFormComponent {
  @Input() redirectTo: string | null = '/dashboard';
  @Output() created = new EventEmitter<{ name: string; slug: string }>();

  api = environment.apiBase;
  submitting = signal(false);

  salesChannels = signal<string[]>([]);
  businessType = signal<'new'|'existing'|''>('');
  otherPlatforms = signal<string[]>([]);
  planToSell = signal<'own-products'|'digital'|'dropshipping'|'services'|'print-on-demand'|'later'|''>('');
  name = signal('');
  slug = signal('');

  constructor(private http: HttpClient, private router: Router) {}

  toggle(list: any, value: string) {
    const sig = list as ReturnType<typeof signal<string[]>>;
    const next = new Set((sig as any)());
    if (next.has(value)) next.delete(value); else next.add(value);
    (sig as any).set(Array.from(next));
  }

  onNameChange(v: string) {
    this.name.set(v);
    if (!this.slug()) {
      this.slug.set(this.slugify(v));
    }
  }

  slugify(v: string) {
    return (v||'').toLowerCase().trim().replace(/[^a-z0-9]+/g,'-').replace(/^-+|-+$/g,'');
  }

  submit() {
    if (this.submitting()) return;
    const name = this.name().trim();
    const slug = (this.slug().trim() || this.slugify(name));
    if (!name || !slug) return;

    const meta = {
      salesChannels: this.salesChannels(),
      businessType: this.businessType(),
      otherPlatforms: this.otherPlatforms(),
      planToSell: this.planToSell(),
    } as any;

    const body: any = { name, slug, description: '', meta };
    this.submitting.set(true);
    this.http.post(`${this.api}/v1/shops`, body, { withCredentials: true }).subscribe({
      next: () => {
        this.submitting.set(false);
        this.created.emit({ name, slug });
        if (this.redirectTo) this.router.navigateByUrl(this.redirectTo);
      },
      error: () => { this.submitting.set(false); }
    });
  }
}

