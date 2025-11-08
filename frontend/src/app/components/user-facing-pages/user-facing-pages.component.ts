import { Component, signal } from '@angular/core';
import { DarkModeToggleComponent } from "../dark-mode-toggle/dark-mode-toggle.component";
import { RouterLink, RouterOutlet } from '@angular/router';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { ProductCardComponent } from 'src/app/shared/components/product-card/product-card.component';
import { environment } from 'src/environments/environment';

type Product = { uuid: string; title: string; priceCents: number; currency: string; imageUrl?: string; shopName: string; shopSlug: string };
type Shop = { uuid: string; name: string; slug: string; description: string };

@Component({
  selector: 'app-user-facing-pages',
  standalone: true,
  imports: [RouterOutlet, DarkModeToggleComponent, CommonModule, FormsModule],
  templateUrl: './user-facing-pages.component.html',
  styleUrl: './user-facing-pages.component.css'
})
export class UserFacingPagesComponent {

  copyRightYear = new Date().getFullYear();
  products = signal<Product[]>([]);
  shops = signal<Shop[]>([]);
  loading = signal(true);
  page = signal<number>(1);
  perPage = signal<number>(24);
  sort = signal<string>('');
  categories = signal<string[]>([]);
  selectedCategory = signal<string>('');
  minPrice = signal<string>('');
  maxPrice = signal<string>('');
  q = signal<string>('');
  showFilters = signal<boolean>(false);
  categorySamples = signal<Record<string, Product[]>>({});
  api = environment.apiBase;
  constructor(private http: HttpClient) {}
  ngOnInit(): void { this.fetchAll(); this.fetchCategories(); }
  prevPage(){ this.page.set(Math.max(1, this.page()-1)); this.fetchAll(); }
  nextPage(){ this.page.set(this.page()+1); this.fetchAll(); }

  fetchAll(){
    this.loading.set(true);
    const params: any = {};
    if (this.selectedCategory()) params.category = this.selectedCategory();
    if (this.minPrice()) params.minPrice = this.minPrice();
    if (this.maxPrice()) params.maxPrice = this.maxPrice();
    if (this.q()) params.q = this.q();
    if (this.page()) params.page = this.page();
    if (this.perPage()) params.limit = this.perPage();
    if (this.sort()) params.sort = this.sort();
    const qs = new URLSearchParams(params).toString();
    Promise.all([
      this.http.get<any>(`${this.api}/v1/products${qs ? '?' + qs : ''}`).toPromise(),
      this.http.get<any>(`${this.api}/v1/shops`).toPromise(),
    ]).then(([p,s])=>{ this.products.set(p?.data||[]); this.shops.set(s?.data||[]); })
      .finally(()=>this.loading.set(false));
  }
  fetchCategories(){
    this.http.get<any>(`${this.api}/v1/categories`).subscribe(async r => {
      const cats: string[] = r.data||[];
      this.categories.set(cats);
      // Fetch a small sample for each category (up to 6) for homepage sections
      const samples: Record<string, Product[]> = {};
      for (const c of cats.slice(0, 6)) { // cap sections to avoid over-fetching
        try {
          const res: any = await this.http.get(`${this.api}/v1/products?category=${encodeURIComponent(c)}`).toPromise();
          samples[c] = (res as any)?.data?.slice(0, 6) || [];
        } catch {}
      }
      this.categorySamples.set(samples);
    });
  }
  clearFilters(){ this.selectedCategory.set(''); this.minPrice.set(''); this.maxPrice.set(''); this.q.set(''); this.fetchAll(); }

}
