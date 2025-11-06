import { Component } from '@angular/core';
import { DarkModeToggleComponent } from "../dark-mode-toggle/dark-mode-toggle.component";
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-user-facing-pages',
  standalone: true,
  imports: [RouterOutlet, DarkModeToggleComponent],
  templateUrl: './user-facing-pages.component.html',
  styleUrl: './user-facing-pages.component.css'
})
export class UserFacingPagesComponent {

  copyRightYear = new Date().getFullYear();

}
