import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class CartApiService {
  private api = environment.apiBase;
  constructor(private http: HttpClient) {}

  getCart(){
    return this.http.get<any>(`${this.api}/v1/cart`, { withCredentials: true });
  }
  putCart(items: any[]){
    return this.http.put<any>(`${this.api}/v1/cart`, { items }, { withCredentials: true });
  }
}

