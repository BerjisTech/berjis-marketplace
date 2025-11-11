import { Component, EventEmitter, Input, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

@Component({
  standalone: true,
  selector: 'app-filters-rail',
  imports: [CommonModule, FormsModule],
  templateUrl: './filters-rail.component.html',
  styleUrls: ['./filters-rail.component.css']
})
export class FiltersRailComponent {
  @Input() open = false;
  @Output() openChange = new EventEmitter<boolean>();

  @Input() title = 'Filters';
  @Input() shopName: string | null = null;

  @Input() q = '';
  @Output() qChange = new EventEmitter<string>();

  @Input() minPrice = '';
  @Output() minPriceChange = new EventEmitter<string>();

  @Input() maxPrice = '';
  @Output() maxPriceChange = new EventEmitter<string>();

  @Input() sort = '';
  @Output() sortChange = new EventEmitter<string>();

  @Input() showCategory = false;
  @Input() categories: string[] = [];
  @Input() selectedCategory = '';
  @Output() selectedCategoryChange = new EventEmitter<string>();

  @Input() collections: { slug: string; title: string }[] = [];
  @Input() selectedCollection = '';
  @Output() selectedCollectionChange = new EventEmitter<string>();

  @Output() apply = new EventEmitter<void>();
  @Output() clear = new EventEmitter<void>();
}
