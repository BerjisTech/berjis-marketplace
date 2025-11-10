import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { ProductCardComponent } from '../app/shared/components/product-card/product-card.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { HttpClient } from '@angular/common/http';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { QueryParamsService } from '../app/core/services/query-params.service';
import { environment } from '../environments/environment';
import { forkJoin, firstValueFrom } from 'rxjs';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-home-page',
  imports: [CommonModule, RouterLink, FormsModule, ProductCardComponent, FiltersRailComponent, PaginationComponent],
  templateUrl: './home-page.component.html',
  styleUrls: ['./home-page.component.css']
})
export class HomePageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly qps = inject(QueryParamsService);

  products = signal<ProductSummary[]>([]);
  shops = signal<ShopSummary[]>([]);
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
  categorySamples = signal<Record<string, ProductSummary[]>>({});
  total = signal<number|null>(null);
  readonly api = environment.apiBase;
  ngOnInit(): void {
    this.qps.normalizeListing(this.router, this.route);
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
  applyFilters(){ this.qps.merge(this.router, this.route, { q: this.q() || null, category: this.selectedCategory() || null, minPrice: this.minPrice() || null, maxPrice: this.maxPrice() || null, sort: this.sort() || null, page: 1, limit: this.perPage() || null, filters: null }); this.showFilters.set(false); }
  clearAllFilters(){
    this.router.navigate([], { relativeTo: this.route, queryParams: { q: null, minPrice: null, maxPrice: null, category: null, page: 1, filters: null }, queryParamsHandling: 'merge' });
  }
  closeFilters(){
    this.showFilters.set(false);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: null }, queryParamsHandling: 'merge' });
  }
  onOpenChange(v: boolean){ this.showFilters.set(v); this.qps.toggleFilters(this.router, this.route, v); }
  fetchAll(){
    this.loading.set(true);
    const params = new URLSearchParams();
    if (this.selectedCategory()) params.set('category', this.selectedCategory());
    if (this.minPrice()) params.set('minPrice', this.minPrice());
    if (this.maxPrice()) params.set('maxPrice', this.maxPrice());
    if (this.q()) params.set('q', this.q());
    params.set('page', String(this.page()));
    params.set('limit', String(this.perPage()));
    if (this.sort()) params.set('sort', this.sort());

    const products$ = this.http.get<ApiResponse<ProductSummary[]>>(
      `${this.api}/v1/products${params.toString() ? `?${params.toString()}` : ''}`,
      { withCredentials: true }
    );
    const shops$ = this.http.get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/shops`, { withCredentials: true });

    forkJoin([products$, shops$]).subscribe({
      next: ([productRes, shopRes]) => {
        this.products.set(productRes?.data ?? []);
        this.total.set(typeof productRes?.total === 'number' ? productRes.total : null);
        this.shops.set(shopRes?.data ?? []);
        this.loading.set(false);
      },
      error: () => this.loading.set(false)
    });
  }
  hasNext(){ const t = this.total(); return t!==null ? (this.page()*this.perPage() < t) : (this.products().length===this.perPage()); }
  fetchCategories(){
    this.http.get<ApiResponse<string[]>>(`${this.api}/v1/categories`).subscribe(async r => {
      const cats: string[] = r.data||[];
      this.categories.set(cats);
      // Fetch a small sample for each category (up to 6) for homepage sections
      const samples: Record<string, ProductSummary[]> = {};
      for (const c of cats.slice(0, 6)) { // cap sections to avoid over-fetching
        try {
          const res = await firstValueFrom(
            this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/products?category=${encodeURIComponent(c)}`, { withCredentials: true })
          );
          samples[c] = res?.data?.slice(0, 6) ?? [];
        } catch {
          continue;
        }
      }
      this.categorySamples.set(samples);
    });
  }
  clearFilters(){ this.selectedCategory.set(''); this.minPrice.set(''); this.maxPrice.set(''); this.q.set(''); this.fetchAll(); }
}

export interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description: string;
}

