import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface StoreTeamMember {
  uuid: string;
  storeUuid: string;
  userUuid: string;
  role: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface StoreInvitation {
  uuid: string;
  storeUuid: string;
  email: string;
  role: string;
  token?: string;
  status: string;
  invitedBy: string;
  expiresAt: string;
  createdAt: string;
  updatedAt: string;
  acceptedAt?: string | null;
  shopName?: string;
  shopSlug?: string;
}

export interface ShopTeamPayload {
  members: StoreTeamMember[];
  invitations: StoreInvitation[];
}

export interface AddTeamMemberPayload {
  userUuid: string;
  role: string;
}

export interface UpdateTeamRolePayload {
  role: string;
}

export interface CreateInvitationPayload {
  email: string;
  role: string;
}

export interface TransferOwnershipPayload {
  newOwnerUuid: string;
}

export interface TransferOwnershipResponse {
  ownerUuid: string;
}

@Injectable({ providedIn: 'root' })
export class TeamService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  list(slug: string): Observable<ApiResponse<ShopTeamPayload>> {
    return this.http.get<ApiResponse<ShopTeamPayload>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/team`,
      { withCredentials: true }
    );
  }

  addMember(slug: string, payload: AddTeamMemberPayload): Observable<ApiResponse<StoreTeamMember>> {
    return this.http.post<ApiResponse<StoreTeamMember>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/team`,
      payload,
      { withCredentials: true }
    );
  }

  updateMemberRole(slug: string, memberUuid: string, payload: UpdateTeamRolePayload): Observable<ApiResponse<unknown>> {
    return this.http.patch<ApiResponse<unknown>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/team/${encodeURIComponent(memberUuid)}`,
      payload,
      { withCredentials: true }
    );
  }

  removeMember(slug: string, memberUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/team/${encodeURIComponent(memberUuid)}`,
      { withCredentials: true }
    );
  }

  createInvitation(slug: string, payload: CreateInvitationPayload): Observable<ApiResponse<StoreInvitation>> {
    return this.http.post<ApiResponse<StoreInvitation>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/invitations`,
      payload,
      { withCredentials: true }
    );
  }

  getInvitation(token: string): Observable<ApiResponse<StoreInvitation>> {
    return this.http.get<ApiResponse<StoreInvitation>>(
      `${this.api}/v1/invitations/${encodeURIComponent(token)}`,
      { withCredentials: true }
    );
  }

  acceptInvitation(token: string): Observable<ApiResponse<unknown>> {
    return this.http.post<ApiResponse<unknown>>(
      `${this.api}/v1/invitations/${encodeURIComponent(token)}/accept`,
      {},
      { withCredentials: true }
    );
  }

  transferOwnership(
    slug: string,
    payload: TransferOwnershipPayload
  ): Observable<ApiResponse<TransferOwnershipResponse>> {
    return this.http.post<ApiResponse<TransferOwnershipResponse>>(
      `${this.api}/v1/shops/${encodeURIComponent(slug)}/transfer`,
      payload,
      { withCredentials: true }
    );
  }
}
