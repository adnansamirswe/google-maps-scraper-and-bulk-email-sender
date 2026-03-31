import { writable } from 'svelte/store';

export type AlertType = 'success' | 'error' | 'warning' | 'info';

export interface Toast {
	id: string;
	type: AlertType;
	message: string;
}

export interface Confirmation {
	title: string;
	message: string;
	resolve: (val: boolean) => void;
}

// Svelte 5 Class-based State for Alerts
class AlertManager {
	toasts = $state<Toast[]>([]);
	confirmation = $state<Confirmation | null>(null);

	private add(type: AlertType, message: string) {
		const id = Math.random().toString(36).substring(2);
		this.toasts.push({ id, type, message });
		
		// Auto-remove after 4 seconds
		setTimeout(() => {
			this.toasts = this.toasts.filter(t => t.id !== id);
		}, 4000);
	}

	success(msg: string) { this.add('success', msg); }
	error(msg: string) { this.add('error', msg); }
	warning(msg: string) { this.add('warning', msg); }
	info(msg: string) { this.add('info', msg); }

	confirm(title: string, message: string): Promise<boolean> {
		return new Promise((resolve) => {
			this.confirmation = { 
				title, 
				message, 
				resolve: (val: boolean) => {
					this.confirmation = null;
					resolve(val);
				}
			};
		});
	}
}

export const alerts = new AlertManager();
