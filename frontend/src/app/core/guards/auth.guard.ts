import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { from, map, catchError, of } from 'rxjs';
import { CoreAuthService } from '../services/core-auth.service';

function redirectToCentralLogin(currentUrl: string) {
  if (typeof window === 'undefined') return;
  const host = window.location.hostname;
  const m = host.match(/(^|\.)berjis\.(test|tech|com)$/i);
  const root = m ? `berjis.${m[2].toLowerCase()}` : 'berjis.tech';
  const origin = window.location.origin;
  const absolute = currentUrl?.startsWith('http') ? currentUrl : origin + currentUrl;
  const target = `${window.location.protocol}//${root}/auth/login?returnUrl=${encodeURIComponent(absolute)}`;
  window.location.href = target;
}

export const authGuard: CanActivateFn = (_route, state) => {
  const core = inject(CoreAuthService);
  const router = inject(Router);
  return from(core.ensureAuth({ maxAgeMs: 1500 })).pipe(
    map(res => {
      const valid = !!res?.data?.valid;
      if (!valid) redirectToCentralLogin(state.url || '/');
      return valid;
    }),
    catchError(() => { redirectToCentralLogin(state.url || '/'); return of(false); })
  );
};
