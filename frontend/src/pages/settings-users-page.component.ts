import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { environment } from '../environments/environment';
import {
  TeamService,
  StoreTeamMember,
  StoreInvitation,
  ShopTeamPayload,
} from '../app/core/services/team.service';
import { ApiResponse } from '../app/core/services/product.service';
import { CoreAuthService } from '../app/core/services/core-auth.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  ownerUuid: string;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

interface EnsureAuthResult {
  success: boolean;
  data?: {
    uuid?: string;
    platformRoles?: string[];
  };
}

@Component({
  standalone: true,
  selector: 'app-settings-users-page',
  imports: [CommonModule, FormsModule],
  templateUrl: './settings-users-page.component.html',
  styleUrls: ['./settings-users-page.component.css'],
})
export class SettingsUsersPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly team = inject(TeamService);
  private readonly coreAuth = inject(CoreAuthService);
  private readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly selectedShopSlug = signal<string>('');
  readonly members = signal<StoreTeamMember[]>([]);
  readonly invitations = signal<StoreInvitation[]>([]);
  readonly loading = signal<boolean>(false);
  readonly inviteBusy = signal<boolean>(false);
  readonly mutationBusy = signal<Record<string, boolean>>({});
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly inviteEmail = signal<string>('');
  readonly inviteRole = signal<'manager' | 'staff'>('manager');
  readonly addUuid = signal<string>('');
  readonly addUuidRole = signal<'manager' | 'staff'>('staff');
  readonly addUuidBusy = signal<boolean>(false);

  private readonly currentUserUuid = signal<string>('');
  private readonly platformRoles = signal<string[]>([]);

  readonly selectedShop = computed(() => {
    const slug = this.selectedShopSlug();
    return this.shops().find((shop) => shop.slug === slug) ?? null;
  });

  readonly currentRole = computed(() => {
    const userUuid = this.currentUserUuid();
    const shop = this.selectedShop();
    if (!shop || !userUuid) {
      return '';
    }
    if (shop.ownerUuid && shop.ownerUuid.toLowerCase() === userUuid.toLowerCase()) {
      return 'owner';
    }
    const member = this.members().find(
      (item) => item.userUuid?.toLowerCase() === userUuid.toLowerCase()
    );
    if (member) {
      return member.role?.toLowerCase();
    }
    if (this.platformRoles().some((role) => /^platform\./i.test(role))) {
      return 'platform';
    }
    return '';
  });

  readonly canManage = computed(() => {
    const role = this.currentRole();
    return role === 'owner' || role === 'platform';
  });

  readonly allowedRoles: ('manager' | 'staff')[] = ['manager', 'staff'];

  ngOnInit(): void {
    this.bootstrapSession();
    this.loadShops();
  }

  onShopChange(slug: string): void {
    this.selectedShopSlug.set(slug);
    this.loadTeam(slug);
  }

  invite(): void {
    const slug = this.selectedShopSlug();
    if (!slug || !this.canManage()) {
      return;
    }
    const email = this.inviteEmail().trim().toLowerCase();
    const role = this.inviteRole();
    if (!email) {
      this.error.set('Please enter an email address.');
      return;
    }
    this.clearMessages();
    this.inviteBusy.set(true);
    this.team
      .createInvitation(slug, { email, role })
      .subscribe({
        next: (response) => {
          const invite = response?.data;
          if (invite) {
            this.invitations.set([invite, ...this.invitations()]);
            this.message.set('Invitation sent.');
            this.inviteEmail.set('');
          }
        },
        error: (err: HttpErrorResponse) => {
          this.error.set(this.describeError(err, 'Could not send invitation.'));
        },
      })
      .add(() => this.inviteBusy.set(false));
  }

  addMemberByUuid(): void {
    const slug = this.selectedShopSlug();
    if (!slug || !this.canManage()) {
      return;
    }
    const uuid = this.addUuid().trim();
    if (!uuid) {
      this.error.set('Please provide a Core user UUID.');
      return;
    }
    this.clearMessages();
    this.addUuidBusy.set(true);
    this.team
      .addMember(slug, { userUuid: uuid, role: this.addUuidRole() })
      .subscribe({
        next: (response) => {
          const member = response?.data;
          if (member) {
            const existing = this.members()
              .filter((m) => m.uuid !== member.uuid && m.userUuid !== member.userUuid);
            this.members.set([...existing, member]);
            this.message.set('Team member added.');
            this.addUuid.set('');
          }
        },
        error: (err: HttpErrorResponse) => {
          this.error.set(this.describeError(err, 'Could not add team member.'));
        },
      })
      .add(() => this.addUuidBusy.set(false));
  }

  updateRole(member: StoreTeamMember, role: 'manager' | 'staff'): void {
    const slug = this.selectedShopSlug();
    if (!slug || !this.canManage()) {
      return;
    }
    if ((member.role ?? '').toLowerCase() === role) {
      return;
    }
    this.setMemberBusy(member.uuid, true);
    this.clearMessages();
    this.team
      .updateMemberRole(slug, member.uuid, { role })
      .subscribe({
        next: () => {
          this.members.set(
            this.members().map((m) =>
              m.uuid === member.uuid
                ? { ...m, role }
                : m
            )
          );
          this.message.set('Role updated.');
        },
        error: (err: HttpErrorResponse) => {
          this.error.set(this.describeError(err, 'Could not update role.'));
        },
      })
      .add(() => this.setMemberBusy(member.uuid, false));
  }

  removeMember(member: StoreTeamMember): void {
    const slug = this.selectedShopSlug();
    if (!slug || !this.canManage()) {
      return;
    }
    if (!confirm('Remove this team member? They will lose access to the shop.')) {
      return;
    }
    this.setMemberBusy(member.uuid, true);
    this.clearMessages();
    this.team
      .removeMember(slug, member.uuid)
      .subscribe({
        next: () => {
          this.members.set(this.members().filter((m) => m.uuid !== member.uuid));
          this.message.set('Team member removed.');
        },
        error: (err: HttpErrorResponse) => {
          this.error.set(this.describeError(err, 'Could not remove team member.'));
        },
      })
      .add(() => this.setMemberBusy(member.uuid, false));
  }

  copyInvite(invite: StoreInvitation): void {
    if (!invite.token) {
      this.error.set('Invitation is no longer active.');
      return;
    }
    const link = this.buildInviteLink(invite.token);
    if (navigator?.clipboard?.writeText) {
      navigator.clipboard
        .writeText(link)
        .then(() => this.message.set('Invitation link copied to clipboard.'))
        .catch(() => (this.error.set('Could not copy link. Please copy manually.'), this.fallbackCopy(link)));
    } else {
      this.fallbackCopy(link);
    }
  }

  inviteLink(invite: StoreInvitation): string {
    return invite.token ? this.buildInviteLink(invite.token) : '';
  }

  trackByMember(_: number, member: StoreTeamMember): string {
    return member.uuid;
  }

  trackByInvite(_: number, invite: StoreInvitation): string {
    return invite.uuid;
  }

  formatRelative(timestamp?: string): string {
    if (!timestamp) {
      return '';
    }
    const date = new Date(timestamp);
    if (Number.isNaN(date.getTime())) {
      return timestamp;
    }
    return date.toLocaleString();
  }

  private loadShops(): void {
    this.loading.set(true);
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const shops = response?.data ?? [];
          this.shops.set(shops);
          const current = this.selectedShopSlug();
          const nextSlug = current && shops.some((s) => s.slug === current)
            ? current
            : shops[0]?.slug ?? '';
          this.selectedShopSlug.set(nextSlug);
          if (nextSlug) {
            this.loadTeam(nextSlug);
          } else {
            this.members.set([]);
            this.invitations.set([]);
          }
        },
        error: (err: HttpErrorResponse) => {
          this.error.set(this.describeError(err, 'Could not load shops or permissions.'));
        },
      })
      .add(() => this.loading.set(false));
  }

  private loadTeam(slug: string): void {
    if (!slug) {
      return;
    }
    this.loading.set(true);
    this.team
      .list(slug)
      .subscribe({
        next: (response) => {
          const data: ShopTeamPayload | undefined = response?.data;
          this.members.set(data?.members ?? []);
          this.invitations.set(data?.invitations ?? []);
          this.message.set('');
          this.error.set('');
        },
        error: (err: HttpErrorResponse) => {
          this.members.set([]);
          this.invitations.set([]);
          this.error.set(this.describeError(err, 'Could not load team members.'));
        },
      })
      .add(() => this.loading.set(false));
  }

  private bootstrapSession(): void {
    this.coreAuth
      .ensureAuth({ force: true, maxAgeMs: 0 })
      .then((result: EnsureAuthResult) => {
        if (result?.success && result.data) {
          this.currentUserUuid.set(result.data.uuid ?? '');
          const platformRoles = Array.isArray(result.data.platformRoles)
            ? result.data.platformRoles
            : [];
          this.platformRoles.set(platformRoles);
        }
      })
      .catch(() => {
        this.currentUserUuid.set('');
        this.platformRoles.set([]);
      });
  }

  private clearMessages(): void {
    this.message.set('');
    this.error.set('');
  }

  private setMemberBusy(uuid: string, busy: boolean): void {
    const current = this.mutationBusy();
    this.mutationBusy.set({ ...current, [uuid]: busy });
  }

  private buildInviteLink(token: string): string {
    const origin = typeof window !== 'undefined' && window.location ? window.location.origin : '';
    return `${origin}/invite/${token}`;
  }

  private fallbackCopy(value: string): void {
    try {
      const textarea = document.createElement('textarea');
      textarea.value = value;
      textarea.style.position = 'fixed';
      textarea.style.opacity = '0';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      this.message.set('Invitation link copied to clipboard.');
    } catch {
      this.error.set('Copy not supported. Use the link below.');
    }
  }

  private describeError(error: HttpErrorResponse, fallback: string): string {
    if (!error) {
      return fallback;
    }
    if (error.status === 401) {
      return 'Please sign in to manage this shop.';
    }
    if (error.status === 403) {
      return 'You do not have permission to manage this shop.';
    }
    if (error.status === 404) {
      return 'The requested resource was not found.';
    }
    if (error.status === 409) {
      return 'A conflicting record already exists.';
    }
    const message = (error.error && error.error.message) || error.message;
    return message || fallback;
  }
}


