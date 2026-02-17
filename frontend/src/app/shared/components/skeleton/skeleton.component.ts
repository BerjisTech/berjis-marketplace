import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  standalone: true,
  selector: 'app-skeleton',
  imports: [CommonModule],
  template: `
    @switch (type) {
      @case ('card') {
        <div class="rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden animate-pulse">
          <div class="aspect-square bg-slate-200 dark:bg-slate-800"></div>
          <div class="p-4 space-y-2">
            <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded w-3/4"></div>
            <div class="h-3 bg-slate-200 dark:bg-slate-800 rounded w-1/2"></div>
            <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded w-1/3"></div>
          </div>
        </div>
      }
      @case ('line') {
        <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded animate-pulse" [style.width]="width"></div>
      }
      @case ('text') {
        <div class="space-y-2 animate-pulse">
          <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded w-full"></div>
          <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded w-5/6"></div>
          <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded w-4/6"></div>
        </div>
      }
      @case ('image') {
        <div class="aspect-square bg-slate-200 dark:bg-slate-800 rounded-lg animate-pulse"></div>
      }
      @default {
        <div class="h-4 bg-slate-200 dark:bg-slate-800 rounded animate-pulse" [style.width]="width" [style.height]="height"></div>
      }
    }
  `,
})
export class SkeletonComponent {
  @Input() type: 'card' | 'line' | 'text' | 'image' | 'custom' = 'line';
  @Input() width = '100%';
  @Input() height = '1rem';
}
