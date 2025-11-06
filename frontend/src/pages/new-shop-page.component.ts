import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NewShopFormComponent } from './new-shop-form.component';

@Component({
  standalone: true,
  selector: 'new-shop-page',
  imports: [CommonModule, NewShopFormComponent],
  templateUrl: './new-shop-page.component.html',
  styleUrls: ['./new-shop-page.component.css']
})
export class NewShopPageComponent {}
