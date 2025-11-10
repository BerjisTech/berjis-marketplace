import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NewShopFormComponent } from './new-shop-form.component';

@Component({
  standalone: true,
  selector: 'app-get-started-page',
  imports: [CommonModule, NewShopFormComponent],
  templateUrl: './get-started-page.component.html',
  styleUrls: ['./get-started-page.component.css']
})
export class GetStartedPageComponent {}


