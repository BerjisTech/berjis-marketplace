import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'dashboard-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './dashboard-overview-page.component.html',
  styleUrls: ['./dashboard-overview-page.component.css']
})
export class DashboardOverviewPageComponent implements OnInit {
  private api = environment.apiBase;
  shops = signal<any[]>([]);
  selectedShop: any = null;
  products = signal<any[]>([]);

  constructor(private http: HttpClient) {}

  ngOnInit(): void {
    this.load();
  }

  private load(){
    this.http.get<any>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => {
      const arr = r?.data || []; this.shops.set(arr);
      this.selectedShop = arr[0] || null;
      if (this.selectedShop) this.loadProducts(this.selectedShop.slug);
    });
  }

  private loadProducts(slug: string){
    this.http.get<any>(`${this.api}/v1/my/shops/${slug}/products`, { withCredentials: true }).subscribe(r => this.products.set(r?.data || []));
  }

  isGenericShopName(name?: string): boolean {
    if (!name) return true;
    const n = name.trim().toLowerCase();
    return /(my\s+shop|new\s+shop|untitled|demo\s+shop|store|shop\s*\d+)/.test(n);
  }
}

