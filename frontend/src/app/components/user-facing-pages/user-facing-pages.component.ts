import { Component, OnInit, inject, signal } from '@angular/core';
import { DarkModeToggleComponent } from "../dark-mode-toggle/dark-mode-toggle.component";
import { ActivatedRoute, RouterLink, RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { QueryParamsService } from '../../core/services/query-params.service';
import { CartService } from '../../core/services/cart.service';
import { WishlistService } from '../../core/services/wishlist.service';
import { MiniCartComponent } from '../../shared/components/mini-cart/mini-cart.component';
import { ToastContainerComponent } from '../../shared/components/toast/toast-container.component';
import { SearchAutocompleteComponent } from '../../shared/components/search-autocomplete/search-autocomplete.component';

@Component({
  selector: 'app-user-facing-pages',
  standalone: true,
  imports: [RouterOutlet, RouterLink, DarkModeToggleComponent, CommonModule, FormsModule, MiniCartComponent, ToastContainerComponent, SearchAutocompleteComponent],
  templateUrl: './user-facing-pages.component.html',
  styleUrl: './user-facing-pages.component.css'
})
export class UserFacingPagesComponent implements OnInit {

  copyRightYear = new Date().getFullYear();
  sort = signal<string>('');
  q = signal<string>('');
  showFilters = signal<boolean>(false);
  showFilterControls = signal<boolean>(true);
  miniCartOpen = signal<boolean>(false);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly qps = inject(QueryParamsService);
  private readonly cart = inject(CartService);
  private readonly wishlist = inject(WishlistService);

  ngOnInit(): void {
    this.route.queryParamMap.subscribe(qp => {
      this.q.set(qp.get('q') || '');
      this.sort.set(qp.get('sort') || '');
      this.showFilters.set((qp.get('filters') || '') === '1');
    });
    // toggle visibility of Filters/Clear on non-listing routes
    const updateControls = () => {
      const url = this.router.url.split('?')[0];
      const isDetail = /^\/product\//.test(url) || /^\/checkout(\/|$)/.test(url) || /^\/wishlist(\/|$)/.test(url);
      this.showFilterControls.set(!isDetail);
    };
    updateControls();
    this.router.events.subscribe(ev => { if (ev instanceof NavigationEnd) updateControls(); });
  }
  applyQuery(){ this.qps.merge(this.router, this.route, { q: this.q() || null, sort: this.sort() || null }); }
  toggleFilters(){
    const next = !this.showFilters(); this.showFilters.set(next);
    this.qps.toggleFilters(this.router, this.route, next);
  }
  clearFilters(){
    this.qps.clear(this.router, this.route);
  }
  cartCount(){ return this.cart.count(); }
  wishlistCount(){ return this.wishlist.items().length; }
  toggleMiniCart(){ this.miniCartOpen.set(!this.miniCartOpen()); }

}
