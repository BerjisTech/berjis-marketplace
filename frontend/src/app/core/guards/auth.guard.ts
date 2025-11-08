import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

export const authGuard: CanActivateFn = (_route, state) => {
  try {
    const token = localStorage.getItem('auth_token');
    if (token && token.length > 10) return true;
  } catch {}
  const router = inject(Router);
  router.navigateByUrl('/get-started?redirect=' + encodeURIComponent(state.url));
  return false;
};

