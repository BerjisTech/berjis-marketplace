import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { QueryParamsService } from '../app/core/services/query-params.service';
import { environment } from '../environments/environment';
import { ApiResponse, ProductSummary } from '../app/core/services/product.service';

@Component({
  standalone: true,
  selector: 'app-category-page',
  imports: [CommonModule, FormsModule, RouterLink, FiltersRailComponent, PaginationComponent],
  templateUrl: './category-page.component.html',
  styleUrls: ['./category-page.component.css']
})
export class CategoryPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly http = inject(HttpClient);
  private readonly qps = inject(QueryParamsService);

  readonly api = environment.apiBase;
  category = signal<string>('');
  products = signal<ProductSummary[]>([]);
  loading = signal(true);
  page = signal<number>(1);
  perPage = signal<number>(24);
  total = signal<number|null>(null);
  // filters
  showFilters = signal<boolean>(false);
  q = signal<string>('');
  minPrice = signal<string>('');
  maxPrice = signal<string>('');
  sort = signal<string>('');

  ngOnInit(): void {
    this.qps.normalizeListing(this.router, this.route);
    const id = this.route.snapshot.paramMap.get('id') || '';
    this.category.set(id);
    this.route.queryParamMap.subscribe(qp => {
      this.q.set(qp.get('q') || '');
      this.minPrice.set(qp.get('minPrice') || '');
      this.maxPrice.set(qp.get('maxPrice') || '');
      this.sort.set(qp.get('sort') || '');
      this.page.set(parseInt(qp.get('page') || '1', 10));
      this.perPage.set(parseInt(qp.get('limit') || '24', 10));
      this.showFilters.set((qp.get('filters') || '') === '1');
      this.fetch();
    });
  }
  fetch(): void {
    this.loading.set(true);
    const params = new URLSearchParams({ category: this.category() });
    if (this.q()) params.set('q', this.q());
    if (this.minPrice()) params.set('minPrice', this.minPrice());
    if (this.maxPrice()) params.set('maxPrice', this.maxPrice());
    if (this.sort()) params.set('sort', this.sort());
    params.set('page', String(this.page()));
    params.set('limit', String(this.perPage()));

    this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/products?${params.toString()}`, { withCredentials: true })
      .subscribe({
        next: response => {
          this.products.set(response?.data ?? []);
          this.total.set(typeof response?.total === 'number' ? response.total : null);
          this.loading.set(false);
        },
        error: () => this.loading.set(false)
      });
  }
  applyFilters(){ this.qps.merge(this.router, this.route, { q: this.q() || null, minPrice: this.minPrice() || null, maxPrice: this.maxPrice() || null, sort: this.sort() || null, page: 1, limit: this.perPage() || null, filters: null }); this.showFilters.set(false); }
  clearAllFilters(){ this.qps.merge(this.router, this.route, { q: null, minPrice: null, maxPrice: null, sort: null, filters: null }); this.showFilters.set(false); }
  prevPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: Math.max(1, this.page()-1) }, queryParamsHandling: 'merge' }); }
  nextPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: this.page()+1, limit: this.perPage() || null }, queryParamsHandling: 'merge' }); }
  onOpenChange(v: boolean){ this.showFilters.set(v); this.qps.toggleFilters(this.router, this.route, v); }
  hasNext(){ const t = this.total(); return t!=null ? (this.page()*this.perPage() < t) : (this.products().length===this.perPage()); }
}

