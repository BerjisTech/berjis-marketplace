import { ComponentFixture, TestBed } from '@angular/core/testing';

import { UserFacingPagesComponent } from './user-facing-pages.component';

describe('UserFacingPagesComponent', () => {
  let component: UserFacingPagesComponent;
  let fixture: ComponentFixture<UserFacingPagesComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [UserFacingPagesComponent]
    })
    .compileComponents();

    fixture = TestBed.createComponent(UserFacingPagesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
