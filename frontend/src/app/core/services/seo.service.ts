import { Injectable, inject } from '@angular/core';
import { DOCUMENT } from '@angular/common';
import { Title, Meta } from '@angular/platform-browser';

@Injectable({ providedIn: 'root' })
export class SeoService {
  private readonly title = inject(Title);
  private readonly meta = inject(Meta);
  private readonly doc = inject(DOCUMENT);

  private readonly defaultTitle = 'Berjis Marketplace';

  set(opts: { title?: string; description?: string; image?: string; url?: string }): void {
    const pageTitle = opts.title ? `${opts.title} — ${this.defaultTitle}` : this.defaultTitle;
    this.title.setTitle(pageTitle);

    if (opts.description) {
      this.meta.updateTag({ name: 'description', content: opts.description });
      this.meta.updateTag({ property: 'og:description', content: opts.description });
    }

    this.meta.updateTag({ property: 'og:title', content: pageTitle });
    this.meta.updateTag({ property: 'og:type', content: 'website' });

    if (opts.image) {
      this.meta.updateTag({ property: 'og:image', content: opts.image });
    }

    if (opts.url) {
      this.meta.updateTag({ property: 'og:url', content: opts.url });
      this.updateCanonical(opts.url);
    }
  }

  reset(): void {
    this.title.setTitle(this.defaultTitle);
    this.meta.removeTag("name='description'");
    this.meta.removeTag("property='og:description'");
    this.meta.removeTag("property='og:image'");
    this.meta.removeTag("property='og:url'");
    this.removeCanonical();
  }

  private updateCanonical(url: string): void {
    let link: HTMLLinkElement | null = this.doc.querySelector("link[rel='canonical']");
    if (!link) {
      link = this.doc.createElement('link');
      link.setAttribute('rel', 'canonical');
      this.doc.head.appendChild(link);
    }
    link.setAttribute('href', url);
  }

  private removeCanonical(): void {
    const link = this.doc.querySelector("link[rel='canonical']");
    link?.remove();
  }
}
