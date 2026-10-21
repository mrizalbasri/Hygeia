<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import HealthIcon from '#lib/HealthIcon.svelte';
	import HeroPhone from '#lib/HeroPhone.svelte';
	import './landing.css';
	let selectedRole = $state(0);
	let ready = $state(false);
	onMount(() => {
		ready = true;
	});
	const audiences = [
		{
			label: 'Pasien',
			title: 'Pahami apa yang sedang berjalan.',
			description:
				'Lihat hubungan perawatan, status pemeriksaan, resep, dan riwayat kejadian dari ruang pasien Anda.',
			items: [
				'Ikuti permintaan pemeriksaan dan statusnya di laboratorium.',
				'Baca nama obat, dosis, serta petunjuk yang tercantum pada resep.',
				'Lihat perkembangan perawatan dalam timeline yang tersusun.'
			],
			access:
				'Pasien melihat data perawatannya sendiri. Berbagi dengan penyedia lain mengikuti persetujuan dan cakupan akses yang berlaku.'
		},
		{
			label: 'Dokter',
			title: 'Lanjutkan perawatan dengan konteks yang jelas.',
			description:
				'Alur dokter menghubungkan pasien, permintaan pemeriksaan, hasil, dan resep melalui episode perawatan.',
			items: [
				'Mulai hubungan perawatan melalui undangan kepada pasien.',
				'Ajukan pemeriksaan dan baca hasil sesuai hak akses.',
				'Terbitkan resep berisi obat, dosis, jumlah, dan aturan pakai.'
			],
			access:
				'Hubungan dalam satu klinik tidak otomatis memberikan akses ke seluruh pasien. Akses klinis mengikuti episode aktif atau izin lain yang sah.'
		},
		{
			label: 'Laboratorium',
			title: 'Dari permintaan menjadi hasil pemeriksaan.',
			description:
				'Laboratorium menerima penugasan pemeriksaan setelah pasien menyetujui pengiriman permintaan tersebut.',
			items: [
				'Terima permintaan yang ditujukan ke laboratorium Anda.',
				'Perbarui proses dari diterima menjadi sedang diperiksa.',
				'Kirim hasil terstruktur untuk perjalanan perawatan terkait.'
			],
			access:
				'Penugasan lab berfokus pada pemeriksaan yang disetujui. Penugasan tersebut tidak membuka seluruh riwayat klinis pasien.'
		},
		{
			label: 'Apotek',
			title: 'Siapkan resep yang ditujukan kepada Anda.',
			description:
				'Pasien memilih apotek dan menyetujui pengiriman resep. Apotek tujuan melanjutkan proses penyiapan.',
			items: [
				'Terima resep yang sudah dibagikan ke apotek Anda.',
				'Ikuti tahap diterima, disiapkan, siap diambil, hingga selesai.',
				'Gunakan detail obat dan aturan pakai yang diterbitkan dokter.'
			],
			access:
				'Akses apotek dibatasi pada resep yang ditugaskan selama prosesnya aktif. Data pemeriksaan lain tidak ikut dibuka.'
		}
	];
	let audience = $derived(audiences[selectedRole]);
	const questions = [
		{
			question: 'Apakah semua dokter bisa melihat data saya?',
			answer:
				'Tidak. Dokter memerlukan hubungan perawatan aktif atau izin lain yang sah untuk data terkait. Dokter lain, termasuk rekan satu klinik, tidak otomatis mendapat akses.'
		},
		{
			question: 'Apakah riwayat perawatan sama dengan riwayat akses?',
			answer:
				'Berbeda. Timeline perawatan mencatat undangan, proses lab, resep, dan persetujuan. Riwayat akses mencatat aktivitas melihat data. Keduanya tersedia di menu terpisah pada ruang pasien.'
		},
		{
			question: 'Bagaimana resep sampai ke apotek?',
			answer:
				'Dokter menerbitkan resep, lalu pasien memilih apotek dan menyetujui pengirimannya dari menu Resep & apotek. Pasien bisa mengikuti status penyiapannya. Antarmuka petugas apotek masih dalam pengembangan.'
		},
		{
			question: 'Apakah Hygeia menentukan obat atau diagnosis?',
			answer:
				'Tidak. Informasi obat berasal dari resep dokter. Hygeia menghubungkan informasi perawatan dan menampilkan perkembangannya; aplikasi tidak membuat keputusan diagnosis atau resep otomatis.'
		},
		{
			question: 'Apa yang sudah bisa saya coba sekarang?',
			answer:
				'Dengan akun demo dari pengelola, pasien dapat menyetujui perawatan dan pemeriksaan, memilih apotek, mengatur izin, serta membagikan CarePass. Dokter dapat mengundang pasien, meminta pemeriksaan, menerbitkan resep, dan membaca data sesuai izin. Data demo bersifat sintetis.'
		}
	];
	const features = [
		{
			icon: 'care' as const,
			title: 'Perawatan yang terhubung',
			text: 'Ikuti hubungan perawatan dengan dokter dan perkembangan setiap perjalanan Anda.'
		},
		{
			icon: 'lab' as const,
			title: 'Pemeriksaan lebih mudah diikuti',
			text: 'Lihat permintaan pemeriksaan dan status prosesnya di laboratorium.'
		},
		{
			icon: 'medicine' as const,
			title: 'Resep dalam satu tempat',
			text: 'Baca obat yang diresepkan, aturan pakai, dan status penyiapan oleh apotek.'
		},
		{
			icon: 'shield' as const,
			title: 'Akses dengan persetujuan',
			text: 'Akses data mengikuti izin pasien dan hubungan perawatan yang berlaku.'
		}
	];
</script>

<svelte:head>
	<link rel="preload" as="image" href="/hero-phone-front.png" />
	<title>Hygeia · Perawatan terhubung, kendali di tangan Anda</title>
	<meta
		name="description"
		content="Ikuti perjalanan perawatan, pemeriksaan, dan resep Anda bersama Hygeia. Platform pertukaran data kesehatan dengan persetujuan pasien."
	/>
</svelte:head>

<div class="landing">
	<a class="landing-skip" href="#utama">Lewati ke konten</a>
	<header class="landing-header">
		<a class="brand" href={resolve('/')} aria-label="Hygeia, beranda"
			><img src="/logo-light.png" width="42" height="42" alt="" /><span>Hygeia</span></a
		>
		<nav aria-label="Navigasi beranda">
			<a href="#fitur">Fitur</a><a href="#cara-kerja">Cara kerja</a><a href="#tentang">Tentang</a>
		</nav>
		<a class="button primary" href={resolve('/login')}>Masuk ke Hygeia</a>
	</header>
	<main id="utama">
		<section class="landing-hero" aria-labelledby="landing-title">
			<div class="hero-copy">
				<p class="landing-kicker">Pasien · Dokter · Laboratorium · Apotek</p>
				<h1 id="landing-title">
					Perawatan<br />terhubung.<br /><span>Kendali di tangan Anda.</span>
				</h1>
				<p class="hero-description">
					Ikuti pemeriksaan dari dokter sampai laboratorium, dan resep sampai apotek. Lihat
					perkembangannya dari ruang pasien Anda.
				</p>
				<div class="hero-actions">
					<a class="button primary" href={resolve('/login')}>Masuk ke ruang Anda</a><a
						class="landing-text-link"
						href="#cara-kerja">Kenali cara kerjanya</a
					>
				</div>
				<p class="hero-note">Prototipe Hygeia · Lingkungan demo dengan data sintetis</p>
			</div>
			<div class="hero-product" aria-label="Ilustrasi dashboard pasien">
				<HeroPhone />
			</div>
		</section>

		<section class="landing-features" id="fitur" aria-labelledby="features-title">
			<div class="features-intro">
				<h2 id="features-title">Dari pemeriksaan<br />sampai resep.</h2>
				<p>
					Permintaan lab sedang diproses? Resep sudah siap diambil? Dashboard memperlihatkan status
					dari layanan yang menangani Anda.
				</p>
			</div>
			<div class="feature-list">
				{#each features as feature (feature.title)}<article>
						<span class="feature-icon"><HealthIcon name={feature.icon} /></span>
						<div>
							<h3>{feature.title}</h3>
							<p>{feature.text}</p>
						</div>
					</article>{/each}
			</div>
		</section>

		<section class="landing-audiences" id="untuk-siapa" aria-labelledby="audiences-title">
			<div class="audiences-heading">
				<h2 id="audiences-title">Empat peran.<br />Satu perjalanan perawatan.</h2>
				<p>
					Kebutuhan setiap pihak berbeda. Informasi yang dibagikan mengikuti peran dan hubungan
					perawatannya.
				</p>
			</div>
			<div class="audience-explorer">
				<div class="audience-options" aria-label="Pilih peran untuk mengetahui alurnya">
					{#each audiences as item, index (item.label)}<button
							disabled={!ready}
							aria-pressed={selectedRole === index}
							onclick={() => (selectedRole = index)}
							>{item.label}<span aria-hidden="true">→</span></button
						>{/each}
				</div>
				<div class="audience-detail" aria-live="polite">
					<h3>{audience.title}</h3>
					<p>{audience.description}</p>
					<ul>
						{#each audience.items as item (item)}<li>{item}</li>{/each}
					</ul>
					<div class="audience-access">
						<HealthIcon name="shield" />
						<p>{audience.access}</p>
					</div>
				</div>
			</div>
			<p class="audience-implementation-note">
				Ruang pasien dan dokter sudah terhubung ke API. Antarmuka petugas laboratorium dan apotek
				masih dalam pengembangan.
			</p>
		</section>

		<section class="landing-flow" id="cara-kerja" aria-labelledby="flow-title">
			<div class="flow-heading">
				<h2 id="flow-title">Perawatan dimulai<br />dengan persetujuan Anda.</h2>
				<p>
					Hubungan dengan dokter menjadi dasar perjalanan perawatan. Akses data mengikuti izin yang
					berlaku.
				</p>
			</div>
			<ol class="flow-steps">
				<li>
					<h3>Dokter mengundang</h3>
					<p>Dokter mengirim undangan untuk memulai hubungan perawatan dengan pasien.</p>
				</li>
				<li>
					<h3>Pasien menyetujui</h3>
					<p>Pasien meninjau undangan dan menentukan apakah perawatan dapat dimulai.</p>
				</li>
				<li>
					<h3>Pemeriksaan dikirim ke lab</h3>
					<p>
						Dokter mengajukan pemeriksaan. Setelah pasien menyetujui pengiriman, laboratorium tujuan
						menerima permintaan dan memprosesnya.
					</p>
				</li>
				<li>
					<h3>Hasil menjadi bagian dari perawatan</h3>
					<p>
						Laboratorium mengirim hasil terstruktur. Dokter dapat meninjaunya sesuai hak akses untuk
						melanjutkan penilaian klinis.
					</p>
				</li>
				<li>
					<h3>Resep dilanjutkan ke apotek pilihan</h3>
					<p>
						Jika dokter menerbitkan resep, pasien memilih apotek dan menyetujui pengiriman. Status
						penyiapan dapat diikuti sampai resep selesai.
					</p>
				</li>
			</ol>
		</section>

		<section class="landing-permission" id="izin-data" aria-labelledby="permission-title">
			<div class="permission-intro">
				<HealthIcon name="shield" />
				<h2 id="permission-title">Berbagi seperlunya.<br />Dengan tujuan yang jelas.</h2>
				<p>
					Persetujuan bukan izin untuk membuka semua data. Penerima, jenis data, dan masa akses
					mengikuti hubungan perawatan atau izin yang diberikan.
				</p>
			</div>
			<dl class="permission-definitions">
				<div>
					<dt>Siapa yang menerima?</dt>
					<dd>
						Dokter yang memiliki hubungan perawatan, atau penyedia yang dituju pada permintaan dan
						resep.
					</dd>
				</div>
				<div>
					<dt>Data apa yang dibagikan?</dt>
					<dd>
						Cakupannya mengikuti kebutuhan. Penugasan laboratorium dan apotek tidak memberi akses ke
						seluruh riwayat pasien.
					</dd>
				</div>
				<div>
					<dt>Berapa lama akses berlaku?</dt>
					<dd>
						Episode, penugasan penyedia, dan izin berbatas waktu memiliki aturan masa akses
						masing-masing.
					</dd>
				</div>
				<div>
					<dt>Apa yang dapat ditelusuri?</dt>
					<dd>
						Backend mencatat akses terlindungi. Timeline perawatan dan riwayat akses merupakan dua
						catatan berbeda.
					</dd>
				</div>
			</dl>
		</section>

		<section class="landing-faq" id="pertanyaan" aria-labelledby="faq-title">
			<div class="faq-intro">
				<h2 id="faq-title">Kenali Hygeia<br />sebelum mulai.</h2>
				<p>Tentang akses data, alur resep, dan apa yang tersedia di versi demo.</p>
			</div>
			<div class="faq-list">
				{#each questions as item (item.question)}<details>
						<summary>{item.question}<span aria-hidden="true">+</span></summary>
						<p>{item.answer}</p>
					</details>{/each}
			</div>
		</section>

		<section class="landing-about" id="tentang" aria-labelledby="about-title">
			<div>
				<h2 id="about-title">Tentang Hygeia</h2>
				<p>
					Hygeia adalah prototipe pertukaran data kesehatan antara pasien, dokter, laboratorium, dan
					apotek, dengan persetujuan pasien dan riwayat akses yang dapat diperiksa.
				</p>
				<p class="about-demo">
					Saat ini tersedia sebagai lingkungan demo dengan data sintetis. Gunakan akun yang
					diberikan pengelola untuk mencoba ruang Anda.
				</p>
			</div>
			<a class="button primary" href={resolve('/login')}>Masuk ke Hygeia</a>
		</section>
	</main>
	<footer class="landing-footer">
		<a class="brand" href={resolve('/')}
			><img src="/logo-light.png" width="32" height="32" alt="" /><span>Hygeia</span></a
		>
		<p>Perawatan terhubung. Kendali di tangan Anda.</p>
		<nav aria-label="Informasi tambahan">
			<a href="#untuk-siapa">Untuk siapa</a><a href="#izin-data">Izin data</a><a href="#pertanyaan"
				>Pertanyaan</a
			><a href={resolve('/login')}>Masuk</a>
		</nav>
	</footer>
</div>
