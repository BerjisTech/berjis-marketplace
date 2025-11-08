import { Injectable } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

export type ListingParams = {
  q?: string | null;
  category?: string | null;
  minPrice?: string | null;
  maxPrice?: string | null;
  sort?: string | null;
  page?: number | null;
  limit?: number | null;
  filters?: '1' | null;
};

@Injectable({ providedIn: 'root' })
export class QueryParamsService {
  read(route: ActivatedRoute): ListingParams {
    const qp = route.snapshot.queryParamMap;
    return {
      q: qp.get('q'),
      category: qp.get('category'),
      minPrice: qp.get('minPrice'),
      maxPrice: qp.get('maxPrice'),
      sort: qp.get('sort'),
      page: qp.has('page') ? parseInt(qp.get('page') || '1', 10) : null,
      limit: qp.has('limit') ? parseInt(qp.get('limit') || '24', 10) : null,
      filters: qp.get('filters') as any
    };
  }

  merge(router: Router, route: ActivatedRoute, params: ListingParams){
    return router.navigate([], { relativeTo: route, queryParams: params, queryParamsHandling: 'merge' });
  }

  clear(router: Router, route: ActivatedRoute){
    return this.merge(router, route, { q: null, minPrice: null, maxPrice: null, category: null, sort: null, page: 1, limit: null, filters: null });
  }

  toggleFilters(router: Router, route: ActivatedRoute, open: boolean){
    return this.merge(router, route, { filters: open ? '1' : null });
  }

  ensureNumber(n: any, fallback: number, min?: number, max?: number){
    let v = parseInt(n, 10); if (isNaN(v)) v = fallback; if (typeof min==='number') v = Math.max(min, v); if (typeof max==='number') v = Math.min(max, v); return v;
  }

  normalizeListing(router: Router, route: ActivatedRoute){
    const qp = route.snapshot.queryParamMap;
    const page = this.ensureNumber(qp.get('page'), 1, 1);
    const limit = this.ensureNumber(qp.get('limit'), 24, 1, 100);
    if (String(page) !== (qp.get('page')||'') || String(limit)!==(qp.get('limit')||'')) {
      return this.merge(router, route, { page, limit });
    }
    return Promise.resolve(true);
  }
}
