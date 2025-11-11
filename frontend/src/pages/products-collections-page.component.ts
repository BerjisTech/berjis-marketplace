import { CommonModule } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { environment } from '../environments/environment';
import { ApiResponse, ProductService, ProductSummary } from '../app/core/services/product.service';
import {
  Collection,
  CollectionRule,
  CollectionService,
  CollectionSummary,
  CreateCollectionPayload,
  UpdateCollectionPayload,
} from '../app/core/services/collection.service';

interface ShopSummary {
  uuid: string;
  name: string;
  slug: string;
  description?: string;
}

interface CollectionFormModel {
  title: string;
  slug: string;
  description: string;
  isAutomatic: boolean;
  sortOrder: number;
  isActive: boolean;
  rules: CollectionRule[];
  productUuids: string[];
}

interface RuleOperatorConfig {
  value: string;
  label: string;
  input: 'string' | 'number' | 'boolean' | 'range';
}

interface RuleFieldConfig {
  field: string;
  label: string;
  operators: RuleOperatorConfig[];
}

@Component({
  standalone: true,
  selector: 'app-products-collections-page',
  imports: [CommonModule, FormsModule, RouterLink],
  templateUrl: './products-collections-page.component.html',
  styleUrls: ['./products-collections-page.component.css'],
})
export class ProductsCollectionsPageComponent implements OnInit {
  private readonly http = inject(HttpClient);
  private readonly collectionsApi = inject(CollectionService);
  private readonly productsApi = inject(ProductService);

  readonly api = environment.apiBase;

  readonly shops = signal<ShopSummary[]>([]);
  readonly shopSlug = signal<string>('');
  readonly collections = signal<Collection[]>([]);
  readonly publicCollections = signal<CollectionSummary[]>([]);
  readonly products = signal<ProductSummary[]>([]);
  readonly loading = signal<boolean>(false);
  readonly productsLoading = signal<boolean>(false);
  readonly message = signal<string>('');
  readonly error = signal<string>('');
  readonly saving = signal<boolean>(false);
  readonly deleting = signal<boolean>(false);
  readonly rebuilding = signal<boolean>(false);
  readonly creatingNew = signal<boolean>(false);
  readonly selectedCollection = signal<Collection | null>(null);
  readonly collectionForm = signal<CollectionFormModel>(this.createEmptyForm());

  readonly ruleFields: RuleFieldConfig[] = [
    {
      field: 'title',
      label: 'Title',
      operators: [
        { value: 'contains', label: 'contains', input: 'string' },
        { value: 'starts_with', label: 'starts with', input: 'string' },
        { value: 'ends_with', label: 'ends with', input: 'string' },
        { value: 'equals', label: 'equals', input: 'string' },
      ],
    },
    {
      field: 'category',
      label: 'Category',
      operators: [
        { value: 'equals', label: 'equals', input: 'string' },
        { value: 'contains', label: 'contains', input: 'string' },
      ],
    },
    {
      field: 'price',
      label: 'Price (cents)',
      operators: [
        { value: 'gte', label: '≥', input: 'number' },
        { value: 'lte', label: '≤', input: 'number' },
        { value: 'between', label: 'between', input: 'range' },
      ],
    },
    {
      field: 'stock',
      label: 'Stock',
      operators: [
        { value: 'gte', label: '≥', input: 'number' },
        { value: 'lte', label: '≤', input: 'number' },
      ],
    },
    {
      field: 'published',
      label: 'Published status',
      operators: [{ value: 'equals', label: 'equals', input: 'boolean' }],
    },
  ];

  readonly hasCollections = computed(() => !this.loading() && this.collections().length > 0);

  ngOnInit(): void {
    this.bootstrap();
  }

  bootstrap(): void {
    this.loading.set(true);
    this.http
      .get<ApiResponse<ShopSummary[]>>(`${this.api}/v1/my/shops`, { withCredentials: true })
      .subscribe({
        next: (response) => {
          const list = response?.data ?? [];
          this.shops.set(list);
          const current = this.shopSlug();
          const initial = current && list.some((shop) => shop.slug === current) ? current : list[0]?.slug ?? '';
          this.shopSlug.set(initial);
          if (initial) {
            this.loadCollections();
            this.loadProducts();
            this.loadPublicCollections(initial);
          } else {
            this.loading.set(false);
          }
        },
        error: () => {
          this.loading.set(false);
          this.error.set('Could not load shops.');
        },
      });
  }

  changeShop(slug: string): void {
    this.shopSlug.set(slug);
    this.message.set('');
    this.error.set('');
    this.selectedCollection.set(null);
    this.creatingNew.set(false);
    this.collectionForm.set(this.createEmptyForm());
    this.loadCollections();
    this.loadProducts();
    this.loadPublicCollections(slug);
  }

  loadCollections(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.collections.set([]);
      this.loading.set(false);
      return;
    }
    this.loading.set(true);
    this.collectionsApi.listMyCollections(slug).subscribe({
      next: (response) => {
        this.collections.set(response?.data ?? []);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.error.set('Could not load collections.');
      },
    });
  }

  loadPublicCollections(slug: string): void {
    if (!slug) {
      this.publicCollections.set([]);
      return;
    }
    this.collectionsApi.listPublicCollections(slug).subscribe({
      next: (response) => {
        this.publicCollections.set(response?.data ?? []);
      },
      error: () => {
        this.publicCollections.set([]);
      },
    });
  }

  loadProducts(): void {
    const slug = this.shopSlug();
    if (!slug) {
      this.products.set([]);
      return;
    }
    this.productsLoading.set(true);
    this.productsApi.listMyShopProducts(slug).subscribe({
      next: (response) => {
        this.products.set(response?.data ?? []);
        this.productsLoading.set(false);
      },
      error: () => {
        this.productsLoading.set(false);
        this.products.set([]);
      },
    });
  }

  startCreate(): void {
    this.creatingNew.set(true);
    this.selectedCollection.set(null);
    this.collectionForm.set(this.createEmptyForm());
    this.message.set('');
    this.error.set('');
  }

  openCollection(collection: Collection): void {
    this.creatingNew.set(false);
    this.selectedCollection.set(collection);
    this.collectionForm.set(this.toFormModel(collection));
    this.message.set('');
    this.error.set('');
    if (!collection.isAutomatic) {
      this.collectionsApi
        .listCollectionProducts(this.shopSlug(), collection.uuid)
        .subscribe({
          next: (response) => {
            const products = response?.data ?? [];
            const nextForm = { ...this.collectionForm() };
            nextForm.productUuids = products.map((p) => p.uuid);
            this.collectionForm.set(nextForm);
          },
          error: () => {
            // ignore load failures but keep selection empty
          },
        });
    }
  }

  closeEditor(): void {
    this.creatingNew.set(false);
    this.selectedCollection.set(null);
    this.collectionForm.set(this.createEmptyForm());
    this.message.set('');
    this.error.set('');
  }

  addRule(): void {
    const form = this.collectionForm();
    const defaultField = this.ruleFields[0];
    const defaultOperator = defaultField.operators[0];
    const nextRules = [...form.rules, { field: defaultField.field, operator: defaultOperator.value, value: '' }];
    this.collectionForm.set({ ...form, rules: nextRules });
  }

  removeRule(index: number): void {
    const form = this.collectionForm();
    const nextRules = form.rules.filter((_, i) => i !== index);
    this.collectionForm.set({ ...form, rules: nextRules });
  }

  updateRuleField(index: number, fieldValue: string): void {
    const form = this.collectionForm();
    const rules = form.rules.slice();
    const fieldConfig = this.ruleFields.find((f) => f.field === fieldValue) ?? this.ruleFields[0];
    const operatorConfig = fieldConfig.operators[0];
    rules[index] = {
      field: fieldConfig.field,
      operator: operatorConfig.value,
      value: operatorConfig.input === 'boolean' ? true : operatorConfig.input === 'number' ? 0 : '',
      min: null,
      max: null,
    };
    this.collectionForm.set({ ...form, rules });
  }

  updateRuleOperator(index: number, operatorValue: string): void {
    const form = this.collectionForm();
    const rules = form.rules.slice();
    const rule = { ...rules[index] };
    rule.operator = operatorValue;
    rule.value = undefined;
    rule.min = null;
    rule.max = null;
    const operatorConfig = this.getOperatorConfig(rule);
    if (operatorConfig?.input === 'boolean') {
      rule.value = true;
    } else if (operatorConfig?.input === 'number') {
      rule.value = 0;
    } else if (operatorConfig?.input === 'string') {
      rule.value = '';
    }
    rules[index] = rule;
    this.collectionForm.set({ ...form, rules });
  }

  toggleProductSelection(productUuid: string): void {
    const form = this.collectionForm();
    const set = new Set(form.productUuids);
    if (set.has(productUuid)) {
      set.delete(productUuid);
    } else {
      set.add(productUuid);
    }
    this.collectionForm.set({ ...form, productUuids: Array.from(set) });
  }

  isProductSelected(productUuid: string): boolean {
    return this.collectionForm().productUuids.includes(productUuid);
  }

  onAutomaticChange(value: boolean): void {
    const form = this.collectionForm();
    const next: CollectionFormModel = {
      ...form,
      isAutomatic: value,
      productUuids: value ? [] : form.productUuids,
      rules: value ? (form.rules.length > 0 ? form.rules : [{ field: 'title', operator: 'contains', value: '' }]) : [],
    };
    this.collectionForm.set(next);
  }

  maybeGenerateSlug(): void {
    const form = this.collectionForm();
    if (!form.slug.trim() && form.title.trim()) {
      this.collectionForm.set({ ...form, slug: this.slugify(form.title) });
    }
  }

  async saveCollection(): Promise<void> {
    if (this.saving()) {
      return;
    }
    const slug = this.shopSlug();
    if (!slug) {
      this.error.set('Select a shop first.');
      return;
    }
    const form = this.collectionForm();
    if (!form.title.trim()) {
      this.error.set('Title is required.');
      return;
    }
    if (!form.slug.trim()) {
      form.slug = this.slugify(form.title);
    }
    this.saving.set(true);
    this.error.set('');
    const payload = this.buildPayload(form);
    const request = this.creatingNew()
      ? this.collectionsApi.createCollection(slug, payload as CreateCollectionPayload)
      : this.collectionsApi.updateCollection(slug, this.selectedCollection()!.uuid, payload as UpdateCollectionPayload);
    request.subscribe({
      next: () => {
        this.saving.set(false);
        this.message.set(this.creatingNew() ? 'Collection created.' : 'Collection updated.');
        this.loadCollections();
        this.closeEditor();
      },
      error: () => {
        this.saving.set(false);
        this.error.set('Could not save the collection.');
      },
    });
  }

  rebuildCollection(): void {
    if (this.rebuilding() || !this.selectedCollection()?.isAutomatic) {
      return;
    }
    const slug = this.shopSlug();
    const collection = this.selectedCollection();
    if (!slug || !collection) {
      return;
    }
    this.rebuilding.set(true);
    this.collectionsApi.rebuildCollection(slug, collection.uuid).subscribe({
      next: () => {
        this.rebuilding.set(false);
        this.message.set('Collection rebuilt.');
        this.loadCollections();
      },
      error: () => {
        this.rebuilding.set(false);
        this.error.set('Could not rebuild collection.');
      },
    });
  }

  deleteCollection(): void {
    if (this.deleting() || (!this.creatingNew() && !this.selectedCollection())) {
      return;
    }
    const slug = this.shopSlug();
    if (!slug) {
      return;
    }
    if (this.creatingNew()) {
      this.closeEditor();
      return;
    }
    const collection = this.selectedCollection();
    if (!collection) {
      return;
    }
    if (!window.confirm(`Delete collection "${collection.title}"?`)) {
      return;
    }
    this.deleting.set(true);
    this.collectionsApi.deleteCollection(slug, collection.uuid).subscribe({
      next: () => {
        this.deleting.set(false);
        this.message.set('Collection deleted.');
        this.loadCollections();
        this.closeEditor();
      },
      error: () => {
        this.deleting.set(false);
        this.error.set('Could not delete collection.');
      },
    });
  }

  formatRule(rule: CollectionRule): string {
    const field = this.ruleFields.find((f) => f.field === rule.field);
    const operator = this.getOperatorConfig(rule);
    if (!field || !operator) {
      return `${rule.field} ${rule.operator}`;
    }
    if (operator.input === 'range') {
      return `${field.label} ${operator.label} ${rule.min ?? ''} – ${rule.max ?? ''}`;
    }
    if (operator.input === 'boolean') {
      return `${field.label} ${operator.label} ${rule.value ? 'true' : 'false'}`;
    }
    return `${field.label} ${operator.label} ${rule.value ?? ''}`;
  }

  private createEmptyForm(): CollectionFormModel {
    return {
      title: '',
      slug: '',
      description: '',
      isAutomatic: false,
      sortOrder: 0,
      isActive: true,
      rules: [],
      productUuids: [],
    };
  }

  private toFormModel(collection: Collection): CollectionFormModel {
    const rules = Array.isArray(collection.rules) ? collection.rules.map((rule) => ({ ...rule })) : [];
    return {
      title: collection.title,
      slug: collection.slug,
      description: collection.description ?? '',
      isAutomatic: collection.isAutomatic,
      sortOrder: collection.sortOrder ?? 0,
      isActive: collection.isActive ?? true,
      rules,
      productUuids: [],
    };
  }

  private buildPayload(form: CollectionFormModel): CreateCollectionPayload | UpdateCollectionPayload {
    const payload: CreateCollectionPayload = {
      title: form.title.trim(),
      slug: this.slugify(form.slug),
      description: form.description.trim(),
      isAutomatic: form.isAutomatic,
      sortOrder: form.sortOrder ?? 0,
      isActive: form.isActive,
    };
    if (form.isAutomatic) {
      payload.rules = form.rules.map((rule) => this.normalizeRule(rule));
      payload.productUuids = [];
    } else {
      payload.productUuids = form.productUuids;
      payload.rules = [];
    }
    return payload;
  }

  private normalizeRule(rule: CollectionRule): CollectionRule {
    const operatorConfig = this.getOperatorConfig(rule);
    const normalized: CollectionRule = { field: rule.field, operator: rule.operator };
    if (operatorConfig?.input === 'range') {
      normalized.min = rule.min != null ? Number(rule.min) : 0;
      normalized.max = rule.max != null ? Number(rule.max) : 0;
    } else if (operatorConfig?.input === 'number') {
      normalized.value = rule.value != null ? Number(rule.value) : 0;
    } else if (operatorConfig?.input === 'boolean') {
      normalized.value = Boolean(rule.value);
    } else {
      normalized.value = typeof rule.value === 'string' ? rule.value.trim() : '';
    }
    return normalized;
  }

  operatorOptions(rule: CollectionRule): RuleOperatorConfig[] {
    const field = this.ruleFields.find((f) => f.field === rule.field);
    return field?.operators ?? [];
  }

  private getOperatorConfig(rule: CollectionRule): RuleOperatorConfig | undefined {
    const field = this.ruleFields.find((f) => f.field === rule.field);
    return field?.operators.find((op) => op.value === rule.operator);
  }

  trackCollectionBy(_index: number, item: Collection): string {
    return item.uuid;
  }

  trackProductBy(_index: number, item: ProductSummary): string {
    return item.uuid;
  }

  private slugify(value: string): string {
    return value
      .toLowerCase()
      .trim()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/(^-|-$)+/g, '');
  }
}
