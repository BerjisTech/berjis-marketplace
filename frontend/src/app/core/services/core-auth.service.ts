import { Injectable } from '@angular/core';

type VerifyResult = { success: boolean; data?: { valid: boolean; uid?: number; uuid?: string; email?: string; name?: string }; message?: string };

@Injectable({ providedIn: 'root' })
export class CoreAuthService {
  // Pin to central Core API; marketplace only needs verification, not base override.
  private base = 'https://api.berjis.tech';

  private authCache: { ts: number; result: VerifyResult } | null = null;
  private authInFlight: Promise<VerifyResult> | null = null;

  async verify(): Promise<VerifyResult> {
    const r = await fetch(this.base + '/v1/auth/verify', {
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: '{}'
    });
    return (await r.json()) as VerifyResult;
  }

  async refresh() {
    try {
      const r = await fetch(this.base + '/v1/auth/refresh', {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: '{}'
      });
      return await r.json();
    } catch { return null; }
  }

  async ensureAuth(opts?: { force?: boolean; maxAgeMs?: number }): Promise<VerifyResult> {
    const force = !!opts?.force; const maxAge = opts?.maxAgeMs ?? 1500; const now = Date.now();
    if (!force && this.authCache && now - this.authCache.ts < maxAge) return this.authCache.result;
    if (!force && this.authInFlight) return this.authInFlight;
    const runner = this.runEnsureAuth().then(result => { this.authCache = { ts: Date.now(), result }; return result; })
      .catch(err => { this.authCache = null; throw err; })
      .finally(() => { this.authInFlight = null; });
    this.authInFlight = runner; return runner;
  }

  private async runEnsureAuth(): Promise<VerifyResult> {
    try {
      const v = await this.verify();
      if (v?.data?.valid) return v;
      await this.refresh();
      return await this.verify();
    } catch {
      try { await this.refresh(); return await this.verify(); } catch { return { success: true, data: { valid: false } } as VerifyResult; }
    }
  }
}

