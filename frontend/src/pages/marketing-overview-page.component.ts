import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'marketing-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './marketing-overview-page.component.html',
  styleUrls: ['./marketing-overview-page.component.css']
})
export class MarketingOverviewPageComponent {}

