import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'order-confirmation-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './order-confirmation-page.component.html',
  styleUrls: ['./order-confirmation-page.component.css']
})
export class OrderConfirmationPageComponent {
  id: string;
  constructor(route: ActivatedRoute){ this.id = route.snapshot.paramMap.get('id') || ''; }
}

