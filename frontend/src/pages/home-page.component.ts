import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

type Product = { uuid: string; title: string; priceCents: number; currency: string; imageUrl?: string; shopName: string; shopSlug: string };
type Shop = { uuid: string; name: string; slug: string; description: string };

@Component({
  standalone: true,
  selector: 'home-page',
  imports: [CommonModule, RouterLink, FormsModule],
  templateUrl: './home-page.component.html',
  styleUrls: ['./home-page.component.css']
})
export class HomePageComponent implements OnInit {
  products = signal<Product[]>([]);
  shops = signal<Shop[]>([]);
  loading = signal(true);
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
  fetchAll(){
    this.loading.set(true);
    const params: any = {};
    if (this.selectedCategory()) params.category = this.selectedCategory();
    if (this.minPrice()) params.minPrice = this.minPrice();
    if (this.maxPrice()) params.maxPrice = this.maxPrice();
    if (this.q()) params.q = this.q();
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
