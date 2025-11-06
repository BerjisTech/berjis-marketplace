import { Routes } from '@angular/router';
import { ShopPageComponent } from './pages/shop-page.component';
import { HomePageComponent } from './pages/home-page.component';
import { ProductPageComponent } from './pages/product-page.component';
import { DashboardPageComponent } from './pages/dashboard-page.component';
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
  { path: 'dashboard', component: DashboardPageComponent, children: [] },
  { path: 'dashboard/shops/new', component: NewShopPageComponent },
  { path: '**', redirectTo: '' }
];
