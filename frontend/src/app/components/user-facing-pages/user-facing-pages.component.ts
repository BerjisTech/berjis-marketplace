import { Component, signal } from '@angular/core';
import { DarkModeToggleComponent } from "../dark-mode-toggle/dark-mode-toggle.component";
import { ActivatedRoute, RouterLink, RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-user-facing-pages',
  standalone: true,
  imports: [RouterOutlet, DarkModeToggleComponent, CommonModule, FormsModule],
  templateUrl: './user-facing-pages.component.html',
  styleUrl: './user-facing-pages.component.css'
})
export class UserFacingPagesComponent {

  copyRightYear = new Date().getFullYear();
  sort = signal<string>('');
  q = signal<string>('');
  showFilters = signal<boolean>(false);
  showFilterControls = signal<boolean>(true);
  constructor(private router: Router, private route: ActivatedRoute, private _http: HttpClient) {}
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
  applyQuery(){
    this.router.navigate([], { relativeTo: this.route, queryParams: { q: this.q() || null, sort: this.sort() || null }, queryParamsHandling: 'merge' });
  }
  toggleFilters(){
    const next = !this.showFilters();
    this.showFilters.set(next);
    this.router.navigate([], { relativeTo: this.route, queryParams: { filters: next ? '1' : null }, queryParamsHandling: 'merge' });
  }
  clearFilters(){
    this.router.navigate([], { relativeTo: this.route, queryParams: { q: null, sort: null, minPrice: null, maxPrice: null, category: null, page: null, limit: null, filters: null }, queryParamsHandling: 'merge' });
  }

}
