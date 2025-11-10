import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ProductCardComponent } from '../app/shared/components/product-card/product-card.component';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { QueryParamsService } from '../app/core/services/query-params.service';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-shop-page',
  imports: [CommonModule, FormsModule, RouterLink, ProductCardComponent, FiltersRailComponent, PaginationComponent],
  templateUrl: './shop-page.component.html',
  styleUrls: ['./shop-page.component.css']
})
export class ShopPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly http = inject(HttpClient);
  private readonly qps = inject(QueryParamsService);
  readonly api = environment.apiBase;
  shop = signal<ShopDetail | null>(null);
  products = signal<ProductSummary[]>([]);
  page = signal<number>(1);
  perPage = signal<number>(24);
  total = signal<number|null>(null);
  // filters via query params
  showFilters = signal<boolean>(false);
  q = signal<string>('');
  minPrice = signal<string>('');
  maxPrice = signal<string>('');
  sort = signal<string>('');
  ngOnInit(): void {
    this.qps.normalizeListing(this.router, this.route);
    const slug = this.route.snapshot.paramMap.get('slug')!;
    this.http.get<ApiResponse<ShopDetail>>(`${this.api}/v1/shops/${slug}`).subscribe(r => this.shop.set(r.data ?? null));
    this.route.queryParamMap.subscribe(qp => {
      this.q.set(qp.get('q') || '');
      this.minPrice.set(qp.get('minPrice') || '');
      this.maxPrice.set(qp.get('maxPrice') || '');
      this.sort.set(qp.get('sort') || '');
      this.page.set(parseInt(qp.get('page') || '1', 10));
      this.perPage.set(parseInt(qp.get('limit') || '24', 10));
      this.showFilters.set((qp.get('filters') || '') === '1');
      this.fetchProducts();
    });
  }
  fetchProducts(){
    const slug = this.route.snapshot.paramMap.get('slug')!;
    const params = new URLSearchParams();
    if (this.q()) params.set('q', this.q());
    if (this.minPrice()) params.set('minPrice', this.minPrice());
    if (this.maxPrice()) params.set('maxPrice', this.maxPrice());
    if (this.sort()) params.set('sort', this.sort());
    params.set('page', String(this.page()));
    params.set('limit', String(this.perPage()));
    const query = params.toString();
    this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/shops/${slug}/products${query ? '?' + query : ''}`, { withCredentials: true }).subscribe(r => {
      this.products.set(r?.data ?? []);
      this.total.set(typeof r?.total === 'number' ? r.total : null);
    });
  }
  applyFilters(){ this.qps.merge(this.router, this.route, { q: this.q() || null, minPrice: this.minPrice() || null, maxPrice: this.maxPrice() || null, sort: this.sort() || null, page: 1, limit: this.perPage() || null, filters: null }); this.showFilters.set(false); }
  clearAllFilters(){ this.qps.merge(this.router, this.route, { q: null, minPrice: null, maxPrice: null, sort: null, filters: null }); this.showFilters.set(false); }
  prevPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: Math.max(1, this.page()-1) }, queryParamsHandling: 'merge' }); }
  nextPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: this.page()+1, limit: this.perPage() || null }, queryParamsHandling: 'merge' }); }
  hasNext(){ const t = this.total(); return t!==null ? (this.page()*this.perPage() < t) : (this.products().length===this.perPage()); }
  handleFilterToggle(open: boolean){
    this.showFilters.set(open);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: open ? '1' : null }, queryParamsHandling: 'merge' });
  }
}

export interface ShopDetail {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

