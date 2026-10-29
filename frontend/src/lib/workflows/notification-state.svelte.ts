import { getContext, setContext } from 'svelte';
import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { api, parseJSON, type Entry } from './api.js';

const notificationContext = Symbol('dashboard-notifications');

// One instance per dashboard, never a module-level singleton shared by SSR users.
export class NotificationState {
	unread = $state<number | null>(null);
	revision = $state(0);
	connection = $state('Menghubungkan pembaruan…');
	failed = $state(false);
	private source: EventSource | null = null;
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
		void api<{ notifications: Entry[] }>(
			'/notifications?limit=1',
			'GET',
			undefined,
			controller.signal
		)
			.then((data) => {
				if (controller.signal.aborted) return;
				const last = data.notifications[0]?.id || '0';
				const source = new EventSource(
					`/api/v1/notifications/stream?after_id=${encodeURIComponent(String(last))}`
				);
				this.source = source;
				const invalidate = () => {
					if (controller.signal.aborted) return;
					this.revision++;
					void this.refreshUnread();
				};
				source.onopen = () => {
					if (controller.signal.aborted) return;
					this.failed = false;
					this.connection = 'Pembaruan langsung aktif';
					// Also catch up after reconnecting: events may have arrived offline.
					invalidate();
				};
				source.onerror = () => {
					if (controller.signal.aborted) return;
					this.failed = true;
					this.connection =
						'Koneksi terputus; mencoba menyambungkan kembali. Anda dapat mencoba lagi.';
				};
				source.addEventListener('notification', (event) => {
					if (controller.signal.aborted) return;
					try {
						parseJSON((event as MessageEvent).data);
						invalidate();
					} catch {
						this.failed = true;
						this.connection = 'Pembaruan tidak dapat dibaca. Coba sambungkan kembali.';
					}
				});
				source.addEventListener('unavailable', () => {
					if (controller.signal.aborted) return;
					this.failed = true;
					this.connection = 'Layanan pembaruan terganggu; mencoba menyambungkan kembali.';
				});
				source.addEventListener('session-ended', () => {
					if (controller.signal.aborted) return;
					this.disconnect();
					void goto(resolve('/login'), { replace: true, refreshAll: true });
				});
			})
			.catch(() => {
				if (controller.signal.aborted) return;
				this.failed = true;
				this.connection = 'Pembaruan langsung belum tersedia. Coba sambungkan kembali.';
			});
		return () => {
			// An old dashboard cleanup must not close a newer connection.
			if (this.controller === controller) this.disconnect();
		};
	}

	disconnect() {
		this.controller?.abort();
		this.controller = null;
		this.source?.close();
		this.source = null;
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
