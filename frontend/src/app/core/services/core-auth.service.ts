import { Injectable } from '@angular/core';
import { CoreAuthService as SharedCoreAuthService, CoreAuthSession } from '@berjis/angular-auth';

type VerifyResult = { success: boolean; data?: CoreAuthSession; message?: string };

@Injectable({ providedIn: 'root' })
export class CoreAuthService {
  constructor(private core: SharedCoreAuthService) {}

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

