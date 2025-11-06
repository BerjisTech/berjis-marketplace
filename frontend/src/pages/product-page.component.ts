import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'product-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './product-page.component.html',
  styleUrls: ['./product-page.component.css']
})
export class ProductPageComponent implements OnInit {
  api = environment.apiBase;
  product = signal<any>(null);
  loading = signal(true);

  constructor(private route: ActivatedRoute, private http: HttpClient) {}
  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id')!;
    this.http.get<any>(`${this.api}/v1/products/${id}`).subscribe(r => { this.product.set(r.data); this.loading.set(false); });
  }
}
