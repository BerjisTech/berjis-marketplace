import { Injectable, signal } from '@angular/core';

export type Toast = { id: string; type: 'success'|'error'|'info'; message: string };

@Injectable({ providedIn: 'root' })
export class ToastService {
  toasts = signal<Toast[]>([]);
  show(message: string, type: Toast['type']='success', ms=2500){
    const t: Toast = { id: Date.now().toString(36)+Math.random().toString(36).slice(2,6), type, message };
    this.toasts.set([...this.toasts(), t]);
    setTimeout(()=> this.dismiss(t.id), ms);
  }
  dismiss(id: string){ this.toasts.set(this.toasts().filter(t => t.id !== id)); }
}

