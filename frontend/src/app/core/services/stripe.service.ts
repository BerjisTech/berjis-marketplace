import { Injectable, signal } from '@angular/core';

declare global {
  interface Window {
    Stripe?: (key: string) => StripeInstance;
  }
}

interface StripeInstance {
  elements(options?: Record<string, unknown>): StripeElements;
  confirmCardPayment(clientSecret: string, data?: Record<string, unknown>): Promise<StripePaymentResult>;
}

interface StripeElements {
  create(type: string, options?: Record<string, unknown>): StripeElement;
}

interface StripeElement {
  mount(selector: string | HTMLElement): void;
  unmount(): void;
  destroy(): void;
  on(event: string, handler: (event: StripeElementEvent) => void): void;
}

interface StripeElementEvent {
  complete?: boolean;
  empty?: boolean;
  error?: { message: string };
}

interface StripePaymentResult {
  error?: { message: string; type?: string };
  paymentIntent?: { id: string; status: string };
}

@Injectable({ providedIn: 'root' })
export class StripeService {
  private stripe: StripeInstance | null = null;
  private elements: StripeElements | null = null;
  private cardElement: StripeElement | null = null;
  private loaded = false;
  private loading = false;

  cardReady = signal(false);
  cardError = signal('');

  async loadStripe(publishableKey: string): Promise<void> {
    if (this.loaded && this.stripe) return;
    if (this.loading) return;
    this.loading = true;

    try {
      if (!window.Stripe) {
        await this.loadScript('https://js.stripe.com/v3/');
      }
      if (!window.Stripe) {
        throw new Error('Stripe.js failed to load');
      }
      this.stripe = window.Stripe(publishableKey);
      this.loaded = true;
    } finally {
      this.loading = false;
    }
  }

  mountCardElement(container: string | HTMLElement): void {
    if (!this.stripe) {
      throw new Error('Stripe not loaded');
    }
    this.elements = this.stripe.elements();
    this.cardElement = this.elements.create('card', {
      style: {
        base: {
          fontSize: '16px',
          color: '#1e3a5f',
          '::placeholder': { color: '#94a3b8' },
        },
        invalid: { color: '#ef4444' },
      },
    });
    this.cardElement.mount(container);
    this.cardElement.on('change', (event: StripeElementEvent) => {
      this.cardReady.set(!!event.complete);
      this.cardError.set(event.error?.message ?? '');
    });
  }

  unmountCardElement(): void {
    if (this.cardElement) {
      this.cardElement.destroy();
      this.cardElement = null;
    }
    this.elements = null;
    this.cardReady.set(false);
    this.cardError.set('');
  }

  async confirmCardPayment(clientSecret: string): Promise<{ success: boolean; error?: string }> {
    if (!this.stripe) {
      return { success: false, error: 'Stripe not loaded' };
    }

    const result = await this.stripe.confirmCardPayment(clientSecret, {
      payment_method: { card: this.cardElement as unknown as Record<string, unknown> },
    });

    if (result.error) {
      return { success: false, error: result.error.message };
    }

    if (result.paymentIntent?.status === 'succeeded') {
      return { success: true };
    }

    return { success: false, error: 'Payment was not completed' };
  }

  private loadScript(src: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const existing = document.querySelector(`script[src="${src}"]`);
      if (existing) {
        resolve();
        return;
      }
      const script = document.createElement('script');
      script.src = src;
      script.async = true;
      script.onload = () => resolve();
      script.onerror = () => reject(new Error(`Failed to load ${src}`));
      document.head.appendChild(script);
    });
  }
}
