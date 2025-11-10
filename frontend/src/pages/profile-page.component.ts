import { CommonModule } from '@angular/common';
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProfileService, UpdateProfilePayload, UserAddress, UserProfile } from '../app/core/services/profile.service';
import { ModalComponent } from '../app/shared/components/modal/modal.component';

@Component({
  standalone: true,
  selector: 'app-profile-page',
  imports: [CommonModule, FormsModule, ModalComponent],
  templateUrl: './profile-page.component.html',
  styleUrls: ['./profile-page.component.css'],
})
export class ProfilePageComponent implements OnInit {
  private readonly profileService = inject(ProfileService);

  readonly loading = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');

  readonly profile = signal<UserProfile | null>(null);
  readonly addresses = signal<UserAddress[]>([]);

  readonly profileForm = signal<ProfileForm>({
    displayName: '',
    email: '',
    phone: '',
    avatarUrl: '',
    timezone: '',
    marketingOptIn: false,
  });
  readonly profileBusy = signal<boolean>(false);

  readonly addressModalOpen = signal<boolean>(false);
  readonly editingAddress = signal<UserAddress | null>(null);
  readonly addressBusy = signal<boolean>(false);
  readonly addressForm = signal<AddressForm>({
    label: '',
    recipientName: '',
    line1: '',
    line2: '',
    city: '',
    region: '',
    postalCode: '',
    country: 'US',
    phone: '',
    defaultShipping: false,
    defaultBilling: false,
  });

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.profileService.getProfile().subscribe({
      next: (response) => {
        const data = response?.data;
        if (data?.profile) {
          this.profile.set(data.profile);
          this.profileForm.set({
            displayName: data.profile.displayName ?? '',
            email: data.profile.email ?? '',
            phone: data.profile.phone ?? '',
            avatarUrl: data.profile.avatarUrl ?? '',
            timezone: data.profile.timezone ?? 'UTC',
            marketingOptIn: data.profile.marketingOptIn ?? false,
          });
        }
        this.addresses.set(data?.addresses ?? []);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Could not load profile.');
      },
    });
  }

  saveProfile(): void {
    const form = this.profileForm();
    const payload: UpdateProfilePayload = {
      displayName: form.displayName,
      email: form.email,
      phone: form.phone,
      avatarUrl: form.avatarUrl || undefined,
      timezone: form.timezone || undefined,
      marketingOptIn: form.marketingOptIn,
    };
    this.profileBusy.set(true);
    this.profileService.updateProfile(payload).subscribe({
      next: (response) => {
        const data = response?.data;
        if (data?.profile) {
          this.profile.set(data.profile);
        }
        if (data?.addresses) {
          this.addresses.set(data.addresses);
        }
        this.profileBusy.set(false);
        this.message.set('Profile updated.');
      },
      error: () => {
        this.profileBusy.set(false);
        this.error.set('Could not update profile.');
      },
    });
  }

  openAddressModal(address?: UserAddress): void {
    if (address) {
      this.editingAddress.set(address);
      this.addressForm.set({
        label: address.label,
        recipientName: address.recipientName,
        line1: address.line1,
        line2: address.line2,
        city: address.city,
        region: address.region,
        postalCode: address.postalCode,
        country: address.country,
        phone: address.phone,
        defaultShipping: address.isDefaultShipping,
        defaultBilling: address.isDefaultBilling,
      });
    } else {
      this.editingAddress.set(null);
      this.addressForm.set({
        label: '',
        recipientName: '',
        line1: '',
        line2: '',
        city: '',
        region: '',
        postalCode: '',
        country: 'US',
        phone: '',
        defaultShipping: false,
        defaultBilling: false,
      });
    }
    this.addressModalOpen.set(true);
    this.addressBusy.set(false);
    this.error.set('');
    this.message.set('');
  }

  closeAddressModal(): void {
    this.addressModalOpen.set(false);
    this.addressBusy.set(false);
    this.editingAddress.set(null);
  }

  saveAddress(): void {
    const form = this.addressForm();
    if (!form.line1 || !form.city || !form.region || !form.postalCode || !form.country) {
      this.error.set('Please provide the required address fields.');
      return;
    }
    this.addressBusy.set(true);
    const payload = {
      label: form.label,
      recipientName: form.recipientName,
      line1: form.line1,
      line2: form.line2,
      city: form.city,
      region: form.region,
      postalCode: form.postalCode,
      country: form.country,
      phone: form.phone,
      defaultShipping: form.defaultShipping,
      defaultBilling: form.defaultBilling,
    };
    const address = this.editingAddress();
    const request$ = address
      ? this.profileService.updateAddress(address.uuid, payload)
      : this.profileService.createAddress(payload);
    request$.subscribe({
      next: (response) => {
        const updated = response?.data;
        if (updated) {
          if (address) {
            this.addresses.set(
              this.addresses().map((item) => (item.uuid === updated.uuid ? updated : item)),
            );
          } else {
            this.addresses.set([updated, ...this.addresses()]);
          }
        }
        this.addressBusy.set(false);
        this.closeAddressModal();
        this.message.set('Address saved.');
      },
      error: () => {
        this.addressBusy.set(false);
        this.error.set('Could not save address.');
      },
    });
  }

  deleteAddress(address: UserAddress): void {
    if (!address) {
      return;
    }
    this.addressBusy.set(true);
    this.profileService.deleteAddress(address.uuid).subscribe({
      next: () => {
        this.addressBusy.set(false);
        this.addresses.set(this.addresses().filter((item) => item.uuid !== address.uuid));
        this.message.set('Address removed.');
      },
      error: () => {
        this.addressBusy.set(false);
        this.error.set('Could not delete address.');
      },
    });
  }
}

interface ProfileForm {
  displayName: string;
  email: string;
  phone: string;
  avatarUrl: string;
  timezone: string;
  marketingOptIn: boolean;
}

interface AddressForm {
  label: string;
  recipientName: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postalCode: string;
  country: string;
  phone: string;
  defaultShipping: boolean;
  defaultBilling: boolean;
}
