import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../../src/environments/environment';

@Injectable({ providedIn: 'root' })
export class ProductService {
  private api = environment.apiBase;
  constructor(private http: HttpClient) {}

  listMyShopProducts(shopSlug: string){
    return this.http.get<any>(`${this.api}/v1/my/shops/${shopSlug}/products`, { withCredentials: true });
  }

  createProduct(body: any){
    return this.http.post<any>(`${this.api}/v1/products`, body, { withCredentials: true });
  }

  upload(file: File){
    const fd = new FormData(); fd.append('file', file);
    return this.http.post<any>(`${this.api}/v1/uploads`, fd, { withCredentials: true });
  }
}

