import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'app-order-confirmation-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './order-confirmation-page.component.html',
  styleUrls: ['./order-confirmation-page.component.css']
})
export class OrderConfirmationPageComponent {
  private readonly route = inject(ActivatedRoute);
  id: string = this.route.snapshot.paramMap.get('id') || '';
}


