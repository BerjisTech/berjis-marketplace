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
import { CategoryPageComponent } from './pages/category-page.component';
import { NewShopPageComponent } from './pages/new-shop-page.component';
import { GetStartedPageComponent } from './pages/get-started-page.component';
import { UserFacingPagesComponent } from './app/components/user-facing-pages/user-facing-pages.component';

export const routes: Routes = [
  {
    path: '', component: UserFacingPagesComponent, children: [
      { path: '', component: HomePageComponent },
      { path: 'get-started', component: GetStartedPageComponent },
      { path: 'product/:id', component: ProductPageComponent },
      { path: 'shop/:slug', component: ShopPageComponent },
      { path: 'category/:id', component: CategoryPageComponent },
    ]
  },
  { path: 'dashboard', component: DashboardPageComponent, children: [
    { path: '', pathMatch: 'full', component: ProductsOverviewPageComponent },
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
    // Settings
    { path: 'settings', component: SettingsPageComponent },
  ] },
  { path: 'dashboard/shops/new', component: NewShopPageComponent },
  { path: '**', redirectTo: '' }
];
