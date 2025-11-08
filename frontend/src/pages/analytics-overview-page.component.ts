import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'analytics-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './analytics-overview-page.component.html',
  styleUrls: ['./analytics-overview-page.component.css']
})
export class AnalyticsOverviewPageComponent {}

