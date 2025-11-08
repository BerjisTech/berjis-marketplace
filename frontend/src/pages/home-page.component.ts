import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ProductCardComponent } from '../app/shared/components/product-card/product-card.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { HttpClient } from '@angular/common/http';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { environment } from '../environments/environment';

type Product = { uuid: string; title: string; priceCents: number; currency: string; imageUrl?: string; shopName: string; shopSlug: string };
type Shop = { uuid: string; name: string; slug: string; description: string };

@Component({
  standalone: true,
  selector: 'home-page',
  imports: [CommonModule, RouterLink, FormsModule, ProductCardComponent, FiltersRailComponent, PaginationComponent],
  templateUrl: './home-page.component.html',
  styleUrls: ['./home-page.component.css']
})
export class HomePageComponent implements OnInit {
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
  constructor(private http: HttpClient, public route: ActivatedRoute, public router: Router) {}
  ngOnInit(): void {
    this.route.queryParamMap.subscribe(qp => {
      this.q.set(qp.get('q') || '');
      this.selectedCategory.set(qp.get('category') || '');
      this.minPrice.set(qp.get('minPrice') || '');
      this.maxPrice.set(qp.get('maxPrice') || '');
      this.sort.set(qp.get('sort') || '');
      this.page.set(parseInt(qp.get('page') || '1', 10));
      this.perPage.set(parseInt(qp.get('limit') || '24', 10));
      this.showFilters.set((qp.get('filters') || '') === '1');
      this.fetchAll();
    });
    this.fetchCategories();
  }
  prevPage(){ this.page.set(Math.max(1, this.page()-1)); this.fetchAll(); }
  nextPage(){ this.page.set(this.page()+1); this.fetchAll(); }
  applyFilters(){
    this.router.navigate([], { relativeTo: this.route, queryParams: {
      q: this.q() || null,
      category: this.selectedCategory() || null,
      minPrice: this.minPrice() || null,
      maxPrice: this.maxPrice() || null,
      sort: this.sort() || null,
      page: 1,
      limit: this.perPage() || null,
      filters: null
    }, queryParamsHandling: 'merge' });
    this.showFilters.set(false);
  }
  clearAllFilters(){
    this.router.navigate([], { relativeTo: this.route, queryParams: { q: null, minPrice: null, maxPrice: null, category: null, page: 1, filters: null }, queryParamsHandling: 'merge' });
  }
  closeFilters(){
    this.showFilters.set(false);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: null }, queryParamsHandling: 'merge' });
  }
  onOpenChange(v: boolean){
    this.showFilters.set(v);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: v ? '1' : null }, queryParamsHandling: 'merge' });
  }
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
