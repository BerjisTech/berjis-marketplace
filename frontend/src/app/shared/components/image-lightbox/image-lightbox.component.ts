import { Component, Input, Output, EventEmitter, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  standalone: true,
  selector: 'app-image-lightbox',
  imports: [CommonModule],
  template: `
    @if (open) {
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm" (click)="close.emit()">
        <div class="relative max-w-4xl max-h-[90vh] mx-4" (click)="$event.stopPropagation()">
          <button type="button"
            class="absolute -top-10 right-0 text-white text-2xl hover:text-slate-300 transition"
            (click)="close.emit()" aria-label="Close">
            &#10005;
          </button>
          <img [src]="images[currentIndex()]" [alt]="alt"
            class="max-w-full max-h-[80vh] object-contain rounded-lg shadow-2xl" loading="lazy" />

          @if (images.length > 1) {
            <button type="button"
              class="absolute left-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/70 text-white rounded-full w-10 h-10 flex items-center justify-center transition"
              (click)="prev()" aria-label="Previous image">
              &#8249;
            </button>
            <button type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 bg-black/50 hover:bg-black/70 text-white rounded-full w-10 h-10 flex items-center justify-center transition"
              (click)="next()" aria-label="Next image">
              &#8250;
            </button>
            <div class="flex justify-center gap-2 mt-3">
              @for (img of images; track img; let i = $index) {
                <button type="button"
                  class="w-12 h-12 rounded border-2 overflow-hidden transition"
                  [class.border-white]="i === currentIndex()"
                  [class.border-transparent]="i !== currentIndex()"
                  [class.opacity-60]="i !== currentIndex()"
                  (click)="currentIndex.set(i)">
                  <img [src]="img" class="w-full h-full object-cover" loading="lazy" [alt]="'Thumbnail ' + (i + 1)" />
                </button>
              }
            </div>
          }
        </div>
      </div>
    }
  `,
})
export class ImageLightboxComponent {
  @Input() images: string[] = [];
  @Input() open = false;
  @Input() startIndex = 0;
  @Input() alt = '';
  @Output() close = new EventEmitter<void>();

  currentIndex = signal(0);

  ngOnChanges() {
    if (this.open) {
      this.currentIndex.set(this.startIndex);
    }
  }

  prev() {
    const idx = this.currentIndex();
    this.currentIndex.set(idx > 0 ? idx - 1 : this.images.length - 1);
  }

  next() {
    const idx = this.currentIndex();
    this.currentIndex.set(idx < this.images.length - 1 ? idx + 1 : 0);
  }
}
