import { Component, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../environments/environment';
import { SkeletonComponent } from '../app/shared/components/skeleton/skeleton.component';

interface OrderItem {
  title: string;
  quantity: number;
  priceCents: number;
  currency: string;
  imageUrl?: string;
}

interface OrderData {
  uuid: string;
  status: string;
  subtotalCents: number;
  shippingCents: number;
  taxCents: number;
  totalCents: number;
  currency: string;
  items: OrderItem[];
  shippingAddress?: {
    name?: string;
    line1?: string;
    line2?: string;
    city?: string;
    state?: string;
    postalCode?: string;
    country?: string;
  };
  createdAt: string;
}

@Component({
  standalone: true,
  selector: 'app-order-confirmation-page',
  imports: [CommonModule, RouterLink, SkeletonComponent],
  templateUrl: './order-confirmation-page.component.html',
  styleUrls: ['./order-confirmation-page.component.css']
})
export class OrderConfirmationPageComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  id: string = this.route.snapshot.paramMap.get('id') || '';
  order = signal<OrderData | null>(null);
  loading = signal(true);

  ngOnInit(): void {
    if (this.id) {
      this.http.get<any>(`${this.api}/v1/my/orders/${this.id}`, { withCredentials: true }).subscribe({
        next: r => {
          this.order.set(r?.data ?? null);
          this.loading.set(false);
        },
        error: () => this.loading.set(false),
      });
    } else {
      this.loading.set(false);
    }
  }
}
