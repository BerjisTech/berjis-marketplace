import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'products-overview-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-overview-page.component.html',
  styleUrls: ['./products-overview-page.component.css']
})
export class ProductsOverviewPageComponent implements OnInit {
  api = environment.apiBase;
  shops = signal<any[]>([]);
  shopSlug = signal<string>('');
  items = signal<any[]>([]);
  loading = signal<boolean>(false);
  q = signal<string>('');

  constructor(private http: HttpClient) {}
  ngOnInit(): void {
    this.http.get<any>(`${this.api}/v1/my/shops`, { withCredentials: true }).subscribe(r => {
      const arr = r?.data || []; this.shops.set(arr); if (arr.length) { this.shopSlug.set(arr[0].slug); this.load(); }
    });
  }
  load(){
    if (!this.shopSlug()) return; this.loading.set(true);
    const qs = this.q() ? `?q=${encodeURIComponent(this.q())}` : '';
    this.http.get<any>(`${this.api}/v1/my/shops/${this.shopSlug()}/products${qs}`, { withCredentials: true })
      .subscribe({ next: r => { this.items.set(r?.data || []); this.loading.set(false); }, error: () => this.loading.set(false) });
  }
}
