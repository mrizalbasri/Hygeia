import type { Clerk } from '@clerk/clerk-js';

let loading: Promise<Clerk> | undefined;
export function loadClerk(publishableKey: string) {
	if (!loading)
		loading = (async () => {
			const [{ Clerk }, { ui }, { idID }] = await Promise.all([
				import('@clerk/clerk-js'),
				import('@clerk/ui'),
				import('@clerk/localizations/id-ID')
			]);
			const client = new Clerk(publishableKey);
			await client.load({
				ui,
				localization: {
					...idID,
					formFieldInputPlaceholder__emailAddress: 'nama@email.com',
					formFieldInputPlaceholder__password: 'Masukkan kata sandi Anda',
					formFieldInputPlaceholder__signUpPassword: 'Buat kata sandi Anda',
					signIn: {
						...idID.signIn,
						start: {
							...idID.signIn?.start,
							title: 'Masuk ke ruang Anda',
							subtitle: 'Lanjutkan perjalanan perawatan dengan akun Hygeia Anda.'
						}
					},
					signUp: {
						...idID.signUp,
						start: {
							...idID.signUp?.start,
							title: 'Daftar sebagai pasien',
							subtitle: 'Satu akun untuk mengikuti perawatan dan mengatur akses data Anda.'
						}
					}
				}
			});
			return client;
		})().catch((error) => {
			loading = undefined;
			throw error;
		});
	return loading;
}

export async function authConfig() {
	const response = await fetch('/api/v1/auth/config', { cache: 'no-store' });
	if (!response.ok) throw new Error('Layanan autentikasi belum tersedia.');
	return (await response.json()) as { provider?: 'local' | 'clerk'; publishable_key?: string };
}
