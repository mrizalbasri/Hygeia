import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import { api, parseJSON, type Entry } from './api.js';

type Listener = { refresh: () => void; status?: (message: string) => void };
const listeners = new Set<Listener>();
let stop: (() => void) | undefined;
let message = 'Menghubungkan pembaruan…';

// One authenticated stream per browser tab, shared by lists, details and summaries.
export function subscribeRealtime(refresh: () => void, status?: Listener['status']) {
	const listener = { refresh, status };
	listeners.add(listener);
	status?.(message);
	if (!stop) stop = connect();
	return () => {
		listeners.delete(listener);
		if (!listeners.size) {
			stop?.();
			stop = undefined;
		}
	};
}

function connect() {
	let source: EventSource | null = null;
	let disposed = false;
	let retry: ReturnType<typeof setTimeout> | undefined;
	let batch: ReturnType<typeof setTimeout> | undefined;
	const controller = new AbortController();
	const announce = (value: string) => {
		message = value;
		for (const listener of listeners) listener.status?.(value);
	};
	const refresh = () => {
		if (disposed || batch) return;
		batch = setTimeout(() => {
			batch = undefined;
			if (!disposed) for (const listener of listeners) listener.refresh();
		}, 150);
	};
	async function open() {
		try {
			const data = await api<{ notifications: Entry[] }>(
				'/notifications?limit=1',
				'GET',
				undefined,
				controller.signal
			);
			if (disposed) return;
			const last = data.notifications?.[0]?.id || '0';
			source = new EventSource(
				`/api/v1/notifications/stream?after_id=${encodeURIComponent(String(last))}`
			);
			source.onopen = () => {
				announce('Pembaruan langsung aktif');
				// Includes changes during cursor bootstrap and disconnected periods.
				refresh();
			};
			source.onerror = () =>
				announce(
					'Koneksi terputus; mencoba menyambungkan kembali. Daftar juga diperbarui berkala.'
				);
			source.addEventListener('notification', (event) => {
				try {
					parseJSON((event as MessageEvent).data);
					refresh();
				} catch {
					announce('Pembaruan tidak dapat dibaca. Gunakan muat ulang daftar.');
				}
			});
			source.addEventListener('unavailable', () =>
				announce('Layanan pembaruan terganggu; mencoba menyambungkan kembali.')
			);
			source.addEventListener('session-ended', () => {
				dispose();
				void goto(resolve('/login'), { replace: true, refreshAll: true });
			});
		} catch {
			if (disposed) return;
			announce('Pembaruan langsung belum tersedia. Daftar diperbarui berkala.');
			retry = setTimeout(() => void open(), 5000);
		}
	}
	const foreground = () => {
		if (document.visibilityState === 'visible') refresh();
	};
	document.addEventListener('visibilitychange', foreground);
	window.addEventListener('online', refresh);
	const fallback = setInterval(() => {
		if (document.visibilityState === 'visible') refresh();
	}, 30000);
	function dispose() {
		disposed = true;
		controller.abort();
		source?.close();
		clearTimeout(retry);
		clearTimeout(batch);
		clearInterval(fallback);
		document.removeEventListener('visibilitychange', foreground);
		window.removeEventListener('online', refresh);
	}
	void open();
	return dispose;
}
