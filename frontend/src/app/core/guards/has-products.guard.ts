import { HttpClient } from '@angular/common/http';
import { inject } from '@angular/core';
import { CanActivateFn, Router, UrlTree } from '@angular/router';
import { forkJoin, Observable, of } from 'rxjs';
import { catchError, map, switchMap } from 'rxjs/operators';
import { environment } from '../../../environments/environment';
import { ApiResponse, ProductSummary } from '../services/product.service';
import { ShopStateService, ShopSummary } from '../services/shop-state.service';

const buildProductCheck = (http: HttpClient, slug: string): Observable<boolean> => {
  const limitParam = '?limit=1';
  return http
    .get<ApiResponse<ProductSummary[]>>(
      `${environment.apiBase}/v1/my/shops/${encodeURIComponent(slug)}/products${limitParam}`,
      { withCredentials: true },
    )
    .pipe(
      map((response) => (response?.data?.length ?? 0) > 0),
      catchError(() => of(false)),
    );
};

export const hasProductsGuard: CanActivateFn = (): Observable<boolean | UrlTree> => {
  const http = inject(HttpClient);
  const router = inject(Router);
  const shopState = inject(ShopStateService);
  return http
    .get<ApiResponse<ShopSummary[]>>(`${environment.apiBase}/v1/my/shops`, { withCredentials: true })
    .pipe(
      switchMap((response) => {
        const shops = response?.data ?? [];
        shopState.applyShops(shops);
        if (!shops.length) {
          return of(router.parseUrl('/get-started'));
        }
        const requests = shops.map((shop) => buildProductCheck(http, shop.slug));
        return forkJoin(requests).pipe(
          map((results) => (results.some(Boolean) ? true : router.parseUrl('/dashboard/products/add'))),
        );
      }),
      catchError(() => of(router.parseUrl('/dashboard/products/add'))),
    );
};
