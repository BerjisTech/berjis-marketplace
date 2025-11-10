import { Injectable, inject } from '@angular/core';
import { CoreAuthService as SharedCoreAuthService, CoreAuthSession } from '@berjis/angular-auth';

export interface VerifyResult { success: boolean; data?: CoreAuthSession; message?: string }

@Injectable({ providedIn: 'root' })
export class CoreAuthService {
  private readonly core = inject(SharedCoreAuthService);

  async verify(): Promise<VerifyResult> {
    const data = await this.core.verify();
    return { success: true, data };
  }

  async refresh() {
    await this.core.refresh();
    return { success: true };
  }

  async ensureAuth(opts?: { force?: boolean; maxAgeMs?: number }): Promise<VerifyResult> {
    const data = await this.core.ensureAuth(opts);
    return { success: true, data };
  }
}
