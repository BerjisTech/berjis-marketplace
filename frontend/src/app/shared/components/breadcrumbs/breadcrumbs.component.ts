import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

export interface BreadcrumbItem {
  label: string;
  url?: string;
}

@Component({
  standalone: true,
  selector: 'app-breadcrumbs',
  imports: [CommonModule, RouterLink],
  template: `
    <nav aria-label="Breadcrumb" class="text-sm text-slate-500 dark:text-slate-400 mb-4">
      <ol class="flex items-center gap-1 flex-wrap">
        @for (item of items; track item.label; let last = $last) {
          <li class="flex items-center gap-1">
            @if (item.url && !last) {
              <a [routerLink]="item.url" class="hover:text-blue-600 dark:hover:text-blue-400 transition">{{ item.label }}</a>
            } @else {
              <span [class.text-slate-700]="last" [class.dark:text-white]="last" [class.font-medium]="last">{{ item.label }}</span>
            }
            @if (!last) {
              <span class="text-slate-400">/</span>
            }
          </li>
        }
      </ol>
    </nav>
  `,
})
export class BreadcrumbsComponent {
  @Input() items: BreadcrumbItem[] = [];
}
