import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';

@Component({
  standalone: true,
  selector: 'category-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './category-page.component.html',
  styleUrls: ['./category-page.component.css']
})
export class CategoryPageComponent implements OnInit {
  api = environment.apiBase;
  category = signal<string>('');
  products = signal<any[]>([]);
  loading = signal(true);
  constructor(private route: ActivatedRoute, private http: HttpClient) {}
  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id') || '';
    this.category.set(id);
    this.fetch();
  }
  fetch(){
    this.loading.set(true);
    this.http.get<any>(`${this.api}/v1/products?category=${encodeURIComponent(this.category())}`).subscribe(r => {
      this.products.set(r.data||[]);
      this.loading.set(false);
    }, _ => this.loading.set(false));
  }
}

