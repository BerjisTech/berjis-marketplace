import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, computed, effect, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { ModalComponent } from '../app/shared/components/modal/modal.component';
import { Category, CategoryService, CreateCategoryPayload, UpdateCategoryPayload } from '../app/core/services/category.service';
import { ShopStateService } from '../app/core/services/shop-state.service';

interface CategoryDraft { name: string; slug: string; description: string; }
interface EditCategoryDraft { name: string; slug: string; description: string; isActive: boolean; }

@Component({
  standalone: true,
  selector: 'app-products-categories-page',
  imports: [CommonModule, FormsModule, RouterLink, ModalComponent],
  templateUrl: './products-categories-page.component.html',
  styleUrls: ['./products-categories-page.component.css']
})
export class ProductsCategoriesPageComponent implements OnInit, OnDestroy {
  private readonly categoriesApi = inject(CategoryService);
  private readonly shopState = inject(ShopStateService);

  readonly shops = this.shopState.shops;
  readonly activeShopSlug = this.shopState.activeShopSlug;
  readonly loadingShops = this.shopState.loading;
  categories = signal<Category[]>([]);
  loadingCategories = signal<boolean>(false);
  message = signal<string>('');
  error = signal<string>('');
  submitting = signal<boolean>(false);
  editing = signal<boolean>(false);

  newCategory = signal<CategoryDraft>({ name: '', slug: '', description: '' });
  editDraft = signal<EditCategoryDraft>({ name: '', slug: '', description: '', isActive: true });
  editTarget: Category | null = null;

  readonly activeCategories = computed(() => this.categories().filter((cat) => cat.isActive));
  readonly archivedCategories = computed(() => this.categories().filter((cat) => !cat.isActive));

  private readonly syncActiveShop = effect(() => {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.categories.set([]);
      return;
    }
    this.loadCategories(slug);
  });

  ngOnInit(): void {
    this.shopState.ensureLoaded();
  }

  ngOnDestroy(): void {
    this.syncActiveShop.destroy();
  }

  changeShop(slug: string): void {
    this.shopState.setActiveShopSlug(slug);
    this.message.set('');
    this.error.set('');
  }

  setNewCategory(partial: Partial<CategoryDraft>): void {
    this.newCategory.update((draft) => ({ ...draft, ...partial }));
  }

  setEditDraft(partial: Partial<EditCategoryDraft>): void {
    this.editDraft.update((draft) => ({ ...draft, ...partial }));
  }

  loadCategories(slug?: string): void {
    const target = slug ?? this.shopState.activeShopSlug();
    if (!target) {
      this.categories.set([]);
      return;
    }
    this.loadingCategories.set(true);
    this.categoriesApi.list(target).subscribe({
      next: (response) => {
        this.categories.set(response?.data ?? []);
        this.loadingCategories.set(false);
      },
      error: (err) => {
        this.loadingCategories.set(false);
        this.error.set(this.resolveError(err, 'Could not load categories.'));
      },
    });
  }

  createCategory(): void {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.error.set('Select a shop first.');
      return;
    }
    const name = this.newCategory().name.trim();
    const slugInput = this.newCategory().slug.trim();
    if (!name) {
      this.error.set('Enter a category name.');
      return;
    }
    const payload: CreateCategoryPayload = {
      name,
      slug: slugInput || this.slugify(name),
      description: this.newCategory().description.trim(),
    };
    this.submitting.set(true);
    this.categoriesApi.create(slug, payload).subscribe({
      next: () => {
        this.newCategory.set({ name: '', slug: '', description: '' });
        this.submitting.set(false);
        this.message.set('Category created.');
        this.loadCategories();
      },
      error: (err) => {
        this.submitting.set(false);
        this.error.set(this.resolveError(err, 'Could not create category.'));
      },
    });
  }

  openEdit(category: Category): void {
    this.editTarget = category;
    this.editDraft.set({
      name: category.name,
      slug: category.slug,
      description: category.description,
      isActive: category.isActive,
    });
    this.editing.set(true);
  }

  closeEdit(): void {
    this.editing.set(false);
    this.editTarget = null;
  }

  saveEdit(): void {
    if (!this.editTarget) {
      return;
    }
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      this.error.set('Select a shop first.');
      return;
    }
    const draft = this.editDraft();
    const payload: UpdateCategoryPayload = {
      name: draft.name.trim() || this.editTarget.name,
      slug: (draft.slug.trim() || this.editTarget.slug).toLowerCase(),
      description: draft.description.trim(),
      isActive: draft.isActive,
    };
    this.submitting.set(true);
    this.categoriesApi.update(slug, this.editTarget.uuid, payload).subscribe({
      next: () => {
        this.submitting.set(false);
        this.message.set('Category updated.');
        this.closeEdit();
        this.loadCategories();
      },
      error: (err) => {
        this.submitting.set(false);
        this.error.set(this.resolveError(err, 'Could not update category.'));
      },
    });
  }

  toggleActive(category: Category): void {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      return;
    }
    const payload: UpdateCategoryPayload = { isActive: !category.isActive };
    this.categoriesApi.update(slug, category.uuid, payload).subscribe({
      next: () => {
        this.message.set(`Category ${category.isActive ? 'archived' : 'restored'}.`);
        this.loadCategories();
      },
      error: (err) => {
        this.error.set(this.resolveError(err, 'Could not update category.'));
      },
    });
  }

  deleteCategory(category: Category): void {
    const slug = this.shopState.activeShopSlug();
    if (!slug) {
      return;
    }
    this.categoriesApi.archive(slug, category.uuid).subscribe({
      next: () => {
        this.message.set('Category archived.');
        this.loadCategories();
      },
      error: (err) => {
        this.error.set(this.resolveError(err, 'Could not archive category.'));
      },
    });
  }

  private slugify(value: string): string {
    return value
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '');
  }

  private resolveError(err: unknown, fallback: string): string {
    if (!err) {
      return fallback;
    }
    const httpErr = err as { error?: { message?: string }; message?: string };
    if (httpErr?.error?.message) {
      return httpErr.error.message;
    }
    if (httpErr?.message) {
      return httpErr.message;
    }
    return fallback;
  }
}
