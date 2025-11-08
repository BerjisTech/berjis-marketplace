import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'content-overview-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './content-overview-page.component.html',
  styleUrls: ['./content-overview-page.component.css']
})
export class ContentOverviewPageComponent {}

