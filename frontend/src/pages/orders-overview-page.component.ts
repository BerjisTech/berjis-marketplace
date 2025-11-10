import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'app-orders-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './orders-overview-page.component.html',
  styleUrls: ['./orders-overview-page.component.css']
})
export class OrdersOverviewPageComponent {}


