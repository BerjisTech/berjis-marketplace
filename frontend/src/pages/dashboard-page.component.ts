import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { DarkModeToggleComponent } from "src/app/components/dark-mode-toggle/dark-mode-toggle.component";

@Component({
  standalone: true,
  selector: 'dashboard-page',
  imports: [CommonModule, FormsModule, RouterLink, DarkModeToggleComponent],
  templateUrl: './dashboard-page.component.html',
  styleUrls: ['./dashboard-page.component.css']
})
export class DashboardPageComponent implements OnInit {
  api = environment.apiBase;
  shops = signal<any[]>([]);
  selectedShopSlug = signal<string>('');
  products = signal<any[]>([]);
  // layout
  navCollapsed = signal<boolean>(false);
  toolsOpen = signal<boolean>(false);
  userMenuOpen = signal<boolean>(false);
  // search
  searchQ = signal<string>('');
  searching = signal<boolean>(false);
  searchResults = signal<{products:any[];orders:any[];customers:any[];marketing:any[];discounts:any[];content:any[];markets:any[];analytics:any[]}|null>(null);
  private searchTimer: any;

  // computed widths for grid columns
  get navW() { return this.navCollapsed() ? 60 : 220; }
  get toolsW() { return this.toolsOpen() ? 300 : 0; }
  // forms
  newShop = { name: '', slug: '', description: '' };
  newProduct: any = { title: '', slug: '', summary: '', priceCents: 0, currency: 'USD', stock: 0, published: true, category: '' };
  uploadBusy = signal(false);

  constructor(private http: HttpClient) {}
  ngOnInit(): void { this.loadShops(); }

  loadShops(){ this.http.get<any>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => { this.shops.set(r.data||[]); if (this.shops().length && !this.selectedShopSlug()) { this.selectedShopSlug.set(this.shops()[0].slug); this.loadProducts(); } }); }
  loadProducts(){ const slug = this.selectedShopSlug(); if(!slug) return; this.http.get<any>(`${this.api}/v1/my/shops/${slug}/products`, { withCredentials: true }).subscribe(r => this.products.set(r.data||[])); }
  onShopChange(ev: Event){ const value = (ev.target as HTMLSelectElement).value; this.selectedShopSlug.set(value); this.loadProducts(); }
  onSearchChange(v: string){
    this.searchQ.set(v);
    if (this.searchTimer) clearTimeout(this.searchTimer);
    if (!v || v.trim().length < 2) { this.searchResults.set(null); return; }
    this.searchTimer = setTimeout(() => this.runSearch(), 300);
  }
  private runSearch(){
    const q = this.searchQ().trim(); if (!q) { this.searchResults.set(null); return; }
    this.searching.set(true);
    this.http.get<any>(`${this.api}/v1/search?q=${encodeURIComponent(q)}`, { withCredentials: true }).subscribe({
      next: (r) => { this.searchResults.set(r?.data || null); this.searching.set(false); },
      error: () => { this.searchResults.set(null); this.searching.set(false); }
    });
  }
  
  createShop(){
    const b = this.newShop; if(!b.name || !b.slug) return;
    this.http.post<any>(`${this.api}/v1/shops`, b, { withCredentials: true }).subscribe(()=>{ this.newShop = { name:'', slug:'', description:'' }; this.loadShops(); });
  }
  async onFile(ev: Event){
    const input = ev.target as HTMLInputElement; const file = input.files?.[0]; if(!file) return;
    const fd = new FormData(); fd.append('file', file);
    this.uploadBusy.set(true);
    try {
      const res: any = await this.http.post(`${this.api}/v1/uploads`, fd, { withCredentials: true }).toPromise();
      this.newProduct.imageUrl = (res as any)?.data?.url || '';
    } finally { this.uploadBusy.set(false); }
  }
  createProduct(){
    const slug = this.selectedShopSlug(); if(!slug) return; const b = { ...this.newProduct, shopSlug: slug };
    this.http.post(`${this.api}/v1/products`, b, { withCredentials: true }).subscribe(()=>{ this.newProduct = { title:'', slug:'', summary:'', priceCents:0, currency:'USD', stock:0, published:true, category:'' }; this.loadProducts(); });
  }
  togglePublish(p: any){
    this.http.patch(`${this.api}/v1/products/${p.uuid}`, { published: !p.published }, { withCredentials: true }).subscribe(()=>{ this.loadProducts(); });
  }
  deleteProduct(p: any){
    this.http.delete(`${this.api}/v1/products/${p.uuid}`, { withCredentials: true }).subscribe(()=> this.loadProducts());
  }
  openAI(){ this.toolsOpen.set(true); }
}
