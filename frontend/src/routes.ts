import { Routes } from '@angular/router';
import { ShopPageComponent } from './pages/shop-page.component';
import { HomePageComponent } from './pages/home-page.component';
import { ProductPageComponent } from './pages/product-page.component';
import { DashboardPageComponent } from './pages/dashboard-page.component';
// Admin dashboard pages (skeletons)
import { OrdersOverviewPageComponent } from './pages/orders-overview-page.component';
import { OrdersDraftPageComponent } from './pages/orders-draft-page.component';
import { OrdersAbandonedCheckoutsPageComponent } from './pages/orders-abandoned-checkouts-page.component';
import { OrdersReturnsPageComponent } from './pages/orders-returns-page.component';

import { ProductsOverviewPageComponent } from './pages/products-overview-page.component';
import { DashboardOverviewPageComponent } from './pages/dashboard-overview-page.component';
import { ProductsCollectionsPageComponent } from './pages/products-collections-page.component';
import { ProductsInventoryPageComponent } from './pages/products-inventory-page.component';
import { ProductsPurchaseOrdersPageComponent } from './pages/products-purchase-orders-page.component';
import { ProductsTransfersPageComponent } from './pages/products-transfers-page.component';
import { ProductsGiftCardsPageComponent } from './pages/products-gift-cards-page.component';
import { ProductsNewPageComponent } from './pages/products-new-page.component';

import { CustomersOverviewPageComponent } from './pages/customers-overview-page.component';
import { CustomersSegmentsPageComponent } from './pages/customers-segments-page.component';

import { MarketingOverviewPageComponent } from './pages/marketing-overview-page.component';
import { MarketingCampaignsPageComponent } from './pages/marketing-campaigns-page.component';
import { MarketingAttributionsPageComponent } from './pages/marketing-attributions-page.component';
import { MarketingAutomationsPageComponent } from './pages/marketing-automations-page.component';

import { DiscountsOverviewPageComponent } from './pages/discounts-overview-page.component';

import { ContentOverviewPageComponent } from './pages/content-overview-page.component';
import { ContentFilesPageComponent } from './pages/content-files-page.component';
import { ContentMenusPageComponent } from './pages/content-menus-page.component';
import { ContentBlogPostsPageComponent } from './pages/content-blog-posts-page.component';

import { MarketsOverviewPageComponent } from './pages/markets-overview-page.component';
import { MarketsCatalogsPageComponent } from './pages/markets-catalogs-page.component';

import { AnalyticsOverviewPageComponent } from './pages/analytics-overview-page.component';
import { AnalyticsReportsPageComponent } from './pages/analytics-reports-page.component';
import { AnalyticsLiveViewPageComponent } from './pages/analytics-live-view-page.component';

import { SettingsPageComponent } from './pages/settings-page.component';
import { SettingsGeneralPageComponent } from './pages/settings-general-page.component';
import { SettingsPlanPageComponent } from './pages/settings-plan-page.component';
import { SettingsBillingPageComponent } from './pages/settings-billing-page.component';
import { SettingsUsersPageComponent } from './pages/settings-users-page.component';
import { SettingsPaymentsPageComponent } from './pages/settings-payments-page.component';
import { SettingsCheckoutPageComponent } from './pages/settings-checkout-page.component';
import { SettingsCustomerAccountsPageComponent } from './pages/settings-customer-accounts-page.component';
import { SettingsShippingAndDeliveryPageComponent } from './pages/settings-shipping-and-delivery-page.component';
import { SettingsTaxesAndDutiesPageComponent } from './pages/settings-taxes-and-duties-page.component';
import { SettingsLocationsPageComponent } from './pages/settings-locations-page.component';
import { SettingsAppsAndSalesChannelsPageComponent } from './pages/settings-apps-and-sales-channels-page.component';
import { SettingsDomainsPageComponent } from './pages/settings-domains-page.component';
import { SettingsCustomerEventsPageComponent } from './pages/settings-customer-events-page.component';
import { SettingsNotificationsPageComponent } from './pages/settings-notifications-page.component';
import { SettingsMetafieldsAndMetaobjectsPageComponent } from './pages/settings-metafields-and-metaobjects-page.component';
import { SettingsLanguagesPageComponent } from './pages/settings-languages-page.component';
import { SettingsCustomerPrivacyPageComponent } from './pages/settings-customer-privacy-page.component';
import { SettingsPoliciesPageComponent } from './pages/settings-policies-page.component';
import { CategoryPageComponent } from './pages/category-page.component';
import { NewShopPageComponent } from './pages/new-shop-page.component';
import { GetStartedPageComponent } from './pages/get-started-page.component';
import { UserFacingPagesComponent } from './app/components/user-facing-pages/user-facing-pages.component';
import { InviteAcceptPageComponent } from './pages/invite-accept-page.component';
import { authGuard } from './app/core/guards/auth.guard';

export const routes: Routes = [
  {
    path: '', component: UserFacingPagesComponent, children: [
      { path: '', component: HomePageComponent },
      { path: 'get-started', component: GetStartedPageComponent, canActivate: [authGuard] },
      { path: 'cart', loadComponent: () => import('./pages/cart-page.component').then(m => m.CartPageComponent) },
      { path: 'wishlist', loadComponent: () => import('./pages/wishlist-page.component').then(m => m.WishlistPageComponent) },
      { path: 'checkout', loadComponent: () => import('./pages/checkout-page.component').then(m => m.CheckoutPageComponent) },
      { path: 'checkout/confirmation/:id', loadComponent: () => import('./pages/order-confirmation-page.component').then(m => m.OrderConfirmationPageComponent) },
      { path: 'product/:id', component: ProductPageComponent },
      { path: 'shop/:slug', component: ShopPageComponent },
      { path: 'category/:id', component: CategoryPageComponent },
    ]
  },
  { path: 'dashboard', component: DashboardPageComponent, canActivate: [authGuard], canActivateChild: [authGuard], children: [
    { path: '', pathMatch: 'full', component: DashboardOverviewPageComponent },
    // Orders
    { path: 'orders', component: OrdersOverviewPageComponent },
    { path: 'orders/draft', component: OrdersDraftPageComponent },
    { path: 'orders/abandoned-checkouts', component: OrdersAbandonedCheckoutsPageComponent },
    { path: 'orders/returns', component: OrdersReturnsPageComponent },
    // Products
    { path: 'products', component: ProductsOverviewPageComponent },
    { path: 'products/new', component: ProductsNewPageComponent },
    { path: 'products/collections', component: ProductsCollectionsPageComponent },
    { path: 'products/inventory', component: ProductsInventoryPageComponent },
    { path: 'products/purchase-orders', component: ProductsPurchaseOrdersPageComponent },
    { path: 'products/transfers', component: ProductsTransfersPageComponent },
    { path: 'products/gift-cards', component: ProductsGiftCardsPageComponent },
    // Customers
    { path: 'customers', component: CustomersOverviewPageComponent },
    { path: 'customers/segments', component: CustomersSegmentsPageComponent },
    // Marketing
    { path: 'marketing', component: MarketingOverviewPageComponent },
    { path: 'marketing/campaigns', component: MarketingCampaignsPageComponent },
    { path: 'marketing/attributions', component: MarketingAttributionsPageComponent },
    { path: 'marketing/automations', component: MarketingAutomationsPageComponent },
    // Discounts
    { path: 'discounts', component: DiscountsOverviewPageComponent },
    // Content
    { path: 'content', component: ContentOverviewPageComponent },
    { path: 'content/files', component: ContentFilesPageComponent },
    { path: 'content/menus', component: ContentMenusPageComponent },
    { path: 'content/blog-posts', component: ContentBlogPostsPageComponent },
    // Markets
    { path: 'markets', component: MarketsOverviewPageComponent },
    { path: 'markets/catalogs', component: MarketsCatalogsPageComponent },
    // Analytics
    { path: 'analytics', component: AnalyticsOverviewPageComponent },
    { path: 'analytics/reports', component: AnalyticsReportsPageComponent },
    { path: 'analytics/live-view', component: AnalyticsLiveViewPageComponent },
    // Settings overview + specific pages
    { path: 'settings', component: SettingsPageComponent },
    { path: 'settings/general', component: SettingsGeneralPageComponent },
    { path: 'settings/plan', component: SettingsPlanPageComponent },
    { path: 'settings/billing', component: SettingsBillingPageComponent },
    { path: 'settings/users', component: SettingsUsersPageComponent },
    { path: 'settings/payments', component: SettingsPaymentsPageComponent },
    { path: 'settings/checkout', component: SettingsCheckoutPageComponent },
    { path: 'settings/customer-accounts', component: SettingsCustomerAccountsPageComponent },
    { path: 'settings/shipping-and-delivery', component: SettingsShippingAndDeliveryPageComponent },
    { path: 'settings/taxes-and-duties', component: SettingsTaxesAndDutiesPageComponent },
    { path: 'settings/locations', component: SettingsLocationsPageComponent },
    { path: 'settings/apps-and-sales-channels', component: SettingsAppsAndSalesChannelsPageComponent },
    { path: 'settings/domains', component: SettingsDomainsPageComponent },
    { path: 'settings/customer-events', component: SettingsCustomerEventsPageComponent },
    { path: 'settings/notifications', component: SettingsNotificationsPageComponent },
    { path: 'settings/metafields-and-metaobjects', component: SettingsMetafieldsAndMetaobjectsPageComponent },
    { path: 'settings/languages', component: SettingsLanguagesPageComponent },
    { path: 'settings/customer-privacy', component: SettingsCustomerPrivacyPageComponent },
    { path: 'settings/policies', component: SettingsPoliciesPageComponent },
  ] },
  { path: 'invite/:token', component: InviteAcceptPageComponent },
  { path: 'dashboard/shops/new', component: NewShopPageComponent },
  { path: '**', redirectTo: '' }
];
