import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../../environments/environment';

@Injectable({ providedIn: 'root' })
export class OrderService {
  private api = environment.apiBase;
  constructor(private http: HttpClient) {}

  createOrder(body: any){
    // optimistic create: try API, else mock
    return new Promise<{ id: string }>(async (resolve) => {
      try {
        const res: any = await this.http.post(`${this.api}/v1/orders`, body, { withCredentials: true }).toPromise();
        resolve({ id: res?.data?.uuid || res?.data?.id || res?.id || Date.now().toString() });
      } catch {
        resolve({ id: Date.now().toString() });
      }
    });
  }
}

