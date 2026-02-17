import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  standalone: true,
  selector: 'app-star-rating',
  imports: [CommonModule],
  template: `
    <div class="flex items-center gap-0.5" [attr.aria-label]="'Rating: ' + rating + ' out of 5'">
      @for (star of stars; track star) {
        <button
          *ngIf="interactive; else staticStar"
          type="button"
          (click)="onRate(star)"
          (mouseenter)="hovered = star"
          (mouseleave)="hovered = 0"
          class="text-lg leading-none cursor-pointer transition-colors"
          [class.text-yellow-400]="star <= (hovered || rating)"
          [class.text-slate-300]="star > (hovered || rating)">
          &#9733;
        </button>
        <ng-template #staticStar>
          <span
            class="text-lg leading-none"
            [class.text-yellow-400]="star <= rating"
            [class.text-slate-300]="star > rating">
            &#9733;
          </span>
        </ng-template>
      }
      <span *ngIf="showCount" class="ml-1 text-sm text-slate-500">({{ count }})</span>
    </div>
  `,
})
export class StarRatingComponent {
  @Input() rating = 0;
  @Input() count = 0;
  @Input() showCount = false;
  @Input() interactive = false;
  @Output() rated = new EventEmitter<number>();

  stars = [1, 2, 3, 4, 5];
  hovered = 0;

  onRate(star: number) {
    if (this.interactive) {
      this.rating = star;
      this.rated.emit(star);
    }
  }
}
