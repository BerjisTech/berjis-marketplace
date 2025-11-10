import { DOCUMENT } from '@angular/common';
import { Injectable, OnDestroy, computed, inject, signal } from '@angular/core';

export type ThemeMode = 'light' | 'dark';

@Injectable({ providedIn: 'root' })
export class ThemeService implements OnDestroy {
  private readonly storageKey = 'theme';
  private readonly document = inject(DOCUMENT);
  private readonly prefersDark = typeof window !== 'undefined' && window.matchMedia
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null;

  private manualOverride = this.readStoredMode() !== null;
  private readonly modeSignal = signal<ThemeMode>(this.initialMode());
  readonly mode = this.modeSignal.asReadonly();
  readonly isDark = computed(() => this.mode() === 'dark');

  private readonly mediaListener = (event: MediaQueryListEvent) => {
    if (!this.manualOverride) {
      this.updateMode(event.matches ? 'dark' : 'light', false);
    }
  };

  constructor() {
    this.apply(this.mode(), this.manualOverride);
    if (this.prefersDark) {
      this.prefersDark.addEventListener('change', this.mediaListener);
    }
  }

  ngOnDestroy(): void {
    if (this.prefersDark) {
      this.prefersDark.removeEventListener('change', this.mediaListener);
    }
  }

  toggle(): void {
    this.setTheme(this.mode() === 'dark' ? 'light' : 'dark');
  }

  setTheme(mode: ThemeMode): void {
    this.manualOverride = true;
    this.updateMode(mode, true);
  }

  useSystemPreference(): void {
    this.manualOverride = false;
    const systemMode: ThemeMode = this.prefersDark?.matches ? 'dark' : 'light';
    this.updateMode(systemMode, false);
  }

  private initialMode(): ThemeMode {
    const stored = this.readStoredMode();
    if (stored) {
      return stored;
    }
    return this.prefersDark?.matches ? 'dark' : 'light';
  }

  private updateMode(mode: ThemeMode, persist: boolean): void {
    this.modeSignal.set(mode);
    this.apply(mode, persist);
  }

  private apply(mode: ThemeMode, persist: boolean): void {
    const root = this.document?.documentElement;
    if (root) {
      root.classList.toggle('dark', mode === 'dark');
      root.setAttribute('data-theme', mode);
    }
    if (persist) {
      this.writeStoredMode(mode);
    } else {
      this.clearStoredMode();
    }
  }

  private readStoredMode(): ThemeMode | null {
    try {
      if (typeof window === 'undefined') {
        return null;
      }
      const stored = window.localStorage.getItem(this.storageKey);
      return stored === 'dark' || stored === 'light' ? stored : null;
    } catch {
      return null;
    }
  }

  private writeStoredMode(mode: ThemeMode): void {
    try {
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(this.storageKey, mode);
      }
    } catch {
      /* ignore unwriteable storage */
    }
  }

  private clearStoredMode(): void {
    try {
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem(this.storageKey);
      }
    } catch {
      /* ignore unwriteable storage */
    }
  }
}
