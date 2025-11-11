import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { environment } from '../environments/environment';
import { ApiResponse } from '../app/core/services/product.service';
import {
  GiftCard,
  GiftCardReportSummary,
  GiftCardService,
  GiftCardTransaction,
} from '../app/core/services/gift-card.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

@Component({
  standalone: true,
  selector: 'app-products-gift-cards-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './products-gift-cards-page.component.html',
  styleUrls: ['./products-gift-cards-page.component.css'],
})
export class ProductsGiftCardsPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly giftCardsApi = inject(GiftCardService);

  readonly api = environment.apiBase;
  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly cards = signal<GiftCard[]>([]);
  readonly summary = signal<GiftCardReportSummary | null>(null);
  readonly selectedCard = signal<GiftCard | null>(null);
  readonly transactions = signal<GiftCardTransaction[]>([]);

  readonly loadingShops = signal<boolean>(false);
  readonly loadingCards = signal<boolean>(false);
  readonly loadingSummary = signal<boolean>(false);
  readonly loadingTransactions = signal<boolean>(false);

  readonly shopError = signal<string>('');
  readonly cardsError = signal<string>('');
  readonly summaryError = signal<string>('');
  readonly transactionsError = signal<string>('');

  readonly hasCards = computed(() => !this.loadingCards() && this.cards().length > 0);
  readonly totalIssuedCents = computed(() => this.summary()?.issuedCents ?? 0);
  readonly outstandingCents = computed(() => this.summary()?.outstandingCents ?? 0);
  readonly redeemedCents = computed(() => this.summary()?.redeemedCents ?? 0);
  readonly totalCards = computed(() => this.summary()?.totalCards ?? 0);
  readonly activeCards = computed(() => this.summary()?.activeCards ?? 0);
  readonly redeemedCards = computed(() => this.summary()?.redeemedCards ?? 0);

  ngOnInit(): void {
    this.loadShops();
  }

  loadShops(): void {
    this.loadingShops.set(true);
    this.shopError.set('');
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const list = response?.data ?? [];
          this.shops.set(list);
          const existing = this.shopSlug();
          const initial =
            existing && list.some((shop) => shop.slug === existing)
              ? existing
              : list[0]?.slug ?? '';
          this.shopSlug.set(initial);
          this.loadingShops.set(false);
          if (initial) {
            this.loadSummary();
            this.loadCards();
          }
        },
        error: () => {
          this.loadingShops.set(false);
          this.shopError.set('Could not load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopSlug.set(slug);
    this.cards.set([]);
    this.summary.set(null);
    this.selectedCard.set(null);
    this.transactions.set([]);
    this.cardsError.set('');
    this.summaryError.set('');
    if (slug) {
      this.loadSummary();
      this.loadCards();
    }
  }

  refresh(): void {
    if (!this.shopSlug()) {
      return;
    }
    this.loadSummary(true);
    this.loadCards(true);
  }

  loadSummary(force = false): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.loadingSummary() && !force) {
      return;
    }
    this.loadingSummary.set(true);
    this.summaryError.set('');
    this.giftCardsApi.report(slug).subscribe({
      next: (response) => {
        this.summary.set(response?.data ?? null);
        this.loadingSummary.set(false);
      },
      error: () => {
        this.loadingSummary.set(false);
        this.summaryError.set('Could not load gift card summary.');
      },
    });
  }

  loadCards(force = false): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.loadingCards() && !force) {
      return;
    }
    this.loadingCards.set(true);
    this.cardsError.set('');
    this.giftCardsApi.list(slug).subscribe({
      next: (response) => {
        this.cards.set(response?.data ?? []);
        this.loadingCards.set(false);
      },
      error: () => {
        this.loadingCards.set(false);
        this.cardsError.set('Could not load gift cards.');
      },
    });
  }

  viewTransactions(card: GiftCard): void {
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    this.selectedCard.set(card);
    this.transactions.set([]);
    this.transactionsError.set('');
    this.loadingTransactions.set(true);
    this.giftCardsApi.transactions(slug, card.uuid).subscribe({
      next: (response) => {
        this.transactions.set(response?.data ?? []);
        this.loadingTransactions.set(false);
      },
      error: () => {
        this.loadingTransactions.set(false);
        this.transactionsError.set('Could not load transactions for this card.');
      },
    });
  }

  clearSelection(): void {
    this.selectedCard.set(null);
    this.transactions.set([]);
    this.transactionsError.set('');
  }

  trackShopBy(_index: number, shop: ShopSummary): string {
    return shop.uuid;
  }

  trackCardBy(_index: number, card: GiftCard): string {
    return card.uuid;
  }

  trackTransactionBy(_index: number, tx: GiftCardTransaction): string {
    return tx.uuid;
  }
}


