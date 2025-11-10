# Angular + Go Marketplace Application - Complete Build Specification

## Project Overview
Build a full-stack e-commerce marketplace similar to Shopify with Angular frontend and Go backend. All components must be built from scratch without external libraries (except core Angular/Go frameworks). This is a learning project focused on understanding fundamental concepts.

---

## Design System

### Color Palette
**Light Mode:**
- Primary Background: `#eff6ff` (blue-50)
- Secondary Background: `#f8fafc` (slate-50)
- Accent: `#fbbf24` / `#f59e0b` (gold-400/500)
- Text Primary: `#0f172a` (slate-900)
- Text Secondary: `#475569` (slate-600)
- Borders: `#e2e8f0` (slate-200)
- Shadows: `0 1px 3px rgba(239, 246, 255, 0.3), 0 1px 2px rgba(239, 246, 255, 0.2)`
- Soft Glow (neon accents): `0 0 20px rgba(251, 191, 36, 0.3)`

**Dark Mode:**
- Primary Background: `#0c4a6e` (blue-900)
- Secondary Background: `#0f172a` (slate-900)
- Accent: `#fbbf24` / `#fcd34d` (gold-400/300)
- Text Primary: `#f1f5f9` (slate-100)
- Text Secondary: `#cbd5e1` (slate-300)
- Borders: `#1e293b` (slate-800)
- Shadows: `0 1px 3px rgba(12, 74, 110, 0.5), 0 1px 2px rgba(12, 74, 110, 0.3)`
- Soft Glow (neon accents): `0 0 25px rgba(252, 211, 77, 0.4)`

### Typography
- Primary Font: System UI stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`)
- Headings: Font-weight 700, letter-spacing -0.02em
- Body: Font-weight 400, line-height 1.6
- Small/Meta: Font-weight 500, size 0.875rem

### Component Styling Rules
- Border radius: 0.5rem (8px) for cards, 0.375rem (6px) for buttons
- Transitions: All interactive elements use `transition: all 0.2s ease`
- Hover states: Slightly lighter/darker background with gold accent border
- Focus states: Gold ring outline with soft glow
- Disabled states: 50% opacity with cursor-not-allowed

---

## Technical Architecture

### Frontend (Angular)

#### Project Structure
```
src/
├── app/
│   ├── core/
│   │   ├── services/
│   │   │   ├── auth.service.ts
│   │   │   ├── product.service.ts
│   │   │   ├── cart.service.ts
│   │   │   ├── order.service.ts
│   │   │   ├── user.service.ts
│   │   │   └── theme.service.ts
│   │   ├── guards/
│   │   │   ├── auth.guard.ts
│   │   │   └── role.guard.ts
│   │   ├── interceptors/
│   │   │   ├── auth.interceptor.ts
│   │   │   └── error.interceptor.ts
│   │   └── models/
│   │       ├── user.model.ts
│   │       ├── product.model.ts
│   │       ├── cart.model.ts
│   │       └── order.model.ts
│   ├── shared/
│   │   ├── components/
│   │   │   ├── header/
│   │   │   ├── footer/
│   │   │   ├── sidebar/
│   │   │   ├── product-card/
│   │   │   ├── modal/
│   │   │   ├── toast/
│   │   │   ├── loader/
│   │   │   └── pagination/
│   │   ├── directives/
│   │   └── pipes/
│   ├── features/
│   │   ├── auth/
│   │   │   ├── login/
│   │   │   ├── register/
│   │   │   └── forgot-password/
│   │   ├── products/
│   │   │   ├── product-list/
│   │   │   ├── product-detail/
│   │   │   └── product-search/
│   │   ├── cart/
│   │   ├── checkout/
│   │   ├── orders/
│   │   ├── profile/
│   │   └── admin/
│   │       ├── dashboard/
│   │       ├── product-management/
│   │       ├── order-management/
│   │       └── user-management/
│   └── app.component.ts
└── styles/
    ├── _variables.scss
    ├── _mixins.scss
    ├── _theme-light.scss
    ├── _theme-dark.scss
    └── global.scss
```

#### Key Features to Implement

**1. Authentication System**
- JWT-based authentication (manual implementation)
- Login, Register, Logout functionality
- Password hashing (implement bcrypt logic)
- Session management with localStorage
- Role-based access (User, Seller, Admin)

**2. Product Management**
- CRUD operations for products
- Image upload and storage (base64 encoding initially)
- Category and tag system
- Search and filter functionality
- Sorting (price, date, popularity)
- Product variants (size, color)

**3. Shopping Cart**
- Add/remove items
- Update quantities
- Calculate totals with tax
- Persist cart in localStorage
- Cart badge with item count

**4. Checkout Flow**
- Multi-step form (shipping, payment, review)
- Form validation (manual implementation)
- Order summary
- Mock payment processing
- Order confirmation

**5. Order Management**
- Order history
- Order tracking
- Order status updates
- Invoice generation

**6. Admin Dashboard**
- Sales analytics (charts built from scratch)
- Product inventory management
- Order processing
- User management
- Revenue reports

**Team Management**
- Invite managers/staff to individual shops with role assignments
- Pending invitation tokens with expiry and acceptance flow
- Manage active team members (update roles, remove access)

**7. User Profile**
- Profile information editing
- Address management
- Order history
- Wishlist functionality

#### Custom Components to Build

**Header Component:**
- Logo
- Search bar with autocomplete
- Navigation menu
- Cart icon with badge
- User account dropdown
- Theme toggle (light/dark)

**Product Card:**
- Product image
- Title, price, rating
- Quick view button
- Add to cart button
- Wishlist heart icon
- Golden accent on hover

**Modal System:**
- Backdrop with blur
- Close on escape/outside click
- Animated entrance/exit
- Different sizes (sm, md, lg, xl)

**Toast Notifications:**
- Success, error, warning, info types
- Auto-dismiss after 3 seconds
- Stack multiple toasts
- Slide-in animation from top-right
- Golden accent border

**Pagination:**
- Page numbers
- Previous/Next buttons
- Items per page selector
- Total count display

**Filter Sidebar:**
- Price range slider
- Category checkboxes
- Rating filter
- Brand filter
- Apply/Clear buttons

---

### Backend (Go)

#### Project Structure
```
backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handlers/
│   │   │   ├── auth.go
│   │   │   ├── products.go
│   │   │   ├── cart.go
│   │   │   ├── orders.go
│   │   │   └── users.go
│   │   ├── middleware/
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   ├── logger.go
│   │   │   └── ratelimit.go
│   │   └── routes.go
│   ├── models/
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── cart.go
│   │   └── order.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── product_repo.go
│   │   ├── cart_repo.go
│   │   └── order_repo.go
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── product_service.go
│   │   ├── cart_service.go
│   │   └── order_service.go
│   ├── database/
│   │   ├── connection.go
│   │   └── migrations.go
│   └── utils/
│       ├── jwt.go
│       ├── password.go
│       ├── validator.go
│       └── response.go
└── config/
    └── config.go
```

#### Database Schema

**Users Table:**
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Products Table:**
```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    seller_id INTEGER REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    stock INTEGER DEFAULT 0,
    category VARCHAR(100),
    images TEXT[], -- Array of base64 or file paths
    rating DECIMAL(3,2) DEFAULT 0,
    review_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Carts Table:**
```sql
CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    cart_id INTEGER REFERENCES carts(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Orders Table:**
```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    total_amount DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    shipping_address TEXT,
    payment_method VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL
);
```

#### API Endpoints to Implement

**Authentication:**
- POST `/api/auth/register` - User registration
- POST `/api/auth/login` - User login
- POST `/api/auth/logout` - User logout
- GET `/api/auth/me` - Get current user
- POST `/api/auth/refresh` - Refresh JWT token

**Products:**
- GET `/api/products` - List products (with pagination, filters)
- GET `/api/products/:id` - Get single product
- POST `/api/products` - Create product (seller/admin only)
- PUT `/api/products/:id` - Update product (seller/admin only)
- DELETE `/api/products/:id` - Delete product (admin only)
- GET `/api/products/search?q=query` - Search products

**Cart:**
- GET `/api/cart` - Get user's cart
- POST `/api/cart/items` - Add item to cart
- PUT `/api/cart/items/:id` - Update cart item quantity
- DELETE `/api/cart/items/:id` - Remove item from cart
- DELETE `/api/cart` - Clear cart

**Orders:**
- POST `/api/orders` - Create order from cart
- GET `/api/orders` - Get user's orders
- GET `/api/orders/:id` - Get single order
- PUT `/api/orders/:id/status` - Update order status (admin only)

**Users:**
- GET `/api/users/profile` - Get user profile
- PUT `/api/users/profile` - Update user profile
- GET `/api/users` - List all users (admin only)

**Teams:**
- GET `/v1/shops/{slug}/team` - list team members and pending invitations
- POST `/v1/shops/{slug}/team` - add a team member by Core user UUID
- PATCH `/v1/shops/{slug}/team/{id}` - update role for an existing team member
- DELETE `/v1/shops/{slug}/team/{id}` - remove a team member
- POST `/v1/shops/{slug}/invitations` - create a tokenized invitation for email/role
- GET `/v1/invitations/{token}` - view invitation metadata prior to acceptance
- POST `/v1/invitations/{token}/accept` - accept invitation (requires authenticated user)

---

## Implementation Checklist

### Phase 1: Foundation
- [x] Set up Angular project with routing
- [x] Set up Go project with basic HTTP server
- [x] Implement database connection (PostgreSQL)
- [x] Create database migrations
- [x] Set up CORS middleware
- [x] Implement theme service (light/dark mode)
- [x] Create base styles with color system
- [x] Build responsive layout structure

### Phase 2: Authentication
- [x] Implement JWT generation and validation (Go) *(handled by Core API)*
- [x] Create password hashing utilities (Go) *(handled by Core API)*
- [x] Build auth API endpoints (Go) *(Core API `/v1/auth/*`)*
- [x] Create auth service (Angular)
- [x] Build login component with validation *(shared landing auth UI)*
- [x] Build register component with validation *(shared landing auth UI)*
- [x] Implement auth guard
- [x] Create auth interceptor for JWT *(via `@berjis/angular-auth`)*
- [x] Add persistent login (localStorage) *(via shared Angular auth client)*

**Marketplace role model**
- [x] Define `marketplace.owner`, `marketplace.manager`, `marketplace.staff` app roles
- [x] Consume Core API `platform.*` roles for elevated dashboard access
- [x] Expose role helpers in Go middleware for downstream handlers

### Phase 3: Product System
- [x] Create product models and repository (Go)
- [x] Implement product CRUD endpoints (Go)
- [x] Build product service (Angular)
- [x] Create product list component with grid layout
- [x] Implement product card component
- [x] Add image upload functionality
- [x] Implement search functionality
- [x] Build filter sidebar
- [x] Create pagination component
- [x] Add sorting options
- [x] Build product detail page

### Phase 4: Shopping Cart
- [x] Create cart models and repository (Go)
- [x] Implement cart API endpoints (Go)
- [x] Build cart service (Angular)
- [x] Create cart component/page
- [x] Implement add to cart functionality
- [x] Build cart badge in header
- [x] Add quantity update controls
- [x] Implement cart total calculations
- [x] Add persistent cart (localStorage)

### Phase 5: Checkout & Orders
- [x] Create order models and repository (Go)
- [x] Implement order API endpoints (Go)
- [x] Build order service (Angular)
- [x] Create multi-step checkout form
- [x] Implement form validation
- [x] Build order summary component
- [x] Add mock payment processing
- [x] Create order confirmation page
- [ ] Build order history page
- [ ] Implement order tracking

### Phase 6: Admin Dashboard
- [x] Create admin route guard
- [x] Build admin layout
- [x] Implement team invitation & membership endpoints (Go)
- [ ] Create dashboard component with metrics
- [ ] Implement sales chart (custom built)
- [x] Build product management table
- [ ] Create product edit modal
- [ ] Build order management interface
- [ ] Implement user management table
- [ ] Add user role management

### Phase 7: User Experience
- [ ] Build user profile page
- [ ] Create profile edit form
- [ ] Implement address management
- [x] Add wishlist functionality
- [ ] Build modal system
- [x] Create toast notification system
- [ ] Implement loading states
- [ ] Add error handling
- [ ] Build 404 page
- [ ] Create footer component

### Phase 8: Polish & Optimization
- [ ] Implement lazy loading for routes
- [ ] Add transition animations
- [ ] Optimize images
- [x] Add meta tags for SEO
- [x] Implement form validation messages
- [x] Add accessibility attributes
- [ ] Test responsive design
- [x] Add hover effects and micro-interactions
- [x] Implement keyboard navigation
- [x] Add golden neon glows on interactive elements

---

## Styling Guidelines

### Light Mode Example
```scss
.product-card {
  background: #f8fafc; // slate-50
  border: 1px solid #e2e8f0; // slate-200
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px rgba(239, 246, 255, 0.3);
  transition: all 0.2s ease;
  
  &:hover {
    background: #eff6ff; // blue-50
    border-color: #fbbf24; // gold-400
    box-shadow: 0 0 20px rgba(251, 191, 36, 0.3);
  }
}

.btn-primary {
  background: #fbbf24; // gold-400
  color: #0f172a; // slate-900
  border: none;
  border-radius: 0.375rem;
  font-weight: 600;
  
  &:hover {
    background: #f59e0b; // gold-500
    box-shadow: 0 0 20px rgba(251, 191, 36, 0.3);
  }
}
```

### Dark Mode Example
```scss
[data-theme="dark"] {
  .product-card {
    background: #0f172a; // slate-900
    border: 1px solid #1e293b; // slate-800
    box-shadow: 0 1px 3px rgba(12, 74, 110, 0.5);
    
    &:hover {
      background: #0c4a6e; // blue-900
      border-color: #fcd34d; // gold-300
      box-shadow: 0 0 25px rgba(252, 211, 77, 0.4);
    }
  }
  
  .btn-primary {
    background: #fcd34d; // gold-300
    color: #0f172a; // slate-900
    
    &:hover {
      background: #fbbf24; // gold-400
      box-shadow: 0 0 25px rgba(252, 211, 77, 0.4);
    }
  }
}
```

---

## Key Learning Objectives

1. **Authentication**: Understand JWT, password hashing, session management
2. **State Management**: Learn Angular services and observables
3. **RESTful API**: Design and implement REST principles
4. **Database**: SQL queries, relationships, migrations
5. **Form Handling**: Validation, error messages, user feedback
6. **Responsive Design**: Mobile-first approach, flexbox, grid
7. **Security**: Input validation, SQL injection prevention, XSS protection
8. **Performance**: Lazy loading, pagination, debouncing
9. **UX**: Loading states, error handling, feedback mechanisms
10. **Architecture**: Separation of concerns, clean code principles

---

## Development Notes

- Build all validation logic manually (no external validators)
- Implement custom HTTP interceptors and guards
- Create reusable components and services
- Write clean, documented code
- Use TypeScript strict mode
- Follow Go best practices
- Implement proper error handling
- Add console logs for debugging
- Test all user flows
- Ensure accessibility standards
- Marketplace pricing surfaces server configuration from `/v1/settings/pricing` (see `TAX_RATE_PERCENT` and `SHIPPING_FLAT_CENTS` env vars for overrides).
- Open Graph metadata lives at `public/assets/og-marketplace.png` and is referenced from `index.html` for rich embeds.

---

## Golden Accents Usage

Use gold accents strategically for:
- Primary action buttons
- Active navigation items
- Selected filters
- Price displays
- Rating stars
- Success messages
- Hover states on interactive elements
- Focus rings on form inputs
- Progress indicators
- Badge backgrounds

Add soft neon glows (golden) on:
- Call-to-action buttons
- Add to cart buttons
- Checkout button
- Featured products
- Promotional banners
- Active tab indicators

---

This specification provides everything needed to build a complete marketplace from scratch. Focus on understanding each component and how they interact rather than rushing through implementation.
