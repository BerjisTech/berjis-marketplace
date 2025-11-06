import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'shop-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './shop-page.component.html',
  styleUrls: ['./shop-page.component.css']
})
export class ShopPageComponent implements OnInit {
  api = environment.apiBase;
  shop = signal<any>(null);
  products = signal<any[]>([]);
  constructor(private route: ActivatedRoute, private http: HttpClient) {}
  ngOnInit(): void {
    const slug = this.route.snapshot.paramMap.get('slug')!;
    this.http.get<any>(`${this.api}/v1/shops/${slug}`).subscribe(r => this.shop.set(r.data));
    this.http.get<any>(`${this.api}/v1/shops/${slug}/products`).subscribe(r => this.products.set(r.data));
  }
}
