import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'markets-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './markets-overview-page.component.html',
  styleUrls: ['./markets-overview-page.component.css']
})
export class MarketsOverviewPageComponent {}

