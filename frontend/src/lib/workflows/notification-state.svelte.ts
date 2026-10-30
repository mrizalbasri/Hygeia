import { getContext, setContext } from 'svelte';
import { api } from './api.js';
import { subscribeRealtime, reconnectRealtime } from './realtime.js';

const notificationContext = Symbol('dashboard-notifications');

// One instance per dashboard, never a module-level singleton shared by SSR users.
export class NotificationState {
	unread = $state<number | null>(null);
	revision = $state(0);
	connection = $state('Menghubungkan pembaruan…');
	failed = $state(false);
	private unsubscribe: (() => void) | null = null;
	private controller: AbortController | null = null;
	private countVersion = 0;

	async refreshUnread() {
		const controller = this.controller;
		if (!controller || controller.signal.aborted) return;
		const version = ++this.countVersion;
		try {
			const response = await api<{ unread_count: number }>(
				'/notifications/unread-count',
				'GET',
				undefined,
				controller.signal
			);
			if (controller.signal.aborted || version !== this.countVersion) return;
			this.unread =
				Number.isSafeInteger(response.unread_count) && response.unread_count >= 0
					? response.unread_count
					: null;
		} catch {
			if (!controller.signal.aborted && version === this.countVersion) this.unread = null;
		}
	}

	connect() {
		this.disconnect();
		const controller = new AbortController();
		this.controller = controller;
		this.unread = null;
		this.failed = false;
		this.connection = 'Menghubungkan pembaruan…';
		void this.refreshUnread();
		reconnectRealtime();
		this.unsubscribe = subscribeRealtime(
			() => {
				if (controller.signal.aborted) return;
				this.revision++;
				void this.refreshUnread();
			},
			(message) => {
				if (controller.signal.aborted) return;
				this.connection = message;
				this.failed =
					message !== 'Pembaruan langsung aktif' && message !== 'Menghubungkan pembaruan…';
			}
		);
		return () => {
			if (this.controller === controller) this.disconnect();
		};
	}

	disconnect() {
		this.controller?.abort();
		this.controller = null;
		this.unsubscribe?.();
		this.unsubscribe = null;
		this.countVersion++;
		this.unread = null;
	}
}

export function provideNotifications() {
	return setContext(notificationContext, new NotificationState());
}

export function useNotifications(): NotificationState {
	return getContext(notificationContext);
}
