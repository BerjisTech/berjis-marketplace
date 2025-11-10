import { Component, OnDestroy, OnInit, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, ParamMap, Router, RouterLink } from '@angular/router';
import { Subscription } from 'rxjs';
import { TeamService, StoreInvitation } from '../app/core/services/team.service';
import { CoreAuthService } from '../app/core/services/core-auth.service';
import { HttpErrorResponse } from '@angular/common/http';

interface EnsureAuthResult {
  success: boolean;
  data?: {
    uuid?: string;
    email?: string;
  };
}

@Component({
  standalone: true,
  selector: 'app-invite-accept-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './invite-accept-page.component.html',
  styleUrls: ['./invite-accept-page.component.css'],
})
export class InviteAcceptPageComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly team = inject(TeamService);
  private readonly coreAuth = inject(CoreAuthService);

  readonly invitation = signal<StoreInvitation | null>(null);
  readonly loading = signal<boolean>(true);
  readonly error = signal<string>('');
  readonly accepting = signal<boolean>(false);
  readonly sessionValid = signal<boolean>(false);
  readonly sessionEmail = signal<string>('');

  private routeSub: Subscription | null = null;
  private token = '';

  ngOnInit(): void {
    this.routeSub = this.route.paramMap.subscribe((params: ParamMap) => {
      const token = params.get('token')?.trim() ?? '';
      this.token = token;
      if (!token) {
        this.error.set('Invitation token is missing.');
        this.loading.set(false);
        return;
      }
      this.bootstrapSession();
      this.fetchInvitation(token);
    });
  }

  ngOnDestroy(): void {
    if (this.routeSub) {
      this.routeSub.unsubscribe();
    }
  }

  async accept(): Promise<void> {
    if (!this.token || this.accepting() || !this.invitation()) {
      return;
    }
    if (!this.sessionValid()) {
      this.goToLogin();
      return;
    }
    this.accepting.set(true);
    this.error.set('');
    this.team
      .acceptInvitation(this.token)
      .subscribe({
        next: () => {
          const invite = this.invitation();
          if (invite) {
            this.invitation.set({ ...invite, status: 'accepted', token: '' });
          }
        },
        error: (err: HttpErrorResponse) => {
          if (err.status === 401) {
            this.sessionValid.set(false);
            this.error.set('Please sign in to accept this invitation.');
          } else {
            this.error.set('Could not accept the invitation. Please try again.');
          }
        },
      })
      .add(() => this.accepting.set(false));
  }

  goToLogin(): void {
    if (!this.token) {
      this.router.navigateByUrl('/auth/login');
      return;
    }
    const redirect = encodeURIComponent(`/invite/${this.token}`);
    this.router.navigateByUrl(`/auth/login?redirect=${redirect}`);
  }

  viewDashboard(): void {
    this.router.navigateByUrl('/dashboard');
  }

  private fetchInvitation(token: string): void {
    this.loading.set(true);
    this.error.set('');
    this.team
      .getInvitation(token)
      .subscribe({
        next: (response) => {
          this.invitation.set(response?.data ?? null);
        },
        error: (err: HttpErrorResponse) => {
          if (err.status === 404) {
            this.error.set('Invitation not found. It may have been rescinded.');
          } else if (err.status === 410) {
            this.error.set('This invitation has expired.');
          } else {
            this.error.set('Unable to load invitation details.');
          }
          this.invitation.set(null);
        },
      })
      .add(() => this.loading.set(false));
  }

  private bootstrapSession(): void {
    this.coreAuth
      .ensureAuth({ force: true, maxAgeMs: 0 })
      .then((result: EnsureAuthResult) => {
        if (result?.success && result.data?.uuid) {
          this.sessionValid.set(true);
          this.sessionEmail.set(result.data.email ?? '');
        } else {
          this.sessionValid.set(false);
          this.sessionEmail.set('');
        }
      })
      .catch(() => {
        this.sessionValid.set(false);
        this.sessionEmail.set('');
      });
  }
}
