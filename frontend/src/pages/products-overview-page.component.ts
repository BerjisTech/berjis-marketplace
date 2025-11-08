import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'products-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './products-overview-page.component.html',
  styleUrls: ['./products-overview-page.component.css']
})
export class ProductsOverviewPageComponent {}

