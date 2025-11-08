import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { FiltersRailComponent } from '../app/shared/components/filters-rail/filters-rail.component';
import { PaginationComponent } from '../app/shared/components/pagination/pagination.component';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'category-page',
  imports: [CommonModule, FormsModule, RouterLink, FiltersRailComponent, PaginationComponent],
  templateUrl: './category-page.component.html',
  styleUrls: ['./category-page.component.css']
})
export class CategoryPageComponent implements OnInit {
  api = environment.apiBase;
  category = signal<string>('');
  products = signal<any[]>([]);
  loading = signal(true);
  page = signal<number>(1);
  perPage = signal<number>(24);
  // filters
  showFilters = signal<boolean>(false);
  q = signal<string>('');
  minPrice = signal<string>('');
  maxPrice = signal<string>('');
  sort = signal<string>('');
  constructor(public route: ActivatedRoute, public router: Router, private http: HttpClient) {}
  ngOnInit(): void {
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
  fetch(){
    this.loading.set(true);
    const params: any = { category: this.category() };
    if (this.q()) params.q = this.q();
    if (this.minPrice()) params.minPrice = this.minPrice();
    if (this.maxPrice()) params.maxPrice = this.maxPrice();
    if (this.sort()) params.sort = this.sort();
    if (this.page()) params.page = this.page();
    if (this.perPage()) params.limit = this.perPage();
    const qs = new URLSearchParams(params).toString();
    this.http.get<any>(`${this.api}/v1/products?${qs}`).subscribe(r => {
      this.products.set(r.data||[]);
      this.loading.set(false);
    }, _ => this.loading.set(false));
  }
  applyFilters(){
    this.router.navigate([], { relativeTo: this.route, queryParams: {
      q: this.q() || null,
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
    this.router.navigate([], { relativeTo: this.route, queryParams: { q: null, minPrice: null, maxPrice: null, sort: null, filters: null }, queryParamsHandling: 'merge' });
    this.showFilters.set(false);
  }
  prevPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: Math.max(1, this.page()-1) }, queryParamsHandling: 'merge' }); }
  nextPage(){ this.router.navigate([], { relativeTo: this.route, queryParams: { page: this.page()+1, limit: this.perPage() || null }, queryParamsHandling: 'merge' }); }
  onOpenChange(v: boolean){
    this.showFilters.set(v);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: v ? '1' : null }, queryParamsHandling: 'merge' });
  }
}
