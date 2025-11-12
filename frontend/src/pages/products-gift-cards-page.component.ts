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
  CreateGiftCardPayload,
} from '../app/core/services/gift-card.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

type GiftCardStatusOption = 'draft' | 'active' | 'disabled';

interface GiftCardFormModel {
  code: string;
  amount: string;
  currency: string;
  issuedToEmail: string;
  note: string;
  status: GiftCardStatusOption;
  expiresOn: string;
}

const GIFT_CARD_CODE_SEGMENTS = 4;
const GIFT_CARD_CODE_SEGMENT_LENGTH = 4;
const GIFT_CARD_ALPHABET = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789';

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

  readonly showCreateForm = signal<boolean>(false);
  readonly creatingGiftCard = signal<boolean>(false);
  readonly createError = signal<string>('');
  readonly createSuccess = signal<string>('');

  giftCardForm: GiftCardFormModel = this.createDefaultGiftCardForm();
  readonly cardStatusOptions: GiftCardStatusOption[] = ['active', 'draft', 'disabled'];

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

  toggleCreateForm(): void {
    const next = !this.showCreateForm();
    this.showCreateForm.set(next);
    if (next) {
      this.createError.set('');
      this.createSuccess.set('');
    }
  }

  generateNewCode(): void {
    this.createError.set('');
    this.createSuccess.set('');
    this.giftCardForm.code = this.generateRandomGiftCardCode();
  }

  resetGiftCardForm(): void {
    this.giftCardForm = this.createDefaultGiftCardForm();
    this.createError.set('');
    this.createSuccess.set('');
  }

  submitGiftCard(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.createError.set('Select a shop first.');
      return;
    }
    if (this.creatingGiftCard()) {
      return;
    }
    this.createError.set('');
    this.createSuccess.set('');

    const amountValue = Number.parseFloat(this.giftCardForm.amount);
    if (!Number.isFinite(amountValue) || amountValue <= 0) {
      this.createError.set('Enter a gift card amount greater than zero.');
      return;
    }
    const balanceCents = Math.round(amountValue * 100);
    const currency = this.giftCardForm.currency.trim().toUpperCase() || 'USD';
    this.giftCardForm.currency = currency;

    const payload: CreateGiftCardPayload = {
      balanceCents,
      currency,
      status: this.giftCardForm.status,
    };

    const normalizedCode = this.normalizeGiftCardCode(this.giftCardForm.code);
    if (normalizedCode) {
      payload.code = normalizedCode;
      this.giftCardForm.code = normalizedCode;
    }

    const issuedTo = this.giftCardForm.issuedToEmail.trim();
    if (issuedTo) {
      payload.issuedToEmail = issuedTo;
    }

    const note = this.giftCardForm.note.trim();
    if (note) {
      payload.note = note;
    }

    const expiresOn = this.giftCardForm.expiresOn.trim();
    if (expiresOn) {
      const iso = this.toIsoDate(expiresOn);
      if (!iso) {
        this.createError.set('Enter a valid expiration date (YYYY-MM-DD).');
        return;
      }
      payload.expiresAt = iso;
    }

    this.creatingGiftCard.set(true);
    this.giftCardsApi.create(slug, payload).subscribe({
      next: (response) => {
        this.creatingGiftCard.set(false);
        const responseCode = response?.data?.code ?? normalizedCode;
        if (responseCode) {
          this.createSuccess.set(`Gift card created. Code ${responseCode}`);
        } else {
          this.createSuccess.set('Gift card created.');
        }
        this.giftCardForm = this.createDefaultGiftCardForm();
        this.loadSummary(true);
        this.loadCards(true);
      },
      error: (err) => {
        this.creatingGiftCard.set(false);
        const message =
          err?.error?.message ||
          err?.message ||
          'Could not create gift card.';
        this.createError.set(message);
      },
    });
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

  private createDefaultGiftCardForm(): GiftCardFormModel {
    return {
      code: '',
      amount: '',
      currency: 'USD',
      issuedToEmail: '',
      note: '',
      status: 'active',
      expiresOn: '',
    };
  }

  private normalizeGiftCardCode(code: string): string {
    if (!code) {
      return '';
    }
    const cleaned = code
      .toUpperCase()
      .replace(/[^A-Z0-9-]/g, '')
      .replace(/^-+/, '')
      .replace(/-+$/, '')
      .replace(/-{2,}/g, '-');
    return cleaned;
  }

  private generateRandomGiftCardCode(): string {
    const segments: string[] = [];
    const segmentCount = Math.max(0, GIFT_CARD_CODE_SEGMENTS);
    Array.from({ length: segmentCount }).forEach(() => {
      segments.push(this.randomCodeSegment(GIFT_CARD_CODE_SEGMENT_LENGTH));
    });
    return segments.join('-');
  }

  private randomCodeSegment(length: number): string {
    const alphabetLength = GIFT_CARD_ALPHABET.length;
    if (length <= 0 || alphabetLength === 0) {
      return '';
    }
    let result = '';
    if (typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function') {
      const buffer = new Uint32Array(length);
      crypto.getRandomValues(buffer);
      for (const value of buffer) {
        const index = value % alphabetLength;
        result += GIFT_CARD_ALPHABET[index];
      }
      return result;
    }
    const segmentLength = Math.max(0, length);
    Array.from({ length: segmentLength }).forEach(() => {
      const index = Math.floor(Math.random() * alphabetLength);
      result += GIFT_CARD_ALPHABET[index];
    });
    return result;
  }

  private toIsoDate(dateValue: string): string | null {
    if (!dateValue) {
      return null;
    }
    const isoCandidate = new Date(`${dateValue}T00:00:00Z`);
    if (Number.isNaN(isoCandidate.getTime())) {
      return null;
    }
    return isoCandidate.toISOString();
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


