import { HttpClient } from '@angular/common/http';
import { inject } from '@angular/core';
import { CanActivateFn, Router, UrlTree } from '@angular/router';
import { Observable, of } from 'rxjs';
import { catchError, map } from 'rxjs/operators';
import { environment } from '../../../environments/environment';
import { ApiResponse } from '../services/product.service';
import { ShopStateService, ShopSummary } from '../services/shop-state.service';

export const hasShopGuard: CanActivateFn = (): Observable<boolean | UrlTree> => {
  const http = inject(HttpClient);
  const router = inject(Router);
  const shopState = inject(ShopStateService);
  return http
    .get<ApiResponse<ShopSummary[]>>(`${environment.apiBase}/v1/my/shops`, { withCredentials: true })
    .pipe(
      map((response) => {
        const shops = response?.data ?? [];
        shopState.applyShops(shops);
        return shops.length > 0 ? true : router.parseUrl('/get-started');
      }),
      catchError(() => of(shopState.hasStoredShop() ? true : router.parseUrl('/get-started'))),
    );
};
