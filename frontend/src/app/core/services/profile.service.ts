import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { ApiResponse } from './product.service';

export interface UserProfile {
  userUuid: string;
  displayName: string;
  email: string;
  phone: string;
  avatarUrl?: string;
  timezone: string;
  marketingOptIn: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface UserAddress {
  uuid: string;
  userUuid: string;
  label: string;
  recipientName: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postalCode: string;
  country: string;
  phone: string;
  isDefaultShipping: boolean;
  isDefaultBilling: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ProfileResponse {
  profile: UserProfile;
  addresses: UserAddress[];
}

export interface UpdateProfilePayload {
  displayName?: string;
  email?: string;
  phone?: string;
  avatarUrl?: string;
  timezone?: string;
  marketingOptIn?: boolean;
}

export interface CreateAddressPayload {
  label?: string;
  recipientName?: string;
  line1: string;
  line2?: string;
  city: string;
  region: string;
  postalCode: string;
  country: string;
  phone?: string;
  defaultShipping?: boolean;
  defaultBilling?: boolean;
}

export interface UpdateAddressPayload {
  label?: string;
  recipientName?: string;
  line1?: string;
  line2?: string;
  city?: string;
  region?: string;
  postalCode?: string;
  country?: string;
  phone?: string;
  defaultShipping?: boolean;
  defaultBilling?: boolean;
}

@Injectable({ providedIn: 'root' })
export class ProfileService {
  private readonly http = inject(HttpClient);
  private readonly api = environment.apiBase;

  getProfile(): Observable<ApiResponse<ProfileResponse>> {
    return this.http.get<ApiResponse<ProfileResponse>>(`${this.api}/v1/me/profile`, { withCredentials: true });
  }

  updateProfile(payload: UpdateProfilePayload): Observable<ApiResponse<ProfileResponse>> {
    return this.http.put<ApiResponse<ProfileResponse>>(`${this.api}/v1/me/profile`, payload, {
      withCredentials: true,
    });
  }

  listAddresses(): Observable<ApiResponse<UserAddress[]>> {
    return this.http.get<ApiResponse<UserAddress[]>>(`${this.api}/v1/me/addresses`, { withCredentials: true });
  }

  createAddress(payload: CreateAddressPayload): Observable<ApiResponse<UserAddress>> {
    return this.http.post<ApiResponse<UserAddress>>(`${this.api}/v1/me/addresses`, payload, { withCredentials: true });
  }

  updateAddress(addressUuid: string, payload: UpdateAddressPayload): Observable<ApiResponse<UserAddress>> {
    return this.http.patch<ApiResponse<UserAddress>>(
      `${this.api}/v1/me/addresses/${encodeURIComponent(addressUuid)}`,
      payload,
      { withCredentials: true },
    );
  }

  deleteAddress(addressUuid: string): Observable<ApiResponse<unknown>> {
    return this.http.delete<ApiResponse<unknown>>(
      `${this.api}/v1/me/addresses/${encodeURIComponent(addressUuid)}`,
      { withCredentials: true },
    );
  }
}
