import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { ProductCardComponent } from '../app/shared/components/product-card/product-card.component';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { QueryParamsService } from '../app/core/services/query-params.service';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'shop-page',
  imports: [CommonModule, FormsModule, RouterLink, ProductCardComponent, FiltersRailComponent, PaginationComponent],
  templateUrl: './shop-page.component.html',
  styleUrls: ['./shop-page.component.css']
})
export class ShopPageComponent implements OnInit {
  api = environment.apiBase;
  shop = signal<any>(null);
  products = signal<any[]>([]);
  page = signal<number>(1);
  perPage = signal<number>(24);
  total = signal<number|null>(null);
  // filters via query params
  showFilters = signal<boolean>(false);
  q = signal<string>('');
  minPrice = signal<string>('');
  maxPrice = signal<string>('');
  sort = signal<string>('');
  constructor(public route: ActivatedRoute, public router: Router, private http: HttpClient, private qps: QueryParamsService) {}
  ngOnInit(): void {
    this.qps.normalizeListing(this.router, this.route);
    const slug = this.route.snapshot.paramMap.get('slug')!;
    this.http.get<any>(`${this.api}/v1/shops/${slug}`).subscribe(r => this.shop.set(r.data));
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
    const params: any = {};
    if (this.q()) params.q = this.q();
    if (this.minPrice()) params.minPrice = this.minPrice();
    if (this.maxPrice()) params.maxPrice = this.maxPrice();
    if (this.sort()) params.sort = this.sort();
    if (this.page()) params.page = this.page();
    if (this.perPage()) params.limit = this.perPage();
    const qs = new URLSearchParams(params).toString();
    this.http.get<any>(`${this.api}/v1/shops/${slug}/products${qs ? '?' + qs : ''}`).subscribe(r => { this.products.set(r.data||[]); this.total.set(typeof r?.total === 'number' ? r.total : null); });
  }
  applyFilters(){ this.qps.merge(this.router, this.route, { q: this.q() || null, minPrice: this.minPrice() || null, maxPrice: this.maxPrice() || null, sort: this.sort() || null, page: 1, limit: this.perPage() || null, filters: null }); this.showFilters.set(false); }
  clearAllFilters(){ this.qps.merge(this.router, this.route, { q: null, minPrice: null, maxPrice: null, sort: null, filters: null }); this.showFilters.set(false); }
  prevPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: Math.max(1, this.page()-1) }, queryParamsHandling: 'merge' }); }
  nextPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: this.page()+1, limit: this.perPage() || null }, queryParamsHandling: 'merge' }); }
  hasNext(){ const t = this.total(); return t!=null ? (this.page()*this.perPage() < t) : (this.products().length===this.perPage()); }
}
