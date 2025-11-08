import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'customers-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './customers-overview-page.component.html',
  styleUrls: ['./customers-overview-page.component.css']
})
export class CustomersOverviewPageComponent {}

