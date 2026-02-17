import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  standalone: true,
  selector: 'app-settings-page',
  imports: [CommonModule, RouterLink],
  templateUrl: './settings-page.component.html',
  styleUrls: ['./settings-page.component.css']
})
export class SettingsPageComponent {
  sections = [
    { label: 'General', route: '/dashboard/settings/general', desc: 'Store name, contact email, preferences' },
    { label: 'Plan', route: '/dashboard/settings/plan', desc: 'Subscription plan management' },
    { label: 'Billing', route: '/dashboard/settings/billing', desc: 'Invoices and payment methods' },
    { label: 'Users', route: '/dashboard/settings/users', desc: 'Staff accounts and permissions' },
    { label: 'Payments', route: '/dashboard/settings/payments', desc: 'Payment gateways and methods' },
    { label: 'Checkout', route: '/dashboard/settings/checkout', desc: 'Customer checkout flow' },
    { label: 'Shipping', route: '/dashboard/settings/shipping-and-delivery', desc: 'Shipping rates and zones' },
    { label: 'Taxes', route: '/dashboard/settings/taxes-and-duties', desc: 'Tax rates and duty rules' },
    { label: 'Locations', route: '/dashboard/settings/locations', desc: 'Fulfillment locations' },
    { label: 'Domains', route: '/dashboard/settings/domains', desc: 'Custom domain configuration' },
    { label: 'Notifications', route: '/dashboard/settings/notifications', desc: 'Email and SMS templates' },
    { label: 'Customer accounts', route: '/dashboard/settings/customer-accounts', desc: 'Login and registration' },
    { label: 'Languages', route: '/dashboard/settings/languages', desc: 'Storefront translations' },
    { label: 'Policies', route: '/dashboard/settings/policies', desc: 'Legal pages and policies' },
    { label: 'Apps', route: '/dashboard/settings/apps-and-sales-channels', desc: 'Installed apps and channels' },
    { label: 'Metafields', route: '/dashboard/settings/metafields-and-metaobjects', desc: 'Custom data definitions' },
    { label: 'Webhooks', route: '/dashboard/settings/webhooks', desc: 'HTTP callbacks for shop events' },
    { label: 'Customer events', route: '/dashboard/settings/customer-events', desc: 'Event tracking pixels' },
    { label: 'Customer privacy', route: '/dashboard/settings/customer-privacy', desc: 'GDPR and privacy controls' },
  ];
}
