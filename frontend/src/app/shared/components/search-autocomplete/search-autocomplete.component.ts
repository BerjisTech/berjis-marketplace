import { Component, Input, Output, EventEmitter, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../../environments/environment';

interface Suggestion {
  uuid: string;
  title: string;
  type: string;
}

@Component({
  standalone: true,
  selector: 'app-search-autocomplete',
  imports: [CommonModule, FormsModule],
  template: `
    <div class="relative">
      <input
        type="text"
        [placeholder]="placeholder"
        [(ngModel)]="query"
        (input)="onInput()"
        (focus)="showSuggestions.set(true)"
        (blur)="hideSuggestionsDelayed()"
        (keyup.enter)="onSubmit()"
        (keyup.escape)="showSuggestions.set(false)"
        class="w-full rounded-full border border-blue-900/20 dark:border-slate-700 bg-white/90 dark:bg-slate-900/70 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-400" />

      @if (showSuggestions() && suggestions().length > 0) {
        <div class="absolute top-full left-0 right-0 mt-1 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-lg z-50 overflow-hidden">
          @for (item of suggestions(); track item.uuid) {
            <button type="button"
              class="w-full px-4 py-2 text-left text-sm hover:bg-blue-50 dark:hover:bg-slate-800 transition flex items-center gap-2"
              (mousedown)="selectSuggestion(item)">
              <span class="text-xs text-slate-400 uppercase">{{ item.type }}</span>
              <span class="text-slate-700 dark:text-slate-200">{{ item.title }}</span>
            </button>
          }
        </div>
      }
    </div>
  `,
})
export class SearchAutocompleteComponent {
  @Input() placeholder = 'Search products...';
  @Output() search = new EventEmitter<string>();

  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly api = environment.apiBase;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  query = '';
  suggestions = signal<Suggestion[]>([]);
  showSuggestions = signal(false);

  onInput() {
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (this.query.length < 2) {
      this.suggestions.set([]);
      return;
    }
    this.debounceTimer = setTimeout(() => this.fetchSuggestions(), 250);
  }

  onSubmit() {
    this.showSuggestions.set(false);
    this.search.emit(this.query);
  }

  selectSuggestion(item: Suggestion) {
    this.showSuggestions.set(false);
    this.query = item.title;
    if (item.type === 'product') {
      this.router.navigate(['/product', item.uuid]);
    } else {
      this.search.emit(item.title);
    }
  }

  hideSuggestionsDelayed() {
    setTimeout(() => this.showSuggestions.set(false), 200);
  }

  private async fetchSuggestions() {
    try {
      const res: any = await this.http.get(
        `${this.api}/v1/search/autocomplete?q=${encodeURIComponent(this.query)}`
      ).toPromise();
      this.suggestions.set(res?.data ?? []);
    } catch {
      this.suggestions.set([]);
    }
  }
}
