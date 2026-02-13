import { Component, OnDestroy, OnInit, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RouterLink, RouterOutlet } from '@angular/router';
import { environment } from '../environments/environment';
import { DarkModeToggleComponent } from '../app/components/dark-mode-toggle/dark-mode-toggle.component';
import { ProfileService, UserProfile } from '../app/core/services/profile.service';
import { ApiResponse, CreateProductPayload, ProductSummary } from '../app/core/services/product.service';
import { ShopStateService, ShopSummary } from '../app/core/services/shop-state.service';

@Component({
  standalone: true,
  selector: 'app-dashboard-page',
  imports: [CommonModule, FormsModule, RouterLink, RouterOutlet, DarkModeToggleComponent],
  templateUrl: './dashboard-page.component.html',
  styleUrls: ['./dashboard-page.component.css']
})
export class DashboardPageComponent implements OnInit, OnDestroy {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;
  private readonly profileService = inject(ProfileService);
  private readonly shopState = inject(ShopStateService);

  readonly shops = this.shopState.shops;
  readonly activeShopSlug = this.shopState.activeShopSlug;
  products = signal<ProductSummary[]>([]);
  // layout
  navCollapsed = signal<boolean>(false);
  toolsOpen = signal<boolean>(false);
  userMenuOpen = signal<boolean>(false);
  // nav accordion: only one group open at a time
  openGroup = signal<null | 'orders' | 'products' | 'customers' | 'marketing' | 'discounts' | 'content' | 'markets' | 'analytics'>(null);
  // search
  searchQ = signal<string>('');
  searching = signal<boolean>(false);
  searchResults = signal<DashboardSearchResults | null>(null);
  private searchTimer: ReturnType<typeof setTimeout> | null = null;
  profile = signal<UserProfile | null>(null);
  profileLoading = signal<boolean>(false);

  // computed widths for grid columns
  get navW() { return this.navCollapsed() ? 60 : 220; }
  get toolsW() { return this.toolsOpen() ? 300 : 0; }
  // forms
  newShop = { name: '', slug: '', description: '' };
  newProduct: CreateProductPayload = { title: '', slug: '', summary: '', priceCents: 0, currency: 'USD', stock: 0, published: true, category: '' };
  uploadBusy = signal(false);
  private readonly syncActiveShop = effect(() => {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.products.set([]);
      return;
    }
    this.loadProducts(slug);
  });

  ngOnInit(): void {
    this.shopState.ensureLoaded();
    this.loadProfile();
  }
  ngOnDestroy(): void {
    if (this.searchTimer) {
      clearTimeout(this.searchTimer);
    }
    this.syncActiveShop.destroy();
  }

  loadShops(): void {
    this.http.get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(response => {
      const shops = response?.data ?? [];
      this.shopState.applyShops(shops);
      const active = this.shopState.activeShopSlug();
      if (active) {
        this.loadProducts(active);
      } else {
        this.products.set([]);
      }
    });
  }
  loadProducts(slug: string): void {
    if(!slug) {
      this.products.set([]);
      return;
    }
    this.http.get<ApiResponse<ProductSummary[]>>(`${this.api}/v1/my/shops/${slug}/products`, { withCredentials: true })
      .subscribe(r => this.products.set(r?.data ?? []));
  }
  onShopChange(ev: Event){
    const value = (ev.target as HTMLSelectElement).value;
    this.shopState.setActiveShopSlug(value);
  }
  onSearchChange(v: string){
    this.searchQ.set(v);
    if (this.searchTimer) clearTimeout(this.searchTimer);
    if (!v || v.trim().length < 2) { this.searchResults.set(null); return; }
    this.searchTimer = setTimeout(() => this.runSearch(), 300);
  }
  open(group: 'orders' | 'products' | 'customers' | 'marketing' | 'discounts' | 'content' | 'markets' | 'analytics'){
    this.openGroup.set(this.openGroup() === group ? null : group);
  }
  private runSearch(){
    const q = this.searchQ().trim(); if (!q) { this.searchResults.set(null); return; }
    this.searching.set(true);
    this.http.get<ApiResponse<DashboardSearchResults>>(`${this.api}/v1/search?q=${encodeURIComponent(q)}`, { withCredentials: true }).subscribe({
      next: (r) => { this.searchResults.set(r?.data ?? null); this.searching.set(false); },
      error: () => { this.searchResults.set(null); this.searching.set(false); }
    });
  }
  
  createShop(){
    const b = this.newShop; if(!b.name || !b.slug) return;
    const desiredSlug = (b.slug || '').trim().toLowerCase();
    this.http.post<unknown>(`${this.api}/v1/shops`, b, { withCredentials: true }).subscribe(()=>{
      this.newShop = { name:'', slug:'', description:'' };
      if (desiredSlug) {
        this.shopState.setActiveShopSlug(desiredSlug);
      }
      this.loadShops();
    });
  }
  async onFile(ev: Event){
    const input = ev.target as HTMLInputElement; const file = input.files?.[0]; if(!file) return;
    const fd = new FormData(); fd.append('file', file);
    this.uploadBusy.set(true);
    try {
      const res = await this.http.post<ApiResponse<{ url: string }>>(`${this.api}/v1/uploads`, fd, { withCredentials: true }).toPromise();
      this.newProduct = { ...this.newProduct, imageUrl: res?.data?.url ?? '' };
    } finally { this.uploadBusy.set(false); }
  }
  createProduct(){
    const slug = this.shopState.activeShopSlug(); if(!slug) return; const b = { ...this.newProduct, shopSlug: slug };
    this.http.post(`${this.api}/v1/products`, b, { withCredentials: true }).subscribe(()=>{
      this.newProduct = { title:'', slug:'', summary:'', priceCents:0, currency:'USD', stock:0, published:true, category:'' };
      this.loadProducts(slug);
    });
  }
  togglePublish(p: ProductSummary){
    const slug = this.shopState.activeShopSlug();
    this.http.patch(`${this.api}/v1/products/${p.uuid}`, { published: !p.published }, { withCredentials: true }).subscribe(()=>{
      if (slug) {
        this.loadProducts(slug);
      }
    });
  }
  deleteProduct(p: ProductSummary){
    const slug = this.shopState.activeShopSlug();
    this.http.delete(`${this.api}/v1/products/${p.uuid}`, { withCredentials: true }).subscribe(()=>{
      if (slug) {
        this.loadProducts(slug);
      }
    });
  }
  openAI(){ this.toolsOpen.set(true); }

  loadProfile(): void {
    this.profileLoading.set(true);
    this.profileService.getProfile().subscribe({
      next: (response) => {
        const data = response?.data;
        if (data?.profile) {
          this.profile.set(data.profile);
        }
        this.profileLoading.set(false);
      },
      error: () => {
        this.profileLoading.set(false);
      }
    });
  }

  profileName(): string {
    const profile = this.profile();
    if (!profile) {
      return '';
    }
    if (profile.displayName && profile.displayName.trim().length > 0) {
      return profile.displayName.trim();
    }
    return (profile.email ?? '').trim();
  }

  profileEmail(): string {
    return this.profile()?.email ?? '';
  }

  profileInitials(): string {
    const profile = this.profile();
    const fallback = profile?.displayName?.trim() || profile?.email?.trim() || '';
    if (!fallback) {
      return '?';
    }
    const parts = fallback.split(/[\s@._-]+/).filter(Boolean).slice(0, 2);
    if (!parts.length) {
      return '?';
    }
    return parts.map(part => part.charAt(0).toUpperCase()).join('').slice(0, 2);
  }

  toggleUserMenu(): void {
    this.userMenuOpen.set(!this.userMenuOpen());
  }
}

export interface DashboardSearchResults {
  products: ProductSummary[];
  orders: unknown[];
  customers: unknown[];
  marketing: unknown[];
  discounts: unknown[];
  content: unknown[];
  markets: unknown[];
  analytics: unknown[];
}
